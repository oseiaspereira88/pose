package cli

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	posemodel "github.com/harne8/pose-mcp/internal/pose"
)

// Spec pose-recoverable-closeout-plan.

func countFiles(t *testing.T, dir string) int {
	t.Helper()
	files, _ := filepath.Glob(filepath.Join(dir, "*.json"))
	return len(files)
}

func TestClosePlanPreviewsWithoutWriting(t *testing.T) {
	root := reviewBundleCLIFixture(t)
	code, out := runPose(t, root, "close", "spec:bundle", "--plan")
	if code != 0 || !strings.Contains(out, "closeout_plan.step.seal=pending") || !strings.Contains(out, "closeout_plan.step.transition=pending") {
		t.Fatalf("plan: %d %s", code, out)
	}
	if countFiles(t, filepath.Join(root, ".pose/review-bundles")) != 0 {
		t.Fatal("--plan sealed a bundle")
	}
	_, raw := runPose(t, root, "close", "spec:bundle", "--plan", "--json")
	var plan posemodel.CloseoutPlan
	if err := json.Unmarshal([]byte(raw), &plan); err != nil || plan.Digest == "" || plan.Terminal {
		t.Fatalf("json plan: %v %s", err, raw)
	}
}

func TestAnInterruptedCloseoutResumesWithoutDuplication(t *testing.T) {
	root := reviewBundleCLIFixture(t)
	// No reviewer: the plan seals, then stops at the mechanical attestation.
	code, out := runPose(t, root, "close", "spec:bundle", "--apply")
	if code == 0 || !strings.Contains(out, "--reviewer is required") {
		t.Fatalf("the plan invented a reviewer or did not stop: %d %s", code, out)
	}
	if countFiles(t, filepath.Join(root, ".pose/review-bundles")) != 1 {
		t.Fatal("the seal step did not run before the stop")
	}
	_, plan := runPose(t, root, "close", "spec:bundle", "--plan")
	if !strings.Contains(plan, "closeout_plan.interrupted=") || !strings.Contains(plan, "closeout_plan.step.seal=done") {
		t.Fatalf("the interruption is not visible: %s", plan)
	}
	code, out = runPose(t, root, "close", "spec:bundle", "--resume", "--reviewer", "agent:closeout-reviewer")
	if code != 0 || !strings.Contains(out, "closeout_plan.result=closed spec:bundle") {
		t.Fatalf("resume: %d %s", code, out)
	}
	if countFiles(t, filepath.Join(root, ".pose/review-bundles")) != 1 || countFiles(t, filepath.Join(root, ".pose/review-attestations")) != 1 {
		t.Fatal("the resume sealed or attested twice")
	}
	spec, _ := os.ReadFile(filepath.Join(root, ".pose/specs/bundle/spec.md"))
	if !strings.Contains(string(spec), "status: done") {
		t.Fatalf("the transition did not apply:\n%s", spec)
	}
	if _, err := os.Stat(filepath.Join(root, ".pose/closeout-plans")); err == nil {
		t.Fatal("the checkpoint survived a completed closeout")
	}
	code, out = runPose(t, root, "close", "spec:bundle", "--resume", "--reviewer", "agent:closeout-reviewer")
	if code != 0 || countFiles(t, filepath.Join(root, ".pose/review-attestations")) != 1 {
		t.Fatalf("a second resume was not a no-op: %d %s", code, out)
	}
	err := posemodel.WithGovernanceWriteGuard("test.hold-terminal-plan-read", []posemodel.Store{cliGovernedStore(root)}, func([]posemodel.Store) error {
		result := applyClosePlan(root, cliGovernedStore(root), "spec:bundle", "resume", "", "agent:closeout-reviewer", false)
		if result.Outcome != posemodel.OutcomeAlreadySatisfied || len(result.Effects) != 0 || len(result.StepsRun) != 0 {
			return errors.New("completed closeout plan tried to acquire a writer lease or run an effect")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("terminal closeout plan was not read-only: %v", err)
	}
}

func TestGovernanceReadViewCloseoutReportsPublishedBundleWithUnconfirmedDurability(t *testing.T) {
	root := reviewBundleCLIFixture(t)
	posemodel.PublishSyncHook = func(kind, path string) error {
		if kind == "dir" && strings.Contains(path, filepath.Join(".pose", "review-bundles")) {
			return errors.New("injected directory sync failure")
		}
		return nil
	}
	t.Cleanup(func() { posemodel.PublishSyncHook = nil })
	var result closeApplyJSON
	code, _ := runPoseJSON(t, root, &result, "close", "spec:bundle", "--apply", "--json")
	if code != 1 || result.Outcome != posemodel.OutcomeFailed || result.Step != posemodel.CloseoutStepMechanical {
		t.Fatalf("closeout did not stop at the reviewer requirement after sealing: code=%d result=%+v", code, result.OperationResult)
	}
	sealed := false
	uncertain := false
	for _, effect := range result.Effects {
		if effect.Step == posemodel.CloseoutStepSeal && effect.Effect == "bundle-sealed" && effect.Ref != "" {
			sealed = true
		}
	}
	for _, item := range result.Unconfirmed {
		if item.Step == "seal-durability" && strings.Contains(item.Reason, "published and readable") {
			uncertain = true
		}
	}
	if !sealed || !uncertain || countFiles(t, filepath.Join(root, ".pose", "review-bundles")) != 1 {
		t.Fatalf("committed seal was hidden or misreported: effects=%+v unconfirmed=%+v", result.Effects, result.Unconfirmed)
	}
}

func TestTheClosePlanStopsAtJudgmentAndNeverAnswersIt(t *testing.T) {
	root := reviewBundleCLIFixture(t)
	writeCloseoutCLIFile(t, root, ".pose/review-profiles/spec-closeout.json", `{
  "schema_version":2,"id":"spec-closeout","version":2,"scope":"spec",
  "criteria":[{"id":"security","description":"Authority boundaries are safe."}]
}`)
	commitSnapshotFixture(t, root)
	code, out := runPose(t, root, "close", "spec:bundle", "--apply", "--reviewer", "agent:closeout-reviewer")
	if code != 3 || !strings.Contains(out, "closeout_plan.pending.security=") || !strings.Contains(out, "waiting on a reviewer") {
		t.Fatalf("the plan did not stop at the judgment: %d %s", code, out)
	}
	if countFiles(t, filepath.Join(root, ".pose/review-attestations")) != 0 {
		t.Fatal("the plan wrote an attestation for a pending judgment")
	}
}

func TestAStalePlanDigestIsRefused(t *testing.T) {
	root := reviewBundleCLIFixture(t)
	code, out := runPose(t, root, "close", "spec:bundle", "--apply", "--digest", "sha256:old", "--reviewer", "agent:r")
	if code == 0 || !strings.Contains(out, "the repository changed since the plan was computed") {
		t.Fatalf("a stale plan was applied: %d %s", code, out)
	}
}
