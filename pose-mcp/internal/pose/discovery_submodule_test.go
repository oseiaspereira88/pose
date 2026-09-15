package pose_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/harne8/pose-mcp/internal/pose"
)

func gitIn(t *testing.T, dir string, args ...string) {
	t.Helper()
	base := []string{"-C", dir, "-c", "user.name=pose-test", "-c", "user.email=pose-test@example.invalid", "-c", "protocol.file.allow=always", "-c", "commit.gpgsign=false"}
	if out, err := exec.Command("git", append(base, args...)...).CombinedOutput(); err != nil {
		t.Fatalf("git %s: %v: %s", strings.Join(args, " "), err, out)
	}
}

func writeFixtureFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// submoduleWithIgnoredCache builds a parent repository with a submodule at
// mod/ whose own .gitignore excludes cache/, and creates mod/cache/ with a
// manifest and a README in the checkout — the shape of a pytest or build
// cache left inside a vendored submodule.
func submoduleWithIgnoredCache(t *testing.T) string {
	t.Helper()
	upstream := t.TempDir()
	gitIn(t, upstream, "init", "-q")
	writeFixtureFile(t, filepath.Join(upstream, ".gitignore"), "cache/\n")
	writeFixtureFile(t, filepath.Join(upstream, "tool", "go.mod"), "module example.com/tool\n")
	gitIn(t, upstream, "add", ".")
	gitIn(t, upstream, "commit", "-q", "-m", "upstream")

	root := t.TempDir()
	gitIn(t, root, "init", "-q")
	gitIn(t, root, "submodule", "add", "-q", upstream, "mod")
	writeFixtureFile(t, filepath.Join(root, "mod", "cache", "go.mod"), "module example.com/cache\n")
	writeFixtureFile(t, filepath.Join(root, "mod", "cache", "README.md"), "cache\n")
	return root
}

// A repository's `git ls-files --ignored` never descends into a submodule,
// so a path the submodule itself ignores used to reach every discovery
// walker as if it were tracked content (spec
// pose-discovery-gitignore-inside-submodules).
func TestGitIgnoredPathsReportsPathsIgnoredInsideASubmodule(t *testing.T) {
	root := submoduleWithIgnoredCache(t)

	ignored := pose.GitIgnoredPaths(root)
	if !ignored["mod/cache/"] {
		t.Fatalf("GitIgnoredPaths = %v, want mod/cache/ (ignored by the submodule's own .gitignore)", ignored)
	}
	if ignored["mod/tool/"] {
		t.Fatalf("GitIgnoredPaths marked tracked submodule content mod/tool/ as ignored: %v", ignored)
	}
}

// git's line-oriented output C-quotes a non-ASCII name ("caf\303\251/"), and
// trimming each line strips a leading space that belongs to the name: either
// way the reported path never matched the directory a walker visits. Reading
// NUL-separated output keeps names byte for byte, at the root and inside a
// submodule. core.quotePath is forced on so a developer who turned it off
// globally still sees the test fail without the fix.
func TestGitIgnoredPathsKeepsNamesGitWouldQuote(t *testing.T) {
	t.Setenv("GIT_CONFIG_COUNT", "1")
	t.Setenv("GIT_CONFIG_KEY_0", "core.quotePath")
	t.Setenv("GIT_CONFIG_VALUE_0", "true")
	root := submoduleWithIgnoredCache(t)
	writeFixtureFile(t, filepath.Join(root, ".gitignore"), "/café/\n/ spaced/\n")
	writeFixtureFile(t, filepath.Join(root, "café", "go.mod"), "module example.com/cafe\n")
	writeFixtureFile(t, filepath.Join(root, " spaced", "go.mod"), "module example.com/spaced\n")
	writeFixtureFile(t, filepath.Join(root, "mod", ".gitignore"), "cache/\ncafé/\n")
	writeFixtureFile(t, filepath.Join(root, "mod", "café", "go.mod"), "module example.com/modcafe\n")

	ignored := pose.GitIgnoredPaths(root)
	for _, want := range []string{"café/", " spaced/", "mod/café/", "mod/cache/"} {
		if !ignored[want] {
			t.Errorf("GitIgnoredPaths is missing %q: %v", want, ignored)
		}
	}
}

// An uninitialised submodule is an empty directory. Running git inside it
// resolves to the parent repository, which lists the same gitlink again, so
// descending into it without a check never returns.
func TestGitIgnoredPathsSkipsAnUninitialisedSubmodule(t *testing.T) {
	root := submoduleWithIgnoredCache(t)
	writeFixtureFile(t, filepath.Join(root, ".gitignore"), "/scratch/\n")
	writeFixtureFile(t, filepath.Join(root, "scratch", "notes.md"), "x\n")
	gitIn(t, root, "add", ".gitignore")
	gitIn(t, root, "submodule", "deinit", "-q", "-f", "mod")

	done := make(chan map[string]bool, 1)
	go func() { done <- pose.GitIgnoredPaths(root) }()
	var ignored map[string]bool
	select {
	case ignored = <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("GitIgnoredPaths did not return: it keeps descending into the uninitialised submodule mod/")
	}
	if !ignored["scratch/"] {
		t.Fatalf("GitIgnoredPaths = %v, want the parent's own scratch/", ignored)
	}
	for path := range ignored {
		if strings.HasPrefix(path, "mod/") {
			t.Fatalf("GitIgnoredPaths reported %q under an uninitialised submodule: %v", path, ignored)
		}
	}
}
