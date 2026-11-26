package pose

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// Spec pose-validate-reads-leave-the-index-alone R1, R2: the review history
// diff against the work tree leaves a stale index byte-identical and answers
// as a plain diff does.
func TestReviewHistoryDiffLeavesTheIndexAlone(t *testing.T) {
	root := t.TempDir()
	for _, args := range [][]string{{"init", "-q"}} {
		if out, err := exec.Command("git", append([]string{"-C", root}, args...)...).CombinedOutput(); err != nil {
			t.Fatal(err, string(out))
		}
	}
	if err := os.WriteFile(filepath.Join(root, "a.txt"), []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"add", "a.txt"}, {"-c", "user.name=t", "-c", "user.email=t@t", "commit", "-q", "-m", "i"}} {
		if out, err := exec.Command("git", append([]string{"-C", root}, args...)...).CombinedOutput(); err != nil {
			t.Fatal(err, string(out))
		}
	}
	later := time.Now().Add(2 * time.Second)
	if err := os.Chtimes(filepath.Join(root, "a.txt"), later, later); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(filepath.Join(root, ".git", "index"))
	if err != nil {
		t.Fatal(err)
	}
	out, err := reviewHistoryGit(root, "diff", "HEAD", "--name-status", "--no-renames", "--")
	if err != nil || len(bytes.TrimSpace(out)) != 0 {
		t.Fatalf("the diff against the work tree: %q %v", out, err)
	}
	after, err := os.ReadFile(filepath.Join(root, ".git", "index"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Fatal("the review history diff rewrote .git/index")
	}
}
