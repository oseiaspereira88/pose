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
	// Review round 1 of spec pose-review-declared-limits-followups, N1: a
	// limit that cannot be sealed is said in the plan, not dropped silently.
	if err := os.WriteFile(path, []byte(strings.Replace(string(raw), "### Technical risks", "### Accepted limits\n- L1: An escape is open. Accepted because <!-- x --> it is rare.\n\n### Technical risks", 1)), 0o644); err != nil {
		t.Fatal(err)
	}
	refused, err := store.ReviewPlan("spec:backend")
	if err != nil || len(refused.AcceptedLimits) != 0 || !strings.Contains(strings.Join(refused.Warnings, ";"), "accepted limits of spec backend are not all sealed") {
		t.Fatalf("a refused limit was not said in the plan: %+v %v %v", refused.AcceptedLimits, refused.Warnings, err)
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
	body := "### Accepted limits\n- L1: one escape. Accepted because it is rare.\n"
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

// Review round 6, N1: the bundle's plan digest covers the sealed limits, and
// a plan without limits keeps the digest it had before them.
func TestTheBundlePlanDigestCoversTheAcceptedLimits(t *testing.T) {
	var plan ReviewPlan
	plan.SchemaVersion, plan.Independence = 1, "different-actor"
	empty, err := reviewBundlePlanDigest(plan)
	if err != nil {
		t.Fatal(err)
	}
	legacy, _ := digestJSON(struct {
		SchemaVersion             int                   `json:"schema_version"`
		PolicySchemaVersion       int                   `json:"policy_schema_version"`
		BaseProfile               string                `json:"base_profile"`
		Independence              string                `json:"independence"`
		Components                []ReviewPlanComponent `json:"components"`
		SelectedProfiles          []ReviewPlanProfile   `json:"selected_profiles"`
		Criteria                  []ReviewPlanCriterion `json:"criteria"`
		Tools                     []ReviewPlanTool      `json:"tools"`
		Limitations               []string              `json:"limitations,omitempty"`
		DelegatedRunProofRequired bool                  `json:"delegated_run_proof_required,omitempty"`
	}{1, 0, "", "different-actor", []ReviewPlanComponent{}, []ReviewPlanProfile{}, []ReviewPlanCriterion{}, []ReviewPlanTool{}, nil, false})
	if empty != legacy {
		t.Fatal("a plan without limits changed its bundle plan digest")
	}
	plan.AcceptedLimits = []ReviewAcceptedLimit{{ID: "L1", Text: "An escape. Accepted because a shim covers it."}}
	one, _ := reviewBundlePlanDigest(plan)
	plan.AcceptedLimits[0].Text = "An escape. Accepted because nothing else covers it."
	other, _ := reviewBundlePlanDigest(plan)
	if one == empty || one == other {
		t.Fatalf("the bundle plan digest does not cover the accepted limits: %s %s %s", empty, one, other)
	}
}

// Review round 8, N1, N2: a wrapped entry keeps its continuation; fenced and
// commented examples are not sealed.
func TestAcceptedLimitsReadWrappedEntriesAndSkipExamples(t *testing.T) {
	// A commented example outside the section is not read; one inside the
	// section refuses it (spec pose-review-declared-limits-followups R2).
	body := "### Accepted limits\n- L1: An escape. Accepted because a shim covers it\n  only on Linux and only for children.\n### Notes\n<!--\n### Accepted limits\n- L3: A hidden one. Accepted because nobody sees it.\n-->\n```\n### Accepted limits\n- L2: An example. Accepted because it is only shown.\n```\n"
	limits, problems := ParseAcceptedLimits(body)
	if len(problems) != 0 || len(limits) != 1 {
		t.Fatalf("limits %+v problems %v", limits, problems)
	}
	if !strings.HasSuffix(limits[0].Text, "only on Linux and only for children.") {
		t.Fatalf("the continuation line was cut: %q", limits[0].Text)
	}
}

// Review round 9, N2, N4: an entry ends where its list item ends; a wrapped
// '#' is no heading; a nested list or a borrowed reason is refused and not
// sealed; fences close only with their own marker.
func TestAcceptedLimitsEndWhereTheirItemEnds(t *testing.T) {
	cases := []struct {
		name, body string
		want       map[string]string
		problem    string
	}{
		{"wrapped hash", "### Accepted limits\n- L1: An escape is open. Accepted because see issue\n  #42 only on Linux.\n", map[string]string{"L1": "An escape is open. Accepted because see issue #42 only on Linux."}, ""},
		{"indented paragraph", "### Accepted limits\n- L1: An escape is open. Accepted because it is rare\n\n  and only on Linux.\n", map[string]string{"L1": "An escape is open. Accepted because it is rare and only on Linux."}, ""},
		{"nested list", "### Accepted limits\n- L1: An escape is open. Accepted because it is rare\n  - only on Linux\n", map[string]string{}, "nested list"},
		{"later paragraph", "### Accepted limits\n- L1: An escape is open. Accepted because it is rare.\n\nSome note that follows the list.\n* L2: Another escape. Accepted because it is rarer.\n", map[string]string{"L2": "Another escape. Accepted because it is rarer."}, "holds only limits"},
		{"borrowed reason", "### Accepted limits\n- L1: no reason here\n\nSome unrelated note. Accepted because other words follow.\n", map[string]string{}, "holds only limits"},
		// Review round 10, N1: a lazy continuation and a fence inside an
		// item are reported, and the limit is not sealed in part.
		{"lazy continuation", "### Accepted limits\n- L1: An escape is open. Accepted because a shim covers it\nonly on Linux and only for children.\n", map[string]string{}, "is not sealed"},
		{"fence in item", "### Accepted limits\n- L1: An escape is open. Accepted because it is rare\n  ```\n  code\n  ```\n  and only on Linux.\n", map[string]string{}, "code fence"},
		{"four backticks", "### Accepted limits\n````\n```\n- L1: An example escape. Accepted because it is only shown.\n```\n````\n", map[string]string{}, "code fence"},
		// Review round 10, N2: valid list forms are read; others are reported.
		{"indented items", "  ### Accepted limits\n  - L1: An escape is open. Accepted because it is rare.\n+ L2: Another escape. Accepted because it is rarer.\n", map[string]string{"L1": "An escape is open. Accepted because it is rare.", "L2": "Another escape. Accepted because it is rarer."}, ""},
		{"ordered item", "### Accepted limits\n1. L1: An escape is open. Accepted because it is rare.\n", map[string]string{}, "is not written as"},
		// Review round 10, N4: inline code is no fence; an unterminated
		// comment is reported.
		{"inline backticks", "### Notes\n```a``` is inline\n### Accepted limits\n- L1: An escape is open. Accepted because it is rare.\n", map[string]string{"L1": "An escape is open. Accepted because it is rare."}, ""},
		{"unterminated comment", "<!-- draft\n### Accepted limits\n- L1: An escape is open. Accepted because it is rare.\n", map[string]string{}, "unterminated HTML comment"},
	}
	for _, c := range cases {
		limits, problems := ParseAcceptedLimits(c.body)
		got := map[string]string{}
		for _, l := range limits {
			got[l.ID] = l.Text
		}
		if len(got) != len(c.want) {
			t.Fatalf("%s: limits %+v, want %v (problems %v)", c.name, limits, c.want, problems)
		}
		for id, text := range c.want {
			if got[id] != text {
				t.Fatalf("%s: %s = %q, want %q", c.name, id, got[id], text)
			}
		}
		joined := strings.Join(problems, "; ")
		if (c.problem == "") != (joined == "") || !strings.Contains(joined, c.problem) {
			t.Fatalf("%s: problems %q, want %q", c.name, joined, c.problem)
		}
	}
}

// Spec pose-review-declared-limits-followups R1, R2: a heading-like line
// indented under a limit is reported, not taken for a heading that cuts it;
// any unterminated comment, and any comment marker in the section, inline
// code included, is reported and nothing is sealed cut by it.
func TestAcceptedLimitsRefuseWhatWouldCutThem(t *testing.T) {
	cases := []struct{ name, body, problem string }{
		{"indented heading", "### Accepted limits\n- L1: An escape is open. Accepted because a shim covers it\n  # of cases on Linux and only for children.\n", "heading inside a limit"},
		{"second unterminated comment", "<!-- closed -->\n### Accepted limits\n- L1: An escape is open. Accepted because it is rare <!-- and so on\n", "HTML comment"},
		{"unterminated after a closed one", "<!-- closed -->\n<!-- open\n### Accepted limits\n- L1: An escape is open. Accepted because it is rare.\n", "unterminated HTML comment"},
		{"unterminated after a closed one, in code", "A spec may name `<!--` in its prose.\n<!-- closed -->\n### Accepted limits\n- L1: An escape is open. Accepted because it is rare <!-- and so on\n", "HTML comment"},
		{"comment in inline code", "### Accepted limits\n- L1: An escape is open. Accepted because `<!-- x -->` hides it.\n", "comment marker"},
	}
	for _, c := range cases {
		limits, problems := ParseAcceptedLimits(c.body)
		if len(limits) != 0 || !strings.Contains(strings.Join(problems, "; "), c.problem) {
			t.Fatalf("%s: limits %+v problems %v", c.name, limits, problems)
		}
	}
}

// Spec pose-review-declared-limits-followups R4: a follow-up naming a limit
// the bundle did not seal is not recorded within-limit, whatever the draft's
// decision; one naming a sealed limit is.
func TestOnlyASealedLimitIsRecordedWithinLimit(t *testing.T) {
	bundle := ReviewBundle{BundleID: "rvb-x"}
	bundle.Payload.Plan.AcceptedLimits = []ReviewAcceptedLimit{{ID: "L1", Text: "x"}}
	for _, decision := range []string{"approved", "changes-requested", "rejected"} {
		draft := ReviewDraft{Decision: decision, Findings: []ReviewDraftFinding{
			{ID: "N1", Severity: "low", Disposition: "follow-up", Summary: "inside", Limit: "L1"},
			{ID: "N2", Severity: "low", Disposition: "follow-up", Summary: "elsewhere", Limit: "L9"},
		}}
		att := reviewAttestationFromDraft(ReviewRun{RunID: "rrn-x"}, bundle, draft, "agent:t")
		got := map[string]string{}
		for _, f := range att.Findings {
			got[f.ID] = f.Disposition + "/" + f.Limit
		}
		if got["N1"] != "within-limit/L1" || strings.HasPrefix(got["N2"], "within-limit") || strings.HasSuffix(got["N2"], "L9") {
			t.Fatalf("%s: %v", decision, got)
		}
	}
}

// A spec that names the comment marker in inline code outside the section
// hides nothing and is not reported (found by the ready check of this spec).
func TestAcceptedLimitsIgnoreAMarkerNamedInCode(t *testing.T) {
	body := "## Requirements\n- R2: An unterminated `<!--` anywhere is reported.\n\n### Accepted limits\n- L1: An escape is open. Accepted because it is rare.\n"
	limits, problems := ParseAcceptedLimits(body)
	if len(problems) != 0 || len(limits) != 1 {
		t.Fatalf("a marker named in code: limits %+v problems %v", limits, problems)
	}
}

// Review round 1 of spec pose-review-declared-limits-followups, N2–N5: a
// comment on the heading line, before it, or in a limit after an indented
// heading refuses the section; a fenced or commented example elsewhere
// leaves the real section whole; an indented heading under an indented
// marker is reported.
func TestAcceptedLimitsReadCommentsWhereTheSectionIs(t *testing.T) {
	refused := []struct{ name, body string }{
		{"trailing on the heading", "### Accepted limits <!-- note -->\n- L1: An escape is open. Accepted because it is rare\n<!-- hidden -->  and only on Linux.\n"},
		{"leading on the heading", "<!-- note --> ### Accepted limits\n- L1: An escape is open. Accepted because it is rare.\n"},
		{"ending on the heading", "<!--\nold notes\n--> ### Accepted limits\n- L1: An escape is open. Accepted because it is rare.\n"},
		{"after an indented heading", "### Accepted limits\n- L1: An escape is open. Accepted because it is rare.\n  # not a heading\n- L2: Another escape. Accepted because <!-- cut --> it is rarer.\n"},
	}
	for _, c := range refused {
		if limits, problems := ParseAcceptedLimits(c.body); len(limits) != 0 || len(problems) == 0 {
			t.Fatalf("%s: limits %+v problems %v", c.name, limits, problems)
		}
	}
	whole := []struct{ name, body string }{
		{"fenced example with a comment", "### Notes\n```\n### Accepted limits\n- L9: Example. Accepted because <!-- x --> shown.\n```\n### Accepted limits\n- L1: An escape is open. Accepted because it is rare.\n"},
		{"unterminated marker in a fence", "### Notes\n```\n<!-- an opening shown as code\n```\n### Accepted limits\n- L1: An escape is open. Accepted because it is rare.\n"},
		{"second comment on a line", "<!-- a --> text <!-- ### Accepted limits\n- L9: Example. Accepted because hidden. -->\n### Accepted limits\n- L1: An escape is open. Accepted because it is rare.\n"},
	}
	for _, c := range whole {
		limits, problems := ParseAcceptedLimits(c.body)
		if len(problems) != 0 || len(limits) != 1 || limits[0].ID != "L1" {
			t.Fatalf("%s: limits %+v problems %v", c.name, limits, problems)
		}
	}
	indented := "### Accepted limits\n  - L1: An escape is open. Accepted because a shim covers it\n  # of cases on Linux.\n"
	if limits, problems := ParseAcceptedLimits(indented); len(limits) != 0 || !strings.Contains(strings.Join(problems, ";"), "heading inside a limit") {
		t.Fatalf("an indented heading under an indented marker: %+v %v", limits, problems)
	}
}
