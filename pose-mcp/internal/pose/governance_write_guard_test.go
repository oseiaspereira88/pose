package pose

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"testing"
	"time"
)

const governanceWriteGuardChildRootEnv = "POSE_GOVERNANCE_WRITE_GUARD_CHILD_ROOT"

type governanceGuardTestFilesWrapper struct{ Files }

func (f governanceGuardTestFilesWrapper) UnderlyingFiles() Files { return f.Files }

func governanceWriteGuardTestStore(t *testing.T) Store {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".pose"), 0o755); err != nil {
		t.Fatal(err)
	}
	return Store{Root: root}
}

func TestGovernanceWriteGuardCallbackErrorReleasesLease(t *testing.T) {
	store := governanceWriteGuardTestStore(t)
	callbackErr := errors.New("stop after protected work")
	var retained Store
	err := WithGovernanceWriteGuard("test.callback-error", []Store{store}, func(guarded []Store) error {
		retained = guarded[0]
		lease, ok := retained.GovernanceWriteLease()
		if !ok || !lease.Active() || lease.Preview() {
			t.Fatalf("callback did not receive an active write lease: lease=%v ok=%v", lease, ok)
		}
		return callbackErr
	})
	if !errors.Is(err, callbackErr) {
		t.Fatalf("callback error was not returned: %v", err)
	}
	if lease, ok := retained.GovernanceWriteLease(); ok || lease != nil {
		t.Fatalf("released lease remained usable: %+v", lease)
	}
	writePath := filepath.Join(store.Root, ".pose", "after-release.json")
	if err := retained.FS.WriteFile(writePath, []byte("{}\n"), 0o644); !errors.Is(err, ErrGovernanceWriteGuardReleased) {
		t.Fatalf("a retained Store wrote after release: %v", err)
	}
	if _, err := os.Lstat(writePath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("post-release write created an artifact: %v", err)
	}
	if err := WithGovernanceWriteGuard("test.after-release", []Store{store}, func([]Store) error { return nil }); err != nil {
		t.Fatalf("callback error leaked the kernel lock: %v", err)
	}
}

func TestGovernanceWriteGuardCanonicalRootOrderAndIndependence(t *testing.T) {
	first := governanceWriteGuardTestStore(t)
	second := governanceWriteGuardTestStore(t)
	var expected []string
	for _, store := range []Store{first, second} {
		root, err := canonicalGovernanceRoot(store.Root)
		if err != nil {
			t.Fatal(err)
		}
		expected = append(expected, governanceRootKey(root))
	}
	sort.Strings(expected)

	// Input order is reversed; locks and the lease's root vector use canonical order.
	err := WithGovernanceWriteGuard("test.root-order", []Store{second, first}, func(guarded []Store) error {
		lease, ok := guarded[0].GovernanceWriteLease()
		if !ok || len(lease.roots) != 2 {
			return fmt.Errorf("lease roots = %+v, ok=%v", lease, ok)
		}
		if lease.roots[0].key != expected[0] || lease.roots[1].key != expected[1] {
			return fmt.Errorf("root lock order = %v, want %v", []string{lease.roots[0].key, lease.roots[1].key}, expected)
		}
		for _, store := range guarded {
			if !lease.Covers(store) {
				return fmt.Errorf("lease did not cover root %s", store.FederatedProjectID)
			}
		}
		if _, present := guarded[0].GovernanceWriteLease(); !present {
			return errors.New("Store clone lost the explicit lease")
		}
		if err := WithGovernanceWriteGuard("test.independent-root", []Store{governanceWriteGuardTestStore(t)}, func([]Store) error { return nil }); err != nil {
			return fmt.Errorf("independent root was serialized with this lease: %w", err)
		}
		if err := WithGovernanceWriteGuard("test.implicit-reentry", guarded, func([]Store) error { return nil }); !errors.Is(err, ErrGovernanceWriteGuardReentrant) {
			return fmt.Errorf("implicit same-root reacquisition = %v, want reentrant refusal", err)
		}
		wrapped := guarded[0]
		wrapped.FS = governanceGuardTestFilesWrapper{Files: wrapped.FS}
		if _, present := wrapped.GovernanceWriteLease(); !present {
			return errors.New("lease was lost under an UnderlyingFiles wrapper")
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestGovernanceWriteGuardPreviewDoesNotCreateLockPaths(t *testing.T) {
	store := governanceWriteGuardTestStore(t)
	preview := NewPreviewFiles(time.Date(2026, 11, 3, 0, 0, 0, 0, time.UTC))
	store.FS = preview
	writePath := filepath.Join(store.Root, ".pose", "preview-only.json")
	err := WithGovernanceWriteGuard("test.preview", []Store{store}, func(guarded []Store) error {
		lease, ok := guarded[0].GovernanceWriteLease()
		if !ok || !lease.Preview() || !lease.Active() {
			return fmt.Errorf("preview lease = %+v, ok=%v", lease, ok)
		}
		return guarded[0].FS.WriteFile(writePath, []byte("{}\n"), 0o644)
	})
	if err != nil {
		t.Fatal(err)
	}
	if !preview.Staged(writePath) {
		t.Fatal("preview write was not staged")
	}
	if _, err := os.Lstat(writePath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("preview wrote to the worktree: %v", err)
	}
	if _, err := os.Lstat(filepath.Join(store.Root, ".pose", "transfers")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("preview created the transfer lock directory: %v", err)
	}

	mixed := store
	mixed.FS = NewPreviewFiles(time.Now())
	if err := WithGovernanceWriteGuard("test.mixed-preview", []Store{store, Store{Root: store.Root}}, func([]Store) error { return nil }); err == nil {
		t.Fatal("mixed preview/native roots were accepted")
	} else if !errors.Is(err, ErrGovernanceWriteGuardMixedPreview) {
		t.Fatalf("mixed preview/native roots = %v", err)
	}
}

func TestGovernanceWriteGuardCanonicalizesStoreRootBeforePathChecks(t *testing.T) {
	store := governanceWriteGuardTestStore(t)
	alias := filepath.Join(t.TempDir(), "root-alias")
	if err := os.Symlink(store.Root, alias); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	store.Root = alias
	var canonical string
	err := WithGovernanceWriteGuard("test.symlink-root-alias", []Store{store}, func(guarded []Store) error {
		canonical = guarded[0].Root
		if canonical == alias {
			return errors.New("guarded Store retained the symlink spelling")
		}
		return guarded[0].FS.WriteFile(filepath.Join(guarded[0].Root, ".pose", "alias-write.json"), []byte("{}\n"), 0o644)
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(canonical, ".pose", "alias-write.json")); err != nil {
		t.Fatalf("write through a root alias was rejected or lost: %v", err)
	}
}

func TestGovernanceWriteGuardAllowsAValidatedFirstWriteWithoutPose(t *testing.T) {
	root := t.TempDir()
	store := Store{Root: root}
	if _, err := os.Lstat(filepath.Join(root, ".pose")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("fixture already has .pose: %v", err)
	}
	path := filepath.Join(root, ".pose", "policy", "review.json")
	err := WithGovernanceWriteGuard("test.first-governance-write", []Store{store}, func(stores []Store) error {
		if err := stores[0].FS.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}
		return stores[0].FS.WriteFile(path, []byte("{}\n"), 0o644)
	})
	if err != nil {
		t.Fatalf("guard rejected a validated first write: %v", err)
	}
	if raw, err := os.ReadFile(path); err != nil || string(raw) != "{}\n" {
		t.Fatalf("first governance write was not published: %q err=%v", raw, err)
	}
	if _, err := os.Lstat(filepath.Join(root, ".pose", "transfers", ".authority-transfer.lock")); err != nil {
		t.Fatalf("root lock was not retained: %v", err)
	}
}

func TestGovernanceWriteGuardRootLimitAndLinks(t *testing.T) {
	stores := make([]Store, 0, MaxGovernanceWriteRoots+1)
	for i := 0; i < MaxGovernanceWriteRoots+1; i++ {
		stores = append(stores, governanceWriteGuardTestStore(t))
	}
	called := false
	err := WithGovernanceWriteGuard("test.root-limit", stores, func([]Store) error { called = true; return nil })
	if !errors.Is(err, ErrGovernanceWriteGuardTooManyRoots) || called {
		t.Fatalf("root limit result = %v, callback called=%v", err, called)
	}
	for _, store := range stores {
		if _, err := os.Lstat(filepath.Join(store.Root, ".pose", "transfers")); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("root limit created a lock directory: %v", err)
		}
	}

	store := governanceWriteGuardTestStore(t)
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(store.Root, ".pose", "transfers")); err != nil {
		t.Skipf("symlink fixture unavailable: %v", err)
	}
	called = false
	err = WithGovernanceWriteGuard("test.linked-lock-path", []Store{store}, func([]Store) error { called = true; return nil })
	if !errors.Is(err, ErrGovernanceWriteGuardRootBinding) || called {
		t.Fatalf("linked lock path result = %v, callback called=%v", err, called)
	}
	if _, err := os.Lstat(filepath.Join(outside, ".authority-transfer.lock")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("guard wrote through the linked lock directory: %v", err)
	}
}

func TestGovernanceWriteGuardKernelContentionAcrossHandlesAndProcesses(t *testing.T) {
	store := governanceWriteGuardTestStore(t)
	err := WithGovernanceWriteGuard("test.kernel-owner", []Store{store}, func([]Store) error {
		if err := WithGovernanceWriteGuard("test.kernel-second-handle", []Store{store}, func([]Store) error { return nil }); !errors.Is(err, ErrGovernanceWriteGuardBusy) {
			return fmt.Errorf("second handle contention = %v, want busy", err)
		}
		command := exec.Command(os.Args[0], "-test.run=^TestGovernanceWriteGuardSubprocessBusyHelper$")
		command.Env = append(os.Environ(), governanceWriteGuardChildRootEnv+"="+store.Root)
		if output, err := command.CombinedOutput(); err != nil {
			return fmt.Errorf("child process did not observe kernel contention: %v: %s", err, output)
		}
		return nil
	})
	if errors.Is(err, ErrGovernanceWriteGuardUnsupported) {
		t.Skipf("platform does not support the shared file lock: %v", err)
	}
	if err != nil {
		t.Fatal(err)
	}
}

func TestGovernanceWriteGuardSubprocessBusyHelper(t *testing.T) {
	root := os.Getenv(governanceWriteGuardChildRootEnv)
	if root == "" {
		return
	}
	err := WithGovernanceWriteGuard("test.child-contention", []Store{{Root: root}}, func([]Store) error { return nil })
	if !errors.Is(err, ErrGovernanceWriteGuardBusy) {
		t.Fatalf("child lock acquisition = %v, want busy", err)
	}
}

func TestGovernanceWriteGuardStoreBindingAndRelease(t *testing.T) {
	store := governanceWriteGuardTestStore(t)
	other := governanceWriteGuardTestStore(t)
	var retained Store
	var lease *GovernanceWriteLease
	err := WithGovernanceWriteGuard("test.binding", []Store{store}, func(guarded []Store) error {
		retained = guarded[0]
		var ok bool
		lease, ok = retained.GovernanceWriteLease()
		if !ok || !lease.Covers(retained) {
			return errors.New("lease did not bind its authorized Store")
		}
		wrongRoot := retained
		wrongRoot.Root = other.Root
		if _, ok := wrongRoot.GovernanceWriteLease(); ok {
			return errors.New("lease bound to a different physical root")
		}
		wrongProject := retained
		wrongProject.FederatedProjectID = "proj.other"
		if _, ok := wrongProject.GovernanceWriteLease(); ok {
			return errors.New("lease bound to a different project selector")
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	lease.Release()
	lease.Release()
	if lease.Active() {
		t.Fatal("idempotent release left the lease active")
	}
	if _, ok := retained.GovernanceWriteLease(); ok {
		t.Fatal("released lease remained available through Store")
	}
}
