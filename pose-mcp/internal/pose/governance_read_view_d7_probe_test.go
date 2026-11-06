package pose

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type d7DirectoryAuditFiles struct {
	Files
	path  string
	opens int
}

func (f *d7DirectoryAuditFiles) Open(path string) (fs.File, error) {
	if path == f.path {
		f.opens++
	}
	return f.Files.Open(path)
}

func TestGovernanceReadViewDirectoryInfoAuditIsLinearAndDetectsChange(t *testing.T) {
	for _, mutate := range []bool{false, true} {
		t.Run(fmt.Sprint(mutate), func(t *testing.T) {
			root := t.TempDir()
			for i := 0; i < 200; i++ {
				if err := os.WriteFile(filepath.Join(root, fmt.Sprintf("entry-%03d", i)), []byte("before"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			files := &d7DirectoryAuditFiles{Files: NativeFiles(), path: root}
			view := newGovernanceReadView(root, files)
			entries, err := view.filesFor("probe").ReadDir(root)
			if err != nil {
				t.Fatal(err)
			}
			for _, entry := range entries {
				if _, err := entry.Info(); err != nil {
					t.Fatal(err)
				}
			}
			if mutate {
				if err := os.WriteFile(filepath.Join(root, "entry-050"), []byte("changed size"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			files.opens = 0
			audit := view.auditSingle()
			if files.opens != 1 {
				t.Fatalf("audit listed one directory %d times", files.opens)
			}
			if len(audit.UnknownProducers) != 0 {
				t.Fatalf("fitting audit lost coverage: %+v", audit)
			}
			if audit.ChangedProducers["probe"] != mutate {
				t.Fatalf("change=%t audit=%+v", mutate, audit)
			}
		})
	}
}

type d7ProbeContainmentFiles struct {
	Files
	failPath string
	cause    error
}

func (f d7ProbeContainmentFiles) EvalSymlinks(path string) (string, error) {
	if filepath.Clean(path) == filepath.Clean(f.failPath) {
		return "", fmt.Errorf("injected containment read loss: %w", f.cause)
	}
	return f.Files.EvalSymlinks(path)
}

func TestGovernanceReadViewD7ProbeContainmentPreservesUnavailableCause(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "fixture.md")
	if err := os.WriteFile(path, []byte("---\nslug: fixture\nstatus: draft\n---\n"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, failPath := range []string{root, path} {
		t.Run(filepath.Base(failPath), func(t *testing.T) {
			store := Store{Root: root, FS: d7ProbeContainmentFiles{Files: NativeFiles(), failPath: failPath, cause: errGovernanceReadViewBudget}}
			_, err := store.readListedSpec(Spec{Slug: "fixture", Path: path})
			if !errors.Is(err, errGovernanceReadViewBudget) || strings.Contains(err.Error(), "escapes project") {
				t.Fatalf("unavailable containment was misclassified or lost its cause: %v", err)
			}
		})
	}
	outside := filepath.Join(t.TempDir(), "outside.md")
	if err := os.WriteFile(outside, []byte("outside"), 0600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "escaping.md")
	if err := os.Symlink(outside, link); err != nil {
		t.Fatal(err)
	}
	if _, err := (Store{Root: root}).readListedSpec(Spec{Slug: "escape", Path: link}); err == nil || !strings.Contains(err.Error(), "escapes project") {
		t.Fatalf("actual escaping link was not refused as an escape: %v", err)
	}
}

func TestGovernanceReadViewD7ProbeAuditExhaustionIsUnknownWithoutMutation(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "input.json")
	if err := os.WriteFile(path, []byte("unchanged"), 0600); err != nil {
		t.Fatal(err)
	}
	view := newGovernanceReadView(root, NativeFiles())
	if _, err := view.filesFor("d7-probe").ReadFile(path); err != nil {
		t.Fatal(err)
	}
	audit := view.auditSingleWithBudget(&governanceReadViewAuditBudget{records: governanceReadViewMaxRecords})
	if len(audit.UnknownProducers["d7-probe"]) == 0 {
		t.Fatalf("audit exhaustion was certified: %+v", audit)
	}
	if len(audit.ChangedPaths) != 0 || audit.ChangedProducers["d7-probe"] {
		t.Fatalf("unchanged input was labelled as an observed mutation: %+v", audit)
	}
}

func d7ProbeCoordinatorRoot(t *testing.T) (string, string) {
	t.Helper()
	root := t.TempDir()
	for _, rel := range []string{".pose/specs", ".pose/review-bundles"} {
		if err := os.MkdirAll(filepath.Join(root, rel), 0700); err != nil {
			t.Fatal(err)
		}
	}
	path := filepath.Join(root, "scope-input.json")
	if err := os.WriteFile(path, []byte(strings.Repeat("x", 32)), 0600); err != nil {
		t.Fatal(err)
	}
	return root, path
}

func TestGovernanceReadViewD7ProbeSaturatedCoordinatorDoesNotConsumeScopeBudget(t *testing.T) {
	root, path := d7ProbeCoordinatorRoot(t)
	count := 0
	err := (Store{Root: root}).WithReviewCoordinator(func(coordinator Store) error {
		// Warm the bounded catalog before admitting independent scopes. Simulate
		// previous coordinator consumption with counters, not huge allocations.
		if err := coordinator.WithReviewScopeOperationView(func(Store) error { return nil }); err != nil {
			return err
		}
		rv, _ := governanceReadViewFromFiles(coordinator.fsys())
		rv.operation.budgetMu.Lock()
		rv.operation.inputBytes = governanceReadViewMaxInputBytes - 8
		rv.operation.metadataBytes = governanceReadViewMaxMetadata - 8
		rv.operation.records = governanceReadViewMaxRecords - 1
		rv.operation.budgetMu.Unlock()
		for i := 0; i < 2; i++ {
			if err := coordinator.WithReviewScopeOperationView(func(scope Store) error {
				raw, err := scope.fsys().ReadFile(path)
				if err == nil && len(raw) != 32 {
					return fmt.Errorf("scope returned %d bytes", len(raw))
				}
				if err == nil {
					count++
				}
				return err
			}); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil || count != 2 {
		t.Fatalf("fitting scope reads inherited coordinator exhaustion: count=%d err=%v", count, err)
	}
}

func TestGovernanceReadViewD7ProbeScopeLossDoesNotPoisonHealthySibling(t *testing.T) {
	root, path := d7ProbeCoordinatorRoot(t)
	var scopeLoss error
	healthy := false
	err := (Store{Root: root}).WithReviewCoordinator(func(coordinator Store) error {
		scopeLoss = coordinator.WithReviewScopeOperationView(func(scope Store) error {
			rv, _ := governanceReadViewFromFiles(scope.fsys())
			rv.operation.budgetMu.Lock()
			rv.operation.records = governanceReadViewMaxRecords
			rv.operation.budgetMu.Unlock()
			_, err := scope.fsys().ReadFile(path)
			return err
		})
		siblingErr := coordinator.WithReviewScopeOperationView(func(scope Store) error {
			_, err := scope.fsys().ReadFile(path)
			healthy = err == nil
			return err
		})
		return errors.Join(scopeLoss, siblingErr)
	})
	if scopeLoss == nil || !healthy || !errors.Is(err, ErrGovernanceReadViewUnknown) {
		t.Fatalf("loss was suppressed or poisoned sibling: scope=%v healthy=%t coordinator=%v", scopeLoss, healthy, err)
	}
}
