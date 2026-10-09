package pose

import (
	"os/exec"
	"strings"
	"testing"
)

// The snapshot answers what one `git status -- path` per path answered, and the
// commit cache what one `git diff-tree -- path` per pair answered (spec
// pose-check-spawns-fewer-git-processes).
func TestCheckGitSpawnsSnapshotMatchesStatusPerPath(t *testing.T) {
	root := t.TempDir()
	designDeltaGit(t, root, "init", "-q")
	designDeltaGit(t, root, "config", "user.email", "pose@example.test")
	designDeltaGit(t, root, "config", "user.name", "POSE Test")
	writeReviewFixture(t, root, "api/clean.go", "package api\n")
	writeReviewFixture(t, root, "api/edited.go", "package api\n")
	writeReviewFixture(t, root, "api/old_name.go", "package api\n")
	designDeltaGit(t, root, "add", "--", ".")
	designDeltaGit(t, root, "commit", "-q", "-m", "base")
	writeReviewFixture(t, root, "api/edited.go", "package api\n\n// edited\n")
	writeReviewFixture(t, root, "docs/new/untracked.md", "# new\n")
	designDeltaGit(t, root, "mv", "api/old_name.go", "api/new_name.go")

	snapshot := takeWorkingTreeSnapshot(root)
	for _, path := range []string{"api/clean.go", "api/edited.go", "docs/new", "docs/new/untracked.md", "api/new_name.go", "api/old_name.go", "api", "missing.go"} {
		out, err := exec.Command("git", "-C", root, "status", "--porcelain=v1", "--untracked-files=all", "--", path).Output()
		if err != nil {
			t.Fatal(err)
		}
		want := strings.TrimSpace(string(out)) != ""
		if got, _ := snapshot.change(path); got != want {
			t.Errorf("%s: snapshot dirty=%v, git status dirty=%v", path, got, want)
		}
	}
	if dirty, detail := snapshot.change("api/edited.go"); !dirty || detail != " (git status M)" {
		t.Errorf("edited file detail = %q", detail)
	}
	if dirty, _ := takeWorkingTreeSnapshot(t.TempDir()).change("anything"); dirty {
		t.Error("a tree without Git reported a change")
	}
}

func TestCheckGitSpawnsCommitCacheMatchesDiffTreePerPath(t *testing.T) {
	root := t.TempDir()
	designDeltaGit(t, root, "init", "-q")
	designDeltaGit(t, root, "config", "user.email", "pose@example.test")
	designDeltaGit(t, root, "config", "user.name", "POSE Test")
	writeReviewFixture(t, root, "a.txt", "a\n")
	designDeltaGit(t, root, "add", "--", ".")
	designDeltaGit(t, root, "commit", "-q", "-m", "root")
	rootCommit := strings.TrimSpace(string(mustDesignDeltaGitOutput(t, root, "rev-parse", "HEAD")))
	writeReviewFixture(t, root, "b/c.txt", "c\n")
	writeReviewFixture(t, root, "a.txt", "a2\n")
	designDeltaGit(t, root, "add", "--", ".")
	designDeltaGit(t, root, "commit", "-q", "-m", "second")
	second := strings.TrimSpace(string(mustDesignDeltaGitOutput(t, root, "rev-parse", "HEAD")))
	store := Store{Root: root}
	for _, commit := range []string{rootCommit, second} {
		for _, path := range []string{"a.txt", "b/c.txt", "b", "missing"} {
			out, err := exec.Command("git", "-C", root, "diff-tree", "--root", "--no-commit-id", "--name-only", "--no-renames", "-r", commit, "--", path).Output()
			if err != nil {
				t.Fatal(err)
			}
			want := false
			for _, changed := range strings.Split(strings.TrimSpace(string(out)), "\n") {
				if changed == path {
					want = true
				}
			}
			if got := store.reviewBundleCommitChangedPath(commit, path); got != want {
				t.Errorf("%s %s: cached=%v diff-tree=%v", commit[:8], path, got, want)
			}
		}
	}
	if store.reviewBundleCommitChangedPath("not-a-sha", "a.txt") {
		t.Error("a non-object-id commit was resolved")
	}
}
