package pose

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// Spec pose-action-answers-reverify-on-read.
func replaceAnswerEvent(t *testing.T, s Store, id string, mutate func(*ActionEvent)) {
	t.Helper()
	path := actionJournalPath(s.Root, id)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	events, err := parseActionJournal(raw)
	if err != nil {
		t.Fatal(err)
	}
	mutate(&events[len(events)-1])
	raw = nil
	for _, event := range events {
		line, err := json.Marshal(event)
		if err != nil {
			t.Fatal(err)
		}
		raw = append(raw, append(line, '\n')...)
	}
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}
}

func issuerAnswerFixture(t *testing.T, issued, expires time.Time) (Store, ActionRequestView, ActionResolution) {
	t.Helper()
	s := actionFixture(t)
	public, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	pin := "fixture:issuer#" + digestBytes(public)
	policy := map[string]any{"schema_version": 2, "enabled": true,
		"profiles":          map[string]string{"spec": "spec-closeout@1"},
		"authority_project": "proj.fixture", "authority_audience": "proj.fixture",
		"trusted_attestation_issuers": []string{pin}, "human_authority_issuers": []string{pin}}
	raw, _ := json.Marshal(policy)
	if err := os.WriteFile(filepath.Join(s.Root, ".pose/policy/review.json"), raw, 0o644); err != nil {
		t.Fatal(err)
	}
	view := openedDecision(t, s)
	res := answer(view, "human:maintainer", "preserve-v1", "issuer-answer")
	claim := &ActionAuthorityClaim{SchemaVersion: 1, Project: "proj.fixture", Audience: "proj.fixture",
		RequestID: view.Request.ID, RequestDigest: view.Request.RequestDigest,
		Principal: res.Actor, Role: "human", Answer: res.Answer, Issuer: "fixture:issuer",
		IssuedAt: issued.UTC().Format(time.RFC3339Nano)}
	if !expires.IsZero() {
		claim.ExpiresAt = expires.UTC().Format(time.RFC3339Nano)
	}
	canonical, _ := json.Marshal(claim)
	res.Claim = claim
	res.Envelope = &ActionClaimEnvelope{Issuer: claim.Issuer, Algorithm: "ed25519",
		PublicKey: base64.StdEncoding.EncodeToString(public),
		Signature: base64.StdEncoding.EncodeToString(ed25519.Sign(private, canonical))}
	return s, view, res
}

func TestActionReadRefusesProoflessVerifiedAndDegradesConsumers(t *testing.T) {
	s := actionFixture(t)
	view := openedDecision(t, s)
	if _, err := s.ResolveActionRequest(answer(view, "human:maintainer", "preserve-v1", "plain"), time.Now()); err != nil {
		t.Fatal(err)
	}
	replaceAnswerEvent(t, s, view.Request.ID, func(e *ActionEvent) { e.Assurance = ReviewIdentityAssuranceVerified })
	if _, err := s.LoadActionRequest(view.Request.ID); !errors.Is(err, ErrActionVerificationFailed) {
		t.Fatalf("a label established proof: %v", err)
	}
	if _, err := s.ListActionRequests(); !errors.Is(err, ErrActionVerificationFailed) {
		t.Fatalf("listing hid the invalid proof: %v", err)
	}
	if _, err := s.GovernedEffectRestrictions(PhaseCloseout, []string{"storage"}); !errors.Is(err, ErrActionVerificationFailed) {
		t.Fatalf("the gate consumed the forged answer: %v", err)
	}
	report, err := s.ProjectObligations(ObligationQuery{})
	if err != nil {
		t.Fatal(err)
	}
	for _, coverage := range report.Coverage {
		if coverage.Producer == "action-requests" {
			if coverage.State != CoverageStateUnavailable || report.Complete {
				t.Fatalf("invalid proof became complete coverage: %+v", report)
			}
			return
		}
	}
	t.Fatal("missing action-requests coverage")
}

func TestActionReadReverifiesSSHBindingsAndMetadata(t *testing.T) {
	cases := map[string]func(*ActionEvent){
		"answer":       func(e *ActionEvent) { e.Answer = "break-v1" },
		"actor":        func(e *ActionEvent) { e.Actor = "human:other" },
		"digest":       func(e *ActionEvent) { e.RequestDigest = "sha256:other" },
		"reason":       func(e *ActionEvent) { e.Reason = "edited" },
		"idempotency":  func(e *ActionEvent) { e.IdempotencyKey = "other" },
		"signature":    func(e *ActionEvent) { e.SSHSignature.Signature = "invalid" },
		"statement":    func(e *ActionEvent) { e.SSHSignature.Statement = "other" },
		"namespace":    func(e *ActionEvent) { e.SSHSignature.Namespace = "other" },
		"key":          func(e *ActionEvent) { e.SSHSignature.Key = "other" },
		"mixed proof":  func(e *ActionEvent) { e.Claim = &ActionAuthorityClaim{SchemaVersion: 1} },
		"fingerprint":  func(e *ActionEvent) { e.SSHSignature.Fingerprint = "other" },
		"presence":     func(e *ActionEvent) { e.SSHSignature.UserPresent = !e.SSHSignature.UserPresent },
		"verification": func(e *ActionEvent) { e.SSHSignature.UserVerified = !e.SSHSignature.UserVerified },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			s := actionFixture(t)
			key := newSSHTestKey(t, false)
			writeActionPolicy(t, s, keysPolicy("verified", map[string][]string{"human:maintainer": {key.line("")}}, nil))
			view := openedDecision(t, s)
			if _, err := s.ResolveActionRequest(signedAnswer(view, key, "human:maintainer", "preserve-v1", "ssh-answer", 0), time.Now()); err != nil {
				t.Fatal(err)
			}
			replaceAnswerEvent(t, s, view.Request.ID, mutate)
			if _, err := s.LoadActionRequest(view.Request.ID); !errors.Is(err, ErrActionVerificationFailed) {
				t.Fatalf("tampered %s accepted: %v", name, err)
			}
		})
	}
}

func TestActionReadReverifiesIssuerBindings(t *testing.T) {
	cases := map[string]func(*ActionEvent){
		"answer":   func(e *ActionEvent) { e.Answer = "break-v1" },
		"actor":    func(e *ActionEvent) { e.Actor = "human:other" },
		"digest":   func(e *ActionEvent) { e.RequestDigest = "sha256:other" },
		"reason":   func(e *ActionEvent) { e.Reason = "edited" },
		"claim":    func(e *ActionEvent) { e.Claim.Answer = "break-v1" },
		"envelope": func(e *ActionEvent) { e.Envelope = nil },
		"time":     func(e *ActionEvent) { e.At = "invalid" },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			now := time.Now().UTC().Truncate(time.Second)
			s, view, res := issuerAnswerFixture(t, now.Add(-time.Minute), now.Add(time.Hour))
			if _, err := s.ResolveActionRequest(res, now); err != nil {
				t.Fatal(err)
			}
			replaceAnswerEvent(t, s, view.Request.ID, mutate)
			if _, err := s.LoadActionRequest(view.Request.ID); !errors.Is(err, ErrActionVerificationFailed) {
				t.Fatalf("tampered %s accepted: %v", name, err)
			}
		})
	}
}

func TestActionReadKeepsHistoricalExpiryAndSecondPrecision(t *testing.T) {
	at := time.Date(2026, 10, 4, 12, 0, 0, 500000000, time.UTC)
	s, view, res := issuerAnswerFixture(t, at.Add(-time.Nanosecond), at.Add(time.Nanosecond))
	if _, err := s.ResolveActionRequest(res, at); err != nil {
		t.Fatalf("the supplied historical clock was ignored: %v", err)
	}
	after, err := s.LoadActionRequest(view.Request.ID)
	if err != nil || after.Assurance != ReviewIdentityAssuranceVerified || after.Satisfaction != SatisfactionSatisfied {
		t.Fatalf("a valid historical answer expired on read: %+v %v", after, err)
	}
}

func TestActionClaimRefusesFutureOrReversedExpiryAtWrite(t *testing.T) {
	now := time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC)
	for _, test := range []struct {
		name            string
		issued, expires time.Time
	}{
		{"future", now.Add(time.Minute), now.Add(time.Hour)},
		{"reversed", now.Add(-time.Minute), now.Add(-2 * time.Minute)},
		{"expired", now.Add(-time.Hour), now.Add(-time.Minute)},
	} {
		t.Run(test.name, func(t *testing.T) {
			s, view, res := issuerAnswerFixture(t, test.issued, test.expires)
			if _, err := s.ResolveActionRequest(res, now); !errors.Is(err, ErrActionVerificationFailed) {
				t.Fatalf("bad claim accepted: %v", err)
			}
			after, err := s.LoadActionRequest(view.Request.ID)
			if err != nil || after.Revision != 1 {
				t.Fatalf("refusal wrote an event: %+v %v", after, err)
			}
		})
	}
}

func TestActionReadDerivesProofAndPreservesDeclaredAndRotatedSSH(t *testing.T) {
	s := actionFixture(t)
	view := openedDecision(t, s)
	if _, err := s.ResolveActionRequest(answer(view, "human:maintainer", "preserve-v1", "declared"), time.Now()); err != nil {
		t.Fatal(err)
	}
	after, err := s.LoadActionRequest(view.Request.ID)
	if err != nil || after.Assurance != ReviewIdentityAssuranceDeclared || after.Satisfaction != SatisfactionSatisfied {
		t.Fatalf("declared history changed: %+v %v", after, err)
	}
	s = actionFixture(t)
	key := newSSHTestKey(t, false)
	writeActionPolicy(t, s, keysPolicy("verified", map[string][]string{"human:maintainer": {key.line("")}}, nil))
	view = openedDecision(t, s)
	if _, err := s.ResolveActionRequest(signedAnswer(view, key, "human:maintainer", "preserve-v1", "signed", 0), time.Now()); err != nil {
		t.Fatal(err)
	}
	writeActionPolicy(t, s, keysPolicy("verified", nil, nil))
	replaceAnswerEvent(t, s, view.Request.ID, func(e *ActionEvent) { e.Assurance = "declared" })
	after, err = s.LoadActionRequest(view.Request.ID)
	if err != nil || after.Assurance != ReviewIdentityAssuranceVerified {
		t.Fatalf("proof was replaced by a label or current key membership: %+v %v", after, err)
	}
}
