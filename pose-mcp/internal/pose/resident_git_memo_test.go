package pose

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Spec pose-worktree-runtime-resident-cache.

// R1: the memo is off by default, bounded in bytes, least recently used first, and keyed by the
// commit, the shallow state and the exact command.
func TestTheResidentGitMemoIsOffByDefaultBoundedAndKeyedByTheCommit(t *testing.T) {
	if residentGit.Load() != nil {
		t.Fatal("the resident memo is on without being asked")
	}
	root := snapshotRepo(t, map[string]string{"a.md": "one\n"})
	m := EnableResidentGitMemo(400)
	defer DisableResidentGitMemo()
	args := []string{"log", "-1", "--format=%H", "--", "a.md"}
	k1, ok := residentGitKey(root, "head1", args)
	if !ok {
		t.Fatal("a plain .git directory is not memoizable")
	}
	k2, _ := residentGitKey(root, "head2", args)
	if k1 == k2 {
		t.Fatal("two commits share a key")
	}
	if err := os.WriteFile(filepath.Join(root, ".git", "shallow"), []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if k3, _ := residentGitKey(root, "head1", args); k3 == k1 {
		t.Fatal("a shallow repository shares the key of a full one")
	}
	m.put("a", []byte(strings.Repeat("x", 150)))
	m.put("b", []byte(strings.Repeat("y", 150)))
	if _, hit := m.get("a"); !hit {
		t.Fatal("a fresh entry missed")
	}
	m.put("c", []byte(strings.Repeat("z", 150))) // evicts b, the least recently used
	if _, hit := m.get("b"); hit {
		t.Fatal("the least recently used entry was kept over the budget")
	}
	if _, hit := m.get("a"); !hit {
		t.Fatal("the recently used entry was evicted")
	}
	if _, _, _, bytes := ResidentGitMemoStats(); bytes > 400 {
		t.Fatalf("the memo retains %d bytes over its budget of 400", bytes)
	}
	m.put("huge", []byte(strings.Repeat("h", 1000)))
	if _, hit := m.get("huge"); hit {
		t.Fatal("an entry larger than the budget was retained")
	}
	// A linked worktree (.git is a file) is not memoized.
	linked := t.TempDir()
	if err := os.WriteFile(filepath.Join(linked, ".git"), []byte("gitdir: elsewhere\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, ok := residentGitKey(linked, "h", args); ok {
		t.Fatal("a linked worktree was memoized")
	}
}

// R2: a second operation of a process with the memo on is answered from it, with the answer
// Git gives, and a new commit is a miss; the answers equal those of a process without it.
func TestASecondOperationIsAnsweredFromTheMemoAndANewCommitMissesIt(t *testing.T) {
	root := snapshotRepo(t, map[string]string{"a.md": "one\n", "b.md": "two\n"})
	capture := func() string {
		base, view := (Store{Root: root}).beginGovernanceReadView()
		store := base.withGovernanceReadView(view, "memo-test")
		out, err := captureGovernanceGit(store.fsys(), root, "memo-test", "log", "-1", "--format=%H", "--", "a.md")
		if err != nil {
			t.Fatal(err)
		}
		if audit := view.audit(); len(audit.ChangedPaths) != 0 || audit.ChangedProducers["memo-test"] {
			t.Fatalf("a memoized read made the audit report a change: %+v", audit)
		}
		return strings.TrimSpace(string(out))
	}
	without := capture()
	EnableResidentGitMemo(1 << 20)
	defer DisableResidentGitMemo()
	first, second := capture(), capture()
	if first != without || second != without {
		t.Fatalf("answers differ: without=%s first=%s second=%s", without, first, second)
	}
	if hits, misses, _, _ := ResidentGitMemoStats(); hits != 1 || misses != 1 {
		t.Fatalf("hits %d misses %d, want 1 and 1", hits, misses)
	}
	if err := os.WriteFile(filepath.Join(root, "a.md"), []byte("one changed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	snapshotTestGit(t, root, "add", "a.md")
	snapshotTestGit(t, root, "commit", "-q", "-m", "change a")
	third := capture()
	if third == without {
		t.Fatalf("a new commit that changed the path returned the memoized commit %s", third)
	}
	if hits, misses, _, _ := ResidentGitMemoStats(); hits != 1 || misses != 2 {
		t.Fatalf("after the new commit hits %d misses %d, want 1 and 2", hits, misses)
	}
}
