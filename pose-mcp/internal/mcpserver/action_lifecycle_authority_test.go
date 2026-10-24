package mcpserver

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/harne8/pose-mcp/internal/pose"
)

func TestActionLifecycleMCPProofOnlyAndBoundIssuerControl(t *testing.T) {
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
	write(".pose/specs/storage.md", "---\nslug: storage\nstatus: in-progress\n---\n## 2. Requirements\n- R1: Preserve data.\n")
	write(".pose/policy/actions.json", `{"schema_version":2,"roles":{"maintainer":["human:maintainer"]}}`)
	store := pose.Store{Root: root}
	now := time.Now().UTC()
	v, e := store.OpenActionRequest(pose.ActionRequest{Origin: "spec:storage", Kind: pose.ActionInput, Question: "Material decision?", RequestedBy: pose.ActionPrincipal{Principal: "agent:impl"}, Recipient: pose.ObligationActor{Role: "maintainer"}, Targets: []pose.NodeRef{{Artifact: "self"}}, Effects: []pose.ObligationEffect{{Phase: pose.PhaseExecution, Mode: pose.EffectBlock}}}, now)
	if e != nil {
		t.Fatal(e)
	}
	ts := httptest.NewServer(New(store).Handler("", ""))
	t.Cleanup(ts.Close)
	call := func(name string, args map[string]any) map[string]any {
		t.Helper()
		raw, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": map[string]any{"name": name, "arguments": args}})
		_, out := post(t, ts, string(raw))
		return out.Result
	}
	args := map[string]any{"id": v.Request.ID, "operation": "waived", "actor": "human:maintainer", "request_digest": v.Request.RequestDigest, "expected_revision": 1, "idempotency_key": "w1", "reason": "reviewed dispensation"}
	if out := call("pose_action_resolve", args); out["isError"] == true {
		t.Fatalf("preview: %+v", out)
	}
	args["apply"] = true
	if out := call("pose_action_resolve", args); out["isError"] != true {
		t.Fatalf("MCP label waived without proof: %+v", out)
	}
	private := ed25519.NewKeyFromSeed(make([]byte, ed25519.SeedSize))
	public := private.Public().(ed25519.PublicKey)
	sum := sha256.Sum256(public)
	pin := "fixture:lifecycle#sha256:" + hex.EncodeToString(sum[:])
	write(".pose/policy/review.json", `{"schema_version":2,"enabled":true,"profiles":{"spec":"spec-closeout@1"},"authority_project":"proj.fixture","authority_audience":"fixture-verifier","trusted_attestation_issuers":["`+pin+`"],"human_authority_issuers":["`+pin+`"]}`)
	claim := pose.ActionAuthorityClaim{SchemaVersion: 2, Project: "proj.fixture", Audience: "fixture-verifier", RequestID: v.Request.ID, RequestDigest: v.Request.RequestDigest, Principal: "human:maintainer", Role: "human", Operation: pose.ActionEventWaived, ExpectedRevision: 1, IdempotencyKey: "w1", Reason: "reviewed dispensation", Issuer: "fixture:lifecycle", IssuedAt: now.Add(-time.Minute).Format(time.RFC3339Nano), ExpiresAt: now.Add(time.Hour).Format(time.RFC3339Nano)}
	raw, _ := json.Marshal(claim)
	args["claim"] = claim
	args["envelope"] = pose.ActionClaimEnvelope{Issuer: claim.Issuer, Algorithm: "ed25519", PublicKey: base64.StdEncoding.EncodeToString(public), Signature: base64.StdEncoding.EncodeToString(ed25519.Sign(private, raw))}
	if out := call("pose_action_resolve", args); out["isError"] == true {
		t.Fatalf("valid issuer waiver: %+v", out)
	}
	got, e := store.LoadActionRequest(v.Request.ID)
	if e != nil || got.Assurance != pose.ReviewIdentityAssuranceVerified || got.Satisfaction != pose.SatisfactionWaived {
		t.Fatalf("verified waiver not projected: %+v %v", got, e)
	}
	// Roundtrip a prepared replacement across an arbitrarily long signing interval.
	nextArgs := map[string]any{"origin": "spec:storage", "kind": "input", "question": "Replacement material question?", "requested_by": "agent:impl", "recipient_role": "maintainer", "targets": []string{"self"}, "effects": []string{"execution:block"}, "supersedes": v.Request.ID, "requested_at": "2026-10-23T01:02:03Z"}
	preview := call("pose_action_open", nextArgs)
	if preview["isError"] == true {
		t.Fatalf("replacement preview: %+v", preview)
	}
	sc, ok := preview["structuredContent"].(map[string]any)
	if !ok {
		t.Fatalf("missing structured preview: %+v", preview)
	}
	rawNext, err := json.Marshal(sc["request"])
	if err != nil {
		t.Fatal(err)
	}
	var prepared pose.ActionRequest
	if err := json.Unmarshal(rawNext, &prepared); err != nil {
		t.Fatal(err)
	}
	replacement := claim
	replacement.Operation = pose.ActionEventSuperseded
	replacement.Principal = "agent:impl"
	replacement.Role = "agent"
	replacement.ExpectedRevision = got.Revision
	replacement.IdempotencyKey = "replacement"
	replacement.Reason = "revised question"
	replacement.SuccessorID = prepared.ID
	replacement.SuccessorDigest = prepared.RequestDigest
	signed, _ := json.Marshal(replacement)
	envelope := pose.ActionClaimEnvelope{Issuer: claim.Issuer, Algorithm: "ed25519", PublicKey: base64.StdEncoding.EncodeToString(public), Signature: base64.StdEncoding.EncodeToString(ed25519.Sign(private, signed))}
	nextArgs["authority"] = pose.ActionResolution{Actor: replacement.Principal, RequestDigest: v.Request.RequestDigest, ExpectedRevision: got.Revision, IdempotencyKey: replacement.IdempotencyKey, Reason: replacement.Reason, SuccessorID: prepared.ID, SuccessorDigest: prepared.RequestDigest, Claim: &replacement, Envelope: &envelope}
	nextArgs["apply"] = true
	if out := call("pose_action_open", nextArgs); out["isError"] == true {
		t.Fatalf("signed replacement apply: %+v", out)
	}
	old, err := store.LoadActionRequest(v.Request.ID)
	if err != nil || old.State != pose.ActionStateSuperseded || old.Events[len(old.Events)-1].SupersededBy != prepared.ID {
		t.Fatalf("replacement lineage: %+v %v", old, err)
	}
	args["operation"] = "invalidated"
	if out := call("pose_action_resolve", args); out["isError"] != true {
		t.Fatalf("actor invalidation: %+v", out)
	}
}
