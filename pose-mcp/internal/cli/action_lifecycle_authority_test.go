package cli

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"

	pose "github.com/harne8/pose-mcp/internal/pose"
)

func TestActionLifecycleCLIRefusesLabelsAndPrintsOperationStatement(t *testing.T) {
	root := actionCLIFixture(t)
	code, out := runPose(t, root, append(append([]string{}, openArgs...), "--apply", "--json")...)
	if code != 0 {
		t.Fatal(out)
	}
	var v pose.ActionRequestView
	if err := json.Unmarshal([]byte(out), &v); err != nil {
		t.Fatal(err)
	}
	mustWrite(t, filepath.Join(root, ".pose/policy/actions.json"), `{"schema_version":2,"identity_assurance":"verified","roles":{"maintainer":["human:maintainer"]}}`)
	args := []string{"action", "waive", v.Request.ID, "--actor", "human:maintainer", "--request-digest", v.Request.RequestDigest, "--expected-revision", "1", "--idempotency-key", "w1", "--reason", "explicit risk", "--json"}
	if code, out := runPose(t, root, args...); code != 0 || !strings.Contains(out, pose.ActionLifecycleNamespace) {
		t.Fatalf("preview statement: %d %s", code, out)
	}
	if code, out := runPose(t, root, append(args, "--apply")...); code == 0 || !strings.Contains(out, "proof") {
		t.Fatalf("unsigned verified waiver: %d %s", code, out)
	}
	store := pose.Store{Root: root}
	if got, err := store.LoadActionRequest(v.Request.ID); err != nil || got.Revision != 1 {
		t.Fatalf("refusal changed request: %+v %v", got, err)
	}
	code, out = runPose(t, root, "action", "statement", v.Request.ID, "--actor", "human:maintainer", "--operation", "waived", "--expected-revision", "1", "--idempotency-key", "w1", "--reason", "explicit risk")
	if code != 0 || !strings.Contains(out, `"operation":"waived"`) || !strings.Contains(out, pose.ActionLifecycleNamespace) {
		t.Fatalf("statement: %d %s", code, out)
	}
	// Declared local behavior remains usable when verified is not adopted.
	mustWrite(t, filepath.Join(root, ".pose/policy/actions.json"), `{"schema_version":2,"roles":{"maintainer":["human:maintainer"]}}`)
	if code, out := runPose(t, root, append(args, "--apply")...); code != 0 || !strings.Contains(out, `"state": "waived"`) {
		t.Fatalf("declared waiver: %d %s", code, out)
	}
	// A separate question cannot be retired by changing its origin/requester.
	next := v.Request
	next.ID = ""
	next.RequestedAt = ""
	next.RequestDigest = ""
	next.Question = "Different question"
	next.Supersedes = v.Request.ID
	next.RequestedBy.Principal = "agent:intruder"
	if _, err := store.OpenActionRequest(next, time.Now()); err == nil {
		t.Fatal("third-party replacement authorized")
	}
}

func TestActionLifecycleCLISupersessionPreservesPreviewIdentity(t *testing.T) {
	root := actionCLIFixture(t)
	code, out := runPose(t, root, append(append([]string{}, openArgs...), "--apply", "--json")...)
	if code != 0 {
		t.Fatal(out)
	}
	var first pose.ActionRequestView
	if err := json.Unmarshal([]byte(out), &first); err != nil {
		t.Fatal(err)
	}
	args := append([]string{}, openArgs...)
	for i := range args {
		if args[i] == "Keep reading schema v1?" {
			args[i] = "Replacement after external signing?"
		}
	}
	// A stable earlier timestamp makes loss of preview identity deterministic,
	// without a wall-clock sleep in the test.
	args = append(args, "--supersedes", first.Request.ID, "--requested-at", "2026-10-23T01:02:03Z", "--json")
	code, out = runPose(t, root, args...)
	if code != 0 {
		t.Fatal(out)
	}
	var prepared pose.ActionRequest
	if err := json.Unmarshal([]byte(out), &prepared); err != nil {
		t.Fatal(err)
	}
	// Declared policy still requires predecessor authority and exact binding.
	auth := pose.ActionResolution{Actor: "agent:impl", RequestDigest: first.Request.RequestDigest, ExpectedRevision: 1, IdempotencyKey: "replacement", Reason: "revised material question", SuccessorID: prepared.ID, SuccessorDigest: prepared.RequestDigest}
	private := ed25519.NewKeyFromSeed(make([]byte, ed25519.SeedSize))
	public := private.Public().(ed25519.PublicKey)
	sum := sha256.Sum256(public)
	pin := "fixture:lifecycle#sha256:" + hex.EncodeToString(sum[:])
	mustWrite(t, filepath.Join(root, ".pose/policy/review.json"), `{"schema_version":2,"enabled":true,"profiles":{"spec":"spec-closeout@1"},"authority_project":"proj.storage-test","authority_audience":"fixture-verifier","trusted_attestation_issuers":["`+pin+`"]}`)
	mustWrite(t, filepath.Join(root, ".pose/policy/actions.json"), `{"schema_version":2,"identity_assurance":"verified","roles":{"maintainer":["human:maintainer"]}}`)
	claim := pose.ActionAuthorityClaim{SchemaVersion: 2, Project: "proj.storage-test", Audience: "fixture-verifier", RequestID: first.Request.ID, RequestDigest: first.Request.RequestDigest, Principal: "agent:impl", Role: "agent", Operation: pose.ActionEventSuperseded, ExpectedRevision: 1, IdempotencyKey: auth.IdempotencyKey, Reason: auth.Reason, SuccessorID: prepared.ID, SuccessorDigest: prepared.RequestDigest, Issuer: "fixture:lifecycle", IssuedAt: time.Now().Add(-time.Minute).UTC().Format(time.RFC3339Nano)}
	signed, e := json.Marshal(claim)
	if e != nil {
		t.Fatal(e)
	}
	auth.Claim = &claim
	auth.Envelope = &pose.ActionClaimEnvelope{Issuer: claim.Issuer, Algorithm: "ed25519", PublicKey: base64.StdEncoding.EncodeToString(public), Signature: base64.StdEncoding.EncodeToString(ed25519.Sign(private, signed))}
	raw, err := json.Marshal(auth)
	if err != nil {
		t.Fatal(err)
	}
	mustWrite(t, filepath.Join(root, "authority.json"), string(raw))
	code, out = runPose(t, root, append(args, "--authority", "authority.json", "--apply")...)
	if code != 0 {
		t.Fatalf("prepared successor did not survive apply: %s", out)
	}
	var next pose.ActionRequestView
	if err := json.Unmarshal([]byte(out), &next); err != nil {
		t.Fatal(err)
	}
	if next.Request.ID != prepared.ID {
		t.Fatal("successor identity changed")
	}
	old, err := (pose.Store{Root: root}).LoadActionRequest(first.Request.ID)
	if err != nil || old.State != pose.ActionStateSuperseded || old.Events[1].SupersededBy != prepared.ID {
		t.Fatalf("lineage: %+v %v", old, err)
	}
	args = append(append([]string{}, openArgs...), "--requested-at", "not-a-date", "--apply")
	if code, out := runPose(t, root, args...); code == 0 {
		t.Fatalf("malformed prepared timestamp accepted: %s", out)
	}
}
