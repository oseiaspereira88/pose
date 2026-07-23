package cli

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/harne8/pose-mcp/internal/pose"
)

func stateTestRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	runStateGit(t, root, "init")
	runStateGit(t, root, "config", "user.email", "test@example.com")
	runStateGit(t, root, "config", "user.name", "State Test")
	writeStateTestFile(t, root, ".pose/specs/alpha/spec.md", "---\nslug: alpha\nstatus: done\ncompleted_at: 2026-07-01\n---\n\n# Spec: alpha\n")
	writeStateTestFile(t, root, ".pose/specs/beta/spec.md", "---\nslug: beta\nstatus: draft\n---\n\n# Spec: beta\n\n## 7. Final Report\n\n### Follow-ups\n\n- [open] Fazer coisa X. (owner:@team crit:high review:2020-01-01)\n")
	runStateGit(t, root, "add", "-A")
	runStateGit(t, root, "commit", "-m", "seed")
	return root
}

func writeStateTestFile(t *testing.T, root, rel, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func runStateGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v: %s", args, err, out)
	}
	return string(out)
}

func runState(t *testing.T, root string, args ...string) (int, string, string) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	code := cmdState(root, args, &stdout, &stderr)
	return code, stdout.String(), stderr.String()
}

func TestCmdState_ValidateWithoutArtifactIsAdditive(t *testing.T) {
	root := stateTestRoot(t)
	code, out, _ := runState(t, root)
	if code != 0 || !strings.Contains(out, "not initialized") {
		t.Fatalf("missing artifact must be a valid, non-failing state: code=%d out=%q", code, out)
	}
}

func TestCmdState_InitRefuseReinit(t *testing.T) {
	root := stateTestRoot(t)
	if code, out, _ := runState(t, root, "init"); code != 0 || !strings.Contains(out, "initialized") {
		t.Fatalf("init failed: code=%d out=%q", code, out)
	}
	if code, _, errOut := runState(t, root, "init"); code == 0 || !strings.Contains(errOut, "already exists") {
		t.Fatalf("re-init must refuse: code=%d stderr=%q", code, errOut)
	}
}

func TestCmdState_RefreshWithoutInitFails(t *testing.T) {
	root := stateTestRoot(t)
	if code, _, errOut := runState(t, root, "refresh"); code == 0 || !strings.Contains(errOut, "init") {
		t.Fatalf("refresh without init must fail nominally: code=%d stderr=%q", code, errOut)
	}
}

func TestCmdState_InitContentReflectsRepo(t *testing.T) {
	root := stateTestRoot(t)
	if code, _, errOut := runState(t, root, "init"); code != 0 {
		t.Fatalf("init: %v", errOut)
	}
	raw, err := os.ReadFile(filepath.Join(root, ".pose", "state", "project-state.md"))
	if err != nil {
		t.Fatal(err)
	}
	content := string(raw)
	for _, want := range []string{
		"schema_version: 1", "staleness_policy: max_age_days=7,max_commits=20",
		"## Resumo executivo", "<!-- state:curated -->",
		"## Specs & Roadmaps", "total=2", "done=1", "draft=1", "spec:alpha",
		"## Follow-ups", "abertos: 1", "owner:@team",
		"## Arquitetura", "status:unavailable",
	} {
		if !strings.Contains(content, want) {
			t.Errorf("project-state.md missing %q\n---\n%s", want, content)
		}
	}
	code, out, _ := runState(t, root)
	if code != 0 || !strings.Contains(out, "Result: SUCCESS") {
		t.Fatalf("fresh init must validate clean: code=%d out=%q", code, out)
	}
}

func TestCmdState_RefreshPreservesCuratedEditsAndUpdatesDerived(t *testing.T) {
	root := stateTestRoot(t)
	if code, _, errOut := runState(t, root, "init"); code != 0 {
		t.Fatalf("init: %v", errOut)
	}
	path := filepath.Join(root, ".pose", "state", "project-state.md")
	raw, _ := os.ReadFile(path)
	edited := strings.Replace(string(raw), curatedExecSummaryPlaceholder, "Este projeto governa X.", 1)
	if err := os.WriteFile(path, []byte(edited), 0o644); err != nil {
		t.Fatal(err)
	}

	writeStateTestFile(t, root, ".pose/specs/gamma/spec.md", "---\nslug: gamma\nstatus: blocked\n---\n\n# Spec: gamma\n")

	if code, _, errOut := runState(t, root, "refresh"); code != 0 {
		t.Fatalf("refresh: %v", errOut)
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	content := string(after)
	if !strings.Contains(content, "Este projeto governa X.") {
		t.Error("refresh discarded the curated edit")
	}
	if !strings.Contains(content, "total=3") || !strings.Contains(content, "blocked=1") {
		t.Errorf("refresh did not pick up the new spec: %s", content)
	}
}

func TestCmdState_RefreshTwiceWithNoRepoChangeIsContentStable(t *testing.T) {
	root := stateTestRoot(t)
	if code, _, errOut := runState(t, root, "init"); code != 0 {
		t.Fatalf("init: %v", errOut)
	}
	entriesAfterInit := readHistoryForTest(t, root)
	if code, _, errOut := runState(t, root, "refresh"); code != 0 {
		t.Fatalf("refresh 1: %v", errOut)
	}
	if code, _, errOut := runState(t, root, "refresh"); code != 0 {
		t.Fatalf("refresh 2: %v", errOut)
	}
	entries := readHistoryForTest(t, root)
	if len(entries) != len(entriesAfterInit)+2 {
		t.Fatalf("history entries = %d, want %d", len(entries), len(entriesAfterInit)+2)
	}
	last, prev := entries[len(entries)-1], entries[len(entries)-2]
	for name, prevSec := range prev.Sections {
		lastSec, ok := last.Sections[name]
		if !ok {
			t.Fatalf("section %q missing from the later refresh", name)
			continue
		}
		if lastSec.Hash != prevSec.Hash {
			t.Errorf("section %q content hash changed with no repo change: %q -> %q", name, prevSec.Hash, lastSec.Hash)
		}
	}
}

func readHistoryForTest(t *testing.T, root string) []stateHistoryEntry {
	t.Helper()
	entries, err := readStateHistory(filepath.Join(root, ".pose", "state", "history.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	return entries
}

func TestCmdState_ValidateDetectsTamperedSection(t *testing.T) {
	root := stateTestRoot(t)
	if code, _, errOut := runState(t, root, "init"); code != 0 {
		t.Fatalf("init: %v", errOut)
	}
	path := filepath.Join(root, ".pose", "state", "project-state.md")
	raw, _ := os.ReadFile(path)
	tampered := strings.Replace(string(raw), "total=2", "total=999", 1)
	if err := os.WriteFile(path, []byte(tampered), 0o644); err != nil {
		t.Fatal(err)
	}
	code, out, _ := runState(t, root)
	if code == 0 || !strings.Contains(out, "TAMPERED") || !strings.Contains(out, "Result: FAILURE") {
		t.Fatalf("hand-edited derived section must fail validation: code=%d out=%q", code, out)
	}
}

func TestCmdState_DiffNeedsTwoRefreshes(t *testing.T) {
	root := stateTestRoot(t)
	if code, _, errOut := runState(t, root, "init"); code != 0 {
		t.Fatalf("init: %v", errOut)
	}
	if code, out, _ := runState(t, root, "diff"); code != 0 || !strings.Contains(out, "Not enough history") {
		t.Fatalf("single-entry diff must report insufficient history: code=%d out=%q", code, out)
	}
	writeStateTestFile(t, root, ".pose/specs/gamma/spec.md", "---\nslug: gamma\nstatus: blocked\n---\n\n# Spec: gamma\n")
	if code, _, errOut := runState(t, root, "refresh"); code != 0 {
		t.Fatalf("refresh: %v", errOut)
	}
	code, out, _ := runState(t, root, "diff")
	if code != 0 {
		t.Fatalf("diff: %s", out)
	}
	if !strings.Contains(out, "Specs & Roadmaps") || !strings.Contains(out, "+") {
		t.Fatalf("diff did not surface the added spec: %q", out)
	}
}

func TestProvideCapabilities_AbsentIsHonest(t *testing.T) {
	root := stateTestRoot(t)
	if got := provideCapabilities(pose.Store{Root: root}); !strings.Contains(got, "ausente") {
		t.Errorf("provideCapabilities without an assessment = %q, want it to say ausente", got)
	}
}

// TestNewestReportsFirst_IgnoresFilenamePrefix guards a real bug found while
// dogfooding this spec: sorting .pose/reports/*.md by filename put
// undated legacy names (e.g. README.md, workspace-experience-e2e.md)
// ahead of recently dated reports, because "R"/"w" sort after the digit
// "2" in "2026-...". Recency must come from modification time.
func TestNewestReportsFirst_IgnoresFilenamePrefix(t *testing.T) {
	root := t.TempDir()
	reports := filepath.Join(root, ".pose", "reports")
	if err := os.MkdirAll(reports, 0o755); err != nil {
		t.Fatal(err)
	}
	write := func(name string, age time.Duration) {
		path := filepath.Join(reports, name)
		if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
		modTime := time.Now().Add(-age)
		if err := os.Chtimes(path, modTime, modTime); err != nil {
			t.Fatal(err)
		}
	}
	write("README.md", 365*24*time.Hour)                    // old, undated filename
	write("2020-01-01-ancient-review.md", 200*24*time.Hour) // old, dated filename
	write("2026-07-23-fresh-review.md", 1*time.Hour)        // actually newest

	names := newestReportsFirst(root)
	if len(names) != 3 {
		t.Fatalf("names = %v, want 3 entries", names)
	}
	if names[0] != "2026-07-23-fresh-review.md" {
		t.Fatalf("newest report = %q, want the actually-newest file first (not alphabetical): %v", names[0], names)
	}
}

func TestProvideArchitecture_AlwaysUnavailable(t *testing.T) {
	if got := provideArchitecture(); !strings.Contains(got, "indisponível") {
		t.Errorf("provideArchitecture = %q, want it to say indisponível", got)
	}
}
