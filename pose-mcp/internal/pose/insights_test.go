package pose

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestInsightsAggregatesAndCountsInvalidHistory(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, ".pose", "reports", "history")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	history := "" +
		`{"generated_at":"2026-07-18T00:00:00Z","workflow":"feature","task_slug":"alpha","context":"ci","outcome":"pass"}` + "\n" +
		`{"generated_at":"2026-07-18T00:01:00Z","workflow":"feature","task_slug":"alpha","context":"ci","outcome":"fail"}` + "\n" +
		"invalid\n"
	if err := os.WriteFile(filepath.Join(dir, "runs.jsonl"), []byte(history), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := (Store{Root: root}).Insights("task", 0)
	if err != nil {
		t.Fatal(err)
	}
	if result.GroupBy != "task" || result.RecordsScanned != 2 || result.RecordsSkippedInvalid != 1 {
		t.Fatalf("unexpected result: %+v", result)
	}
	if len(result.Rows) != 1 || result.Rows[0].Key != "alpha" || result.Rows[0].Pass != 1 || result.Rows[0].Fail != 1 {
		t.Fatalf("unexpected rows: %+v", result.Rows)
	}
	if result.Rows[0].PassRate == nil || *result.Rows[0].PassRate != 0.5 {
		t.Fatalf("pass rate = %v, want 0.5", result.Rows[0].PassRate)
	}
}

func TestInsightsAppliesTimeWindow(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, ".pose", "reports", "history")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	history := `{"generated_at":"2000-01-01T00:00:00Z","workflow":"old","outcome":"fail"}` + "\n" +
		`{"generated_at":"` + time.Now().UTC().Format(time.RFC3339) + `","workflow":"current","outcome":"pass"}` + "\n"
	if err := os.WriteFile(filepath.Join(dir, "runs.jsonl"), []byte(history), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := (Store{Root: root}).Insights("workflow", 1)
	if err != nil {
		t.Fatal(err)
	}
	if result.RecordsScanned != 2 || result.RecordsSkippedByWindow != 1 || len(result.Rows) != 1 || result.Rows[0].Key != "current" {
		t.Fatalf("windowed insights = %+v", result)
	}
}

func TestInsightsValidatesInputsAndDefaults(t *testing.T) {
	store := Store{Root: t.TempDir()}
	result, err := store.Insights("", 0)
	if err != nil || result.GroupBy != "workflow" || result.Rows == nil {
		t.Fatalf("default insights = %+v, err=%v", result, err)
	}
	if _, err := store.Insights("owner", 0); err == nil {
		t.Error("invalid group accepted")
	}
	if _, err := store.Insights("workflow", -1); err == nil {
		t.Error("negative since_days accepted")
	}
}
