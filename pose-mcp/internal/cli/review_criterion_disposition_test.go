package cli

import (
	"strings"
	"testing"

	posemodel "github.com/harne8/pose-mcp/internal/pose"
)

func dispositionPlan() posemodel.ReviewPlan {
	return posemodel.ReviewPlan{Criteria: []posemodel.ReviewPlanCriterion{
		{ID: "correctness", Required: true, EvidenceClasses: []string{"unit"}},
		{ID: "frontend-accessibility", Required: true, EvidenceClasses: []string{"a11y"}},
		{ID: "optional-one", Required: false},
	}}
}

// The engine has always modelled three dispositions and this command could
// write only one, so a criterion that genuinely does not apply had no honest
// expression: claim it passed on evidence that does not support it, or leave
// the closeout blocked. `auto-attest` can record not-applicable; a human
// reviewer should not have fewer options than the automated path.
func TestAttestRecordsNotApplicableWithARationale(t *testing.T) {
	criteria, err := reviewCriterionDispositions(dispositionPlan(), []string{"unit:api/go/test"},
		[]string{"frontend-accessibility|not-applicable||the change has no user-visible surface"})
	if err != nil {
		t.Fatal(err)
	}
	byID := map[string]posemodel.ReviewCriterion{}
	for _, c := range criteria {
		byID[c.ID] = c
	}
	if len(criteria) != 2 {
		t.Fatalf("optional criteria must stay out of the attestation: %+v", criteria)
	}
	na := byID["frontend-accessibility"]
	if na.Disposition != "not-applicable" || na.Rationale == "" || na.Evidence != "" {
		t.Errorf("not-applicable criterion = %+v", na)
	}
	// Everything not named keeps the default, so the override is a scalpel.
	if got := byID["correctness"]; got.Disposition != "passed" || got.Evidence != "unit:api/go/test" {
		t.Errorf("an unnamed criterion changed: %+v", got)
	}
}

// The builder decodes; it does not judge. A malformed flag is still the
// command's to refuse, because nothing can be built from it.
func TestAttestRefusesOnlyMalformedCriterionFlags(t *testing.T) {
	for _, arg := range []string{"correctness", "correctness|passed", "Not A Slug|passed||", "correctness|passed|a|b|c"} {
		if _, err := reviewCriterionDispositions(dispositionPlan(), []string{"unit:api/go/test"}, []string{arg}); err == nil || !strings.Contains(err.Error(), "must be ID|disposition|evidence|rationale") {
			t.Fatalf("%q: err = %v", arg, err)
		}
	}
}

// Every answer the reviewer gave reaches the domain as given — an unknown
// criterion, a duplicate, an invalid disposition, a finding nobody recorded —
// so the domain writer, not this builder, names what is wrong (spec
// pose-review-approval-domain-preflight R7). The verdicts themselves are
// exercised through the command in TestReviewDomainPreflightCLISemanticRefusalsComeFromTheDomain.
func TestAttestPassesEveryAnswerToTheDomain(t *testing.T) {
	criteria, err := reviewCriterionDispositions(dispositionPlan(), []string{"unit:api/go/test"}, []string{
		"correctness|finding|f-nope|",
		"correctness|passed|unit:api/go/test|",
		"invented|probably-fine||",
	})
	if err != nil {
		t.Fatal(err)
	}
	got := []string{}
	for _, c := range criteria {
		got = append(got, c.ID+"|"+c.Disposition+"|"+c.Evidence)
	}
	want := []string{"correctness|finding|f-nope", "correctness|passed|unit:api/go/test", "frontend-accessibility|passed|unit:api/go/test", "invented|probably-fine|"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("criteria = %v, want %v", got, want)
	}
}

// A finding criterion names the finding the attestation records; the builder
// keeps both, and the domain ties them together.
func TestAttestTiesAFindingCriterionToARecordedFinding(t *testing.T) {
	criteria, err := reviewCriterionDispositions(dispositionPlan(), []string{"unit:api/go/test"},
		[]string{"correctness|finding|f-1|the race is filed and fixed"})
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range criteria {
		if c.ID == "correctness" && (c.Disposition != "finding" || c.Evidence != "f-1") {
			t.Errorf("criterion = %+v", c)
		}
	}
}
