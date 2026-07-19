package cli

// Runtime guardrail behavior (spec pose-validation-runtime-guardrails):
// timeout and output-limit produce explicit error states, isolation-required
// checks never run locally, and the execution plan binds digests and an
// approval slot.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func guardrailFixture(t *testing.T, matrix map[string]any) string {
	t.Helper()
	root := t.TempDir()
	write := func(rel, content string) {
		path := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("mod/go.mod", "module fixture\n")
	raw, _ := json.Marshal(matrix)
	write(".pose/indexes/validation-matrix.json", string(raw))
	return root
}

func TestGuardrailTimeoutState(t *testing.T) {
	root := guardrailFixture(t, map[string]any{
		"defaults": map[string]any{"mode": "strict"},
		"stacks": map[string]any{"go": map[string]any{"checks": []map[string]any{
			{"name": "hang", "program": "sleep", "args": []string{"5"}, "severity": "required", "timeoutSeconds": 1},
		}}},
	})
	code, _ := runValidate(t, root, "--json", "result.json")
	if code != 1 {
		t.Fatalf("required timeout must fail the run, exit=%d", code)
	}
	run := loadRun(t, root)
	c := run.Checks[0]
	if c.Outcome != "error" || c.LimitState != "timeout" {
		t.Errorf("timeout state = %s/%s, want error/timeout", c.Outcome, c.LimitState)
	}
	if c.DurationSeconds < 0.9 || c.DurationSeconds > 3 {
		t.Errorf("duration = %.2fs, want ~1s (cancelled at the limit)", c.DurationSeconds)
	}
	if !strings.Contains(c.Output, "timeout after 1s") {
		t.Errorf("output = %q", c.Output)
	}
}

func TestGuardrailOutputLimitState(t *testing.T) {
	root := guardrailFixture(t, map[string]any{
		"defaults": map[string]any{"mode": "strict", "maxOutputBytes": 4096},
		"stacks": map[string]any{"go": map[string]any{"checks": []map[string]any{
			{"name": "flood", "program": "yes", "severity": "optional", "timeoutSeconds": 30},
		}}},
	})
	code, _ := runValidate(t, root, "--json", "result.json")
	if code != 0 {
		t.Fatalf("optional flood must not block, exit=%d", code)
	}
	run := loadRun(t, root)
	c := run.Checks[0]
	if c.Outcome != "error" || c.LimitState != "output-limit" {
		t.Errorf("flood state = %s/%s, want error/output-limit", c.Outcome, c.LimitState)
	}
	if c.DurationSeconds > 10 {
		t.Errorf("flood should be cancelled quickly, took %.2fs", c.DurationSeconds)
	}
	if run.Outcome != "partial" {
		t.Errorf("run outcome = %s, want partial", run.Outcome)
	}
}

func TestGuardrailIsolationDelegation(t *testing.T) {
	root := guardrailFixture(t, map[string]any{
		"defaults": map[string]any{"mode": "strict"},
		"stacks": map[string]any{"go": map[string]any{"checks": []map[string]any{
			{"name": "trusted", "program": "true", "severity": "required"},
			{"name": "hostile", "program": "true", "severity": "required", "isolation": "required"},
		}}},
	})
	code, out := runValidate(t, root, "--json", "result.json", "--emit-plan", "plan.json")
	if code != 0 {
		t.Fatalf("exit=%d output=%s", code, out)
	}
	run := loadRun(t, root)
	var hostile checkResult
	for _, c := range run.Checks {
		if c.Name == "hostile" {
			hostile = c
		}
	}
	if hostile.Outcome != "skipped" || hostile.Isolation != "required" || !strings.Contains(hostile.SkipReason, "isolated execution") {
		t.Errorf("isolation-required must be skipped locally with reason: %+v", hostile)
	}
	raw, err := os.ReadFile(filepath.Join(root, "plan.json"))
	if err != nil {
		t.Fatal(err)
	}
	var plan executionPlan
	if err := json.Unmarshal(raw, &plan); err != nil {
		t.Fatal(err)
	}
	if len(plan.Checks) != 1 || plan.Checks[0].Name != "hostile" {
		t.Errorf("plan checks = %+v, want only the isolated check", plan.Checks)
	}
	if len(plan.MatrixSHA256) != 64 || !plan.Approval.Required || plan.Approval.Identity != "" {
		t.Errorf("plan must bind matrix digest and an unstamped approval slot: %+v", plan)
	}
	if plan.ProjectID == "" {
		t.Error("plan must bind the project id")
	}
}
