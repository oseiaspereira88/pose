package pose

import (
	"context"
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
	k1, ok := residentGitKey(root, "head1", "cfg", args)
	if !ok {
		t.Fatal("a plain .git directory is not memoizable")
	}
	k2, _ := residentGitKey(root, "head2", "cfg", args)
	if k1 == k2 {
		t.Fatal("two commits share a key")
	}
	if err := os.WriteFile(filepath.Join(root, ".git", "shallow"), []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if k3, _ := residentGitKey(root, "head1", "cfg", args); k3 == k1 {
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
	// The effective Git configuration is part of the key, and an unreadable one memoizes nothing.
	if os.Remove(filepath.Join(root, ".git", "shallow")) != nil {
		t.Fatal("the shallow file the test wrote is gone")
	}
	kc1, _ := residentGitKey(root, "head1", "cfgA", args)
	kc2, _ := residentGitKey(root, "head1", "cfgB", args)
	if kc1 == kc2 {
		t.Fatal("two configurations share a key")
	}
	if _, ok := residentGitKey(root, "head1", "", args); ok {
		t.Fatal("an unreadable configuration was memoized")
	}
	// A linked worktree (.git is a file) is not memoized.
	linked := t.TempDir()
	if err := os.WriteFile(filepath.Join(linked, ".git"), []byte("gitdir: elsewhere\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, ok := residentGitKey(linked, "h", "cfg", args); ok {
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

// A global Git configuration that changes what a read answers must change the key: the digest of
// the effective configuration follows a change of the user's own configuration file.
func TestTheEffectiveGitConfigurationFollowsTheUsersConfigurationFile(t *testing.T) {
	root := snapshotRepo(t, map[string]string{"a.md": "one\n"})
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, "xdg"))
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	_, view := (Store{Root: root}).beginGovernanceReadView()
	state := view.operation.git
	ctx := context.Background()
	before := state.effectiveGitConfig(ctx, NativeFiles(), root)
	if before == "" {
		t.Skip("the effective Git configuration cannot be read here")
	}
	// The test process points GIT_CONFIG_GLOBAL at a file of its own (testgit.Isolate): that is the
	// user's configuration here.
	userConfig := os.Getenv("GIT_CONFIG_GLOBAL")
	if userConfig == "" {
		userConfig = filepath.Join(home, ".gitconfig")
	}
	original, _ := os.ReadFile(userConfig)
	t.Cleanup(func() { _ = os.WriteFile(userConfig, original, 0o644) })
	if err := os.WriteFile(userConfig, append(append([]byte(nil), original...), []byte("\n[log]\n\tfollow = true\n")...), 0o644); err != nil {
		t.Fatal(err)
	}
	// A new operation reads it again; the same operation keeps the digest it took.
	if same := state.effectiveGitConfig(ctx, NativeFiles(), root); same != before {
		t.Fatal("the digest changed inside one operation")
	}
	_, next := (Store{Root: root}).beginGovernanceReadView()
	after := next.operation.git.effectiveGitConfig(ctx, NativeFiles(), root)
	if after == before {
		t.Fatal("a change of the user's Git configuration did not change the key")
	}
}
