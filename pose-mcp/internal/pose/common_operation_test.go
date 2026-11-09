package pose

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestAnInProcessRunKeepsTheSpawnContract(t *testing.T) {
	spawns, runs := CLISpawnAttempts(), CLIInProcessRuns()
	RegisterInProcessCommand("t-ok", func(root string, args []string, stdout, stderr io.Writer) int {
		_, _ = io.WriteString(stdout, `{"root":"`+root+`","args":"`+strings.Join(args, ",")+`"}`)
		return 0
	})
	RegisterInProcessCommand("t-fail", func(root string, args []string, stdout, stderr io.Writer) int {
		_, _ = io.WriteString(stderr, "no such thing\n")
		return 1
	})
	block := make(chan struct{})
	RegisterInProcessCommand("t-block", func(root string, args []string, stdout, stderr io.Writer) int {
		<-block
		return 0
	})
	defer close(block)
	s := Store{Root: "/r"}

	out, err := s.runCLI(context.Background(), []string{"t-ok", "a", "b"})
	if err != nil || string(out) != `{"root":"/r","args":"a,b"}` {
		t.Fatalf("a successful run: %q %v", out, err)
	}
	if _, err := s.runCLI(context.Background(), []string{"t-fail"}); err == nil || err.Error() != "pose t-fail failed: no such thing" {
		t.Fatalf("a failing run must carry the command's own message, as the spawned one does: %v", err)
	}
	// A deadline is an execution failure with no verdict, never a failed result.
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	_, err = s.runCLI(ctx, []string{"t-block"})
	var exec *GateExecutionError
	if !errors.As(err, &exec) || exec.Cause != ExecutionCauseDeadline {
		t.Fatalf("a deadline must be an execution failure: %v", err)
	}
	cctx, ccancel := context.WithCancel(context.Background())
	ccancel()
	if _, err = s.runCLI(cctx, []string{"t-block"}); !errors.As(err, &exec) || exec.Cause != ExecutionCauseCanceled {
		t.Fatalf("a cancellation must be an execution failure: %v", err)
	}
	if CLISpawnAttempts() != spawns {
		t.Fatalf("an in-process run tried to start the binary")
	}
	if CLIInProcessRuns()-runs != 2 {
		t.Fatalf("in-process runs = %d, want 2 (the finished ones)", CLIInProcessRuns()-runs)
	}
}

func handoffPlan() CloseoutPlan {
	return CloseoutPlan{
		Scope: "spec:x", Digest: "sha256:aaaa", SchemaVersion: 1,
		Inputs: CloseoutPlanInputs{SpecDigest: "sd", Revision: "rev", PolicyDigest: "pd"},
		Steps: []CloseoutPlanStep{
			{ID: CloseoutStepContext, State: CloseoutStateDone},
			{ID: CloseoutStepEvidence, State: CloseoutStatePending},
		},
	}
}

func TestACloseoutHandoffDescribesAndExecutesNothing(t *testing.T) {
	h := CloseoutHandoff(handoffPlan(), HandoffTarget{})
	if h == nil || h.Executes || h.Operation != "close" || h.Scope != "spec:x" || h.Phase != "closeout" {
		t.Fatalf("handoff = %+v", h)
	}
	want := []string{"pose", "close", "spec:x", "--apply", "--digest", "sha256:aaaa"}
	if strings.Join(h.Argv, "\x00") != strings.Join(want, "\x00") {
		t.Fatalf("argv = %q", h.Argv)
	}
	if h.Inputs.PlanDigest != "sha256:aaaa" || h.Inputs.Revision != "rev" || h.Inputs.SpecDigest != "sd" || h.Inputs.PolicyDigest != "pd" {
		t.Fatalf("inputs = %+v", h.Inputs)
	}
	if h.RequiredAuthority == "" || h.Receipt == "" || h.Invalidation == "" || !strings.Contains(h.ContextRequirement, "takes no context flag") {
		t.Fatalf("a handoff must say who may run it, how the outcome returns and when it stops being valid: %+v", h)
	}
}

func TestAHandoffIsDeterministicAndFollowsThePlan(t *testing.T) {
	a, b := CloseoutHandoff(handoffPlan(), HandoffTarget{}), CloseoutHandoff(handoffPlan(), HandoffTarget{})
	if a.RequestID != b.RequestID {
		t.Fatalf("the same plan gave two ids: %s %s", a.RequestID, b.RequestID)
	}
	moved := handoffPlan()
	moved.Digest = "sha256:bbbb"
	if CloseoutHandoff(moved, HandoffTarget{}).RequestID == a.RequestID {
		t.Fatal("a different plan kept the handoff id: a stale handoff would be taken for the new one")
	}
}

func TestNoHandoffWhenThereIsNothingToApply(t *testing.T) {
	terminal := handoffPlan()
	terminal.Terminal = true
	blocked := handoffPlan()
	blocked.Steps = []CloseoutPlanStep{{ID: CloseoutStepTrace, State: CloseoutStateBlocked, Reason: "trace incomplete"}}
	done := handoffPlan()
	done.Steps = []CloseoutPlanStep{{ID: CloseoutStepContext, State: CloseoutStateDone}}
	for name, plan := range map[string]CloseoutPlan{"terminal": terminal, "blocked": blocked, "all done": done, "no scope": {}} {
		if CloseoutHandoff(plan, HandoffTarget{}) != nil {
			t.Errorf("%s: a handoff was offered for nothing to apply", name)
		}
	}
}

func TestAPendingJudgmentNamesTheReviewerAsTheAuthority(t *testing.T) {
	plan := handoffPlan()
	plan.Steps = append(plan.Steps, CloseoutPlanStep{ID: CloseoutStepJudgment, State: CloseoutStateWaiting})
	if h := CloseoutHandoff(plan, HandoffTarget{}); h == nil || !strings.Contains(h.RequiredAuthority, "reviewer") {
		t.Fatalf("a pending judgment is a reviewer's: %+v", h)
	}
}

func TestABlockedOrUnstablePlanGetsNoHandoff(t *testing.T) {
	mixed := handoffPlan() // a changes-requested review: a blocked step and a pending transition
	mixed.Steps = []CloseoutPlanStep{{ID: CloseoutStepTrace, State: CloseoutStateBlocked}, {ID: CloseoutStepEvidence, State: CloseoutStatePending}}
	blockers := handoffPlan()
	blockers.Blockers = []string{"finding B1 is open"}
	unknown := handoffPlan()
	unknown.ReadView = &ReadViewMetadata{Freshness: ReadViewFreshnessUnknown}
	for name, plan := range map[string]CloseoutPlan{"blocked and pending": mixed, "blockers": blockers, "read view unknown": unknown} {
		if CloseoutHandoff(plan, HandoffTarget{}) != nil {
			t.Errorf("%s: a handoff was offered for a plan that cannot be applied", name)
		}
	}
}

func TestTheHandoffKeepsTheSelectedTarget(t *testing.T) {
	h := CloseoutHandoff(handoffPlan(), HandoffTarget{ProjectID: "harne8", Root: "/work/harne8"})
	if h == nil || h.Target.ProjectID != "harne8" || h.Target.Root != "/work/harne8" {
		t.Fatalf("the handoff lost the project the caller selected: %+v", h)
	}
}

func TestACancelledCallerNeverStartsTheCommandAndNeverGetsAVerdict(t *testing.T) {
	var started atomic.Int64
	RegisterInProcessCommand("t-count", func(root string, args []string, stdout, stderr io.Writer) int {
		started.Add(1)
		_, _ = io.WriteString(stdout, "ok")
		return 0
	})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	for i := 0; i < 50; i++ {
		out, err := (Store{Root: "/r"}).runCLI(ctx, []string{"t-count"})
		var exec *GateExecutionError
		if out != nil || !errors.As(err, &exec) || exec.Cause != ExecutionCauseCanceled {
			t.Fatalf("a cancelled caller got a result (%q, %v) instead of an execution failure", out, err)
		}
	}
	if started.Load() != 0 {
		t.Fatalf("the command started %d times for a caller that had already cancelled", started.Load())
	}
}

func TestACompletionThatRacesACancellationIsDiscarded(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	RegisterInProcessCommand("t-race", func(root string, args []string, stdout, stderr io.Writer) int {
		cancel() // the caller cancels while the command finishes
		_, _ = io.WriteString(stdout, "late")
		return 0
	})
	for i := 0; i < 20; i++ {
		c, cc := context.WithCancel(ctx)
		out, err := (Store{Root: "/r"}).runCLI(c, []string{"t-race"})
		cc()
		var exec *GateExecutionError
		if out != nil || !errors.As(err, &exec) {
			t.Fatalf("a result was returned after the cancellation: %q %v", out, err)
		}
	}
}

func TestAbandonedRunsDoNotAccumulateBeyondTheSlots(t *testing.T) {
	block := make(chan struct{})
	var alive atomic.Int64
	RegisterInProcessCommand("t-hold", func(root string, args []string, stdout, stderr io.Writer) int {
		alive.Add(1)
		<-block
		alive.Add(-1)
		return 0
	})
	for i := 0; i < 3*inProcessMaxRuns; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
		_, err := (Store{Root: "/r"}).runCLI(ctx, []string{"t-hold"})
		cancel()
		var exec *GateExecutionError
		if !errors.As(err, &exec) {
			t.Fatalf("call %d: %v", i, err)
		}
	}
	if n := alive.Load(); n > inProcessMaxRuns {
		t.Fatalf("%d abandoned runs are alive against %d slots", n, inProcessMaxRuns)
	}
	close(block)
	deadline := time.Now().Add(2 * time.Second)
	for len(inProcessSlots) != 0 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if len(inProcessSlots) != 0 {
		t.Fatalf("slots were not given back: %d held", len(inProcessSlots))
	}
}

// Spec pose-common-domain-and-cli-mcp-handoff R1 (and pose-mcp-server-survives-self-update R2):
// a command runs in process only when that answers as the spawn of the installed binary would.
func TestAnInProcessRunIsNotTakenWhenTheSpawnWouldRunAnotherEngine(t *testing.T) {
	dir := t.TempDir()
	fake := filepath.Join(dir, "pose-fake")
	write := func(body string) {
		next := fake + ".next"
		if err := os.WriteFile(next, []byte(body), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.Rename(next, fake); err != nil { // `pose update` writes the new engine at the same path
			t.Fatal(err)
		}
	}
	write("#!/bin/sh\necho '{\"engine\":\"old\"}'\n")
	oldPath, oldInfo, oldErr := executableAtStart, executableInfoAtStart, executableAtStartErr
	defer func() { executableAtStart, executableInfoAtStart, executableAtStartErr = oldPath, oldInfo, oldErr }()
	executableAtStart, executableAtStartErr = fake, nil
	executableInfoAtStart = statExecutableAtStart()
	t.Setenv("POSE_EXECUTABLE", "")

	if !inProcessRunsTheSpawnedEngine() {
		t.Fatal("the installed binary is the one running: the command should run in process")
	}
	write("#!/bin/sh\necho '{\"engine\":\"updated\"}'\n")
	if inProcessRunsTheSpawnedEngine() {
		t.Fatal("the installed binary was replaced by an update: the command must be spawned so it runs the updated engine")
	}
	var inProcess atomic.Int64
	RegisterInProcessCommand("t-engine", func(root string, args []string, stdout, stderr io.Writer) int {
		inProcess.Add(1)
		_, _ = io.WriteString(stdout, `{"engine":"in-memory"}`)
		return 0
	})
	out, err := (Store{Root: t.TempDir()}).runCLI(context.Background(), []string{"t-engine"})
	if err != nil || !strings.Contains(string(out), "updated") || inProcess.Load() != 0 {
		t.Fatalf("a replaced binary was not asked: %q %v (in-process runs %d)", out, err, inProcess.Load())
	}
	// POSE_EXECUTABLE keeps its precedence over the in-process route.
	other := filepath.Join(dir, "pose-other")
	if err := os.WriteFile(other, []byte("#!/bin/sh\necho '{\"engine\":\"configured\"}'\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("POSE_EXECUTABLE", other)
	if inProcessRunsTheSpawnedEngine() {
		t.Fatal("POSE_EXECUTABLE names another binary: it takes precedence")
	}
	out, err = (Store{Root: t.TempDir()}).runCLI(context.Background(), []string{"t-engine"})
	if err != nil || !strings.Contains(string(out), "configured") {
		t.Fatalf("POSE_EXECUTABLE was ignored: %q %v", out, err)
	}
}

func TestAPanicInAnInProcessCommandIsAFailedRunNotACrash(t *testing.T) {
	RegisterInProcessCommand("t-panic", func(root string, args []string, stdout, stderr io.Writer) int { panic("boom") })
	_, err := (Store{Root: "/r"}).runCLI(context.Background(), []string{"t-panic"})
	if err == nil || !strings.Contains(err.Error(), "panic: boom") {
		t.Fatalf("a panic must become an error carrying its message: %v", err)
	}
	if len(inProcessSlots) != 0 {
		t.Fatalf("the slot of a panicked run was not given back: %d held", len(inProcessSlots))
	}
}
