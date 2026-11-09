package cli

import (
	"os"
	"path/filepath"
	"strings"
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

// R4, on the path both modes share: a call that ends at the stop clears a checkpoint an
// earlier stop left and says where the state is.
func TestTheStopOnUnstableInputsClearsAnOlderCheckpointAndSaysWhereTheStateIs(t *testing.T) {
	root := reviewBundleCLIFixture(t)
	store := posemodel.Store{Root: root}
	checkpoint := filepath.Join(root, ".pose/closeout-plans/spec-bundle.json")
	for _, resume := range []string{"`pose close spec:bundle --resume`"} {
		mustWrite(t, checkpoint, `{"schema_version":1,"scope":"spec:bundle","started_at":"2026-11-08T00:00:00Z","plan_digest":"sha256:old","completed":["context"],"last_error":"verify: unknown"}`)
		result := posemodel.NewOperationResult("close.apply", "spec:bundle")
		out := stopOnUnstableInputs(store, "spec:bundle", resume, &result, posemodel.CloseoutStepVerify, "governance read view is unknown")
		if out.Outcome != posemodel.OutcomeBlocked || out.Error == nil || out.Error.Cause != "terminal-verification-unstable-inputs" || !out.Resumable {
			t.Fatalf("the stop was not reported as one on unstable inputs: %+v", out)
		}
		if !strings.Contains(out.NextAction, "no checkpoint is kept for this stop") || !strings.Contains(out.NextAction, resume) {
			t.Fatalf("the result does not say where the state is: %q", out.NextAction)
		}
		if _, err := os.Stat(checkpoint); err == nil {
			t.Fatal("a checkpoint of the stop was kept")
		}
	}
}
