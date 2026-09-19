package pose

import (
	"strings"
	"testing"
	"time"
)

// These are the inversions of the four characterization tests recorded against
// 5.0.8 in the ABM review. Each one asserted the defect and passed; here the
// expectation is reversed, so the same fixture now proves the refusal
// (spec pose-abm-review-soundness).

// abmMixedProfileFixture writes a plan with one criterion a check answers and
// one only a reviewer can, which is the shape every distributed profile has:
// `spec-closeout` carries five judged criteria out of seven, `roadmap-outcome`
// six out of six.
func abmMixedProfileFixture(t *testing.T) (string, Store) {
	t.Helper()
	root, store := reviewBundleFixture(t)
	writeReviewFixture(t, root, ".pose/review-profiles/spec-closeout.json", `{
  "schema_version":2,"id":"spec-closeout","version":2,"scope":"spec",
  "criteria":[
    {"id":"correctness","description":"Behavior is correct.","evidence_classes":["integration"]},
    {"id":"operability","description":"Errors and remediation are actionable."}
  ]
}`)
	return root, store
}

// F01. A criterion no registered check reports on was answered from whichever
// sealed result happened to be first in the bundle, with an empty rationale,
// and the bundle verified approved.
func TestABMReviewSoundnessJudgmentIsNotAutoApproved(t *testing.T) {
	root, store := reviewBundleFixture(t)
	writeReviewFixture(t, root, ".pose/review-profiles/spec-closeout.json", `{
  "schema_version":2,"id":"spec-closeout","version":2,"scope":"spec",
  "criteria":[{"id":"security","description":"Authority boundaries are safe."}],
  "tools":[{"id":"review-check","requiredness":"required","criteria":["security"]}]
}`)
	now := time.Date(2026, 9, 18, 15, 0, 0, 0, time.UTC)
	bundle, err := store.SealReviewBundle("spec:backend", now)
	if err != nil {
		t.Fatal(err)
	}
	prepared, err := store.PrepareReviewAttestation(bundle.BundleID, "agent:audit", now)
	if err != nil {
		t.Fatal(err)
	}
	if prepared.Complete {
		t.Fatal("a plan whose only criterion is judgment cannot be completely prepared by automation")
	}
	if len(prepared.Pending) != 1 || prepared.Pending[0].Criterion != "security" {
		t.Fatalf("expected `security` to be pending, got %+v", prepared.Pending)
	}
	for _, criterion := range prepared.Attestation.Criteria {
		if criterion.ID == "security" {
			t.Fatalf("a judgment criterion must carry no prepared disposition: %+v", criterion)
		}
	}
	if _, err := store.AutoAttestReviewBundle(bundle.BundleID, "agent:audit", true, now.Add(time.Minute)); err == nil {
		t.Fatal("--apply must refuse while judgment is pending")
	} else if !strings.Contains(err.Error(), "security") {
		t.Fatalf("the refusal must name the criterion: %v", err)
	}

	// Answered explicitly, the same bundle verifies. The contract asks for a
	// conclusion, not for more ceremony.
	att := approvedBundleAttestation(bundle, "agent:audit")
	if _, err := store.RecordReviewAttestation(att, now.Add(2*time.Minute)); err != nil {
		t.Fatal(err)
	}
	verification, err := store.VerifyReviewBundle("spec:backend")
	if err != nil {
		t.Fatal(err)
	}
	if !verification.Approved {
		t.Fatalf("an explicitly judged attestation must verify: %v", verification.Blockers)
	}
}

// A judged criterion passed without a conclusion is refused, and the message
// says what is missing rather than that something is invalid.
func TestABMReviewSoundnessJudgmentNeedsAConclusion(t *testing.T) {
	_, store := abmMixedProfileFixture(t)
	now := time.Date(2026, 9, 18, 15, 0, 0, 0, time.UTC)
	bundle, err := store.SealReviewBundle("spec:backend", now)
	if err != nil {
		t.Fatal(err)
	}
	att := approvedBundleAttestation(bundle, "agent:audit")
	stripped := false
	for i := range att.Criteria {
		for _, planned := range bundle.Payload.Plan.Criteria {
			if planned.ID == att.Criteria[i].ID && ReviewCriterionKind(planned) == ReviewCriterionKindJudgment && att.Criteria[i].Disposition == "passed" {
				att.Criteria[i].Rationale = ""
				stripped = true
			}
		}
	}
	if !stripped {
		t.Fatal("fixture must contain a judgment criterion passed with evidence")
	}
	blockers := store.validateBundleAttestation(bundle, att)
	if !containsSubstring(blockers, "passed with no conclusion") {
		t.Fatalf("expected a conclusion blocker, got %v", blockers)
	}
}

// F02. The CLI parser refused a criterion naming a finding it does not record;
// the Store accepted it, so every other caller could approve a criterion that
// explicitly did not pass while filing nothing.
func TestABMReviewSoundnessOrphanFindingRefusedByStore(t *testing.T) {
	_, store := reviewBundleFixture(t)
	now := time.Date(2026, 9, 18, 15, 0, 0, 0, time.UTC)
	bundle, err := store.SealReviewBundle("spec:backend", now)
	if err != nil {
		t.Fatal(err)
	}
	att := approvedBundleAttestation(bundle, "agent:audit")
	att.Criteria[0].Disposition = "finding"
	att.Criteria[0].Evidence = "finding-does-not-exist"
	if _, err := store.RecordReviewAttestation(att, now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	verification, err := store.VerifyReviewBundle("spec:backend")
	if err != nil {
		t.Fatal(err)
	}
	if verification.Approved {
		t.Fatal("a criterion naming an unrecorded finding must not verify approved")
	}
	if !containsSubstring(verification.Blockers, "which this attestation does not record") {
		t.Fatalf("expected an orphan-finding blocker, got %v", verification.Blockers)
	}
}

// F04. Every criterion dispensed with, on a scope that does carry a delivery
// target, verified approved on the strength of one arbitrary phrase.
func TestABMReviewSoundnessBlanketNotApplicableRefused(t *testing.T) {
	_, store := reviewBundleFixture(t)
	now := time.Date(2026, 9, 18, 15, 0, 0, 0, time.UTC)
	bundle, err := store.SealReviewBundle("spec:backend", now)
	if err != nil {
		t.Fatal(err)
	}
	att := approvedBundleAttestation(bundle, "agent:audit")
	for i := range att.Criteria {
		att.Criteria[i].Disposition = "not-applicable"
		att.Criteria[i].Evidence = ""
		att.Criteria[i].Rationale = "not relevant"
	}
	if _, err := store.RecordReviewAttestation(att, now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	verification, err := store.VerifyReviewBundle("spec:backend")
	if err != nil {
		t.Fatal(err)
	}
	if verification.Approved {
		t.Fatal("an attestation where the whole plan is inapplicable must not verify approved")
	}
	if !containsSubstring(verification.Blockers, "not-applicable") {
		t.Fatalf("expected an applicability blocker, got %v", verification.Blockers)
	}
}

// Missing judgment is a pendency; missing relevance is an answer. A criterion
// asking for a class the bundle actually seals is applicable by construction.
func TestABMReviewSoundnessNotApplicableContradictedBySealedEvidence(t *testing.T) {
	_, store := reviewBundleFixture(t)
	now := time.Date(2026, 9, 18, 15, 0, 0, 0, time.UTC)
	bundle, err := store.SealReviewBundle("spec:backend", now)
	if err != nil {
		t.Fatal(err)
	}
	sealed := map[string]bool{}
	for _, ev := range bundle.Payload.Evidence {
		sealed[ev.EvidenceClass] = true
	}
	att := approvedBundleAttestation(bundle, "agent:audit")
	target := ""
	for i := range att.Criteria {
		for _, planned := range bundle.Payload.Plan.Criteria {
			if planned.ID != att.Criteria[i].ID {
				continue
			}
			for _, class := range planned.EvidenceClasses {
				if sealed[class] && target == "" {
					target = planned.ID
					att.Criteria[i] = ReviewCriterion{ID: planned.ID, Disposition: "not-applicable", Rationale: "skipped"}
				}
			}
		}
	}
	if target == "" {
		t.Skip("fixture seals no evidence a criterion asks for")
	}
	blockers := store.validateBundleAttestation(bundle, att)
	if !containsSubstring(blockers, "while the bundle seals evidence of class") {
		t.Fatalf("expected an applicability contradiction for %s, got %v", target, blockers)
	}
}

// A profile may raise a collected criterion to a judged one. It may not declare
// a criterion mechanical with nothing able to answer it, which would re-open
// the hole the derivation closes.
func TestABMReviewSoundnessMechanicalCriterionNeedsAProducer(t *testing.T) {
	root, store := reviewBundleFixture(t)
	writeReviewFixture(t, root, ".pose/review-profiles/spec-closeout.json", `{
  "schema_version":2,"id":"spec-closeout","version":2,"scope":"spec",
  "criteria":[{"id":"security","description":"Authority boundaries are safe.","kind":"mechanical"}]
}`)
	if _, _, err := store.loadReviewProfile("spec-closeout@2"); err == nil {
		t.Fatal("a mechanical criterion with no evidence class must not load")
	} else if !strings.Contains(err.Error(), "no registered check can answer it") {
		t.Fatalf("the refusal must say why: %v", err)
	}
}

// R6. A bundle sealed before the contract existed never lists it, so its
// verdict does not change. The 470 attestations already recorded in POSE's own
// repository are readable and auditable exactly as they were.
func TestABMReviewSoundnessUnstampedBundleKeepsItsVerdict(t *testing.T) {
	_, store := abmMixedProfileFixture(t)
	now := time.Date(2026, 9, 18, 15, 0, 0, 0, time.UTC)
	bundle, err := store.SealReviewBundle("spec:backend", now)
	if err != nil {
		t.Fatal(err)
	}
	att := approvedBundleAttestation(bundle, "agent:audit")
	judged := false
	for i := range att.Criteria {
		for _, planned := range bundle.Payload.Plan.Criteria {
			if planned.ID == att.Criteria[i].ID && ReviewCriterionKind(planned) == ReviewCriterionKindJudgment && att.Criteria[i].Disposition == "passed" {
				att.Criteria[i].Rationale = ""
				judged = true
			}
		}
	}
	if !judged {
		t.Fatal("fixture must contain a judgment criterion passed with evidence")
	}
	if blockers := store.validateBundleAttestation(bundle, att); !containsSubstring(blockers, "passed with no conclusion") {
		t.Fatalf("the contract must apply to the bundle that sealed it: %v", blockers)
	}
	// The same attestation against a bundle that predates the contract.
	legacy := bundle
	legacy.Payload.GoverningContracts = []string{"component-aware", "review-bundles", "evidence-vocabulary"}
	if blockers := store.validateBundleAttestation(legacy, att); containsSubstring(blockers, "passed with no conclusion") {
		t.Fatalf("a bundle sealed before the contract must not be judged by it: %v", blockers)
	}
}

func containsSubstring(values []string, want string) bool {
	for _, value := range values {
		if strings.Contains(value, want) {
			return true
		}
	}
	return false
}

// A component the validation matrix declares runs no check produces a required
// tool that nothing can feed. Before this, the reviewer's only options were to
// cite another component's result, which is false, or to stay blocked. It is
// the same failure the profile loader already refuses one level up, per
// component instead of per class (spec pose-abm-review-soundness).
func TestABMReviewSoundnessToolWithoutProducerIsDispensable(t *testing.T) {
	root, store := abmMixedProfileFixture(t)
	writeReviewFixture(t, root, ".pose/indexes/validation-matrix.json", `{
  "defaults":{"mode":"strict"},
  "moduleOverrides":{"docs-only":{"stack":"python","replaceDefaultChecks":true,"checks":[]}}
}`)
	tool := ReviewPlanTool{ID: "validate", Requiredness: "required", Component: "docs-only", EvidenceClasses: []string{"unit"}}
	if !store.componentDeclaresNoChecks("docs-only") {
		t.Fatal("an explicit empty replaceDefaultChecks must be read as no producer")
	}
	if store.componentDeclaresNoChecks("pose-mcp") {
		t.Fatal("a component the matrix does not mention must keep its tool required")
	}

	warnings := store.annotateReviewToolProducerCoverage([]ReviewPlanTool{tool}, nil)
	if len(warnings) != 1 || !strings.Contains(warnings[0], "runs no check") {
		t.Fatalf("the gap must reach the reviewer: %v", warnings)
	}

	// Still refused while the plan does not record the gap.
	_, blockers := evaluateReviewToolCoverage(root, []ReviewPlanTool{tool},
		[]ReviewToolDisposition{{ID: "validate", Component: "docs-only", Disposition: "not-used", Rationale: "skipped"}}, nil, true)
	if !containsSubstring(blockers, "was not used") {
		t.Fatalf("a required tool with a producer must still refuse not-used: %v", blockers)
	}

	// Dispensable once it does, and still owing a reason.
	tool.ProducerCoverage = "none"
	_, blockers = evaluateReviewToolCoverage(root, []ReviewPlanTool{tool},
		[]ReviewToolDisposition{{ID: "validate", Component: "docs-only", Disposition: "not-used", Rationale: "the matrix declares this component runs no check"}}, nil, true)
	if len(blockers) != 0 {
		t.Fatalf("a tool nothing can feed must be dispensable: %v", blockers)
	}
	_, blockers = evaluateReviewToolCoverage(root, []ReviewPlanTool{tool},
		[]ReviewToolDisposition{{ID: "validate", Component: "docs-only", Disposition: "not-used"}}, nil, true)
	if !containsSubstring(blockers, "still needs a not-used rationale") {
		t.Fatalf("the dispensation must state the fact: %v", blockers)
	}
}
