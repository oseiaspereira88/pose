package pose

// Spec pose-proof-presence-channel-parity: caller admission, proof and disclosure.

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestActionPresenceLegacyRequiresMigration(t *testing.T) {
	now := time.Date(2026, 10, 23, 12, 0, 0, 0, time.UTC)
	for _, issuer := range []bool{false, true} {
		name := "declared"
		if issuer {
			name = "issuer"
		}
		t.Run(name, func(t *testing.T) {
			s, view, res := issuerAnswerFixture(t, now, now.Add(time.Hour))
			if !issuer {
				res.Claim, res.Envelope = nil, nil
			}
			writeActionPolicy(t, s, keysPolicy("declared", nil, map[string]any{"require_presence": true}))
			path := actionJournalPath(s.Root, view.Request.ID)
			before, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			_, err = s.ResolveActionRequest(res, now)
			if !errors.Is(err, ErrActionVerificationFailed) || !strings.Contains(err.Error(), "migration") {
				t.Errorf("legacy presence bypass did not require explicit migration: %v", err)
			}
			after, err := os.ReadFile(path)
			if err != nil || !bytes.Equal(before, after) {
				t.Errorf("refusal changed journal: %v", err)
			}
		})
	}
}

func presenceClaim(t *testing.T, s Store, view ActionRequestView, res ActionResolution, native bool, mutate func(*ActionAuthorityClaim)) ActionResolution {
	t.Helper()
	public, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	issuer := "external:fixture"
	if native {
		home := t.TempDir()
		if err := os.Chmod(home, 0700); err != nil {
			t.Fatal(err)
		}
		t.Setenv("POSE_ISSUER_HOME", home)
		key, err := CreateIssuerKey(s.Root, "presence-fixture", time.Date(2026, 10, 23, 12, 0, 0, 0, time.UTC))
		if err != nil {
			t.Fatal(err)
		}
		issuer, public, private = key.Name, key.public, key.private
	}
	pin := issuer + "#" + digestBytes(public)
	policy := map[string]any{"schema_version": 2, "enabled": true, "profiles": map[string]string{"spec": "spec-closeout@1"}, "authority_project": "proj.fixture", "authority_audience": "fixture-audience", "trusted_attestation_issuers": []string{pin}, "human_authority_issuers": []string{pin}}
	raw, err := json.Marshal(policy)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(s.Root, ".pose/policy/review.json"), raw, 0644); err != nil {
		t.Fatal(err)
	}
	role := "human"
	if strings.HasPrefix(res.Actor, "agent:") {
		role = "agent"
	}
	claim := &ActionAuthorityClaim{SchemaVersion: 1, Project: "proj.fixture", Audience: "fixture-audience", RequestID: view.Request.ID, RequestDigest: view.Request.RequestDigest, Principal: res.Actor, Role: role, Answer: res.Answer, Reason: res.Reason, Issuer: issuer, IssuedAt: "2026-10-23T12:00:00Z", ExpiresAt: "2026-10-23T13:00:00Z"}
	if mutate != nil {
		mutate(claim)
	}
	canonical, err := json.Marshal(claim)
	if err != nil {
		t.Fatal(err)
	}
	res.Claim = claim
	res.Envelope = &ActionClaimEnvelope{Issuer: issuer, Algorithm: "ed25519", PublicKey: base64.StdEncoding.EncodeToString(public), Signature: base64.StdEncoding.EncodeToString(ed25519.Sign(private, canonical))}
	return res
}

func TestActionPresenceAdmissionMatrix(t *testing.T) {
	plain, sk := newSSHTestKey(t, false), newSSHTestKey(t, true)
	now := time.Date(2026, 10, 23, 12, 0, 0, 0, time.UTC)
	for _, actor := range []string{"human:maintainer", "agent:maintainer"} {
		for _, assurance := range []string{"declared", "verified"} {
			for _, required := range []bool{false, true} {
				for _, scope := range []string{"universal", "sshsig"} {
					for _, mechanism := range []string{"declared", "plain", "sk-untouched", "sk-uv-only", "sk-touched", "issuer", "combined"} {
						t.Run(fmt.Sprintf("%s/%s/%t/%s/%s", actor, assurance, required, scope, mechanism), func(t *testing.T) {
							s := actionFixture(t)
							extra := map[string]any{"schema_version": 2, "require_presence": required, "presence_scope": scope, "roles": map[string][]string{"maintainer": {actor}}}
							writeActionPolicy(t, s, keysPolicy(assurance, map[string][]string{actor: {plain.line(""), sk.line("")}}, extra))
							view := openedDecision(t, s)
							res := answer(view, actor, "preserve-v1", "presence-matrix")
							res.Channel = "portal-human-touch"
							flags := byte(0)
							signingKey := sk
							switch mechanism {
							case "plain":
								signingKey = plain
							case "sk-touched", "combined":
								flags = sshSKUserPresent
							case "sk-uv-only":
								flags = sshSKUserVerified
							}
							if mechanism != "declared" && mechanism != "issuer" {
								res.Signature = signingKey.sign(ActionAnswerNamespace, AnswerStatement(view.Request, actor, res.Answer, res.IdempotencyKey, "").Canonical(), flags)
							}
							if mechanism == "issuer" || mechanism == "combined" {
								res = presenceClaim(t, s, view, res, false, nil)
							}
							want := !(assurance == "verified" && mechanism == "declared")
							if actor == "human:maintainer" && required && (scope == "universal" || (mechanism != "declared" && mechanism != "issuer")) {
								want = want && (mechanism == "sk-touched" || mechanism == "combined")
							}
							path := actionJournalPath(s.Root, view.Request.ID)
							before, err := os.ReadFile(path)
							if err != nil {
								t.Fatal(err)
							}
							after, err := s.ResolveActionRequest(res, now)
							if want {
								if err != nil || after.Satisfaction != SatisfactionSatisfied {
									t.Fatalf("valid answer refused: %+v %v", after, err)
								}
								if after.Presence == nil {
									t.Fatal("presence not disclosed")
								}
								verifiedPresence := actor == "human:maintainer" && (mechanism == "sk-touched" || mechanism == "combined")
								if (after.Presence.State == "verified-key-assertion") != verifiedPresence {
									t.Fatalf("false presence: %+v", after.Presence)
								}
							} else {
								if !errors.Is(err, ErrActionVerificationFailed) {
									t.Fatalf("invalid answer accepted: %v", err)
								}
								got, e := os.ReadFile(path)
								if e != nil || !bytes.Equal(before, got) {
									t.Fatalf("refusal wrote journal: %v", e)
								}
								current, e := s.LoadActionRequest(view.Request.ID)
								if e != nil || current.Revision != 1 || current.Satisfaction != SatisfactionPending {
									t.Fatalf("refusal satisfied request: %+v %v", current, e)
								}
							}
						})
					}
				}
			}
		}
	}
}

func TestActionPresenceIssuerOriginAndMixedProofBindings(t *testing.T) {
	now := time.Date(2026, 10, 23, 12, 0, 0, 0, time.UTC)
	sk := newSSHTestKey(t, true)
	for _, native := range []bool{false, true} {
		for name, mutate := range map[string]func(*ActionAuthorityClaim){
			"valid": nil, "project": func(c *ActionAuthorityClaim) { c.Project = "proj.other" }, "audience": func(c *ActionAuthorityClaim) { c.Audience = "other" }, "principal": func(c *ActionAuthorityClaim) { c.Principal = "human:other" }, "request": func(c *ActionAuthorityClaim) { c.RequestID = "act-0000000000000000" }, "digest": func(c *ActionAuthorityClaim) { c.RequestDigest = "sha256:other" }, "answer": func(c *ActionAuthorityClaim) { c.Answer = "break-v1" }, "reason": func(c *ActionAuthorityClaim) { c.Reason = "other" },
		} {
			t.Run(fmt.Sprintf("native=%t/%s", native, name), func(t *testing.T) {
				s := actionFixture(t)
				writeActionPolicy(t, s, keysPolicy("verified", map[string][]string{"human:maintainer": {sk.line("")}}, map[string]any{"schema_version": 2, "require_presence": true, "presence_scope": "universal"}))
				view := openedDecision(t, s)
				res := signedAnswer(view, sk, "human:maintainer", "preserve-v1", "mixed", sshSKUserPresent)
				res = presenceClaim(t, s, view, res, native, mutate)
				path := actionJournalPath(s.Root, view.Request.ID)
				before, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				_, err = s.ResolveActionRequest(res, now)
				if name == "valid" {
					if err != nil {
						t.Fatal(err)
					}
				} else {
					if !errors.Is(err, ErrActionVerificationFailed) {
						t.Fatalf("mismatched claim fell back to SSH: %v", err)
					}
					after, e := os.ReadFile(path)
					if e != nil || !bytes.Equal(before, after) {
						t.Fatalf("mixed proof refusal wrote: %v", e)
					}
				}
			})
		}
	}
}

func TestActionPresenceHistoricalDisclosureDoesNotAdoptCurrentPolicy(t *testing.T) {
	now := time.Date(2026, 10, 23, 12, 0, 0, 0, time.UTC)
	s, view, res := issuerAnswerFixture(t, now, now.Add(time.Hour))
	if _, err := s.ResolveActionRequest(res, now); err != nil {
		t.Fatal(err)
	}
	path := actionJournalPath(s.Root, view.Request.ID)
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	writeActionPolicy(t, s, keysPolicy("verified", nil, map[string]any{"schema_version": 2, "require_presence": true, "presence_scope": "universal"}))
	after, err := s.LoadActionRequest(view.Request.ID)
	if err != nil || after.Assurance != "verified" || after.Satisfaction != SatisfactionSatisfied {
		t.Fatalf("current policy rewrote history: %+v %v", after, err)
	}
	if after.Presence == nil || after.Presence.State != "not-established" || after.Presence.HistoricalRequirement != "not-recorded" {
		t.Fatalf("issuer gained presence or historical receipt: %+v", after.Presence)
	}
	got, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(before, got) {
		t.Fatalf("historical read wrote: %v", err)
	}
}

func TestActionPresencePolicyRoundTrip(t *testing.T) {
	for _, scope := range []string{"universal", "sshsig"} {
		t.Run(scope, func(t *testing.T) {
			s := actionFixture(t)
			writeActionPolicy(t, s, keysPolicy("declared", nil, map[string]any{"schema_version": 2, "require_presence": true, "presence_scope": scope, "_comment": "keep"}))
			doc, err := LoadActionPolicyDocument(s.Root)
			if err != nil {
				t.Fatal(err)
			}
			doc.GrantRole("reviewer", "agent:review")
			key, keyErr := ParseAuthorizedKey(newSSHTestKey(t, true).line("presence-roundtrip"))
			if keyErr != nil {
				t.Fatal(keyErr)
			}
			if _, err := doc.AddKey("human:maintainer", key, "presence-roundtrip", "2026-10-23"); err != nil {
				t.Fatal(err)
			}
			if err := doc.Write(s.Root); err != nil {
				t.Fatal(err)
			}
			got, err := LoadActionPolicy(s.Root)
			if err != nil || got.SchemaVersion != 2 || !got.RequirePresence || got.PresenceScope != scope {
				t.Fatalf("policy weakened: %+v %v", got, err)
			}
			raw, err := os.ReadFile(actionPolicyPath(s.Root))
			if err != nil || !bytes.Contains(raw, []byte("keep")) {
				t.Fatalf("metadata lost: %v", err)
			}
		})
	}
}

func TestActionPresenceDiagnosticsAndInvalidMixedProof(t *testing.T) {
	now := time.Date(2026, 10, 23, 12, 0, 0, 0, time.UTC)
	for _, mode := range []string{"missing", "issuer", "invalid-signature"} {
		t.Run(mode, func(t *testing.T) {
			s, view, res := issuerAnswerFixture(t, now, now.Add(time.Hour))
			writeActionPolicy(t, s, keysPolicy("declared", nil, map[string]any{"schema_version": 2, "require_presence": true, "presence_scope": "universal"}))
			if mode == "missing" {
				res.Claim, res.Envelope = nil, nil
			}
			if mode == "invalid-signature" {
				res.Signature = "invalid"
			}
			_, err := s.ResolveActionRequest(res, now)
			if !errors.Is(err, ErrActionVerificationFailed) {
				t.Fatalf("unexpected acceptance: %v", err)
			}
			var presence *ActionPresenceError
			if mode == "invalid-signature" {
				if errors.As(err, &presence) {
					t.Fatalf("invalid proof treated as admitted/missing presence: %v", err)
				}
			} else {
				want := "missing"
				if mode == "issuer" {
					want = "unsupported"
				}
				if !errors.As(err, &presence) || presence.Code != want {
					t.Fatalf("diagnostic %s: %v", want, err)
				}
			}
			after, loadErr := s.LoadActionRequest(view.Request.ID)
			if loadErr != nil || after.Revision != 1 {
				t.Fatalf("invalid proof changed journal: %+v %v", after, loadErr)
			}
		})
	}
}

func TestActionPresenceLegacyMigrationDoesNotRequireAgentTouch(t *testing.T) {
	s := actionFixture(t)
	writeActionPolicy(t, s, keysPolicy("declared", nil, map[string]any{"require_presence": true, "roles": map[string][]string{"maintainer": {"agent:maintainer"}}}))
	view := openedDecision(t, s)
	after, err := s.ResolveActionRequest(answer(view, "agent:maintainer", "preserve-v1", "agent"), time.Now())
	if err != nil || after.Presence == nil || after.Presence.State != "not-applicable" {
		t.Fatalf("agent gained/required human presence: %+v %v", after, err)
	}
}

func TestActionPresenceSchemaContract(t *testing.T) {
	for _, tc := range []struct {
		name           string
		version        int
		required       bool
		scope          any
		include, valid bool
	}{
		{"legacy off", 1, false, nil, false, true},
		{"legacy enabled readable", 1, true, nil, false, true},
		{"legacy scope refused", 1, true, "sshsig", true, false},
		{"v2 disabled", 2, false, nil, false, true},
		{"v2 missing scope", 2, true, nil, false, false},
		{"v2 universal", 2, true, "universal", true, true},
		{"v2 sshsig", 2, true, "sshsig", true, true},
		{"unknown scope", 2, false, "portal", true, false},
		{"wrong scope type", 2, true, 7, true, false},
		{"unknown version", 3, false, nil, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := actionFixture(t)
			extra := map[string]any{"schema_version": tc.version, "require_presence": tc.required}
			if tc.include {
				extra["presence_scope"] = tc.scope
			}
			writeActionPolicy(t, s, keysPolicy("declared", nil, extra))
			_, err := LoadActionPolicy(s.Root)
			if (err == nil) != tc.valid {
				t.Fatalf("valid=%v, error=%v", tc.valid, err)
			}
		})
	}
}

func TestActionPresencePolicyEditingRefusesInvalidBeforePublication(t *testing.T) {
	s := actionFixture(t)
	path := filepath.Join(s.Root, ".pose/policy/actions.json")
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := LoadActionPolicyDocument(s.Root)
	if err != nil {
		t.Fatal(err)
	}
	doc.SchemaVersion = 99
	if err := doc.Write(s.Root); err == nil {
		t.Error("invalid policy was published")
	}
	after, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(before, after) {
		t.Errorf("invalid edit modified policy: %v", err)
	}
}
