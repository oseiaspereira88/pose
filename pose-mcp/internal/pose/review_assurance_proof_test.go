package pose

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

// Spec pose-review-disclosure-requires-proof: coherent claim fields cannot
// authenticate themselves, and semantic rejection does not invalidate identity.
func assertReviewProofNotVerified(t *testing.T, a *ReviewAssurance) {
	t.Helper()
	if a == nil || a.ReviewerIdentity != reviewAssuranceNotVerified || a.SeparationVerified != reviewAssuranceNotVerified || a.VerifiedClaim != nil {
		t.Fatalf("invalid proof disclosed verified assurance: %+v", a)
	}
	if a.Attribution != nil && a.Attribution.ConfirmationAssurance == ReviewIdentityAssuranceVerified {
		t.Fatalf("invalid proof verified human confirmation: %+v", a.Attribution)
	}
	raw, err := json.Marshal(a)
	if err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"reviewer_identity", "separation_verified", "confirmation_assurance"} {
		if strings.Contains(string(raw), `"`+field+`":"verified"`) {
			t.Fatalf("JSON promotes invalid proof: %s", raw)
		}
	}
	rendered := RenderReviewAssurance(*a)
	if strings.Contains(rendered, "verified human identity") || strings.Contains(rendered, "verified agent identity") || !strings.Contains(rendered, "identity not verified") {
		t.Fatalf("human rendering promotes invalid proof: %s", rendered)
	}
}

func TestReviewAssuranceRejectsInvalidCryptographicProof(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*authorityFixture, *ReviewAttestation)
	}{
		{"malformed-signature", func(_ *authorityFixture, a *ReviewAttestation) { a.Envelope.Signature = "!" }},
		{"invalid-signature", func(_ *authorityFixture, a *ReviewAttestation) {
			a.Envelope.Signature = base64.StdEncoding.EncodeToString(make([]byte, ed25519.SignatureSize))
		}},
		{"missing-envelope", func(_ *authorityFixture, a *ReviewAttestation) { a.Envelope = nil }},
		{"unsigned-decision-change", func(_ *authorityFixture, a *ReviewAttestation) { a.Decision = "rejected" }},
		{"unsigned-claim-change", func(_ *authorityFixture, a *ReviewAttestation) { a.Authority.ReviewExecution = "another-review-run" }},
		{"wrong-subject", func(_ *authorityFixture, a *ReviewAttestation) { a.Envelope.Subject = "another-bundle" }},
		{"unsupported-algorithm", func(_ *authorityFixture, a *ReviewAttestation) { a.Envelope.Algorithm = "none" }},
		{"untrusted-key", func(f *authorityFixture, a *ReviewAttestation) {
			other := *f
			other.private = ed25519.NewKeyFromSeed(make([]byte, ed25519.SeedSize))
			other.public = other.private.Public().(ed25519.PublicKey)
			*a = resignAuthorityAttestation(t, other, *a)
		}},
		{"revoked-pin", func(f *authorityFixture, _ *ReviewAttestation) {
			path := filepath.Join(f.root, ".pose/policy/review.json")
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			var policy map[string]any
			if err := json.Unmarshal(raw, &policy); err != nil {
				t.Fatal(err)
			}
			policy["trusted_attestation_issuers"] = []string{}
			raw, err = json.Marshal(policy)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, raw, 0o644); err != nil {
				t.Fatal(err)
			}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := verifiedAuthorityFixture(t, "different-actor")
			att := f.attestation(t, "agent:reviewer", "agent:implementer", "review-run-2", "implementation-run-1")
			valid := f.store.DescribeReviewAssurance(f.bundle, &att)
			if valid.ReviewerIdentity != ReviewIdentityAssuranceVerified {
				t.Fatalf("fixture is not verified: %+v", valid)
			}
			tc.mutate(&f, &att)
			if len(f.store.validateBundleAttestation(f.bundle, att)) == 0 {
				t.Fatal("invalid proof passed the gate")
			}
			a := f.store.DescribeReviewAssurance(f.bundle, &att)
			assertReviewProofNotVerified(t, &a)
		})
	}
}

func TestReviewAssuranceBindsTheSuppliedBundle(t *testing.T) {
	f := verifiedAuthorityFixture(t, "different-actor")
	att := f.attestation(t, "agent:reviewer", "agent:implementer", "review-run-2", "implementation-run-1")
	other := f.bundle
	other.BundleID = "another-bundle"
	// A valid signature over its original bundle must not authenticate a
	// different supplied bundle, even when a caller copies the digest field.
	a := f.store.DescribeReviewAssurance(other, &att)
	assertReviewProofNotVerified(t, &a)
}

func TestReviewAssuranceKeepsIdentitySeparateFromApproval(t *testing.T) {
	for _, decision := range []string{"approved", "rejected"} {
		t.Run(decision, func(t *testing.T) {
			f := verifiedAuthorityFixture(t, "different-actor")
			att := f.attestation(t, "agent:reviewer", "agent:implementer", "review-run-2", "implementation-run-1")
			if len(att.Criteria) == 0 {
				t.Fatal("fixture has no required criteria")
			}
			att.Criteria = nil
			att.Decision = decision
			att = resignAuthorityAttestation(t, f, att)
			v := verifyAuthorityAttestation(t, f, att)
			if v.Approved || !containsSubstring(v.Blockers, "missing criterion") {
				t.Fatalf("invalid criteria passed: %+v", v)
			}
			if v.Assurance == nil || v.Assurance.ReviewerIdentity != ReviewIdentityAssuranceVerified || v.Assurance.VerifiedClaim == nil || v.Assurance.SeparationVerified != "actor-and-execution" {
				t.Fatalf("semantic rejection erased valid identity proof: %+v", v.Assurance)
			}
		})
	}
	att := ReviewAttestation{Reviewer: "human:legacy"}
	a := Store{}.DescribeReviewAssurance(declaredBundle("different-actor"), &att)
	if a.ReviewerIdentity != ReviewIdentityAssuranceDeclared || a.VerifiedClaim != nil {
		t.Fatalf("declared history changed: %+v", a)
	}
}

func TestReviewConsumersAgreeOnTamperedProof(t *testing.T) {
	f := verifiedAuthorityFixture(t, "different-actor")
	enableReviewBundlesForAssurance(t, &f)
	att := f.attestation(t, "agent:reviewer", "agent:implementer", "review-run-2", "implementation-run-1")
	recorded, err := f.store.RecordReviewAttestation(att, f.now.Add(2*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	recorded.Envelope.Signature = base64.StdEncoding.EncodeToString(make([]byte, ed25519.SignatureSize))
	raw, err := json.Marshal(recorded)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(f.root, recorded.Path), raw, 0o644); err != nil {
		t.Fatal(err)
	}
	v, err := f.store.VerifyReviewBundle("spec:backend")
	if err != nil {
		t.Fatal(err)
	}
	if v.Approved || !containsSubstring(v.Blockers, "signature verification failed") {
		t.Fatalf("tampering was not rejected: %+v", v)
	}
	eval, err := f.store.ReviewCheck("spec:backend")
	if err != nil {
		t.Fatal(err)
	}
	closeout, err := f.store.GetCloseoutState("spec:backend")
	if err != nil {
		t.Fatal(err)
	}
	direct := f.store.DescribeReviewAssurance(f.bundle, &recorded)
	for _, a := range []*ReviewAssurance{v.Assurance, eval.Assurance, closeout.Review.Assurance, &direct} {
		assertReviewProofNotVerified(t, a)
		if !reflect.DeepEqual(v.Assurance, a) {
			t.Fatalf("consumers disagree: verify=%+v other=%+v", v.Assurance, a)
		}
	}
	if eval.Approved || closeout.Review.Approved {
		t.Fatal("a consumer approved tampered proof")
	}
}

func TestReviewInvalidProofCannotVerifyHumanConfirmation(t *testing.T) {
	f := verifiedAuthorityFixture(t, "different-actor")
	trustHumanAuthority(t, f)
	completed, canonical, err := f.store.SignableReviewAttestation(confirmationDraft(f, "human:maintainer", "human"), f.now.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	recorded, err := f.store.RecordReviewAttestationEnvelope(signEnvelope(f, completed, canonical), true)
	if err != nil {
		t.Fatal(err)
	}
	valid := f.store.DescribeReviewAssurance(f.bundle, &recorded)
	if valid.Attribution == nil || valid.Attribution.ConfirmationAssurance != ReviewIdentityAssuranceVerified {
		t.Fatalf("fixture confirmation not verified: %+v", valid)
	}
	recorded.Envelope.Signature = base64.StdEncoding.EncodeToString(make([]byte, ed25519.SignatureSize))
	a := f.store.DescribeReviewAssurance(f.bundle, &recorded)
	assertReviewProofNotVerified(t, &a)
	if a.Attribution.ConfirmationAssurance != ReviewIdentityAssuranceDeclared {
		t.Fatalf("damaged confirmation should remain a declaration: %+v", a.Attribution)
	}
}
