package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	usagepkg "github.com/harne8/pose-mcp/internal/usage"
)

func TestUsageRecordsKnownCLICommandAndExcludesQuery(t *testing.T) {
	repo := newGitRepo(t)
	usageDir := t.TempDir()
	t.Setenv("POSE_USAGE_DIR", usageDir)
	t.Setenv("POSE_USAGE_DISABLED", "")

	inDir(t, repo, func() {
		var out, errOut bytes.Buffer
		if code := Main([]string{"new-spec"}, &out, &errOut); code != 2 {
			t.Fatalf("new-spec without slug exit=%d stderr=%s", code, errOut.String())
		}
		out.Reset()
		errOut.Reset()
		if code := Main([]string{"usage", "--since-days", "0", "--json"}, &out, &errOut); code != 0 {
			t.Fatalf("usage exit=%d stderr=%s", code, errOut.String())
		}
		var report usagepkg.Report
		if err := json.Unmarshal(out.Bytes(), &report); err != nil {
			t.Fatalf("decode usage: %v\n%s", err, out.String())
		}
		if len(report.Rows) != 1 || report.Rows[0].Tool != "new-spec" || report.Rows[0].Invalid != 1 {
			t.Fatalf("unexpected usage rows: %+v", report.Rows)
		}
	})
}

func TestUsageRecordingFailureNeverChangesCommandResult(t *testing.T) {
	repo := newGitRepo(t)
	blocked := filepath.Join(t.TempDir(), "not-a-directory")
	if err := os.WriteFile(blocked, []byte("occupied"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("POSE_USAGE_DIR", blocked)
	t.Setenv("POSE_USAGE_DISABLED", "")

	inDir(t, repo, func() {
		var out, errOut bytes.Buffer
		code := Main([]string{"new-spec"}, &out, &errOut)
		if code != 2 {
			t.Fatalf("command result changed by usage storage failure: exit=%d stderr=%s", code, errOut.String())
		}
	})
}

func TestUsageRejectsPartiallyParsedSinceDays(t *testing.T) {
	repo := newGitRepo(t)
	t.Setenv("POSE_USAGE_DIR", t.TempDir())
	inDir(t, repo, func() {
		var out, errOut bytes.Buffer
		if code := Main([]string{"usage", "--since-days", "7days", "--json"}, &out, &errOut); code != 2 {
			t.Fatalf("usage exit=%d, want 2", code)
		}
	})
}

func TestInvalidGateInvocationIsNotASemanticFailure(t *testing.T) {
	result := defaultCommandUsage("validate", 2)
	if result.SemanticOutcome != "unknown" || result.FindingCount != 0 {
		t.Fatalf("invalid invocation summary = %+v", result)
	}
}

func TestValidateUsageUsesStructuredCheckFindings(t *testing.T) {
	root := resultFixture(t)
	t.Setenv("POSE_USAGE_DIR", t.TempDir())
	t.Setenv("POSE_USAGE_DISABLED", "")
	inDir(t, root, func() {
		var out, errOut bytes.Buffer
		if code := Main([]string{"validate", "--json", "result.json"}, &out, &errOut); code != 0 {
			t.Fatalf("validate exit=%d output=%s%s", code, out.String(), errOut.String())
		}
		report, err := usagepkg.Aggregate(root, usagepkg.Query{SinceDays: 0, Tool: "validate", Surface: "cli"})
		if err != nil {
			t.Fatal(err)
		}
		if len(report.Rows) != 1 {
			t.Fatalf("rows = %+v", report.Rows)
		}
		row := report.Rows[0]
		if row.Partial != 1 || row.FindingsObserved != 1 || row.UniqueFindings != 1 {
			t.Fatalf("validate usage = %+v", row)
		}
	})
}
