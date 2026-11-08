package cli

import (
	"testing"

	posemodel "github.com/harne8/pose-mcp/internal/pose"
)

// Spec pose-done-spec-keeps-its-sealed-plan R4, R5.

func unstablePlan(done bool, freshness string) posemodel.CloseoutPlan {
	plan := posemodel.CloseoutPlan{Scope: "spec:x", LifecycleDone: done}
	if freshness != "" {
		plan.ReadView = &posemodel.ReadViewMetadata{Freshness: freshness}
	}
	return plan
}

func TestAStopOnAnUnknownViewOfADoneScopeLeavesNoCheckpoint(t *testing.T) {
	verify := posemodel.CloseoutPlanStep{ID: posemodel.CloseoutStepVerify, State: posemodel.CloseoutStateBlocked}
	if !closeoutStopLeavesNoCheckpoint(unstablePlan(true, posemodel.ReadViewFreshnessUnknown), verify) {
		t.Fatal("the verification of a done scope stopped on an unknown read view must leave no checkpoint")
	}
}

func TestEveryOtherStopStillRecordsItsCheckpoint(t *testing.T) {
	blocked := func(id string) posemodel.CloseoutPlanStep {
		return posemodel.CloseoutPlanStep{ID: id, State: posemodel.CloseoutStateBlocked}
	}
	cases := map[string]struct {
		plan posemodel.CloseoutPlan
		step posemodel.CloseoutPlanStep
	}{
		"a lifecycle not done":       {unstablePlan(false, posemodel.ReadViewFreshnessUnknown), blocked(posemodel.CloseoutStepVerify)},
		"a current read view":        {unstablePlan(true, posemodel.ReadViewFreshnessCurrent), blocked(posemodel.CloseoutStepVerify)},
		"no read view":               {unstablePlan(true, ""), blocked(posemodel.CloseoutStepVerify)},
		"another step":               {unstablePlan(true, posemodel.ReadViewFreshnessUnknown), blocked(posemodel.CloseoutStepMechanical)},
		"a step that is not blocked": {unstablePlan(true, posemodel.ReadViewFreshnessUnknown), posemodel.CloseoutPlanStep{ID: posemodel.CloseoutStepVerify, State: posemodel.CloseoutStatePending}},
	}
	for name, c := range cases {
		if closeoutStopLeavesNoCheckpoint(c.plan, c.step) {
			t.Errorf("%s: a stop that is progress or a real blocker lost its checkpoint", name)
		}
	}
}
