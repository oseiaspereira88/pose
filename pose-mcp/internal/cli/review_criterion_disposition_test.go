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

func TestAttestRejectsDispositionsItCannotStandBehind(t *testing.T) {
	for _, tc := range []struct{ name, arg, want string }{
		{"no rationale", "frontend-accessibility|not-applicable||", "needs a rationale"},
		{"unknown disposition", "correctness|probably-fine||", "invalid disposition"},
		{"criterion not in the plan", "invented|not-applicable||why", "does not match a required criterion"},
		{"optional criterion", "optional-one|not-applicable||why", "does not match a required criterion"},
		{"duplicate", "correctness|passed|unit:api/go/test|", "duplicate"},
		{"malformed", "correctness", "must be ID|disposition|evidence|rationale"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw := []string{tc.arg}
			if tc.name == "duplicate" {
				raw = append(raw, tc.arg)
			}
			_, err := reviewCriterionDispositions(dispositionPlan(), []string{"unit:api/go/test"}, raw)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("err = %v, want one mentioning %q", err, tc.want)
			}
		})
	}
}
