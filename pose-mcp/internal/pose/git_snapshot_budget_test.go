package pose

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
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

// cleanupBudgetFixture registers a detached review worktree of a commit
// whose tree is 4 KiB, with the snapshot margin lowered to 1 KiB, as
// dispatch does, and captures its cleanup ownership.
func cleanupBudgetFixture(t *testing.T, extra int) (Store, string, os.FileInfo, string, os.FileInfo, []byte, *reviewCleanupOwnership) {
	t.Helper()
	original := gitSnapshotMarginBytes
	t.Cleanup(func() { gitSnapshotMarginBytes = original })
	gitSnapshotMarginBytes = 1 << 10
	repo := t.TempDir()
	federatedTestGit(t, repo, "init", "-q")
	for _, kv := range [][]string{{"user.name", "fixture"}, {"user.email", "fixture@example.invalid"}, {"commit.gpgsign", "false"}} {
		federatedTestGit(t, repo, "config", kv[0], kv[1])
	}
	if err := os.WriteFile(filepath.Join(repo, "tracked"), bytes.Repeat([]byte{'x'}, 4<<10), 0o600); err != nil {
		t.Fatal(err)
	}
	federatedTestGit(t, repo, "add", "-A")
	federatedTestGit(t, repo, "commit", "-qm", "fixture")
	tmp := filepath.Join(t.TempDir(), "operation")
	if err := os.Mkdir(tmp, 0o700); err != nil {
		t.Fatal(err)
	}
	owned, err := os.Lstat(tmp)
	if err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(tmp)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { root.Close() })
	worktree := filepath.Join(tmp, "worktree")
	federatedTestGit(t, repo, "worktree", "add", "-q", "--detach", worktree, "HEAD")
	if extra > 0 {
		if err := os.WriteFile(filepath.Join(worktree, "grown"), bytes.Repeat([]byte{'y'}, extra), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	worktreeOwned, err := os.Lstat(worktree)
	if err != nil {
		t.Fatal(err)
	}
	marker, err := os.ReadFile(filepath.Join(worktree, ".git"))
	if err != nil {
		t.Fatal(err)
	}
	store := Store{Root: repo}
	ownership, err := store.captureReviewCleanupOwnership(root, worktree, marker)
	if err != nil {
		t.Fatal(err)
	}
	return store, tmp, owned, worktree, worktreeOwned, marker, ownership
}

// R1: a review worktree whose checked-out tree is past the margin is
// removed whole by its cleanup.
func TestReviewWorktreeCleanupCoversTheCommittedTree(t *testing.T) {
	store, tmp, owned, worktree, worktreeOwned, marker, ownership := cleanupBudgetFixture(t, 0)
	if err := store.cleanupReviewResources(tmp, owned, worktree, worktreeOwned, marker, ownership); err != nil {
		t.Fatalf("cleanup of a worktree past the margin: %v", err)
	}
	if _, err := os.Lstat(worktree); !os.IsNotExist(err) {
		t.Fatalf("the worktree was not removed: %v", err)
	}
}

// R2: a worktree that grew past its committed tree by more than the margin
// keeps failing closed.
func TestReviewWorktreeCleanupStillBoundsGrowth(t *testing.T) {
	store, tmp, owned, worktree, worktreeOwned, marker, ownership := cleanupBudgetFixture(t, 64<<10)
	if err := store.cleanupReviewResources(tmp, owned, worktree, worktreeOwned, marker, ownership); err == nil {
		t.Fatal("cleanup of a worktree grown past its budget succeeded")
	}
	if _, err := os.Lstat(filepath.Join(worktree, "grown")); err != nil {
		t.Fatalf("a refused cleanup removed the worktree's contents: %v", err)
	}
}

// Review N2: a tree listing past its bound is not retained; the comparison
// fails, and the disk snapshot and the cleanup fall back to the margin alone,
// which a tree past the margin exceeds, so all three fail closed.
func TestGitTreeListingIsBounded(t *testing.T) {
	original := gitTreeListingMaxBytes
	t.Cleanup(func() { gitTreeListingMaxBytes = original })
	root, head := snapshotBudgetRepo(t)
	gitTreeListingMaxBytes = 64
	if _, err := listCommittedTree(root, head); err == nil {
		t.Fatal("a listing past its bound was returned")
	}
	if _, err := gitMaterializedChanges(root, head); err == nil {
		t.Fatal("the comparison accepted a listing past its bound")
	}
	if _, err := dispatchDiskSnapshot(root); err == nil {
		t.Fatal("the disk snapshot of a tree past the margin succeeded without its listing")
	}
	store, tmp, owned, worktree, worktreeOwned, marker, ownership := cleanupBudgetFixture(t, 0)
	if ownership.inventoryBytes != gitSnapshotMarginBytes {
		t.Fatalf("the cleanup budget was computed from a listing past its bound: %d", ownership.inventoryBytes)
	}
	if err := store.cleanupReviewResources(tmp, owned, worktree, worktreeOwned, marker, ownership); err == nil {
		t.Fatal("the cleanup of a worktree past the margin succeeded without its listing")
	}
}

// Review N4: the governed comparison a closeout runs under its producer
// bounds the listing by the same bytes and deadline, and keeps working
// within them.
func TestGovernedTreeListingIsBounded(t *testing.T) {
	root, head := snapshotBudgetRepo(t)
	governed := func() error {
		_, view := Store{Root: root}.beginGovernanceReadView()
		if view == nil {
			t.Fatal("no read view")
		}
		_, err := gitMaterializedChangesForIn(view.filesFor("closeout-plan"), root, head, nil, "closeout-plan")
		return err
	}
	if err := governed(); err != nil {
		t.Fatalf("the governed comparison within its bounds: %v", err)
	}
	maxBytes, timeout := gitTreeListingMaxBytes, gitTreeListingTimeout
	t.Cleanup(func() { gitTreeListingMaxBytes, gitTreeListingTimeout = maxBytes, timeout })
	gitTreeListingMaxBytes = 64
	if err := governed(); err == nil {
		t.Fatal("the governed comparison accepted a listing past its byte bound")
	}
	gitTreeListingMaxBytes = maxBytes
	gitTreeListingTimeout = time.Nanosecond
	if err := governed(); err == nil {
		t.Fatal("the governed comparison accepted a listing past its deadline")
	}
	if _, err := listCommittedTree(root, head); err == nil {
		t.Fatal("a listing past its deadline was returned")
	}
}
