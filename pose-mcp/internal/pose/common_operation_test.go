package pose

import (
	"context"
	"errors"
	"io"
	"strings"
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
	h := CloseoutHandoff(handoffPlan())
	if h == nil || h.Executes || h.Operation != "close" || h.Scope != "spec:x" || h.Phase != "closeout" {
		t.Fatalf("handoff = %+v", h)
	}
	want := []string{"pose", "close", "spec:x", "--apply", "--digest", "aaaa"}
	if strings.Join(h.Argv, "\x00") != strings.Join(want, "\x00") {
		t.Fatalf("argv = %q", h.Argv)
	}
	if h.Inputs.PlanDigest != "sha256:aaaa" || h.Inputs.Revision != "rev" || h.Inputs.SpecDigest != "sd" || h.Inputs.PolicyDigest != "pd" {
		t.Fatalf("inputs = %+v", h.Inputs)
	}
	if h.RequiredAuthority == "" || h.Receipt == "" || h.Invalidation == "" || !strings.Contains(h.ContextRequirement, "--expect-context") {
		t.Fatalf("a handoff must say who may run it, how the outcome returns and when it stops being valid: %+v", h)
	}
}

func TestAHandoffIsDeterministicAndFollowsThePlan(t *testing.T) {
	a, b := CloseoutHandoff(handoffPlan()), CloseoutHandoff(handoffPlan())
	if a.RequestID != b.RequestID {
		t.Fatalf("the same plan gave two ids: %s %s", a.RequestID, b.RequestID)
	}
	moved := handoffPlan()
	moved.Digest = "sha256:bbbb"
	if CloseoutHandoff(moved).RequestID == a.RequestID {
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
		if CloseoutHandoff(plan) != nil {
			t.Errorf("%s: a handoff was offered for nothing to apply", name)
		}
	}
}

func TestAPendingJudgmentNamesTheReviewerAsTheAuthority(t *testing.T) {
	plan := handoffPlan()
	plan.Steps = append(plan.Steps, CloseoutPlanStep{ID: CloseoutStepJudgment, State: CloseoutStateWaiting})
	if h := CloseoutHandoff(plan); h == nil || !strings.Contains(h.RequiredAuthority, "reviewer") {
		t.Fatalf("a pending judgment is a reviewer's: %+v", h)
	}
}
