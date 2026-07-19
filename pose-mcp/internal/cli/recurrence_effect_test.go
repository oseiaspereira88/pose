package cli

// Recurrence effectiveness behavior (spec pose-recurrence-effectiveness):
// registration validation, effective/ineffective verdicts, sparse-data and
// incomplete-window warnings, partial telemetry and the opt-in blocking flag.

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func effectFixture(t *testing.T, interventionAgeDays, windowDays int, afterFailures int) string {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, ".pose", "reports", "history")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	at := now.AddDate(0, 0, -interventionAgeDays)
	var history strings.Builder
	rec := func(offsetDays int, outcome string, extra string) {
		ts := at.AddDate(0, 0, offsetDays).Format(time.RFC3339)
		history.WriteString(fmt.Sprintf(`{"generated_at":%q,"task_slug":"flaky-task","outcome":%q%s}`+"\n", ts, outcome, extra))
	}
	// Before the intervention: three failures with telemetry.
	rec(-5, "fail", `,"duration_seconds":120,"cost_usd":2.5`)
	rec(-3, "fail", `,"duration_seconds":100,"cost_usd":2.0`)
	rec(-1, "fail", "")
	// After: configurable failures plus one pass.
	for i := 0; i < afterFailures; i++ {
		rec(2+i, "fail", "")
	}
	rec(1, "pass", "")
	if err := os.WriteFile(filepath.Join(dir, "standard-flaky-task.jsonl"), []byte(history.String()), 0o644); err != nil {
		t.Fatal(err)
	}
	iv := fmt.Sprintf(`{"schema":1,"at":%q,"task_slug":"flaky-task","ref":"rule:flaky-guard","window_days":%d,"rationale":"guard added","author":"@core"}`+"\n", at.Format(time.RFC3339), windowDays)
	if err := os.WriteFile(filepath.Join(dir, "interventions.jsonl"), []byte(iv), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

func runEffect(t *testing.T, root string, args ...string) (int, string) {
	t.Helper()
	var out, errB bytes.Buffer
	code := cmdRecurrenceEffect(root, args, &out, &errB)
	return code, out.String() + errB.String()
}

func TestRecurrenceEffectEffective(t *testing.T) {
	root := effectFixture(t, 40, 30, 0) // window complete, failures dropped 3 → 0
	code, out := runEffect(t, root)
	if code != 0 {
		t.Fatalf("exit=%d: %s", code, out)
	}
	if !strings.Contains(out, "[EFFECTIVE] failures before:3/3 after:0/1") {
		t.Errorf("expected effective verdict, got: %s", out)
	}
	if !strings.Contains(out, "avg_duration_s before:110.00 after:n/a") || !strings.Contains(out, "avg_cost_usd before:2.25") {
		t.Errorf("expected partial telemetry averages, got: %s", out)
	}
}

func TestRecurrenceEffectIneffectiveBlocksOptIn(t *testing.T) {
	root := effectFixture(t, 40, 30, 4) // window complete, failures 3 → 4
	code, out := runEffect(t, root)
	if code != 0 {
		t.Fatalf("default run must not block, exit=%d", code)
	}
	if !strings.Contains(out, "[INEFFECTIVE]") || !strings.Contains(out, "reopen or spawn a governed follow-up") {
		t.Errorf("expected ineffective verdict with governed action, got: %s", out)
	}
	if code, _ := runEffect(t, root, "--fail-ineffective"); code != 1 {
		t.Fatalf("--fail-ineffective must exit 1, got %d", code)
	}
}

func TestRecurrenceEffectWarnings(t *testing.T) {
	root := effectFixture(t, 5, 30, 0) // window incomplete
	_, out := runEffect(t, root, "--min-sample", "10")
	if !strings.Contains(out, "[INCONCLUSIVE]") {
		t.Errorf("expected inconclusive verdict, got: %s", out)
	}
	if !strings.Contains(out, "insufficient sample") || !strings.Contains(out, "observation window incomplete") {
		t.Errorf("expected both data-quality warnings, got: %s", out)
	}
}

func TestRecurrenceEffectRegisterValidation(t *testing.T) {
	root := t.TempDir()
	if code, _ := runEffect(t, root, "--register", "--task", "x", "--ref", "bogus", "--rationale", "r", "--author", "@a"); code == 0 {
		t.Fatal("invalid ref must be rejected")
	}
	if code, _ := runEffect(t, root, "--register", "--task", "x", "--ref", "spec:ghost", "--rationale", "r", "--author", "@a"); code == 0 {
		t.Fatal("spec ref must resolve to an existing spec")
	}
	if code, out := runEffect(t, root, "--register", "--task", "x", "--ref", "rule:guard", "--rationale", "r", "--author", "@a"); code != 0 {
		t.Fatalf("valid registration failed: %s", out)
	}
	code, out := runEffect(t, root)
	if code != 0 || !strings.Contains(out, "effect.interventions=1") {
		t.Fatalf("registered intervention should be reported: %s", out)
	}
}
