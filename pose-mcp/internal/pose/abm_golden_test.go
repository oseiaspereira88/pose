package pose

import (
	"encoding/json"
	"os"
	"testing"
)

// The pilot reuses independently stated contract tests. The manifest records
// the adjudicated oracle for each case and keeps synthetic cases separate from
// dogfood deliveries. Every oracle is checked, not just displayed.
func TestABMGolden(t *testing.T) {
	raw, err := os.ReadFile("testdata/abm/scenarios.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases []struct {
		ID       string `json:"id"`
		Category string `json:"category"`
		Test     string `json:"test"`
		Oracle   string `json:"oracle"`
	}
	if err := json.Unmarshal(raw, &cases); err != nil {
		t.Fatal(err)
	}
	registry := map[string]struct {
		run    func(*testing.T)
		oracle string
	}{
		"TestABMProgressiveReviewTrivialCorpusHasNoAdditionalObligations": {TestABMProgressiveReviewTrivialCorpusHasNoAdditionalObligations, "No extra obligations across criticalities for scopes without a trigger."},
		"TestABMStructuralCausalityRequiresCoverageOfEveryMaterialFact":   {TestABMStructuralCausalityRequiresCoverageOfEveryMaterialFact, "A green build does not waive an unmapped material fact."},
		"TestABMCausalityCloseoutAcceptsAJustifiedLiveMapping":            {TestABMCausalityCloseoutAcceptsAJustifiedLiveMapping, "Accept the live decision basis reaching an active requirement."},
		"TestABMNodeTransitionsInvalidatedAssumptionBlocksActiveDecision": {TestABMNodeTransitionsInvalidatedAssumptionBlocksActiveDecision, "An invalidated assumption blocks a dependent active decision."},
		"TestABMPolicyDowngradeOverlayCannotLowerHumanFloor":              {TestABMPolicyDowngradeOverlayCannotLowerHumanFloor, "An overlay cannot lower the mandatory human floor."},
		"TestABMAtomicStartReconciliationNeverFabricatesHistory":          {TestABMAtomicStartReconciliationNeverFabricatesHistory, "A late start reconciliation never invents an earlier baseline."},
		"TestABMStructuralDeltaSubjectActionsAndUnsupportedCoverage":      {TestABMStructuralDeltaSubjectActionsAndUnsupportedCoverage, "Unsupported formats remain partial coverage."},
		"TestABMReviewAuthorityRejectsReplayAndExpiry":                    {TestABMReviewAuthorityRejectsReplayAndExpiry, "Reject replay to another bundle or expired authority."},
		"TestABMReviewSoundnessJudgmentIsNotAutoApproved":                 {TestABMReviewSoundnessJudgmentIsNotAutoApproved, "Missing reviewer judgment stays pending."},
		"TestABMReviewSoundnessJudgmentNeedsAConclusion":                  {TestABMReviewSoundnessJudgmentNeedsAConclusion, "A blank judgment conclusion does not approve."},
		"TestABMReviewSoundnessOrphanFindingRefusedByStore":               {TestABMReviewSoundnessOrphanFindingRefusedByStore, "An unrecorded finding reference does not approve."},
		"TestABMReviewSoundnessBlanketNotApplicableRefused":               {TestABMReviewSoundnessBlanketNotApplicableRefused, "Blanket inapplicability cannot approve a delivery."},
		"TestABMReviewSoundnessNotApplicableContradictedBySealedEvidence": {TestABMReviewSoundnessNotApplicableContradictedBySealedEvidence, "Sealed applicable evidence contradicts inapplicability."},
		"TestBundlePathWontFixUsesSealedAcceptedRiskGate":                 {TestBundlePathWontFixUsesSealedAcceptedRiskGate, "Only complete risks of a sealed allowed severity can be waived."},
		"TestABMReviewSoundnessReuseCannotLaunderRejectedRisk":            {TestABMReviewSoundnessReuseCannotLaunderRejectedRisk, "Reuse cannot launder an unresolved critical risk or rejection."},
		"TestABMStructuralDeltaObservesBothSidesAndStableIDs":             {TestABMStructuralDeltaObservesBothSidesAndStableIDs, "Detect both subject sides with stable fact identities."},
		"TestABMStructuralMaterialityExcludesTransitiveAndUncertain":      {TestABMStructuralMaterialityExcludesTransitiveAndUncertain, "Transitive and uncertain facts create no material obligation."},
		"TestABMStructuralCausalityBasisMustReachARequirement":            {TestABMStructuralCausalityBasisMustReachARequirement, "A disconnected basis cannot justify added structure."},
		"TestABMCausalityCloseoutRefusesADeadBasis":                       {TestABMCausalityCloseoutRefusesADeadBasis, "A dead basis cannot justify a material fact."},
		"TestABMCausalityCloseoutUnknownIsNotAbsence":                     {TestABMCausalityCloseoutUnknownIsNotAbsence, "Unknown structural coverage cannot count as absence."},
		"TestABMAtomicStartRefusesStaleAndUnreadyPlans":                   {TestABMAtomicStartRefusesStaleAndUnreadyPlans, "A stale or unready digest cannot start execution."},
		"TestABMAtomicStartResumesAfterInterruptionAndCancels":            {TestABMAtomicStartResumesAfterInterruptionAndCancels, "Resume a journaled interruption without duplicate start."},
		"TestABMContractNodesRefusesSchema2WithoutCapability":             {TestABMContractNodesRefusesSchema2WithoutCapability, "Refuse schema 2 without adopted reader capability."},
		"TestABMNodeTransitionsEditorialCannotMaskSemantic":               {TestABMNodeTransitionsEditorialCannotMaskSemantic, "An editorial disposition cannot hide semantic node changes."},
	}
	if len(cases) < 24 {
		t.Fatalf("pilot requires at least 24 adjudicated cases, got %d", len(cases))
	}
	seen := map[string]bool{}
	categories := map[string]bool{}
	for _, scenario := range cases {
		test, ok := registry[scenario.Test]
		if !ok || seen[scenario.Test] || scenario.Oracle != test.oracle || scenario.ID == "" {
			t.Fatalf("unadjudicated or duplicate scenario: %+v", scenario)
		}
		seen[scenario.Test] = true
		categories[scenario.Category] = true
		t.Run(scenario.ID+"/"+scenario.Category, func(t *testing.T) {
			t.Log(scenario.Oracle)
			test.run(t)
			if t.Skipped() {
				t.Fatal("an adjudicated golden case cannot skip")
			}
		})
	}
	for _, category := range []string{"trivial", "overengineered-green", "justified", "false-assumption", "policy-downgrade", "post-hoc", "unsupported"} {
		if !categories[category] {
			t.Fatalf("missing required category %s", category)
		}
	}
}
