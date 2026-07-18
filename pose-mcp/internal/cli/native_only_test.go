package cli

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/crisol/pose-mcp/internal/scaffold"
)

func TestNativeOnlyAdvertisedMaintenanceCommands(t *testing.T) {
	repo := newGitRepo(t)
	var out, errOut bytes.Buffer
	if rc := cmdInstall([]string{repo, "--skip-mcp"}, &out, &errOut); rc != 0 {
		t.Fatalf("install rc=%d out=%s err=%s", rc, out.String(), errOut.String())
	}
	if err := os.WriteFile(filepath.Join(repo, "go.mod"), []byte("module example.invalid/native\n\ngo 1.26\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	inDir(t, repo, func() {
		commands := [][]string{
			{"index"}, {"suggest", "feature", "--json"}, {"stats", "workflows", "--json"},
			{"recurrence-check", "--strict"}, {"knowledge-check", "--strict"},
			{"knowledge-housekeeping", "list-expired"}, {"reports-housekeeping", "list-stale"},
			{"upgrade", "--dry-run"}, {"init", "--wizard", "--yes"},
			{"release-notes", "--version", "v0.1.0"},
		}
		for _, command := range commands {
			out.Reset()
			errOut.Reset()
			if rc := Main(command, &out, &errOut); rc != 0 {
				t.Errorf("%v rc=%d out=%s err=%s", command, rc, out.String(), errOut.String())
			}
		}
	})
}

func TestValidateRejectsLegacyShellCommand(t *testing.T) {
	repo := newGitRepo(t)
	if err := os.MkdirAll(filepath.Join(repo, ".pose", "indexes"), 0o755); err != nil {
		t.Fatal(err)
	}
	matrix := `{"defaults":{"mode":"strict"},"stacks":{"go":{"checks":[{"name":"legacy","command":"echo unsafe","severity":"required"}]}},"moduleOverrides":{}}`
	if err := os.WriteFile(filepath.Join(repo, ".pose", "indexes", "validation-matrix.json"), []byte(matrix), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "go.mod"), []byte("module example.invalid/test\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	inDir(t, repo, func() {
		var out, errOut bytes.Buffer
		if rc := Main([]string{"validate", "--strict"}, &out, &errOut); rc != 2 || !strings.Contains(errOut.String(), "program + args + env") {
			t.Fatalf("legacy command accepted rc=%d out=%s err=%s", rc, out.String(), errOut.String())
		}
	})
}

func TestStatsHTMLIncludesRoadmapInsightsAndEscapesHistory(t *testing.T) {
	repo := newGitRepo(t)
	if err := os.MkdirAll(filepath.Join(repo, ".pose", "reports", "history"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(repo, ".pose", "specs", "done"), 0o755); err != nil {
		t.Fatal(err)
	}
	spec := "---\nslug: done\nstatus: done\ncreated_at: 2026-07-01\ncompleted_at: 2026-07-03\n---\n## 7. Final Report\n### Follow-ups\n- [open] inspect aging\n"
	if err := os.WriteFile(filepath.Join(repo, ".pose", "specs", "done", "spec.md"), []byte(spec), 0o644); err != nil {
		t.Fatal(err)
	}
	history := "{\"generated_at\":\"2026-07-18T00:00:00Z\",\"workflow\":\"<script>alert(1)</script>\",\"task_slug\":\"task\",\"outcome\":\"pass\"}\ninvalid\n"
	if err := os.WriteFile(filepath.Join(repo, ".pose", "reports", "history", "x.jsonl"), []byte(history), 0o644); err != nil {
		t.Fatal(err)
	}
	var out, errOut bytes.Buffer
	path := filepath.Join(repo, ".pose", "reports", "insights.html")
	if rc := cmdStats(repo, []string{"--html", "--out", path}, &out, &errOut); rc != 0 {
		t.Fatalf("rc=%d err=%s", rc, errOut.String())
	}
	raw, e := os.ReadFile(path)
	if e != nil {
		t.Fatal(e)
	}
	text := string(raw)
	for _, want := range []string{"Open follow-ups: 1", "Average lead time: 2.0 days", "&lt;script&gt;alert(1)&lt;/script&gt;", "Invalid skipped: 1", "Outcomes by workflow", "Outcomes by task"} {
		if !strings.Contains(text, want) {
			t.Errorf("HTML missing %q", want)
		}
	}
	if strings.Contains(text, "<script>alert(1)</script>") {
		t.Error("untrusted history was not escaped")
	}
}

func TestIndexUsesConfiguredMetadataDefaults(t *testing.T) {
	repo := newGitRepo(t)
	if err := os.MkdirAll(filepath.Join(repo, ".pose", "indexes"), 0o755); err != nil {
		t.Fatal(err)
	}
	metadata := `{"defaults":{"owner":"platform","criticality":"high","domain":"governance","validationProfile":"strict"},"modules":{}}`
	if err := os.WriteFile(filepath.Join(repo, ".pose", "indexes", "module-metadata.json"), []byte(metadata), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "go.mod"), []byte("module example.invalid/index\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var out, errOut bytes.Buffer
	if rc := cmdIndex(repo, nil, &out, &errOut); rc != 0 {
		t.Fatalf("index rc=%d err=%s", rc, errOut.String())
	}
	raw, err := os.ReadFile(filepath.Join(repo, ".pose", "indexes", "packages.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"owner": "platform"`, `"criticality": "high"`, `"domain": "governance"`, `"validationProfile": "strict"`} {
		if !strings.Contains(string(raw), want) {
			t.Errorf("packages index missing configured default %s", want)
		}
	}
}

func TestEmbeddedDistributionContainsNoLegacyRuntime(t *testing.T) {
	forbidden := []string{"pose", ".pose/scripts", ".pose/hooks", "pre-commit/run-pose-hook"}
	err := fs.WalkDir(scaffold.Dist(), ".", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		for _, prefix := range forbidden {
			if path == prefix || strings.HasPrefix(path, prefix+"/") {
				t.Errorf("legacy runtime embedded: %s", path)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestHooksLinkNativeExecutable(t *testing.T) {
	repo := newGitRepo(t)
	var out, errOut bytes.Buffer
	if rc := cmdHooks(repo, []string{"install"}, &out, &errOut); rc != 0 {
		t.Fatalf("install rc=%d err=%s", rc, errOut.String())
	}
	for _, name := range []string{"pre-commit", "post-merge"} {
		if _, err := os.Readlink(filepath.Join(repo, ".git", "hooks", name)); err != nil {
			t.Errorf("%s is not native symlink: %v", name, err)
		}
	}
	if rc := cmdHooks(repo, []string{"uninstall"}, &out, &errOut); rc != 0 {
		t.Fatalf("uninstall rc=%d", rc)
	}
}
