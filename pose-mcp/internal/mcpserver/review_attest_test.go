package mcpserver

import (
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/harne8/pose-mcp/internal/pose"
)

// Spec pose-mcp-review-attest-signed-only: an agent records a review
// attestation over MCP only inside a trusted issuer's envelope. The signed
// positive path is covered in package pose
// (review_attest_signed_test.go); here the tool's refusals over JSON-RPC.

func reviewAttestServer(t *testing.T) (string, *httptest.Server) {
	t.Helper()
	t.Setenv("POSE_DEFAULT_PROJECT_ID", "proj.review-attest-test")
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".pose", "specs"), 0o755); err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(New(pose.Store{Root: root}).Handler("", ""))
	t.Cleanup(ts.Close)
	return root, ts
}

func reviewAttestCall(t *testing.T, ts *httptest.Server, args map[string]any) rpcResult {
	t.Helper()
	raw, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": map[string]any{"name": "pose_review_attest", "arguments": args}})
	_, out := post(t, ts, string(raw))
	return out
}

// R2: applying without an envelope is refused whatever the policy says, even
// with a complete-looking attestation that names a human confirmer.
func TestToolsCall_ReviewAttest_RefusesAnAttestationWithoutAnEnvelope(t *testing.T) {
	root, ts := reviewAttestServer(t)
	att := map[string]any{"bundle_id": "rvb-0123456789abcdef", "reviewer": "human:maintainer", "decision": "approved",
		"attribution": map[string]any{"schema_version": 1, "confirmed_by": "human:maintainer", "confirmation_mode": "adopted-conclusions"}}
	out := reviewAttestCall(t, ts, map[string]any{"attestation": att, "apply": true})
	if out.Result["isError"] != true || !strings.Contains(toolText(out), "envelope") {
		t.Fatalf("a declared attestation was recorded over MCP: %+v", out)
	}
	if entries, _ := os.ReadDir(filepath.Join(root, ".pose", "review-attestations")); len(entries) != 0 {
		t.Fatalf("a refused call left %d records", len(entries))
	}
}

// R3: an envelope from an issuer the policy does not trust is refused, as a
// preview and with apply, and nothing is recorded.
func TestToolsCall_ReviewAttest_RefusesAnUntrustedEnvelope(t *testing.T) {
	root, ts := reviewAttestServer(t)
	envelope := map[string]any{"schema_version": pose.ReviewBundleSchemaVersion, "issuer": "stranger", "subject": "rvb-0123456789abcdef", "algorithm": "ed25519",
		"public_key": "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=", "signature": strings.Repeat("A", 86) + "==",
		"attestation": map[string]any{"bundle_id": "rvb-0123456789abcdef", "reviewer": "human:maintainer", "decision": "approved"}}
	for _, apply := range []bool{false, true} {
		out := reviewAttestCall(t, ts, map[string]any{"envelope": envelope, "apply": apply})
		if out.Result["isError"] != true || !strings.Contains(toolText(out), "untrusted") {
			t.Fatalf("apply=%v: an untrusted envelope was accepted: %+v", apply, out)
		}
	}
	if entries, _ := os.ReadDir(filepath.Join(root, ".pose", "review-attestations")); len(entries) != 0 {
		t.Fatalf("a refused envelope left %d records", len(entries))
	}
}

// R1: a preview needs a draft or an envelope, and a draft for a bundle that
// does not exist is refused rather than completed.
func TestToolsCall_ReviewAttest_PreviewNeedsADraftForASealedBundle(t *testing.T) {
	_, ts := reviewAttestServer(t)
	out := reviewAttestCall(t, ts, map[string]any{})
	if out.Result["isError"] != true || !strings.Contains(toolText(out), "attestation or envelope") {
		t.Fatalf("an empty preview was accepted: %+v", out)
	}
	out = reviewAttestCall(t, ts, map[string]any{"attestation": map[string]any{"bundle_id": "rvb-0123456789abcdef", "reviewer": "human:maintainer", "decision": "approved"}})
	if out.Result["isError"] != true {
		t.Fatalf("a draft for an unknown bundle was completed: %+v", out)
	}
}
