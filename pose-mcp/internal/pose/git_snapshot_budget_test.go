package pose

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// snapshotBudgetRepo is a committed repository of four 4 KiB files, with the
// snapshot margin lowered to 1 KiB: its committed tree alone is past the
// margin, as pose-dist's 256.8 MiB tree is past the old 256 MiB bound.
func snapshotBudgetRepo(t *testing.T) (string, string) {
	t.Helper()
	original := gitSnapshotMarginBytes
	t.Cleanup(func() { gitSnapshotMarginBytes = original })
	gitSnapshotMarginBytes = 1 << 10
	root := t.TempDir()
	federatedTestGit(t, root, "init", "-q")
	for _, kv := range [][]string{{"user.name", "fixture"}, {"user.email", "fixture@example.invalid"}, {"commit.gpgsign", "false"}} {
		federatedTestGit(t, root, "config", kv[0], kv[1])
	}
	for i := 0; i < 4; i++ {
		name := filepath.Join(root, "data-"+string(rune('a'+i))+".txt")
		if err := os.WriteFile(name, bytes.Repeat([]byte{'x'}, 4<<10), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	federatedTestGit(t, root, "add", "-A")
	federatedTestGit(t, root, "commit", "-q", "-m", "data")
	head, err := gitCommand("-C", root, "rev-parse", "HEAD").Output()
	if err != nil {
		t.Fatal(err)
	}
	return root, strings.TrimSpace(string(head))
}

// Spec pose-git-snapshot-budget-scales-with-the-tree, R1: a committed tree
// larger than the margin is compared and snapshotted whole.
func TestGitSnapshotBudgetCoversTheCommittedTree(t *testing.T) {
	root, head := snapshotBudgetRepo(t)
	changed, err := gitMaterializedChanges(root, head)
	if err != nil || len(changed) != 0 {
		t.Fatalf("a clean tree past the margin: changed=%v err=%v", changed, err)
	}
	entries, err := dispatchDiskSnapshot(root)
	if err != nil || len(entries) < 4 {
		t.Fatalf("a disk snapshot of a tree past the margin: %d entries, err=%v", len(entries), err)
	}
}

// R2: disk that grew past the committed tree by more than the margin still
// fails closed.
func TestGitSnapshotBudgetStillBoundsGrowth(t *testing.T) {
	root, head := snapshotBudgetRepo(t)
	if err := os.WriteFile(filepath.Join(root, "data-a.txt"), bytes.Repeat([]byte{'y'}, 64<<10), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := gitMaterializedChanges(root, head); err == nil || !strings.Contains(err.Error(), "budget") {
		t.Fatalf("a tree that grew past its budget: %v, want a budget failure", err)
	}
	if _, err := dispatchDiskSnapshot(root); err == nil || !strings.Contains(err.Error(), "budget") {
		t.Fatalf("a disk that grew past its budget: %v, want a budget failure", err)
	}
}

// R3: the budget is never below the margin and never above the ceiling.
func TestGitSnapshotBudgetIsBounded(t *testing.T) {
	if got := gitSnapshotBudget(0); got != gitSnapshotMarginBytes {
		t.Fatalf("an empty tree: %d, want the margin %d", got, gitSnapshotMarginBytes)
	}
	if got := gitSnapshotBudget(1 << 40); got != gitSnapshotCeilingBytes {
		t.Fatalf("a huge tree: %d, want the ceiling %d", got, gitSnapshotCeilingBytes)
	}
	if got := gitSnapshotBudget(-5); got != gitSnapshotMarginBytes {
		t.Fatalf("a negative size: %d", got)
	}
}
