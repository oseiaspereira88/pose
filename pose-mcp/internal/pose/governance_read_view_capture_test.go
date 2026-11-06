package pose

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
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

// Intercept only the subject: root-registry/configuration reads retain native
// behavior and cannot become unrelated failures in this callback fixture.
type governanceReadViewCallbackSubjectFiles struct {
	Files
	target                              string
	readFileCalls, openCalls, readCalls int
}

func (f *governanceReadViewCallbackSubjectFiles) ReadFile(name string) ([]byte, error) {
	if filepath.Clean(name) == filepath.Clean(f.target) {
		f.readFileCalls++
		return nil, errors.New("probe: subject used unbounded ReadFile")
	}
	return f.Files.ReadFile(name)
}

func (f *governanceReadViewCallbackSubjectFiles) Open(name string) (fs.File, error) {
	file, err := f.Files.Open(name)
	if filepath.Clean(name) != filepath.Clean(f.target) || err != nil {
		return file, err
	}
	f.openCalls++
	return governanceReadViewProbeTrackedFile{File: file, reads: &f.readCalls}, nil
}

func TestGovernanceReadViewCallbackFailurePreservesAuditedCauses(t *testing.T) {
	for _, mode := range []string{"budget", "invalid", "changed", "audit-budget"} {
		t.Run(mode, func(t *testing.T) {
			root := snapshotRepo(t, map[string]string{"source.txt": "version A\n"})
			path := filepath.Join(root, "source.txt")
			cause := errors.New("callback failure")
			callbackErr := &os.PathError{Op: "callback", Path: "synthetic-input", Err: cause}
			files := &governanceReadViewCallbackSubjectFiles{Files: NativeFiles(), target: path}
			directory := filepath.Join(root, "audit-directory")
			if err := os.Mkdir(directory, 0o755); err != nil {
				t.Fatal(err)
			}
			dirFiles := &governanceReadViewProbeDirectoryFiles{Files: files, root: directory}
			if mode == "budget" {
				if err := os.Truncate(path, governanceReadViewMaxInputBytes+1); err != nil {
					t.Fatal(err)
				}
			}
			calls := 0
			err := (Store{Root: root, FS: dirFiles}).WithGovernanceReadView("index.generate", func(view Store) error {
				calls++
				if mode == "audit-budget" {
					if _, err := view.fsys().ReadDir(directory); err != nil {
						t.Fatal(err)
					}
					dirFiles.entries = make([]fs.DirEntry, governanceReadViewMaxRecords+1)
					for i := range dirFiles.entries {
						dirFiles.entries[i] = governanceReadViewProbeCountedDirEntry{name: fmt.Sprintf("entry-%05d", i), infoCalls: new(int)}
					}
				} else {
					_, readErr := view.fsys().ReadFile(path)
					if mode == "budget" {
						if !errors.Is(readErr, errGovernanceReadViewBudget) {
							t.Fatalf("oversized input was not rejected: %v", readErr)
						}
					} else if readErr != nil {
						t.Fatal(readErr)
					}
					if mode == "changed" {
						if err := os.WriteFile(path, []byte("version B\n"), 0o644); err != nil {
							t.Fatal(err)
						}
					}
				}
				return callbackErr
			})
			var recovered *os.PathError
			if calls != 1 || !errors.Is(err, cause) || !errors.As(err, &recovered) || recovered != callbackErr {
				t.Fatalf("callback cause lost or callback retried: calls=%d err=%v", calls, err)
			}
			wantUnknown := mode == "budget" || mode == "audit-budget"
			if errors.Is(err, ErrGovernanceReadViewUnknown) != wantUnknown {
				t.Fatalf("audited unknown=%t, want %t: %v", errors.Is(err, ErrGovernanceReadViewUnknown), wantUnknown, err)
			}
			if wantUnknown && (!strings.Contains(err.Error(), "budget") || !strings.Contains(err.Error(), "index.generate")) {
				t.Fatalf("unknown lost its producer/reason summary: %v", err)
			}
			if strings.Contains(err.Error(), root) {
				t.Fatalf("audit summary disclosed an observed host path: %v", err)
			}
			if mode == "budget" && (files.openCalls != 0 || files.readFileCalls != 0 || files.readCalls != 0) {
				t.Fatalf("oversized input reached a byte reader: %+v", files)
			}
			if mode == "changed" && !errors.Is(err, ErrGovernanceReadViewChanged) {
				t.Fatalf("callback failure hid changed inputs: %v", err)
			}
			if mode == "invalid" && errors.Is(err, ErrGovernanceReadViewChanged) {
				t.Fatalf("genuine callback failure invented a changed closure: %v", err)
			}
		})
	}
}

func TestGovernanceReadViewUnknownSummaryExcludesResolutionPaths(t *testing.T) {
	err := governanceReadViewUnknownError(governanceReadViewAudit{
		ChangedPaths: []string{"/private/observed-input"},
		UnknownProducers: map[string][]string{"index.generate": {
			"index generation could not be pinned: /private/index/descriptor.json",
			"authorized roots could not be resolved: /private/root/project.json",
			"read view exceeded its operation input byte budget",
		}},
	})
	if !errors.Is(err, ErrGovernanceReadViewUnknown) || strings.Contains(err.Error(), "/private") || !strings.Contains(err.Error(), "index.generate") || !strings.Contains(err.Error(), "byte budget") {
		t.Fatalf("unknown summary lost its cause or exposed observed paths: %v", err)
	}
}
