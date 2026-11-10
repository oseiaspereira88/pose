package pose

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// Spec pose-git-status-leaves-the-git-directory-alone.

// Asking which paths changed writes nothing under .git: a stale stat cache in the index is left
// as it is, so a governed read view that watches .git does not see its own question as a change.
func TestGitChangedPathsWritesNothingUnderTheGitDirectory(t *testing.T) {
	root := snapshotRepo(t, map[string]string{"a.md": "one\n", "b.md": "two\n"})
	// The content is the same and the stat data is not: a plain `git status` refreshes the index.
	later := time.Now().Add(2 * time.Hour)
	if err := os.Chtimes(filepath.Join(root, "a.md"), later, later); err != nil {
		t.Fatal(err)
	}
	indexPath := filepath.Join(root, ".git", "index")
	before, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := gitChangedPaths(root); err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Fatal("asking for the changed paths rewrote .git/index")
	}
	if _, err := os.Lstat(filepath.Join(root, ".git", "index.lock")); err == nil {
		t.Fatal("a lock file was left under .git")
	}
}
