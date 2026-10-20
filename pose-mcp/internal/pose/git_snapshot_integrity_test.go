package pose

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

func TestGitSnapshotPathsUseNULIncludingRename(t *testing.T) {
	want := []string{"new -> \"á\"\n.go", "old name.go", "tab\t.go"}
	got, err := parseGitChangedPaths([]byte("R  new -> \"á\"\n.go\x00old name.go\x00?? tab\t.go\x00"))
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("paths: %q %v", got, err)
	}
	for _, raw := range []string{" M file", "R  new\x00", "bad\x00"} {
		if _, err := parseGitChangedPaths([]byte(raw)); err == nil {
			t.Fatalf("accepted %q", raw)
		}
	}
}

func TestGitSnapshotUnavailableIsTyped(t *testing.T) {
	_, err := gitChangedPaths(t.TempDir())
	var unavailable *GitSnapshotError
	if !errors.As(err, &unavailable) {
		t.Fatalf("not typed: %v", err)
	}
	if _, err := gitMaterializedChanges(t.TempDir(), "HEAD"); !errors.As(err, &unavailable) {
		t.Fatalf("tree not typed: %v", err)
	}
}

func TestGitSnapshotStatusFailurePreservesCause(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX command fixture")
	}
	bin := t.TempDir()
	if err := os.WriteFile(filepath.Join(bin, "git"), []byte("#!/bin/sh\nexit 97\n"), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin)
	_, err := gitChangedPaths(t.TempDir())
	var unavailable *GitSnapshotError
	if !errors.As(err, &unavailable) || !strings.Contains(err.Error(), "exit status 97") {
		t.Fatalf("cause lost: %v", err)
	}
}

func TestGitSnapshotCloseoutOutputsHaveExactOwners(t *testing.T) {
	for _, path := range []string{".pose/indexes/validation-matrix.json", ".pose/results/arbitrary.json", ".pose/state/arbitrary.json", ".pose/review-bundles/arbitrary.json", ".pose/review-bundles/rvb-0123456789abcdef.json"} {
		if closeoutProducedPath(path, "alpha", ".pose/results/validation.json", true) {
			t.Fatalf("authorized unrelated or immutable output: %s", path)
		}
	}
	for _, path := range []string{".pose/indexes/repo-map.json", ".pose/results/validation.json", ".pose/closeout-plans/spec-alpha.json"} {
		if !closeoutProducedPath(path, "alpha", ".pose/results/validation.json", true) {
			t.Fatalf("lost producer output: %s", path)
		}
	}
}

func TestGitSnapshotAuthorizedGenerationRequiresItsExactBytes(t *testing.T) {
	root := t.TempDir()
	designDeltaGit(t, root, "init", "-q")
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("fixture\n"), 0644); err != nil {
		t.Fatal(err)
	}
	designDeltaGit(t, root, "add", "-A")
	head := designDeltaGit(t, root, "-c", "user.name=t", "-c", "user.email=t@t", "-c", "commit.gpgsign=false", "commit", "-qm", "fixture")
	store := Store{Root: root}
	outputs := map[string][]byte{}
	for _, name := range IndexGenerationFiles {
		outputs[name] = []byte("{}\n")
	}
	if _, err := store.PublishIndexGeneration(outputs, map[string]string{}); err != nil {
		t.Fatal(err)
	}
	if dirty, err := store.ungeneratedChanges("alpha", head); err != nil || len(dirty) != 0 {
		t.Fatalf("authorized generation: %q %v", dirty, err)
	}
	view := store.ReadIndexGeneration()
	path := filepath.ToSlash(filepath.Join(".pose/indexes/generations", view.Descriptor.GenerationID, "repo-map.json"))
	if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(path)), []byte("tampered"), 0644); err != nil {
		t.Fatal(err)
	}
	dirty, err := store.ungeneratedChanges("alpha", head)
	if err != nil || !reflect.DeepEqual(dirty, []string{path}) {
		t.Fatalf("tampered generation accepted via legacy fallback: %q %v", dirty, err)
	}
}

func TestGitSnapshotParentLinkIsNotFollowed(t *testing.T) {
	root := t.TempDir()
	designDeltaGit(t, root, "init", "-q")
	if err := os.Mkdir(filepath.Join(root, "dir"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "dir", "file"), []byte("sealed"), 0644); err != nil {
		t.Fatal(err)
	}
	designDeltaGit(t, root, "add", "-A")
	designDeltaGit(t, root, "-c", "user.name=t", "-c", "user.email=t@t", "-c", "commit.gpgsign=false", "commit", "-qm", "fixture")
	if err := os.Rename(filepath.Join(root, "dir"), filepath.Join(root, "saved")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(root, "saved"), filepath.Join(root, "dir")); err != nil {
		t.Skip(err)
	}
	got, err := gitMaterializedChanges(root, "HEAD")
	if err != nil || !reflect.DeepEqual(got, []string{"dir/file"}) {
		t.Fatalf("parent link: %q %v", got, err)
	}
	if _, err := dispatchInputPath(root, "dir/file"); err == nil {
		t.Fatal("dispatch followed a parent link")
	}
}

func TestGitSnapshotSubmoduleHiddenEditIsNotClean(t *testing.T) {
	child, root := t.TempDir(), t.TempDir()
	for _, dir := range []string{child, root} {
		designDeltaGit(t, dir, "init", "-q")
	}
	if err := os.WriteFile(filepath.Join(child, "file"), []byte("sealed"), 0644); err != nil {
		t.Fatal(err)
	}
	designDeltaGit(t, child, "add", "-A")
	designDeltaGit(t, child, "-c", "user.name=t", "-c", "user.email=t@t", "-c", "commit.gpgsign=false", "commit", "-qm", "child")
	designDeltaGit(t, root, "-c", "protocol.file.allow=always", "submodule", "add", "-q", child, "nested")
	designDeltaGit(t, root, "-c", "user.name=t", "-c", "user.email=t@t", "-c", "commit.gpgsign=false", "commit", "-qam", "root")
	if got, err := gitMaterializedChanges(root, "HEAD"); err != nil || len(got) != 0 {
		t.Fatalf("control: %q %v", got, err)
	}
	childHead := designDeltaGit(t, child, "rev-parse", "HEAD")
	bundle := ReviewBundle{Payload: ReviewBundlePayload{Subject: ReviewBundleSubject{Entries: []ReviewBundleSubjectEntry{{Class: "submodule", Path: "nested", Digest: digestBytes([]byte(childHead))}}}}}
	if err := verifyBundleMaterialization(root, bundle); err != nil {
		t.Fatalf("sealed submodule control: %v", err)
	}
	designDeltaGit(t, filepath.Join(root, "nested"), "update-index", "--assume-unchanged", "file")
	if err := os.WriteFile(filepath.Join(root, "nested/file"), []byte("tampered"), 0644); err != nil {
		t.Fatal(err)
	}
	if got, err := gitMaterializedChanges(root, "HEAD"); err != nil || !reflect.DeepEqual(got, []string{"nested"}) {
		t.Fatalf("hidden submodule edit: %q %v", got, err)
	}
}

func TestGitSnapshotHiddenEditsAndTypes(t *testing.T) {
	for _, flag := range []string{"--assume-unchanged", "--skip-worktree"} {
		t.Run(flag, func(t *testing.T) {
			root := t.TempDir()
			designDeltaGit(t, root, "init", "-q")
			path := "space \"á\".txt"
			full := filepath.Join(root, path)
			if err := os.WriteFile(full, []byte("sealed\n"), 0644); err != nil {
				t.Fatal(err)
			}
			designDeltaGit(t, root, "add", "--", path)
			designDeltaGit(t, root, "-c", "user.name=t", "-c", "user.email=t@t", "-c", "commit.gpgsign=false", "commit", "-qm", "fixture")
			if got, err := gitMaterializedChanges(root, "HEAD"); err != nil || len(got) != 0 {
				t.Fatalf("control: %q %v", got, err)
			}
			designDeltaGit(t, root, "update-index", flag, "--", path)
			if err := os.WriteFile(full, []byte("tampered\n"), 0644); err != nil {
				t.Fatal(err)
			}
			if got, err := gitChangedPaths(root); err != nil || len(got) != 0 {
				t.Fatalf("fixture not hidden: %q %v", got, err)
			}
			if got, err := gitMaterializedChanges(root, "HEAD"); err != nil || !reflect.DeepEqual(got, []string{path}) {
				t.Fatalf("hidden edit: %q %v", got, err)
			}
			if err := os.Remove(full); err != nil {
				t.Fatal(err)
			}
			if got, err := gitMaterializedChanges(root, "HEAD"); err != nil || len(got) != 1 {
				t.Fatalf("missing: %q %v", got, err)
			}
			if err := os.Mkdir(full, 0755); err != nil {
				t.Fatal(err)
			}
			if got, err := gitMaterializedChanges(root, "HEAD"); err != nil || len(got) != 1 {
				t.Fatalf("type: %q %v", got, err)
			}
		})
	}
}
