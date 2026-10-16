package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	posemodel "github.com/harne8/pose-mcp/internal/pose"
)

// Spec pose-journal-checkpoint-and-lock-recovery: the CLI reports journals,
// checkpoints and start locks as they are, with the supported next step, in
// both renderers (R9).

type journalFailureJSON struct {
	posemodel.OperationResult
	Data posemodel.ActionJournalError `json:"data"`
}

// V1/V8 — a journal with an interrupted tail is located, in JSON and for
// people, by show and by list; its bytes stay as they were.
func TestAJournalThatDoesNotReadIsLocatedInBothRenderers(t *testing.T) {
	root := actionCLIFixture(t)
	code, out := runPose(t, root, append(append([]string{}, openArgs...), "--apply", "--json")...)
	var opened posemodel.ActionRequestView
	if code != 0 || json.Unmarshal([]byte(out), &opened) != nil {
		t.Fatalf("open: %d %s", code, out)
	}
	id := opened.Request.ID
	path := filepath.Join(root, ".pose", "actions", id+".jsonl")
	intact, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	damaged := append(append([]byte{}, intact...), []byte(`{"schema_version":1,"seq":2,`)...)
	if err := os.WriteFile(path, damaged, 0o644); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"action", "show", id, "--json"}, {"action", "list", "--json"}} {
		var result journalFailureJSON
		code, _ := runPoseJSON(t, root, &result, args...)
		if code != 1 || result.Outcome != posemodel.OutcomeFailed || result.Error == nil || result.Error.Cause != "action-journal-invalid" ||
			result.Data.Kind != posemodel.JournalInterruptedTail || result.Data.Offset != len(intact) || result.Data.Line != 2 || result.Data.ValidEvents != 1 ||
			!strings.Contains(result.NextAction, "truncating the journal to") {
			t.Fatalf("%v: %d %+v", args, code, result)
		}
	}
	for _, args := range [][]string{{"action", "show", id}, {"action", "list"}} {
		code, human := runPose(t, root, args...)
		if code != 1 || !strings.Contains(human, "action.journal=interrupted-tail .pose/actions/"+id+".jsonl") || !strings.Contains(human, "action.journal.next_action=") {
			t.Fatalf("%v: %d %s", args, code, human)
		}
	}
	if after, _ := os.ReadFile(path); !bytes.Equal(after, damaged) {
		t.Fatal("reading the journal changed it")
	}
}

type checkpointPlanJSON struct {
	posemodel.CloseoutPlan
}

// V2 — a checkpoint that does not read is shown by the plan and stops an
// apply before any step runs; it is never removed by the engine.
func TestACheckpointThatDoesNotReadStopsTheApplyBeforeAnyStep(t *testing.T) {
	root := reviewBundleCLIFixture(t)
	path := filepath.Join(root, ".pose", "closeout-plans", "spec-bundle.json")
	writeCloseoutCLIFile(t, root, ".pose/closeout-plans/spec-bundle.json", `{"schema_version":1,`)
	var plan checkpointPlanJSON
	if code, _ := runPoseJSON(t, root, &plan, "close", "spec:bundle", "--plan", "--json"); code != 0 || plan.Checkpoint == nil || plan.Checkpoint.State != posemodel.CheckpointCorrupt {
		t.Fatalf("the plan hid the checkpoint: %d %+v", code, plan.Checkpoint)
	}
	if code, human := runPose(t, root, "close", "spec:bundle", "--plan"); code != 0 || !strings.Contains(human, "closeout_plan.checkpoint=corrupt .pose/closeout-plans/spec-bundle.json") {
		t.Fatalf("the human plan hid the checkpoint: %d %s", code, human)
	}
	for _, mode := range []string{"--apply", "--resume"} {
		var result closeApplyJSON
		code, _ := runPoseJSON(t, root, &result, "close", "spec:bundle", mode, "--reviewer", "agent:closeout-reviewer", "--json")
		if code != 1 || result.Outcome != posemodel.OutcomeBlocked || result.Error == nil || result.Error.Cause != "checkpoint-corrupt" || len(result.StepsRun) != 0 || result.Resumable {
			t.Fatalf("%s ran over a corrupt checkpoint: %d %+v", mode, code, result.OperationResult)
		}
	}
	if code, human := runPose(t, root, "close", "spec:bundle", "--apply", "--reviewer", "agent:closeout-reviewer"); code != 1 || !strings.Contains(human, "closeout_plan.next_action=inspect .pose/closeout-plans/spec-bundle.json") {
		t.Fatalf("the human refusal: %d %s", code, human)
	}
	if _, err := os.Stat(filepath.Join(root, ".pose", "review-bundles")); !os.IsNotExist(err) {
		t.Fatal("a step ran over the corrupt checkpoint")
	}
	if raw, _ := os.ReadFile(path); string(raw) != `{"schema_version":1,` {
		t.Fatal("the corrupt checkpoint was changed")
	}
}

// V4 — a checkpoint that records an effect the repository does not reflect
// is a conflict for --resume; --apply with the digest of the plan the caller
// reviewed starts a new run and names the checkpoint it replaced.
func TestACheckpointIsProgressNeverAReceipt(t *testing.T) {
	root := reviewBundleCLIFixture(t)
	writeCloseoutCLIFile(t, root, ".pose/closeout-plans/spec-bundle.json", `{"schema_version":1,"scope":"spec:bundle","started_at":"2026-10-12T12:00:00Z","plan_digest":"sha256:earlier","completed":["context","trace","seal"]}`)
	var resumed closeApplyJSON
	code, _ := runPoseJSON(t, root, &resumed, "close", "spec:bundle", "--resume", "--reviewer", "agent:closeout-reviewer", "--json")
	if code != 1 || resumed.Outcome != posemodel.OutcomeConflict || resumed.Error.Cause != "checkpoint-progress-mismatch" || len(resumed.StepsRun) != 0 ||
		!strings.Contains(resumed.Error.Message, "seal") || !strings.Contains(resumed.NextAction, "--apply --digest") {
		t.Fatalf("the checkpoint replayed or completed a step: %d %+v", code, resumed.OperationResult)
	}
	if code, _ := runPoseJSON(t, root, &resumed, "close", "spec:bundle", "--apply", "--reviewer", "agent:closeout-reviewer", "--json"); code != 1 || resumed.Outcome != posemodel.OutcomeConflict {
		t.Fatalf("an apply without the reviewed digest replaced the checkpoint: %d %+v", code, resumed.OperationResult)
	}
	var plan posemodel.CloseoutPlan
	if code, _ := runPoseJSON(t, root, &plan, "close", "spec:bundle", "--plan", "--json"); code != 0 {
		t.Fatal("plan")
	}
	var applied struct {
		posemodel.OperationResult
		Data struct {
			Replaced *posemodel.CloseoutCheckpoint `json:"replaced_checkpoint"`
		} `json:"data"`
	}
	code, stderr := runPoseJSON(t, root, &applied, "close", "spec:bundle", "--apply", "--digest", plan.Digest, "--reviewer", "agent:closeout-reviewer", "--json")
	if code != 0 || applied.Outcome != posemodel.OutcomeCompleted || applied.Data.Replaced == nil || applied.Data.Replaced.PlanDigest != "sha256:earlier" {
		t.Fatalf("the reviewed plan did not replace the stale checkpoint: %d %+v %s", code, applied, stderr)
	}
	if _, err := os.Stat(filepath.Join(root, ".pose", "closeout-plans", "spec-bundle.json")); !os.IsNotExist(err) {
		t.Fatal("the finished run left its checkpoint")
	}
}

// V5/V6/V9 — the start command reports a held lock with its owner's state and
// recovers only the dead lock it names.
func TestTheStartCommandReportsAndRecoversOnlyADeadLock(t *testing.T) {
	root := startCLIFixture(t, true)
	var plan posemodel.StartPlan
	if code, _ := runPoseJSON(t, root, &plan, "start", "spec:work", "--json"); code != 0 {
		t.Fatal("preview")
	}
	lock := filepath.Join(root, ".pose", "starts", ".work.lock")
	if err := os.MkdirAll(filepath.Dir(lock), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(lock, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	var blocked struct {
		posemodel.OperationResult
		Data posemodel.StartLockError `json:"data"`
	}
	code, _ := runPoseJSON(t, root, &blocked, "start", "spec:work", "--apply", "--digest", plan.Digest, "--json")
	if code != 1 || blocked.Outcome != posemodel.OutcomeBlocked || blocked.Error.Cause != "start-lock-owner-unknown" || blocked.Data.State != posemodel.LockOwnerUnknown ||
		!strings.Contains(blocked.NextAction, "confirm no start is running") || blocked.Resumable {
		t.Fatalf("an ownerless lock: %d %+v", code, blocked)
	}
	var recovered posemodel.OperationResult
	if code, _ := runPoseJSON(t, root, &recovered, "start", "spec:work", "--recover-lock", blocked.Data.OwnerDigest, "--json"); code != 1 || recovered.Outcome != posemodel.OutcomeBlocked {
		t.Fatalf("an ownerless lock was recovered: %d %+v", code, recovered)
	}
	if _, err := os.Stat(lock); err != nil {
		t.Fatal("the ownerless lock was removed")
	}
	code, human := runPose(t, root, "start", "spec:work", "--status")
	if code != 0 || !strings.Contains(human, "start.lock=owner-unknown .pose/starts/.work.lock") || !strings.Contains(human, "start.lock.owner_digest=") {
		t.Fatalf("status hid the lock: %d %s", code, human)
	}
	for _, args := range [][]string{{"--recover-lock"}, {"--recover-lock", "x", "--apply"}, {"--recover-lock", "x", "--status"}} {
		if code, _ := runPose(t, root, append([]string{"start", "spec:work"}, args...)...); code != 2 {
			t.Fatalf("%v: exit %d", args, code)
		}
	}
}
