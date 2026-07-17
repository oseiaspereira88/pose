package pose

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func fixtureReportsStore(t *testing.T) Store {
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

	// Write two history files
	write(".pose/reports/history/task-a.jsonl", `{"generated_at":"2026-06-11T12:00:00Z","task":"task-a","report_path":"/abs/path/to/2026-06-11-standard-task-a.md","outcome":"pass"}
{"generated_at":"2026-06-11T14:00:00Z","task":"task-a","report_path":"/abs/path/to/2026-06-11-standard-task-a.md","outcome":"fail"}
`)
	write(".pose/reports/history/task-b.jsonl", `{"generated_at":"2026-06-11T13:00:00Z","task":"task-b","report_path":"/abs/path/to/2026-06-11-standard-task-b.md","outcome":"pass"}
`)
	write(".pose/reports/history/retrospective-v1.jsonl", `{"generated_at":"2026-06-11T15:00:00Z","task":"retro","report_path":"/abs/path/to/retro.md","outcome":"pass","retrospective":{"version":"v1","spec_count":2}}
`)

	// Write markdown reports
	write(".pose/reports/2026-06-11-standard-task-a.md", "# Report A\nPassed standard validation.")
	write(".pose/reports/2026-06-11-standard-task-b.md", "# Report B\nPassed with warning.")

	return Store{Root: root}
}

func TestListReports(t *testing.T) {
	store := fixtureReportsStore(t)

	reports, err := store.ListReports()
	if err != nil {
		t.Fatalf("ListReports failed: %v", err)
	}

	// Total lines across the three files: 2 + 1 + 1 = 4.
	if len(reports) != 4 {
		t.Errorf("expected 4 reports, got %d", len(reports))
	}
	if !strings.Contains(string(reports[0].Retrospective), `"version":"v1"`) {
		t.Fatalf("structured retrospective was not preserved: %s", reports[0].Retrospective)
	}

	// Should be sorted by GeneratedAt DESC:
	// 1st: 2026-06-11T14:00:00Z (task-a, fail)
	// 2nd: 2026-06-11T13:00:00Z (task-b, pass)
	// 3rd: 2026-06-11T12:00:00Z (task-a, pass)

	if reports[1].GeneratedAt != "2026-06-11T14:00:00Z" || reports[1].Outcome != "fail" {
		t.Errorf("1st report mismatch: %+v", reports[0])
	}
	if reports[2].GeneratedAt != "2026-06-11T13:00:00Z" || reports[2].Task != "task-b" {
		t.Errorf("2nd report mismatch: %+v", reports[1])
	}
	if reports[3].GeneratedAt != "2026-06-11T12:00:00Z" || reports[3].Outcome != "pass" {
		t.Errorf("3rd report mismatch: %+v", reports[2])
	}

	// Filename should be extracted
	if reports[1].Filename != "2026-06-11-standard-task-a.md" {
		t.Errorf("expected filename '2026-06-11-standard-task-a.md', got %q", reports[1].Filename)
	}
}

func TestGetReport(t *testing.T) {
	store := fixtureReportsStore(t)

	// Valid read
	rep, err := store.GetReport("2026-06-11-standard-task-a.md")
	if err != nil {
		t.Fatalf("GetReport failed: %v", err)
	}
	if rep.Filename != "2026-06-11-standard-task-a.md" {
		t.Errorf("Filename mismatch: got %q", rep.Filename)
	}
	if !strings.Contains(rep.Body, "Passed standard validation.") {
		t.Errorf("Body content mismatch: got %q", rep.Body)
	}

	// Non-existent
	_, err = store.GetReport("nonexistent.md")
	if err == nil {
		t.Error("expected error for nonexistent file, got nil")
	}

	// Traversal safety checks
	badFilenames := []string{
		"../reports/2026-06-11-standard-task-a.md",
		"sub/2026-06-11-standard-task-a.md",
		"2026-06-11-standard-task-a.jsonl",
		"passwd",
	}

	for _, bad := range badFilenames {
		_, err := store.GetReport(bad)
		if err == nil {
			t.Errorf("expected error for traversal/bad name %q, got nil", bad)
		}
	}
}
