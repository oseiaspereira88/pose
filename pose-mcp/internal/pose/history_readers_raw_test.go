package pose

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// rawHistoryFixture is a file changed twice, with HEAD replaced or grafted so
// that a reader honouring either sees another history than the raw one.
func rawHistoryFixture(t *testing.T) (root, first, second string) {
	t.Helper()
	root = t.TempDir()
	reviewFixtureGit(t, root, "init", "-q")
	write := func(path, body, message string) string {
		t.Helper()
		if err := os.WriteFile(filepath.Join(root, path), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		reviewFixtureGit(t, root, "add", "--all")
		reviewFixtureGit(t, root, "commit", "-qm", message)
		return reviewFixtureGit(t, root, "rev-parse", "HEAD")
	}
	first = write("a.txt", "one\n", "first")
	second = write("b.txt", "two\n", "second")
	return root, first, second
}

func TestGitCommandReadsRawHistory(t *testing.T) {
	check := func(t *testing.T, root, want, second string) {
		t.Helper()
		out, err := gitCommand("-C", root, "rev-parse", second+"^{tree}").Output()
		if err != nil {
			t.Fatal(err)
		}
		if tree := strings.TrimSpace(string(out)); tree != want {
			t.Fatalf("a replace ref reached gitCommand: %s, want %s", tree, want)
		}
		count, err := gitCommand("-C", root, "rev-list", "--count", second).Output()
		if err != nil {
			t.Fatal(err)
		}
		if strings.TrimSpace(string(count)) != "2" {
			t.Fatalf("a graft reached gitCommand: %s commits", count)
		}
		if paths, ok := commitChangedPaths(root, second); !ok || !paths["b.txt"] {
			t.Fatalf("the commit's own paths: %v %v", paths, ok)
		}
	}
	t.Run("replace ref", func(t *testing.T) {
		root, first, second := rawHistoryFixture(t)
		tree := reviewFixtureGit(t, root, "rev-parse", second+"^{tree}")
		reviewFixtureGit(t, root, "replace", second, first)
		check(t, root, tree, second)
	})
	t.Run("repository graft", func(t *testing.T) {
		root, _, second := rawHistoryFixture(t)
		tree := reviewFixtureGit(t, root, "rev-parse", second+"^{tree}")
		if err := os.WriteFile(filepath.Join(root, ".git", "info", "grafts"), []byte(second+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		check(t, root, tree, second)
	})
	t.Run("graft from the environment", func(t *testing.T) {
		root, _, second := rawHistoryFixture(t)
		tree := reviewFixtureGit(t, root, "rev-parse", second+"^{tree}")
		graft := filepath.Join(t.TempDir(), "grafts")
		if err := os.WriteFile(graft, []byte(second+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		t.Setenv("GIT_GRAFT_FILE", graft)
		check(t, root, tree, second)
	})
}

// The readers that did not go through gitCommand.
func TestReviewReadersReadRawHistory(t *testing.T) {
	root, first, second := rawHistoryFixture(t)
	reviewFixtureGit(t, root, "replace", second, first)
	log, err := gitAttributionLog(root, []string{second})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(log), "b.txt") {
		t.Fatalf("attribution read a replaced commit: %s", log)
	}
	exists, err := gitObjectExists(root, second, "b.txt")
	if err != nil || !exists {
		t.Fatalf("the design delta read a replaced commit: %v %v", exists, err)
	}
	raw, err := gitShowBounded(root, second, "b.txt", 1024)
	if err != nil || string(raw) != "two\n" {
		t.Fatalf("%q %v", raw, err)
	}
	reader := &gitBatchReader{root: root}
	defer reader.close()
	if err := reader.start(); err != nil {
		t.Fatal(err)
	}
	if reader.cmd.Env == nil || !strings.Contains(strings.Join(reader.cmd.Env, "\n"), "GIT_NO_REPLACE_OBJECTS=1") {
		t.Fatal("the batch reader is not raw")
	}
}

// The readers of a change set's range take a root change set's base (R3).
func TestRangeReadersTakeTheEmptyTree(t *testing.T) {
	root, first, second := rawHistoryFixture(t)
	if got, err := gitRevListCount(root, GitEmptyTree(first), second); err != nil || got != 2 {
		t.Fatalf("%d %v", got, err)
	}
	baseline, err := (Store{Root: root}).resolveReviewPolicyBaseline(GitEmptyTree(first), "spec")
	if err != nil || !baseline.Resolved || baseline.Policy.Enabled {
		t.Fatalf("no contract governed before the first commit: %+v %v", baseline, err)
	}
}

func TestChangeSetRange(t *testing.T) {
	for _, c := range []struct{ base, want string }{
		{"abc", "abc..h"},
		{gitEmptyTreeSHA1, "h"},
		{gitEmptyTreeSHA256, "h"},
	} {
		if got := ChangeSetRange(c.base, "h"); got != c.want {
			t.Errorf("%s: %s", c.base, got)
		}
	}
	if GitEmptyTree(strings.Repeat("a", 40)) != gitEmptyTreeSHA1 || GitEmptyTree(strings.Repeat("a", 64)) != gitEmptyTreeSHA256 {
		t.Fatal("the object format")
	}
}
