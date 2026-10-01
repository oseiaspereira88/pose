package usage

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestVerdictJoinsAcrossLocalSaltsAndSupersedes(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, ".pose"), 0o755); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)
	for _, disposition := range []string{"valid", "false-positive"} {
		if err := RecordVerdict(root, VerdictInput{Tool: "validate", FindingID: "check-a", Disposition: disposition, Reason: "Reviewed check result", By: "maintainer", At: now}); err != nil {
			t.Fatal(err)
		}
	}
	if err := RecordVerdict(root, VerdictInput{Tool: "validate", FindingID: "check-missing", Disposition: "wont-fix", Reason: "Accepted backlog item", By: "maintainer", At: now}); err != nil {
		t.Fatal(err)
	}
	journal, err := os.ReadFile(verdictPath(root))
	if err != nil || strings.Count(string(journal), "\n") != 3 {
		t.Fatalf("journal lines=%q err=%v", journal, err)
	}
	var salts []string
	for _, dir := range []string{t.TempDir(), t.TempDir()} {
		t.Setenv("POSE_USAGE_DIR", dir)
		if err := Record(root, Observation{At: now, Tool: "validate", Surface: "cli", ExecutionOutcome: "completed", SemanticOutcome: "fail", Findings: []Finding{{ID: "check-a", Severity: "error"}}, FindingSetComplete: true, Scope: "project"}); err != nil {
			t.Fatal(err)
		}
		salt, err := os.ReadFile(filepath.Join(dir, "salt"))
		if err != nil {
			t.Fatal(err)
		}
		salts = append(salts, string(salt))
		report, err := Aggregate(root, Query{SinceDays: 0, Now: now.Add(time.Minute)})
		if err != nil {
			t.Fatal(err)
		}
		if len(report.Adjudications) != 1 {
			t.Fatalf("adjudications=%+v", report.Adjudications)
		}
		row := report.Adjudications[0]
		if row.Tool != "validate" || row.FalsePositive != 1 || row.Valid != 0 || row.WontFix != 0 || row.Unmatched != 1 || row.FalsePositiveRate != 1 {
			t.Fatalf("cross-salt verdict=%+v", row)
		}
		if report.Rows[0].FindingsObserved != 1 || report.Rows[0].UniqueFindings != 1 {
			t.Fatalf("automatic counts changed: %+v", report.Rows[0])
		}
		filtered, err := Aggregate(root, Query{SinceDays: 1, Surface: "mcp", Now: now.AddDate(0, 0, 2)})
		if err != nil || filtered.Available || len(filtered.Adjudications) != 1 || filtered.Adjudications[0].FalsePositive != 1 || filtered.Adjudications[0].Unmatched != 1 {
			t.Fatalf("verdicts should use all local history: %+v err=%v", filtered, err)
		}
	}
	if salts[0] == salts[1] {
		t.Fatal("cross-machine fixture reused the same salt")
	}
}

func TestVerdictUnmatchedWithoutLocalEvents(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, ".pose"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("POSE_USAGE_DIR", filepath.Join(t.TempDir(), "absent"))
	if err := RecordVerdict(root, VerdictInput{Tool: "check", FindingID: "rule-a", Disposition: "valid", Reason: "Observed elsewhere", By: "reviewer"}); err != nil {
		t.Fatal(err)
	}
	report, err := Aggregate(root, Query{Tool: "check"})
	if err != nil || report.Available || len(report.Adjudications) != 1 || report.Adjudications[0].Unmatched != 1 {
		t.Fatalf("unmatched report=%+v err=%v", report, err)
	}
}

func TestVerdictRejectsUnsafeIdentityAndCorruptJournal(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, ".pose"), 0o755); err != nil {
		t.Fatal(err)
	}
	input := VerdictInput{Tool: "check", FindingID: "safe-id", Disposition: "valid", Reason: "Reviewed", By: "reviewer"}
	for _, id := range []string{"/home/private.go:42", "../private.go", "src/../private.go", "customer@example.com", "bad id", "bad\nid"} {
		input.FindingID = id
		if err := RecordVerdict(root, input); err == nil {
			t.Fatalf("accepted unsafe finding ID %q", id)
		}
	}
	input.FindingID = "mod/go/broken"
	if err := RecordVerdict(root, input); err != nil {
		t.Fatalf("rejected structured validation ID: %v", err)
	}
	input.FindingID = "safe-id"
	input.Reason = ""
	if err := RecordVerdict(root, input); err == nil {
		t.Fatal("accepted empty reason")
	}
	path := verdictPath(root)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("not-json\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := RecordVerdict(root, VerdictInput{Tool: "check", FindingID: "safe-id", Disposition: "valid", Reason: "Reviewed", By: "reviewer"}); err == nil {
		t.Fatal("appended to corrupt journal")
	}
	t.Setenv("POSE_USAGE_DIR", filepath.Join(t.TempDir(), "absent"))
	if _, err := Aggregate(root, Query{}); err == nil {
		t.Fatal("report hid corrupt governed verdict")
	}
}

func TestVerdictRejectsSymlinkedJournal(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Dir(verdictPath(root)), 0o755); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(t.TempDir(), "outside.jsonl")
	if err := os.Symlink(outside, verdictPath(root)); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	input := VerdictInput{Tool: "check", FindingID: "safe-id", Disposition: "valid", Reason: "Reviewed", By: "reviewer"}
	if err := RecordVerdict(root, input); err == nil {
		t.Fatal("wrote through symlinked verdict journal")
	}
	if _, err := os.Stat(outside); !os.IsNotExist(err) {
		t.Fatalf("external file was touched: %v", err)
	}
}
