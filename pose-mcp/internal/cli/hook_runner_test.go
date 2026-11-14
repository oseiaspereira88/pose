package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Spec pose-hooks-delegate-to-the-instance-runner.
func hookRunnerRepo(t *testing.T, policy string) (string, string) {
	t.Helper()
	root := t.TempDir()
	artifactGit(t, root, "init", "-q")
	marker := filepath.Join(t.TempDir(), "ran")
	writeArtifactTestFile(t, root, "scripts/hook.sh", "#!/bin/sh\nprintf '%s' \"$1\" > '"+marker+"'\n[ \"$1\" = post-merge ] && exit 3\nexit 0\n")
	if err := os.Chmod(filepath.Join(root, "scripts/hook.sh"), 0o755); err != nil {
		t.Fatal(err)
	}
	if policy != "" {
		writeArtifactTestFile(t, root, ".pose/policy/hooks.json", policy)
	}
	t.Chdir(root)
	return root, marker
}

// R1: a declared runner gets the hook's name and its exit code is the hook's.
func TestAHookDelegatesToTheDeclaredRunner(t *testing.T) {
	_, marker := hookRunnerRepo(t, `{"schema_version":1,"runner":"scripts/hook.sh"}`)
	var out, errOut bytes.Buffer
	if code := HookMain("pre-commit", &out, &errOut); code != 0 {
		t.Fatalf("pre-commit through the runner: %d %s", code, errOut.String())
	}
	if got, _ := os.ReadFile(marker); string(got) != "pre-commit" {
		t.Fatalf("the runner was called with %q", got)
	}
	if code := HookMain("post-merge", &out, &errOut); code != 3 {
		t.Fatalf("the runner's exit code was not the hook's: %d", code)
	}
}

// R2: a runner outside the project, not executable or under an unknown
// schema is refused, and nothing runs.
func TestAnInvalidHookRunnerIsRefused(t *testing.T) {
	for name, policy := range map[string]string{
		"outside":        `{"schema_version":1,"runner":"../hook.sh"}`,
		"absolute":       `{"schema_version":1,"runner":"/bin/true"}`,
		"missing":        `{"schema_version":1,"runner":"scripts/absent.sh"}`,
		"unknown schema": `{"schema_version":2,"runner":"scripts/hook.sh"}`,
		"unknown field":  `{"schema_version":1,"runner":"scripts/hook.sh","shell":true}`,
	} {
		t.Run(name, func(t *testing.T) {
			_, marker := hookRunnerRepo(t, policy)
			var out, errOut bytes.Buffer
			if code := HookMain("pre-commit", &out, &errOut); code != 1 || !strings.Contains(errOut.String(), "hooks policy") {
				t.Fatalf("an invalid runner was not refused: %d %s", code, errOut.String())
			}
			if _, err := os.Stat(marker); err == nil {
				t.Fatal("an invalid runner ran")
			}
		})
	}
}

// R3: status names the runner a managed hook delegates to.
func TestHookStatusNamesTheRunner(t *testing.T) {
	root, _ := hookRunnerRepo(t, `{"schema_version":1,"runner":"scripts/hook.sh"}`)
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(exe, filepath.Join(root, ".git", "hooks", "pre-commit")); err != nil {
		t.Fatal(err)
	}
	var out, errOut bytes.Buffer
	if code := cmdHooks(root, []string{"status"}, &out, &errOut); code != 0 || !strings.Contains(out.String(), "delegates to "+filepath.Join(root, "scripts/hook.sh")) {
		t.Fatalf("status: %d %s %s", code, out.String(), errOut.String())
	}
}
