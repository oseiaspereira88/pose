package pose

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestWithGovernanceReadViewRetriesChangedInputsAndKeepsOnlyRetryOutput(t *testing.T) {
	root := snapshotRepo(t, map[string]string{"source.txt": "version A\n"})
	path := filepath.Join(root, "source.txt")
	var output []byte
	calls := 0
	err := (Store{Root: root}).WithGovernanceReadView("index.generate", func(view Store) error {
		calls++
		var err error
		output, err = view.fsys().ReadFile(path)
		if err != nil {
			return err
		}
		if calls == 1 {
			return os.WriteFile(path, []byte("version B\n"), 0o644)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if calls != 2 || string(output) != "version B\n" {
		t.Fatalf("capture did not discard the changed first attempt: calls=%d output=%q", calls, output)
	}
}

func TestWithGovernanceReadViewRefusesRepeatedInterleaving(t *testing.T) {
	root := snapshotRepo(t, map[string]string{"source.txt": "version A\n"})
	path := filepath.Join(root, "source.txt")
	calls := 0
	err := (Store{Root: root}).WithGovernanceReadView("index.generate", func(view Store) error {
		calls++
		if _, err := view.fsys().ReadFile(path); err != nil {
			return err
		}
		return os.WriteFile(path, []byte("version "+string(rune('A'+calls))+"\n"), 0o644)
	})
	if !errors.Is(err, ErrGovernanceReadViewChanged) || calls != 2 {
		t.Fatalf("repeated interleaving was not refused after one retry: calls=%d err=%v", calls, err)
	}
}

func TestIndexBuildReadViewRequiresPrimaryRevisionOnlyWhenCallbackNeedsIt(t *testing.T) {
	root := t.TempDir()
	project := ProjectIDFor(filepath.Base(root))
	store := Store{
		Root: root, FederatedProjectID: project,
		FederatedResolver: &ArtifactResolver{Roots: NewRoots(RootsConfig{DefaultRoot: root, DefaultProjectID: project})},
	}
	if err := store.WithGovernanceReadViewForIndexBuild("index.build", func(Store) (bool, error) {
		return false, nil
	}); err != nil {
		t.Fatalf("index build with no revision-dependent inputs rejected missing Git HEAD: %v", err)
	}
	err := store.WithGovernanceReadViewForIndexBuild("index.build", func(Store) (bool, error) {
		return true, nil
	})
	if !errors.Is(err, ErrGovernanceReadViewUnknown) {
		t.Fatalf("index build requiring a source revision did not report unknown coverage: %v", err)
	}
}

func TestGovernanceReadViewAuditsStagedPreviewIdentity(t *testing.T) {
	root := snapshotRepo(t, map[string]string{"source.txt": "physical\n"})
	path := filepath.Join(root, "source.txt")
	for _, mutate := range []bool{false, true} {
		t.Run(map[bool]string{false: "stable", true: "replaced"}[mutate], func(t *testing.T) {
			preview := NewPreviewFiles(time.Now())
			if err := preview.WriteFile(path, []byte("staged\n"), 0644); err != nil {
				t.Fatal(err)
			}
			calls := 0
			err := (Store{Root: root, FS: preview}).WithGovernanceReadView("index.generate", func(view Store) error {
				calls++
				raw, err := view.fsys().ReadFile(path)
				if err != nil {
					return err
				}
				if string(raw) != "staged\n" {
					t.Fatalf("captured %q", raw)
				}
				if mutate {
					return preview.WriteFile(path, raw, 0644)
				}
				return nil
			})
			if mutate {
				if !errors.Is(err, ErrGovernanceReadViewChanged) || calls != 2 {
					t.Fatalf("replacement certified: calls=%d err=%v", calls, err)
				}
			} else if err != nil || calls != 1 {
				t.Fatalf("stable staged input refused: calls=%d err=%v", calls, err)
			}
			raw, err := os.ReadFile(path)
			if err != nil || string(raw) != "physical\n" {
				t.Fatalf("preview changed disk: %q %v", raw, err)
			}
		})
	}
}
