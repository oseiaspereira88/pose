package pose

import (
	"strings"
	"testing"
	"time"
)

// Spec pose-review-declared-limits-residuals R1–R4: a comment between the
// heading's words refuses the section; inline code spans are read as
// Markdown reads them; indented code opens no fence; the next section's
// heading closes the section whatever comment it holds; problems are bounded.
func TestAcceptedLimitsResiduals(t *testing.T) {
	limit := "- L1: An escape is open. Accepted because it is rare.\n"
	if limits, problems := ParseAcceptedLimits("### Accepted <!-- x --> limits\n" + limit); len(limits) != 0 || len(problems) == 0 {
		t.Fatalf("R1: a comment in the heading words: %+v %v", limits, problems)
	}
	if limits, problems := ParseAcceptedLimits("A span ``a ` <!-- b`` is code.\n\n### Accepted limits\n" + limit); len(problems) != 0 || len(limits) != 1 {
		t.Fatalf("R2: a double-backtick span: %+v %v", limits, problems)
	}
	if _, problems := ParseAcceptedLimits("A stray ` backtick, then <!-- an open comment\n\n### Accepted limits\n" + limit); !strings.Contains(strings.Join(problems, ";"), "unterminated") {
		t.Fatalf("R2: a stray backtick hid an open comment: %v", problems)
	}
	if limits, problems := ParseAcceptedLimits("Example:\n\n    ```\n    code\n\n### Accepted limits\n" + limit); len(problems) != 0 || len(limits) != 1 {
		t.Fatalf("R3: indented code opened a fence: %+v %v", limits, problems)
	}
	if limits, problems := ParseAcceptedLimits("### Accepted limits\n" + limit + "### Next <!-- note -->\nProse.\n"); len(problems) != 0 || len(limits) != 1 {
		t.Fatalf("R4: a comment on the next heading: %+v %v", limits, problems)
	}
	if _, problems := ParseAcceptedLimits("### Accepted limits\n" + strings.Repeat("prose line\n", 60)); len(problems) > acceptedLimitProblemLimit+1 || !strings.Contains(problems[len(problems)-1], "more problems not shown") {
		t.Fatalf("R4: problems are not bounded: %d", len(problems))
	}
}

// R6: an attestation of any decision naming a limit the bundle did not seal
// is refused.
func TestNoAttestationNamesAnUnsealedLimit(t *testing.T) {
	bundle := ReviewBundle{BundleID: "rvb-x"}
	bundle.Payload.Plan.AcceptedLimits = []ReviewAcceptedLimit{{ID: "L1", Text: "x"}}
	att := ReviewAttestation{Decision: "changes-requested", Findings: []ReviewFinding{{ID: "n1", Disposition: "within-limit", Limit: "L9"}, {ID: "n2", Disposition: "within-limit", Limit: "L1"}}}
	got := strings.Join(Store{Root: t.TempDir()}.reviewApprovalAdmissionBlockers(bundle, att, time.Now()), ";")
	if !strings.Contains(got, "n1 is within-limit") || strings.Contains(got, "n2") {
		t.Fatalf("a non-approving attestation: %s", got)
	}
}

// R8: a run-backed record an earlier engine derived within an unsealed limit
// still matches its draft.
func TestAnEarlierWithinLimitDerivationStillMatches(t *testing.T) {
	bundle := ReviewBundle{BundleID: "rvb-x"}
	draft := ReviewDraft{Schema: ReviewDraftSchema, Decision: "changes-requested", Findings: []ReviewDraftFinding{{ID: "N1", Severity: "low", Disposition: "follow-up", Summary: "inside", Limit: "L9"}}}
	run := ReviewRun{RunID: "rrn-x", Provenance: &ReviewRunProvenance{TemplateVersion: reviewBriefTemplateAcceptedLimits}, Output: "```pose-draft\n" + string(mustJSON(t, draft)) + "\n```\n"}
	legacy := reviewAttestationFromDraftWith(run, bundle, draft, "agent:t", false)
	if why := runConclusionDiffers(run, bundle, legacy); why != "" {
		t.Fatalf("an earlier derivation differs: %s", why)
	}
	current := reviewAttestationFromDraft(run, bundle, draft, "agent:t")
	if why := runConclusionDiffers(run, bundle, current); why != "" {
		t.Fatalf("the current derivation differs: %s", why)
	}
}
