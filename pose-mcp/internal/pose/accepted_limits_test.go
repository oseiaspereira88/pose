package pose

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const limitsSection = `
### Accepted limits
- L1: A process that clears its environment escapes. Accepted because the shim covers it.
- L2: Windows is untested here. Accepted because the host is Linux.
`

// Spec pose-review-declared-threat-model R1: the section is read as
// written, and what does not say why it is accepted is reported.
func TestAcceptedLimitsAreParsed(t *testing.T) {
	limits, problems := ParseAcceptedLimits("## 3. Technical Plan\n" + limitsSection + "\n### Other\n- L9: not a limit here.\n")
	if len(problems) != 0 || len(limits) != 2 || limits[0].ID != "L1" || limits[1].ID != "L2" {
		t.Fatalf("limits %+v problems %v", limits, problems)
	}
	_, problems = ParseAcceptedLimits("### Accepted limits\n- L1: no reason given.\n- L1: twice. Accepted because yes.\n- X1: wrong id. Accepted because no.\n- L3: a marker with nothing after it. Accepted because\n")
	joined := strings.Join(problems, "\n")
	for _, want := range []string{"L1 does not say why it is accepted", "L1 is declared twice", "is not written as", "L3 does not say why it is accepted"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("missing problem %q in %v", want, problems)
		}
	}
}

// R1: the review plan seals the limits the spec declares, and they enter the
// plan digest; a spec without limits keeps its digest.
func TestTheReviewPlanSealsTheAcceptedLimits(t *testing.T) {
	root, store := reviewBundleFixture(t)
	before, err := store.ReviewPlan("spec:backend")
	if err != nil || len(before.AcceptedLimits) != 0 {
		t.Fatalf("plan without limits: %+v %v", before.AcceptedLimits, err)
	}
	path := filepath.Join(root, ".pose/specs/backend/spec.md")
	raw, _ := os.ReadFile(path)
	if err := os.WriteFile(path, []byte(strings.Replace(string(raw), "### Technical risks", strings.TrimPrefix(limitsSection, "\n")+"\n### Technical risks", 1)), 0o644); err != nil {
		t.Fatal(err)
	}
	after, err := store.ReviewPlan("spec:backend")
	if err != nil || len(after.AcceptedLimits) != 2 || after.AcceptedLimits[0].ID != "L1" {
		t.Fatalf("plan with limits: %+v %v", after.AcceptedLimits, err)
	}
	without := after
	without.AcceptedLimits = nil
	if digest, err := digestReviewPlan(without); err != nil || digest == after.PlanDigest {
		t.Fatalf("the accepted limits did not enter the plan digest: %v", err)
	}
}

// R2, R3: a follow-up the reviewer classifies inside a sealed limit is
// recorded within-limit, naming it, and does not block an approval whatever
// its severity; a limit the bundle did not seal accepts nothing.
func TestAFindingWithinASealedLimitDoesNotBlockApproval(t *testing.T) {
	var bundle ReviewBundle
	bundle.Payload.Plan.AcceptedLimits = []ReviewAcceptedLimit{{ID: "L1", Text: "x. Accepted because y."}}
	draft := ReviewDraft{Decision: "approved", Findings: []ReviewDraftFinding{
		{ID: "N1", Severity: "high", Disposition: "follow-up", Summary: "an orphan that clears its environment escapes; that is L1", Limit: "L1"},
	}}
	att := reviewAttestationFromDraft(ReviewRun{RunID: "rrn-0000000000000001"}, bundle, draft, "agent:reviewer")
	if len(att.Findings) != 1 || att.Findings[0].Disposition != "within-limit" || att.Findings[0].Limit != "L1" {
		t.Fatalf("the finding was not recorded within its limit: %+v", att.Findings)
	}
	att.Decision = "approved"
	if blockers := reviewFindingDecisionBlockers(bundle, att); len(blockers) != 0 {
		t.Fatalf("a finding within a sealed limit blocked the approval: %v", blockers)
	}
	att.Findings[0].Limit = "L9"
	if blockers := strings.Join(reviewFindingDecisionBlockers(bundle, att), "\n"); !strings.Contains(blockers, "L9, which this bundle did not seal") {
		t.Fatalf("an unsealed limit was accepted: %s", blockers)
	}
	att.Findings[0].Limit, att.Findings[0].Rationale = "L1", "  \t "
	if blockers := strings.Join(reviewFindingDecisionBlockers(bundle, att), "\n"); !strings.Contains(blockers, "without saying why") {
		t.Fatalf("a finding within a limit without a reason was accepted: %s", blockers)
	}
}

// Review round 1, N1: across the specs of a composite scope each limit is
// qualified by its spec, so a finding names exactly one.
func TestLimitsAcrossSpecsAreQualified(t *testing.T) {
	body := "### Accepted limits\n- L1: one. Accepted because it is rare.\n"
	single := acceptedLimitsOfSpecs([]Spec{{Slug: "alpha", Body: body}})
	if len(single) != 1 || single[0].ID != "L1" || single[0].Spec != "" {
		t.Fatalf("a single spec's limit was qualified: %+v", single)
	}
	both := acceptedLimitsOfSpecs([]Spec{{Slug: "alpha", Body: body}, {Slug: "beta", Body: body}})
	if len(both) != 2 || both[0].ID != "alpha#L1" || both[1].ID != "beta#L1" || both[1].Spec != "beta" {
		t.Fatalf("limits across specs are ambiguous: %+v", both)
	}
}

// Review round 2, N1: punctuation is not a reason, and a reason may start
// with a one-letter word.
func TestAnAcceptedLimitNeedsWordsAsItsReason(t *testing.T) {
	for text, want := range map[string]bool{
		"x. Accepted because ..":                                          false,
		"x. Accepted because -- !":                                        false,
		"x. Accepted because rare":                                        false,
		"An escape. Accepted because a shim covers it.":                   true,
		"Uma fuga. Aceito porque o shim cobre o caso.":                    true,
		"Not accepted because this would violate scope.":                  false,
		"Accepted because it is rare.":                                    false,
		"An escape. Accepted because. Then unrelated prose follows here.": false,
		"An escape. Accepted because a shim covers it. More context.":     true,
		"An escape. Accepted because ; later prose has words.":            false,
		"X. Accepted because a shim covers it.":                           false,
		"x, accepted because it is rare.":                                 false,
	} {
		if got := hasAcceptedLimitReason(text); got != want {
			t.Errorf("%q: reason=%t, want %t", text, got, want)
		}
	}
}
