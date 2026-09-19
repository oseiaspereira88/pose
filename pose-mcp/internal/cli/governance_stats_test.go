package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	posepkg "github.com/harne8/pose-mcp/internal/pose"
)

func TestGovernanceStatsCLIJSONIsReadOnlyAndSeparated(t *testing.T) {
	root := newGitRepo(t)
	history := filepath.Join(root, ".pose", "reports", "history", "runs.jsonl")
	if err := os.MkdirAll(filepath.Dir(history), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(history, []byte(`{"generated_at":"2026-09-19T00:00:00Z","task_slug":"alpha","outcome":"pass"}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	inDir(t, root, func() {
		var out, errOut bytes.Buffer
		if code := Main([]string{"stats", "governance", "--json"}, &out, &errOut); code != 0 {
			t.Fatalf("stats governance exit=%d stderr=%s", code, errOut.String())
		}
		var report posepkg.GovernanceOutcomesReport
		if err := json.Unmarshal(out.Bytes(), &report); err != nil {
			t.Fatalf("decode report: %v\n%s", err, out.String())
		}
		if report.Attempts.AttemptsObserved != 1 || report.Attempts.Pass != 1 {
			t.Fatalf("unexpected attempts: %+v", report.Attempts)
		}
		if strings.Contains(out.String(), root) || strings.Contains(out.String(), "alpha") {
			t.Fatalf("governance JSON leaked path or unit identity: %s", out.String())
		}
		if _, err := os.Stat(filepath.Join(root, ".pose", "review-bundles")); err == nil {
			t.Fatal("read-only stats unexpectedly created review artifacts")
		}
	})
}

func TestGovernanceStatsCLIRejectsUnknownOption(t *testing.T) {
	root := newGitRepo(t)
	inDir(t, root, func() {
		var out, errOut bytes.Buffer
		if code := Main([]string{"stats", "governance", "--wat"}, &out, &errOut); code != 2 {
			t.Fatalf("exit=%d stdout=%s stderr=%s", code, out.String(), errOut.String())
		}
	})
}
