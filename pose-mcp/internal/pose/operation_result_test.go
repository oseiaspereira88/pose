package pose

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Spec pose-cli-mcp-operation-results R5 (PERF-07): a gate subprocess that
// does not finish is an inconclusive execution, never a failed gate.

func fakePoseExecutable(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "pose")
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+body+"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("POSE_EXECUTABLE", path)
	return path
}

func TestAGateKilledByItsDeadlineIsInconclusiveNotFailed(t *testing.T) {
	fakePoseExecutable(t, "echo partial line\nsleep 30")
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	started := time.Now()
	gate, err := Store{Root: t.TempDir()}.Check(ctx, true)
	if elapsed := time.Since(started); elapsed > 10*time.Second {
		t.Fatalf("the stopped gate held the call for %s", elapsed)
	}
	var inconclusive *GateExecutionError
	if gate != nil || !errors.As(err, &inconclusive) {
		t.Fatalf("a killed gate became a verdict: gate=%+v err=%v", gate, err)
	}
	if inconclusive.Cause != ExecutionCauseDeadline || inconclusive.Command != "pose check --strict" || !strings.Contains(inconclusive.PartialOutput, "partial line") || inconclusive.Cleanup == "" {
		t.Fatalf("the inconclusive result lost its cause, command, partial output or cleanup note: %+v", inconclusive)
	}
}

func TestACancelledGateIsInconclusive(t *testing.T) {
	fakePoseExecutable(t, "sleep 30")
	ctx, cancel := context.WithCancel(context.Background())
	go func() { time.Sleep(200 * time.Millisecond); cancel() }()
	_, err := Store{Root: t.TempDir()}.LintSpec(ctx, "", true)
	var inconclusive *GateExecutionError
	if !errors.As(err, &inconclusive) || inconclusive.Cause != ExecutionCauseCanceled {
		t.Fatalf("a cancelled gate was not inconclusive: %v", err)
	}
}

func TestAGateThatExitsNonZeroInTimeIsStillAVerdict(t *testing.T) {
	fakePoseExecutable(t, "echo 'check: 1 error'\nexit 1")
	gate, err := Store{Root: t.TempDir()}.Check(context.Background(), false)
	if err != nil || gate == nil || gate.Passed || gate.ExitCode != 1 || !strings.Contains(gate.Output, "1 error") {
		t.Fatalf("a finished failing gate is not a verdict: %+v %v", gate, err)
	}
	fakePoseExecutable(t, "echo ok")
	gate, err = Store{Root: t.TempDir()}.Check(context.Background(), false)
	if err != nil || gate == nil || !gate.Passed {
		t.Fatalf("a passing gate is not a verdict: %+v %v", gate, err)
	}
}

func TestAJSONQueryKilledByItsDeadlineIsInconclusive(t *testing.T) {
	fakePoseExecutable(t, "sleep 30")
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	_, err := Store{Root: t.TempDir()}.Followups(ctx, false)
	var inconclusive *GateExecutionError
	if !errors.As(err, &inconclusive) || inconclusive.Cause != ExecutionCauseDeadline {
		t.Fatalf("a killed query was not inconclusive: %v", err)
	}
}

func TestOperationResultStartsWithRangeableLists(t *testing.T) {
	result := NewOperationResult("close.apply", "spec:x")
	if result.SchemaVersion != OperationResultSchemaVersion || result.StepsRun == nil || result.Effects == nil {
		t.Fatalf("a new result is not ready to render: %+v", result)
	}
	result.Fail(OutcomeConflict, "", "plan-digest-mismatch", "changed")
	if result.Outcome != OutcomeConflict || result.Error == nil || result.Error.Cause != "plan-digest-mismatch" {
		t.Fatalf("Fail did not type the outcome: %+v", result)
	}
}
