package pose

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Spec pose-governance-git-reads-untranslated: under a translated locale, a
// governance Git query in a directory outside any repository is still
// recognised as such, and a scope consuming an index there is not unknown.
func TestGovernanceGitIsReadUntranslated(t *testing.T) {
	probe := exec.Command("git", "-C", t.TempDir(), "rev-parse", "--show-toplevel")
	probe.Env = append(probe.Environ(), "LC_ALL=pt_BR.UTF-8", "LANGUAGE=pt_BR", "LANG=pt_BR.UTF-8")
	out, _ := probe.CombinedOutput()
	if strings.Contains(strings.ToLower(string(out)), "not a git repository") {
		t.Skip("Git is not translated to pt_BR on this machine")
	}
	t.Setenv("LC_ALL", "pt_BR.UTF-8")
	t.Setenv("LANGUAGE", "pt_BR")
	t.Setenv("LANG", "pt_BR.UTF-8")
	TestScopeIndexInputsAreCheckedOncePerCoordinator(t)
}

// Review round 1 (F1): both governance Git runners, captured and streamed,
// run Git untranslated whatever the session's locale.
func TestEveryGovernanceGitRunnerIsUntranslated(t *testing.T) {
	real, err := exec.LookPath("git")
	if err != nil {
		t.Skip("no git")
	}
	bin := t.TempDir()
	record := filepath.Join(t.TempDir(), "locale")
	script := "#!/bin/sh\nprintf '%s|%s\\n' \"$LC_ALL\" \"$LANGUAGE\" >> " + record + "\nexec " + real + " \"$@\"\n"
	if err := os.WriteFile(filepath.Join(bin, "git"), []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("LC_ALL", "pt_BR.UTF-8")
	t.Setenv("LANGUAGE", "pt_BR")
	dir := t.TempDir()
	admit := func(int64) bool { return true }
	_, _, _ = runGovernanceGitCommandOutputAdmittedContext(context.Background(), nil, dir, []string{"rev-parse", "--git-dir"}, 1<<20, admit)
	_, _, _, _ = runGovernanceGitStreamContext(context.Background(), nil, dir, []string{"rev-parse", "--git-dir"}, 1<<20, admit, nil)
	raw, err := os.ReadFile(record)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(raw)), "\n")
	if len(lines) != 2 {
		t.Fatalf("recorded %d Git commands, want 2: %q", len(lines), raw)
	}
	for i, line := range lines {
		if line != "C|C" {
			t.Fatalf("runner %d ran Git with LC_ALL|LANGUAGE %q", i, line)
		}
	}
}
