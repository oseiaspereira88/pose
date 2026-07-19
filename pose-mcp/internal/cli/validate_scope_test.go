package cli

// Changed-scope behavior (spec pose-changed-scope-validation): deterministic
// selection with reasons, dependency widening, safe-execution fallback,
// machine-readable skip reasons and revision confinement.

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// scopeFixture builds a git repo with modules a and b (b depends on a) and
// one commit; returns root. Checks are trivial `true` programs.
func scopeFixture(t *testing.T, withDeps bool) string {
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
	write("a/go.mod", "module a\n")
	write("b/go.mod", "module b\n")
	write("root.txt", "root\n")
	matrix := `{"defaults":{"mode":"strict"},"stacks":{"go":{"checks":[{"name":"ok","program":"true","severity":"required"}]}}}`
	write(".pose/indexes/validation-matrix.json", matrix)
	if withDeps {
		write(".pose/indexes/module-metadata.json", `{"schemaVersion":1,"modules":{"b":{"dependsOn":["a"]}}}`)
	}
	git := func(args ...string) {
		cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
		cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v %s", args, err, out)
		}
	}
	git("init", "-q")
	git("add", "-A")
	git("commit", "-qm", "base")
	return root
}

func TestChangedScopeSelectsAffectedModule(t *testing.T) {
	root := scopeFixture(t, false)
	if err := os.WriteFile(filepath.Join(root, "a", "changed.go"), []byte("package a\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	code, out := runValidate(t, root, "--changed-from", "HEAD", "--explain", "--json", "result.json")
	if code != 0 {
		t.Fatalf("exit=%d out=%s", code, out)
	}
	if !strings.Contains(out, "+ a: contains changed file: a/changed.go") || !strings.Contains(out, "- b: not affected") {
		t.Errorf("explain output missing decisions: %s", out)
	}
	run := loadRun(t, root)
	var aOutcome, bReason string
	for _, c := range run.Checks {
		if c.Module == "a" {
			aOutcome = c.Outcome
		}
		if c.Module == "b" {
			bReason = c.SkipReason
		}
	}
	if aOutcome != "pass" {
		t.Errorf("module a should run, outcome=%s", aOutcome)
	}
	if !strings.Contains(bReason, "changed-scope: module not affected by HEAD..worktree") {
		t.Errorf("module b skip reason = %q", bReason)
	}
}

func TestChangedScopeDependencyWidening(t *testing.T) {
	root := scopeFixture(t, true)
	if err := os.WriteFile(filepath.Join(root, "a", "changed.go"), []byte("package a\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, out := runValidate(t, root, "--changed-from", "HEAD", "--explain")
	if !strings.Contains(out, "+ b: depends on selected module a") {
		t.Errorf("dependent module must be widened with a reason: %s", out)
	}
}

func TestChangedScopeRootChangeRunsEverything(t *testing.T) {
	root := scopeFixture(t, false)
	if err := os.WriteFile(filepath.Join(root, "root.txt"), []byte("edited\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, out := runValidate(t, root, "--changed-from", "HEAD", "--explain")
	for _, want := range []string{"+ a: root-level change outside any module: root.txt", "+ b: root-level change outside any module: root.txt"} {
		if !strings.Contains(out, want) {
			t.Errorf("safe-execution fallback missing %q in: %s", want, out)
		}
	}
}

func TestChangedScopeNoChangesSkipsAllWithReasons(t *testing.T) {
	root := scopeFixture(t, false)
	code, _ := runValidate(t, root, "--changed-from", "HEAD", "--json", "result.json")
	if code != 0 {
		t.Fatalf("exit=%d", code)
	}
	run := loadRun(t, root)
	if run.Counts.Executed != 0 || run.Counts.Skipped != 2 {
		t.Errorf("counts = %+v, want everything skipped with reasons", run.Counts)
	}
	raw, _ := json.Marshal(run.Checks)
	if !strings.Contains(string(raw), "changed-scope: module not affected") {
		t.Errorf("skip reasons must be machine-readable: %s", raw)
	}
}

func TestChangedScopeRejectsUnsafeRevision(t *testing.T) {
	root := scopeFixture(t, false)
	code, out := runValidate(t, root, "--changed-from", "-rev^{}")
	if code != 2 || !strings.Contains(out, "unsafe git revision") {
		t.Fatalf("unsafe revision must be rejected before git runs: code=%d out=%s", code, out)
	}
	code, out = runValidate(t, root, "--changed-to", "HEAD")
	if code != 2 || !strings.Contains(out, "--changed-from") {
		t.Fatalf("--changed-to alone must be rejected: code=%d out=%s", code, out)
	}
}
