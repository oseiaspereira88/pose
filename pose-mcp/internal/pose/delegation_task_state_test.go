//go:build !windows

package pose

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// spec pose-delegation-task-detach-and-state R5: a relative XDG_STATE_HOME is
// ignored for the platform default; an absolute one is the base.
func TestTaskStateBaseTakesOnlyAnAbsoluteXDGStateHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_STATE_HOME", "relative/state")
	if base, err := taskStateBase(); err != nil || base != filepath.Join(home, ".local", "state") {
		t.Fatalf("relative XDG_STATE_HOME: %q %v", base, err)
	}
	abs := t.TempDir()
	t.Setenv("XDG_STATE_HOME", abs)
	if base, err := taskStateBase(); err != nil || base != abs {
		t.Fatalf("absolute XDG_STATE_HOME: %q %v", base, err)
	}
}

// R6, D2: a base reached through the person's own link is accepted and
// resolved; a base or an ancestor another user can write is refused, unless
// sticky.
func TestTaskStateDirTrustsOnlyAPrivateBase(t *testing.T) {
	root := t.TempDir()
	real := filepath.Join(t.TempDir(), "state")
	if err := os.Mkdir(real, 0o700); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(t.TempDir(), "link")
	if err := os.Symlink(real, link); err != nil {
		t.Fatal(err)
	}
	t.Setenv("XDG_STATE_HOME", link)
	dir, err := taskStateDir(root, true)
	resolved, _ := filepath.EvalSymlinks(real)
	if err != nil || !strings.HasPrefix(dir, filepath.Join(resolved, "pose", "delegate")) {
		t.Fatalf("a linked base: %q %v", dir, err)
	}
	open := filepath.Join(t.TempDir(), "open")
	base := filepath.Join(open, "state")
	if err := os.MkdirAll(base, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(open, 0o777); err != nil {
		t.Fatal(err)
	}
	t.Setenv("XDG_STATE_HOME", base)
	if _, err := taskStateDir(root, true); err == nil || !strings.Contains(err.Error(), open) {
		t.Fatalf("a group-writable ancestor: %v", err)
	}
	if _, err := os.Lstat(filepath.Join(base, "pose")); err == nil {
		t.Fatal("a refused base was written under")
	}
	if err := os.Chmod(open, 0o777|os.ModeSticky); err != nil {
		t.Fatal(err)
	}
	if _, err := taskStateDir(root, true); err != nil {
		t.Fatalf("a sticky ancestor: %v", err)
	}
	if err := os.Chmod(base, 0o770); err != nil {
		t.Fatal(err)
	}
	if _, err := taskStateDir(root, false); err == nil || !strings.Contains(err.Error(), base) {
		t.Fatalf("a group-writable base was read: %v", err)
	}
}

// R7: a component POSE creates is refused, never followed or tightened, when
// it is a link or another user can write it.
func TestTaskStateDirRefusesALinkedOrWritableComponent(t *testing.T) {
	root := t.TempDir()
	base := t.TempDir()
	t.Setenv("XDG_STATE_HOME", base)
	elsewhere := t.TempDir()
	if err := os.Symlink(elsewhere, filepath.Join(base, "pose")); err != nil {
		t.Fatal(err)
	}
	if _, err := taskStateDir(root, true); err == nil || !strings.Contains(err.Error(), "not a private directory") {
		t.Fatalf("a linked component: %v", err)
	}
	if entries, _ := os.ReadDir(elsewhere); len(entries) != 0 {
		t.Fatalf("a linked component was followed: %v", entries)
	}
	if err := os.Remove(filepath.Join(base, "pose")); err != nil {
		t.Fatal(err)
	}
	if _, err := taskStateDir(root, true); err != nil {
		t.Fatal(err)
	}
	delegate := filepath.Join(base, "pose", "delegate")
	if err := os.Chmod(delegate, 0o777|os.ModeSticky); err != nil {
		t.Fatal(err)
	}
	if _, err := taskStateDir(root, true); err == nil || !strings.Contains(err.Error(), delegate) {
		t.Fatalf("a writable component: %v", err)
	}
	if info, _ := os.Lstat(delegate); info.Mode().Perm() != 0o777 {
		t.Fatalf("a refused component was tightened: %v", info.Mode())
	}
	if err := os.Chmod(delegate, 0o700); err != nil {
		t.Fatal(err)
	}
	dir, err := taskStateDir(root, true)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	store := Store{Root: root}
	if got, err := store.taskRunsDir(false, true); err != nil || got != dir {
		t.Fatalf("a readable project directory: %q %v", got, err)
	}
	if info, _ := os.Lstat(dir); info.Mode().Perm() != 0o700 {
		t.Fatalf("the project directory was not made owner-only: %v", info.Mode())
	}
	if runs, lost := store.ReadDelegationTaskRuns(); len(runs) != 0 || len(lost) != 0 {
		t.Fatalf("an empty project directory: %v %v", runs, lost)
	}
	if err := os.Chmod(dir, 0o775); err != nil {
		t.Fatal(err)
	}
	if _, lost := store.ReadDelegationTaskRuns(); len(lost) == 0 {
		t.Fatal("a writable project directory was read without a loss")
	}
}

// R6, R7: a base or an ancestor of another user is refused and one of root is
// accepted; a component POSE makes is refused when root or another user owns
// it.
func TestTaskStateDirRefusesAnotherUsersDirectory(t *testing.T) {
	root := t.TempDir()
	parent := t.TempDir()
	base := filepath.Join(parent, "state")
	if err := os.Mkdir(base, 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("XDG_STATE_HOME", base)
	saved := taskStateOwner
	defer func() { taskStateOwner = saved }()
	owned := func(name string, user, byRoot bool) {
		taskStateOwner = func(info os.FileInfo) (bool, bool) {
			if info.Name() == name {
				return user, byRoot
			}
			return saved(info)
		}
	}
	if _, err := taskStateDir(root, true); err != nil {
		t.Fatalf("baseline: %v", err)
	}
	owned(filepath.Base(parent), false, false)
	if _, err := taskStateDir(root, true); err == nil || !strings.Contains(err.Error(), parent) || !strings.Contains(err.Error(), "another user") {
		t.Fatalf("an ancestor of another user: %v", err)
	}
	owned("state", false, false)
	if _, err := taskStateDir(root, false); err == nil || !strings.Contains(err.Error(), base) {
		t.Fatalf("a base of another user: %v", err)
	}
	owned(filepath.Base(parent), false, true)
	if _, err := taskStateDir(root, true); err != nil {
		t.Fatalf("an ancestor of root: %v", err)
	}
	for _, name := range []string{"pose", "delegate"} {
		for _, byRoot := range []bool{false, true} {
			owned(name, false, byRoot)
			if _, err := taskStateDir(root, true); err == nil || !strings.Contains(err.Error(), "belongs to another user") {
				t.Fatalf("a %s component of another user (root %v): %v", name, byRoot, err)
			}
		}
	}
}

// F2F3-R4-01: the history is keyed by the resolved root; what an earlier
// engine kept under a link's spelling is still read, never written.
func TestTaskHistoryKeyedByTheResolvedRootReadsTheLegacyKey(t *testing.T) {
	root := t.TempDir()
	link := filepath.Join(t.TempDir(), "project")
	if err := os.Symlink(root, link); err != nil {
		t.Fatal(err)
	}
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	viaLink, err := taskStateDir(link, true)
	if err != nil {
		t.Fatal(err)
	}
	if viaRoot, err := taskStateDir(root, false); err != nil || viaRoot != viaLink {
		t.Fatalf("two spellings, two histories: %q %q %v", viaLink, viaRoot, err)
	}
	if dirs, errs := legacyTaskStateDirs(link); len(dirs) != 1 || errs[0] == nil {
		t.Fatalf("an absent legacy directory: %v %v", dirs, errs)
	}
	if dirs, _ := legacyTaskStateDirs(root); len(dirs) != 0 {
		t.Fatal("a resolved root has a legacy spelling")
	}
	// A detached run is pinned to the resolved root; its caller's spelling
	// still names the legacy history.
	DetachedCallerRoot = link
	defer func() { DetachedCallerRoot = "" }()
	if dirs, _ := legacyTaskStateDirs(root); len(dirs) != 1 {
		t.Fatalf("the caller's spelling was not read: %v", dirs)
	}
	DetachedCallerRoot = ""
	abs, _ := filepath.Abs(link)
	legacy, err := taskStateDirFor(abs, true)
	if err != nil {
		t.Fatal(err)
	}
	record := `{"run_id": "trn-legacy0000000001", "purpose": "task", "attempt_id": "tat-legacy"}`
	if err := os.WriteFile(filepath.Join(legacy, "trn-legacy0000000001.json"), []byte(record), 0o600); err != nil {
		t.Fatal(err)
	}
	runs, lost := Store{Root: link}.ReadDelegationTaskRuns()
	if len(lost) != 0 || len(runs) != 1 || runs[0].RunID != "trn-legacy0000000001" {
		t.Fatalf("the legacy history: %v %v", runs, lost)
	}
}
