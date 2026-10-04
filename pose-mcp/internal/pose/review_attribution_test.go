package pose

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// Spec pose-review-attribution-roles: the four cases of the analysis
// (section 8.4) render differently, and the 6.3.0 records stay legacy.

func attributed(reviewer string, a ReviewAttribution) ReviewAttestation {
	att := ReviewAttestation{BundleDigest: "sha256:b", Reviewer: reviewer, Decision: "approved", Criteria: []ReviewCriterion{{ID: "correctness", Disposition: "passed", Evidence: "unit:x"}}}
	a.SchemaVersion = ReviewAttributionSchemaVersion
	att.Attribution = &a
	if a.ConfirmedBy != "" {
		att.Attribution.ConfirmationDigest = ReviewConfirmationDigest(att)
	}
	return att
}

func declaredAssurance() ReviewAssurance {
	return ReviewAssurance{IdentityAssurance: "declared", ReviewerIdentity: "declared"}
}

func TestReviewAttributionDistinguishesTheFourObservedCases(t *testing.T) {
	cases := []struct {
		name string
		att  ReviewAttestation
		want []string
		not  []string
	}{
		{"person wrote the conclusion", attributed("human:p", ReviewAttribution{ConcludedBy: "human:p", AppliedBy: "human:p"}),
			[]string{"concluded by human:p", "no confirmation recorded"}, []string{"agent:"}},
		{"agent wrote, person adopted", attributed("agent:a", ReviewAttribution{PreparedBy: "agent:a", ConcludedBy: "agent:a", ConfirmedBy: "human:p", ConfirmationMode: ReviewConfirmationAdoptedConclusions, AppliedBy: "agent:a"}),
			[]string{"prepared by agent:a", "confirmed by human:p (declared, adopted the conclusions)", "applied by agent:a"}, nil},
		{"person authorized a run", attributed("agent:a", ReviewAttribution{PreparedBy: "agent:a", ConcludedBy: "agent:a", ConfirmedBy: "human:p", ConfirmationMode: ReviewConfirmationAuthorizedOperation, AppliedBy: "agent:a"}),
			[]string{"authorized the operation, did not adopt the conclusions"}, []string{"adopted the conclusions)"}},
		{"agent wrote and applied under a human identity", attributed("human:p", ReviewAttribution{PreparedBy: "agent:a", ConcludedBy: "agent:a", AppliedBy: "agent:a"}),
			[]string{"prepared by agent:a", "concluded by agent:a", "no confirmation recorded", "applied by agent:a"}, []string{"confirmed by"}},
	}
	renders := map[string]bool{}
	for _, tc := range cases {
		if err := ValidateReviewAttribution(tc.att); err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
		r := RenderReviewAttribution(DescribeReviewAttribution(tc.att, declaredAssurance(), nil))
		for _, w := range tc.want {
			if !strings.Contains(r, w) {
				t.Errorf("%s: %q lacks %q", tc.name, r, w)
			}
		}
		for _, n := range tc.not {
			if strings.Contains(r, n) {
				t.Errorf("%s: %q must not contain %q", tc.name, r, n)
			}
		}
		renders[r] = true
	}
	if len(renders) != len(cases) {
		t.Fatalf("the four cases do not render distinctly: %v", renders)
	}
}

func TestReviewConfirmationIsBoundToTheContentItConfirmed(t *testing.T) {
	att := attributed("agent:a", ReviewAttribution{PreparedBy: "agent:a", ConfirmedBy: "human:p", ConfirmationMode: ReviewConfirmationAdoptedConclusions})
	att.Criteria[0].Rationale = "a conclusion edited after the confirmation"
	if err := ValidateReviewAttribution(att); err == nil {
		t.Fatal("a confirmation bound to other content was accepted")
	}
	if d := DescribeReviewAttribution(att, declaredAssurance(), nil); d.ConfirmationAssurance != "unbound" {
		t.Fatalf("an unbound confirmation disclosed as %q", d.ConfirmationAssurance)
	}
}

func TestReviewAttributionRefusesInconsistentConfirmation(t *testing.T) {
	for name, a := range map[string]ReviewAttribution{
		"confirmed without mode":    {ConfirmedBy: "human:p"},
		"mode without confirmer":    {ConcludedBy: "agent:a", ConfirmationMode: ReviewConfirmationAdoptedConclusions},
		"unknown mode":              {ConfirmedBy: "human:p", ConfirmationMode: "nodded"},
		"empty block":               {},
		"malformed principal":       {PreparedBy: "claude"},
		"digest without confirmer":  {ConcludedBy: "agent:a", ConfirmationDigest: "sha256:x"},
	} {
		a.SchemaVersion = ReviewAttributionSchemaVersion
		att := ReviewAttestation{Attribution: &a}
		if err := ValidateReviewAttribution(att); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
}

func TestOnlyAVerifiedHumanClaimVerifiesAConfirmation(t *testing.T) {
	att := attributed("human:p", ReviewAttribution{PreparedBy: "agent:a", ConfirmedBy: "human:p", ConfirmationMode: ReviewConfirmationAdoptedConclusions})
	verified := ReviewAssurance{ReviewerIdentity: "verified", VerifiedClaim: &ReviewAssuranceClaim{Principal: "human:p", Role: "human"}}
	if d := DescribeReviewAttribution(att, verified, nil); d.ConfirmationAssurance != "verified" {
		t.Fatalf("a verified human claim for the confirmer was not used: %+v", d)
	}
	agentClaim := ReviewAssurance{ReviewerIdentity: "verified", VerifiedClaim: &ReviewAssuranceClaim{Principal: "agent:a", Role: "agent"}}
	if d := DescribeReviewAttribution(att, agentClaim, nil); d.ConfirmationAssurance != "declared" {
		t.Fatalf("a claim for another principal verified the confirmation: %+v", d)
	}
}

// The 6.3.0 cycle: human:oseias as reviewer, conclusions written and applied
// by the agent, no attribution block. Schema-valid, and must not read as a
// human review.
func TestThe630CycleAttestationsReadAsLegacyNotHumanReview(t *testing.T) {
	raw := `{"schema_version":2,"attestation_id":"rva-0000000000000000","bundle_id":"rvb-0000000000000000","bundle_digest":"sha256:b","reviewer":"human:oseias","decision":"approved","criteria":[{"id":"correctness","disposition":"passed","evidence":"unit:x","rationale":"written by the agent"}],"findings":[],"attested_at":"2026-10-02T10:00:00Z"}`
	var att ReviewAttestation
	if err := json.Unmarshal([]byte(raw), &att); err != nil {
		t.Fatal(err)
	}
	assurance := Store{}.describeReviewAssurance(declaredBundle("same-actor-separate-execution"), &att)
	attribution := DescribeReviewAttribution(att, assurance, nil)
	assurance.Attribution = &attribution
	r := RenderReviewAssurance(assurance)
	for _, w := range []string{"human:oseias — declared identity", "attribution legacy-undifferentiated", "verified not-verified"} {
		if !strings.Contains(r, w) {
			t.Errorf("render %q lacks %q", r, w)
		}
	}
	for _, n := range []string{"confirmed by", "verified human", "concluded by human"} {
		if strings.Contains(r, n) {
			t.Errorf("render %q reads as a human review: contains %q", r, n)
		}
	}
}

func TestAnAttestationWithoutAttributionSerializesAsBefore(t *testing.T) {
	att := ReviewAttestation{Reviewer: "agent:a", Decision: "approved"}
	raw, _ := json.Marshal(att)
	if strings.Contains(string(raw), "attribution") {
		t.Fatalf("an unattributed record gained a key, changing legacy digests and signatures: %s", raw)
	}
}

func TestAttributionSupplementClarifiesWithoutAlteringOrConfirming(t *testing.T) {
	f := verifiedAuthorityFixture(t, "same-actor-separate-execution")
	att := f.attestation(t, "agent:reviewer", "agent:implementer", "review-run-2", "implementation-run-1")
	recorded, err := f.store.RecordReviewAttestation(att, f.now.Add(2*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	before, _ := f.store.LoadReviewAttestation(recorded.AttestationID)
	if _, err := f.store.RecordReviewAttributionSupplement(ReviewAttributionSupplement{AttestationID: recorded.AttestationID, RecordedBy: "agent:auditor", Note: "n", Attribution: ReviewAttribution{ConfirmedBy: "human:p", ConfirmationMode: ReviewConfirmationAdoptedConclusions}}, f.now); err == nil {
		t.Fatal("a supplement added a confirmation after the fact")
	}
	if _, err := f.store.RecordReviewAttributionSupplement(ReviewAttributionSupplement{AttestationID: recorded.AttestationID, RecordedBy: "agent:auditor", Attribution: ReviewAttribution{PreparedBy: "agent:a"}}, f.now); err == nil {
		t.Fatal("a supplement without a note was accepted")
	}
	sup, err := f.store.RecordReviewAttributionSupplement(ReviewAttributionSupplement{AttestationID: recorded.AttestationID, RecordedBy: "agent:auditor", Note: "cycle report section 3", Evidence: []string{"report:cycle"}, Attribution: ReviewAttribution{PreparedBy: "agent:a", ConcludedBy: "agent:a", AppliedBy: "agent:a"}}, f.now)
	if err != nil || !strings.HasPrefix(sup.SupplementID, "ras-") {
		t.Fatalf("supplement not recorded: %v %+v", err, sup)
	}
	after, _ := f.store.LoadReviewAttestation(recorded.AttestationID)
	a, _ := json.Marshal(before)
	b, _ := json.Marshal(after)
	if string(a) != string(b) {
		t.Fatal("recording a supplement altered the attestation")
	}
	list, _ := f.store.ListReviewAttributionSupplements(recorded.AttestationID)
	if len(list) != 1 {
		t.Fatalf("supplements: %+v", list)
	}
	assurance := f.store.DescribeReviewAssurance(f.bundle, &after)
	if assurance.Attribution == nil || len(assurance.Attribution.Supplements) != 1 || !strings.Contains(RenderReviewAssurance(assurance), "1 attribution supplement(s)") {
		t.Fatalf("the supplement is not disclosed: %+v", assurance.Attribution)
	}
}
