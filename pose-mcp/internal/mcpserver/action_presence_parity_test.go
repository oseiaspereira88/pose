package mcpserver

// Spec pose-proof-presence-channel-parity: caller admission, proof and disclosure.

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/harne8/pose-mcp/internal/pose"
)

func presenceMCPClaim(t *testing.T, root string, view pose.ActionRequestView) map[string]any {
	t.Helper()
	public, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	issuer := "fixture:presence"
	pin := pose.IssuerPin(issuer, public)
	policy := map[string]any{"schema_version": 2, "enabled": true, "profiles": map[string]string{"spec": "spec-closeout@1"}, "authority_project": "proj.storage-test", "authority_audience": "fixture-audience", "trusted_attestation_issuers": []string{pin}, "human_authority_issuers": []string{pin}}
	raw, err := json.Marshal(policy)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".pose/policy/review.json"), raw, 0644); err != nil {
		t.Fatal(err)
	}
	claim := pose.ActionAuthorityClaim{SchemaVersion: 1, Project: "proj.storage-test", Audience: "fixture-audience", RequestID: view.Request.ID, RequestDigest: view.Request.RequestDigest, Principal: "human:maintainer", Role: "human", Answer: "preserve-v1", Issuer: issuer, IssuedAt: time.Now().Add(-time.Second).UTC().Format(time.RFC3339), ExpiresAt: time.Now().Add(time.Hour).UTC().Format(time.RFC3339)}
	canonical, err := json.Marshal(claim)
	if err != nil {
		t.Fatal(err)
	}
	envelope := pose.ActionClaimEnvelope{Issuer: issuer, Algorithm: "ed25519", PublicKey: base64.StdEncoding.EncodeToString(public), Signature: base64.StdEncoding.EncodeToString(ed25519.Sign(private, canonical))}
	return map[string]any{"claim": claim, "envelope": envelope}
}

func TestActionPresenceMCPPreviewAndApplyParity(t *testing.T) {
	for _, tc := range []struct {
		name, policy, diagnostic string
		want                     bool
	}{
		{"legacy", `{"schema_version":1,"roles":{"maintainer":["human:maintainer"]},"require_presence":true}`, "migration-required", false},
		{"universal", `{"schema_version":2,"roles":{"maintainer":["human:maintainer"]},"require_presence":true,"presence_scope":"universal"}`, "unsupported", false},
		{"sshsig", `{"schema_version":2,"roles":{"maintainer":["human:maintainer"]},"require_presence":true,"presence_scope":"sshsig"}`, "", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root, ts, view := actionResolveFixture(t, tc.policy)
			path := filepath.Join(root, ".pose/actions", view.Request.ID+".jsonl")
			before, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			preview := resolveCall(t, ts, view, nil)
			sc, ok := preview.Result["structuredContent"].(map[string]any)
			if !ok || sc["presence_evaluation"] != "not-evaluated" || sc["presence_policy"] == nil {
				t.Fatalf("preview claims evaluated proof/hides policy: %+v", preview)
			}
			args := presenceMCPClaim(t, root, view)
			args["apply"] = true
			out := resolveCall(t, ts, view, args)
			if tc.want {
				if out.Result["isError"] == true || !strings.Contains(toolText(out), "not-established") || !strings.Contains(toolText(out), "issuer-no-presence-contract") {
					t.Fatalf("issuer gained presence/refused: %+v", out)
				}
			} else {
				if out.Result["isError"] != true || !strings.Contains(toolText(out), tc.diagnostic) {
					t.Fatalf("issuer bypass: %+v", out)
				}
				after, err := os.ReadFile(path)
				if err != nil || !bytes.Equal(before, after) {
					t.Fatalf("MCP refusal wrote: %v", err)
				}
				got, err := (pose.Store{Root: root}).LoadActionRequest(view.Request.ID)
				if err != nil || got.Revision != 1 || got.Satisfaction != pose.SatisfactionPending {
					t.Fatalf("MCP refusal satisfied: %+v %v", got, err)
				}
			}
		})
	}
}
