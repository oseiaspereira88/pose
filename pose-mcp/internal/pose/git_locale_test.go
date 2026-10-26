package pose

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// localizedGit puts first on the path a git that answers every command as a git outside any
// repository does, in Portuguese unless the C locale is forced, as a translated git does.
func localizedGit(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("the git shim is a shell script")
	}
	shim := t.TempDir()
	script := "#!/bin/sh\n" +
		"if [ \"$LC_ALL\" = C ]; then\n" +
		"  echo 'fatal: not a git repository (or any of the parent directories): .git' >&2\n" +
		"else\n" +
		"  echo 'fatal: não é um repositório git (ou qualquer pai até o ponto de montagem /)' >&2\n" +
		"fi\n" +
		"exit 128\n"
	if err := os.WriteFile(filepath.Join(shim, "git"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", shim+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("LANG", "pt_BR.UTF-8")
	t.Setenv("LC_ALL", "pt_BR.UTF-8")
	t.Setenv("LC_MESSAGES", "pt_BR.UTF-8")
	t.Setenv("LANGUAGE", "pt_BR")
}

// R1: a directory outside every repository is recognised whatever language git speaks.
func TestGitNotARepositoryIgnoresTheLocale(t *testing.T) {
	localizedGit(t)
	if !GitNotARepository(t.TempDir()) {
		t.Fatal("a directory outside every repository was not recognised under a translated git")
	}
}

// R2: the release readers that depend on it keep "no repository" apart from "git failed".
func TestReleaseTagOutsideARepositoryIsMissingInAnyLocale(t *testing.T) {
	localizedGit(t)
	if _, _, err := ResolveReleaseTag(t.TempDir(), "v1.0.0"); !errors.Is(err, ErrReleaseTagMissing) {
		t.Fatalf("a tag outside every repository must be missing, got %v", err)
	}
}
