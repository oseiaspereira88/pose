package pose

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Spec pose-review-assurance-disclosure.

func declaredBundle(independence string) ReviewBundle {
	return ReviewBundle{Payload: ReviewBundlePayload{
		Plan:  ReviewBundlePlan{Independence: independence},
		Gates: &ReviewBundleGates{IdentityAssurance: ReviewIdentityAssuranceDeclared},
	}}
}

func TestDeclaredHumanReviewerIsNeverRenderedAsAVerifiedPerson(t *testing.T) {
	att := ReviewAttestation{Reviewer: "human:oseias"}
	a := Store{}.DescribeReviewAssurance(declaredBundle("same-actor-separate-execution"), &att)
	if a.IdentityAssurance != ReviewIdentityAssuranceDeclared || a.ReviewerIdentity != ReviewIdentityAssuranceDeclared {
		t.Fatalf("declared assurance not disclosed: %+v", a)
	}
	if a.SeparationRequired != "same-actor-separate-execution" || a.SeparationDeclared != "declared-human" || a.SeparationVerified != "not-verified" {
		t.Fatalf("separation axes collapsed: %+v", a)
	}
	if a.CognitiveIndependence != "not-observable" {
		t.Fatalf("cognitive independence must be stated as not observable: %+v", a)
	}
	rendered := RenderReviewAssurance(a)
	for _, want := range []string{"declared identity", "verified not-verified", "cognitive independence not-observable"} {
		if !strings.Contains(rendered, want) {
			t.Errorf("render %q lacks %q", rendered, want)
		}
	}
	if strings.Contains(rendered, "verified human") {
		t.Errorf("a declared human prefix rendered as a verified person: %q", rendered)
	}
}

func TestDeclaredIndependentAgentPrefixIsADeclaration(t *testing.T) {
	att := ReviewAttestation{Reviewer: "agent:independent-reviewer"}
	a := Store{}.DescribeReviewAssurance(declaredBundle("different-actor"), &att)
	if a.SeparationDeclared != "declared-independent-agent" || a.SeparationVerified != "not-verified" {
		t.Fatalf("an agent:independent- prefix was read as more than a declaration: %+v", a)
	}
	found := false
	for _, l := range a.Limitations {
		found = found || l == ReviewLimitationSeparationDeclared
	}
	if !found {
		t.Fatalf("declared separation limitation missing: %+v", a.Limitations)
	}
}

func TestVerifiedAssuranceDisclosesTheClaimItVerified(t *testing.T) {
	f := verifiedAuthorityFixture(t, "different-actor")
	att := f.attestation(t, "agent:reviewer", "agent:implementer", "review-run-2", "implementation-run-1")
	verification := verifyAuthorityAttestation(t, f, att)
	if !verification.Approved || verification.Assurance == nil {
		t.Fatalf("expected approved verification with assurance: %+v", verification)
	}
	a := *verification.Assurance
	if a.ReviewerIdentity != "verified" || a.SeparationVerified != "actor-and-execution" || a.VerifiedClaim == nil {
		t.Fatalf("verified claim not disclosed: %+v", a)
	}
	if a.VerifiedClaim.ReviewExecution != "review-run-2" || a.VerifiedClaim.ImplementationExecution != "implementation-run-1" || a.VerifiedClaim.Issuer != f.issuer || a.VerifiedClaim.Audience != "fixture-project" {
		t.Fatalf("claim binding incomplete: %+v", a.VerifiedClaim)
	}
	if a.CognitiveIndependence != "not-observable" {
		t.Fatalf("verified separation must still not claim cognitive independence: %+v", a)
	}
}

func TestVerifiedAssuranceWithoutAValidClaimVerifiesNothing(t *testing.T) {
	f := verifiedAuthorityFixture(t, "different-actor")
	att := f.attestation(t, "agent:reviewer", "agent:implementer", "review-run-2", "implementation-run-1")
	att.Authority = nil
	a := f.store.DescribeReviewAssurance(f.bundle, &att)
	if a.ReviewerIdentity != "not-verified" || a.SeparationVerified != "not-verified" || a.VerifiedClaim != nil {
		t.Fatalf("a missing claim produced verified assurance: %+v", a)
	}
}

func TestReviewCheckCarriesTheSameAssuranceAsVerify(t *testing.T) {
	f := verifiedAuthorityFixture(t, "different-actor")
	enableReviewBundlesForAssurance(t, &f)
	att := f.attestation(t, "agent:reviewer", "agent:implementer", "review-run-2", "implementation-run-1")
	verification := verifyAuthorityAttestation(t, f, att)
	eval, err := f.store.ReviewCheck("spec:backend")
	if err != nil {
		t.Fatal(err)
	}
	if eval.Assurance == nil || verification.Assurance == nil || RenderReviewAssurance(*eval.Assurance) != RenderReviewAssurance(*verification.Assurance) {
		t.Fatalf("review-check and verify disclose the same record differently:\n%+v\n%+v\neval bundle=%q approved=%v state=%q blockers=%v", eval.Assurance, verification.Assurance, eval.BundleID, eval.Approved, eval.BundleState, eval.Blockers)
	}
	_ = time.Now
}

// enableReviewBundlesForAssurance turns on the bundle path of review-check,
// which the authority fixture leaves off, and reseals so the bundle matches
// the policy it was sealed under.
func enableReviewBundlesForAssurance(t *testing.T, f *authorityFixture) {
	t.Helper()
	path := filepath.Join(f.root, ".pose/policy/review.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var policy map[string]any
	if err := json.Unmarshal(raw, &policy); err != nil {
		t.Fatal(err)
	}
	policy["review_bundles"] = true
	policy["review_bundles_adopted_at"] = "2026-01-01"
	encoded, _ := json.MarshalIndent(policy, "", "  ")
	if err := os.WriteFile(path, append(encoded, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
	resealAfterPolicyChange(t, f)
}

func TestLegacyAttemptUnderVerifiedPolicyVerifiesNothing(t *testing.T) {
	a := describeAttemptAssurance(ReviewIdentityAssuranceVerified, "mandatory-human", &ReviewAttempt{Reviewer: "human:someone"})
	if a.IdentityAssurance != "verified" || a.ReviewerIdentity != "not-verified" || a.SeparationVerified != "not-verified" {
		t.Fatalf("a legacy attempt under a verified policy was disclosed as verified: %+v", a)
	}
}
