package mcpserver

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/harne8/pose-mcp/internal/pose"
)

type recoveryMCPFault struct {
	pose.Files
	path  string
	fired bool
}

func (f *recoveryMCPFault) CreateExclusive(name string, b []byte, p os.FileMode, d bool) error {
	if err := f.Files.CreateExclusive(name, b, p, d); err != nil {
		return err
	}
	if name == f.path && !f.fired {
		f.fired = true
		return errors.New("synthetic successor Sync failure")
	}
	return nil
}
func TestActionRecoveryMCPPreflightAndSignedResume(t *testing.T) {
	root := t.TempDir()
	write := func(path, body string) {
		t.Helper()
		p := filepath.Join(root, path)
		if e := os.MkdirAll(filepath.Dir(p), 0755); e != nil {
			t.Fatal(e)
		}
		if e := os.WriteFile(p, []byte(body), 0644); e != nil {
			t.Fatal(e)
		}
	}
	t.Setenv("POSE_DEFAULT_PROJECT_ID", "proj.recovery-fixture")
	write(".pose/project.json", `{"schema_version":1,"project_id":"proj.recovery-fixture"}`)
	write(".pose/specs/storage.md", "---\nslug: storage\nstatus: in-progress\n---\n## 2. Requirements\n- R1: Material input.\n")
	write(".pose/policy/actions.json", `{"schema_version":2,"identity_assurance":"verified","roles":{"maintainer":["human:maintainer"]}}`)
	private := ed25519.NewKeyFromSeed(make([]byte, ed25519.SeedSize))
	public := private.Public().(ed25519.PublicKey)
	sum := sha256.Sum256(public)
	pin := "fixture:recovery#sha256:" + hex.EncodeToString(sum[:])
	write(".pose/policy/review.json", `{"schema_version":2,"enabled":true,"profiles":{"spec":"spec-closeout@1"},"authority_project":"proj.recovery-fixture","authority_audience":"fixture-verifier","trusted_attestation_issuers":["`+pin+`"],"human_authority_issuers":["`+pin+`"]}`)
	s := pose.Store{Root: root}
	now := time.Now().UTC()
	r := pose.ActionRequest{Origin: "spec:storage", Kind: pose.ActionInput, Question: "Material input?", RequestedBy: pose.ActionPrincipal{Principal: "agent:impl"}, Recipient: pose.ObligationActor{Role: "maintainer"}, Targets: []pose.NodeRef{{Artifact: "self"}}, Effects: []pose.ObligationEffect{{Phase: pose.PhaseExecution, Mode: pose.EffectBlock}}}
	first, err := s.OpenActionRequest(r, now)
	if err != nil {
		t.Fatal(err)
	}
	next := r
	next.Question = "Materially revised input?"
	next.Supersedes = first.Request.ID
	next.RequestedAt = "2026-10-24T12:00:00Z"
	prepared, err := s.PrepareActionRequest(next, now)
	if err != nil {
		t.Fatal(err)
	}
	fault := &recoveryMCPFault{Files: pose.NativeFiles(), path: filepath.Join(root, ".pose/actions", prepared.ID+".jsonl")}
	serverStore := s
	serverStore.FS = fault
	ts := httptest.NewServer(New(s).Handler("", ""))
	t.Cleanup(ts.Close)
	call := func(name string, args map[string]any) map[string]any {
		t.Helper()
		raw, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": map[string]any{"name": name, "arguments": args}})
		_, out := post(t, ts, string(raw))
		return out.Result
	}
	previewArgs := map[string]any{"id": first.Request.ID, "actor": "agent:intruder", "answer": "input", "idempotency_key": "bad"}
	if out := call("pose_action_resolve", previewArgs); out["isError"] != true {
		t.Fatalf("invalid actor preview accepted: %+v", out)
	}
	previewArgs["actor"] = "human:maintainer"
	out := call("pose_action_resolve", previewArgs)
	sc, ok := out["structuredContent"].(map[string]any)
	if out["isError"] == true || !ok || sc["proof_pending"] != true || sc["acceptance"] != "not-certified" {
		t.Fatalf("unsigned preview certified proof: %+v", out)
	}
	claim := pose.ActionAuthorityClaim{SchemaVersion: 2, Project: "proj.recovery-fixture", Audience: "fixture-verifier", RequestID: first.Request.ID, RequestDigest: first.Request.RequestDigest, Principal: "agent:impl", Role: "agent", Operation: pose.ActionEventSuperseded, ExpectedRevision: 1, IdempotencyKey: "replacement", Reason: "changed material question", SuccessorID: prepared.ID, SuccessorDigest: prepared.RequestDigest, Issuer: "fixture:recovery", IssuedAt: now.Add(-time.Minute).Format(time.RFC3339Nano), ExpiresAt: now.Add(time.Hour).Format(time.RFC3339Nano)}
	sign := func(c pose.ActionAuthorityClaim) pose.ActionClaimEnvelope {
		raw, _ := json.Marshal(c)
		return pose.ActionClaimEnvelope{Issuer: c.Issuer, Algorithm: "ed25519", PublicKey: base64.StdEncoding.EncodeToString(public), Signature: base64.StdEncoding.EncodeToString(ed25519.Sign(private, raw))}
	}
	envelope := sign(claim)
	authority := pose.ActionResolution{Actor: claim.Principal, RequestDigest: first.Request.RequestDigest, ExpectedRevision: 1, IdempotencyKey: claim.IdempotencyKey, Reason: claim.Reason, SuccessorID: prepared.ID, SuccessorDigest: prepared.RequestDigest, Claim: &claim, Envelope: &envelope}
	args := map[string]any{"origin": "spec:storage", "kind": "input", "question": next.Question, "requested_by": "agent:impl", "recipient_role": "maintainer", "targets": []string{"self"}, "effects": []string{"execution:block"}, "supersedes": first.Request.ID, "requested_at": next.RequestedAt, "authority": authority, "apply": true}
	authority.Channel = "mcp"
	if _, err := serverStore.OpenActionRequestAuthorized(prepared, &authority, now); !errors.Is(err, pose.ErrActionRecoveryPending) || !fault.fired {
		t.Fatalf("partial domain operation not exercised: %v fired=%v", err, fault.fired)
	}

	partial, err := s.LoadActionRequest(first.Request.ID)
	if err != nil || partial.Recovery == nil || partial.Recovery.Phase != "successor-opened" {
		t.Fatalf("partial state hidden: %+v %v", partial, err)
	}
	for n := 0; n < 2; n++ {
		if out := call("pose_action_open", args); out["isError"] == true {
			t.Fatalf("signed retry %d: %+v", n, out)
		}
	}
	prior, err := s.LoadActionRequest(first.Request.ID)
	if err != nil || prior.Revision != 2 || prior.Recovery.Phase != "complete" {
		t.Fatalf("retirement diverged: %+v %v", prior, err)
	}
	child, err := s.LoadActionRequest(prepared.ID)
	if err != nil {
		t.Fatal(err)
	}
	answerClaim := pose.ActionAuthorityClaim{SchemaVersion: 1, Project: claim.Project, Audience: claim.Audience, RequestID: child.Request.ID, RequestDigest: child.Request.RequestDigest, Principal: "human:maintainer", Role: "human", Answer: "admitted input", Issuer: claim.Issuer, IssuedAt: claim.IssuedAt, ExpiresAt: claim.ExpiresAt}
	resolve := map[string]any{"id": child.Request.ID, "actor": "human:maintainer", "answer": answerClaim.Answer, "request_digest": child.Request.RequestDigest, "expected_revision": 1, "idempotency_key": "answer", "claim": answerClaim, "envelope": sign(answerClaim), "apply": true}
	if out := call("pose_action_resolve", resolve); out["isError"] == true {
		t.Fatalf("positive answer broken after recovery: %+v", out)
	}
	done, err := s.LoadActionRequest(child.Request.ID)
	if err != nil || done.Satisfaction != pose.SatisfactionSatisfied || done.NextStep != nil {
		t.Fatalf("satisfaction repeated: %+v %v", done, err)
	}
}
