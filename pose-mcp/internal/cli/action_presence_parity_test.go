package cli

// Spec pose-proof-presence-channel-parity: caller admission, proof and disclosure.

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	pose "github.com/harne8/pose-mcp/internal/pose"
)

func presenceCLIRequest(t *testing.T, root string) pose.ActionRequestView {
	t.Helper()
	args := append(append([]string{}, openArgs...), "--apply", "--json")
	code, raw := runPose(t, root, args...)
	var view pose.ActionRequestView
	if err := json.Unmarshal([]byte(raw), &view); code != 0 || err != nil || view.Request.ID == "" {
		t.Fatalf("open: %d %v %s", code, err, raw)
	}
	return view
}

func presenceCLIClaim(t *testing.T, root string, view pose.ActionRequestView) string {
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
	mustWrite(t, filepath.Join(root, ".pose/policy/review.json"), string(raw))
	claim := pose.ActionAuthorityClaim{SchemaVersion: 1, Project: "proj.storage-test", Audience: "fixture-audience", RequestID: view.Request.ID, RequestDigest: view.Request.RequestDigest, Principal: "human:maintainer", Role: "human", Answer: "preserve-v1", Issuer: issuer, IssuedAt: time.Now().Add(-time.Second).UTC().Format(time.RFC3339), ExpiresAt: time.Now().Add(time.Hour).UTC().Format(time.RFC3339)}
	canonical, err := json.Marshal(claim)
	if err != nil {
		t.Fatal(err)
	}
	envelope := pose.ActionClaimEnvelope{Issuer: issuer, Algorithm: "ed25519", PublicKey: base64.StdEncoding.EncodeToString(public), Signature: base64.StdEncoding.EncodeToString(ed25519.Sign(private, canonical))}
	raw, err = json.Marshal(map[string]any{"claim": claim, "envelope": envelope})
	if err != nil {
		t.Fatal(err)
	}
	rel := ".pose/reports/presence-claim.json"
	mustWrite(t, filepath.Join(root, rel), string(raw))
	return rel
}

func TestActionPresenceCLIIssuerAndHistoricalParity(t *testing.T) {
	for _, tc := range []struct {
		name       string
		version    int
		scope      string
		want       bool
		diagnostic string
	}{
		{"legacy", 1, "", false, "migration-required"}, {"universal", 2, "universal", false, "unsupported"}, {"sshsig", 2, "sshsig", true, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := actionCLIFixture(t)
			view := presenceCLIRequest(t, root)
			claim := presenceCLIClaim(t, root, view)
			policy := map[string]any{"schema_version": tc.version, "roles": map[string][]string{"maintainer": {"human:maintainer"}}, "identity_assurance": "verified", "require_presence": true}
			if tc.scope != "" {
				policy["presence_scope"] = tc.scope
			}
			raw, err := json.Marshal(policy)
			if err != nil {
				t.Fatal(err)
			}
			mustWrite(t, filepath.Join(root, ".pose/policy/actions.json"), string(raw))
			path := filepath.Join(root, ".pose/actions", view.Request.ID+".jsonl")
			before, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			args := []string{"action", "resolve", view.Request.ID, "--actor", "human:maintainer", "--answer", "preserve-v1", "--request-digest", view.Request.RequestDigest, "--expected-revision", "1", "--idempotency-key", "presence-cli", "--claim", claim, "--apply", "--json"}
			code, out := runPose(t, root, args...)
			if tc.want {
				var got pose.ActionRequestView
				if err := json.Unmarshal([]byte(out), &got); code != 0 || err != nil || got.Assurance != "verified" || got.Presence == nil || got.Presence.State != "not-established" {
					t.Fatalf("issuer presence overclaim/refusal: %d %v %s", code, err, out)
				}
				// Current universal policy does not retroactively impose a contract.
				policy["presence_scope"] = "universal"
				raw, err = json.Marshal(policy)
				if err != nil {
					t.Fatal(err)
				}
				mustWrite(t, filepath.Join(root, ".pose/policy/actions.json"), string(raw))
				for _, jsonOutput := range []bool{false, true} {
					args := []string{"action", "show", view.Request.ID}
					if jsonOutput {
						args = append(args, "--json")
					}
					code, out := runPose(t, root, args...)
					if code != 0 || !strings.Contains(out, "not-established") || !strings.Contains(out, "not-recorded") {
						t.Fatalf("history disclosure: %d %s", code, out)
					}
				}
			} else {
				if code == 0 || !strings.Contains(out, tc.diagnostic) {
					t.Fatalf("issuer bypass: %d %s", code, out)
				}
				after, err := os.ReadFile(path)
				if err != nil || !bytes.Equal(before, after) {
					t.Fatalf("refusal wrote: %v", err)
				}
			}
		})
	}
}

func TestActionPresenceCLIPreviewAndMigrationBeforeSigner(t *testing.T) {
	root := actionCLIFixture(t)
	view := presenceCLIRequest(t, root)
	mustWrite(t, filepath.Join(root, ".pose/policy/actions.json"), `{"schema_version":1,"roles":{"maintainer":["human:maintainer"]},"require_presence":true}`)
	args := []string{"action", "resolve", view.Request.ID, "--actor", "human:maintainer", "--answer", "preserve-v1", "--request-digest", view.Request.RequestDigest, "--expected-revision", "1", "--idempotency-key", "preview", "--sign", filepath.Join(root, "missing-private-key"), "--json"}
	path := filepath.Join(root, ".pose/actions", view.Request.ID+".jsonl")
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	code, out := runPose(t, root, args...)
	var preview map[string]any
	// B10 exposes this known admission blocker while retaining policy disclosure.
	if err := json.Unmarshal([]byte(out), &preview); code == 0 || err != nil || preview["presence_evaluation"] != "not-evaluated" || !strings.Contains(out, "migration-required") {
		t.Fatalf("preview: %d %v %s", code, err, out)
	}
	policy, ok := preview["presence_policy"].(map[string]any)
	if !ok || policy["migration_required"] != true || policy["scope"] != "legacy-sshsig-only" {
		t.Fatalf("hidden migration: %+v", preview)
	}
	code, out = runPose(t, root, append(args, "--apply")...)
	if code == 0 || !strings.Contains(out, "migration-required") || strings.Contains(out, "signing failed") {
		t.Fatalf("signer ran before migration: %d %s", code, out)
	}
	after, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatalf("preview/refusal wrote: %v", err)
	}
	code, out = runPose(t, root, "identity", "list", "--json")
	if code != 0 || !strings.Contains(out, "legacy-sshsig-only") {
		t.Fatalf("identity hid scope: %d %s", code, out)
	}
}

func TestActionPresenceDoctorIssuerCannotReplaceUniversalPresence(t *testing.T) {
	for _, scope := range []string{"universal", "sshsig"} {
		t.Run(scope, func(t *testing.T) {
			root, _ := installedInstance(t)
			mustWrite(t, filepath.Join(root, ".pose/policy/actions.json"), fmt.Sprintf(`{"schema_version":2,"roles":{"maintainer":["human:maintainer"]},"identity_assurance":"verified","require_presence":true,"presence_scope":%q}`, scope))
			var policy map[string]any
			path := filepath.Join(root, ".pose/policy/review.json")
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal(raw, &policy); err != nil {
				t.Fatal(err)
			}
			public, _, keyErr := ed25519.GenerateKey(rand.Reader)
			if keyErr != nil {
				t.Fatal(keyErr)
			}
			pin := pose.IssuerPin("fixture:issuer", public)
			policy["human_authority_issuers"] = []string{pin}
			policy["trusted_attestation_issuers"] = []string{pin}
			raw, err = json.Marshal(policy)
			if err != nil {
				t.Fatal(err)
			}
			mustWrite(t, path, string(raw))
			code, output := runPose(t, root, "doctor", "--json")
			var report struct {
				Findings []doctorFinding `json:"findings"`
			}
			if err := json.Unmarshal([]byte(output), &report); err != nil {
				t.Fatalf("doctor JSON: %v %s", err, output)
			}
			if (code != 0) != (scope == "universal") {
				t.Fatalf("doctor status: %d %s", code, output)
			}
			findings := report.Findings
			presence, ok := findDoctorFinding(findings, "actions.presence")
			if !ok || !strings.Contains(presence.Message, scope) {
				t.Fatalf("scope absent: %+v", findings)
			}
			keys, _ := findDoctorFinding(findings, "actions.keys")
			if (keys.Level == "error") != (scope == "universal") {
				t.Fatalf("issuer suppressed/unjustifiably acquired presence: %+v", keys)
			}
		})
	}
}
