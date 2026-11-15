package pose

import (
	"os/exec"
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
