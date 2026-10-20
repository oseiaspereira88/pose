package pose

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"
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

// Spec pose-git-and-dispatch-snapshot-integrity.

func snapshotRepo(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for name, body := range files {
		p := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	snapshotTestGit(t, root, "init", "-q")
	snapshotTestGit(t, root, "add", "-A")
	snapshotTestGit(t, root, "commit", "-q", "-m", "base")
	return root
}

func snapshotTestGit(t *testing.T, root string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", root, "-c", "user.name=t", "-c", "user.email=t@t", "-c", "commit.gpgsign=false"}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v %s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

// V1: a status, a HEAD or a tree that cannot be read is unknown, never clean.
func TestGitStateThatCannotBeReadIsUnknownNotClean(t *testing.T) {
	root := snapshotRepo(t, map[string]string{"a.txt": "a"})
	if entries, err := ReadGitStatus(root); err != nil || len(entries) != 0 {
		t.Fatalf("a clean repository: %v %v", entries, err)
	}
	if err := os.WriteFile(filepath.Join(root, ".git", "index"), []byte("not an index"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadGitStatus(root); !errors.Is(err, ErrGitUnknown) {
		t.Fatalf("a damaged index read as a status: %v", err)
	}
	if _, dirtyErr := (Store{Root: root}).ungeneratedChanges("alpha", "HEAD"); !errors.Is(dirtyErr, ErrGitUnknown) {
		t.Fatalf("closeout would take a damaged repository for clean: %v", dirtyErr)
	}
	if err := os.WriteFile(filepath.Join(root, ".git", "HEAD"), []byte("garbage"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadGitHead(root); !errors.Is(err, ErrGitUnknown) {
		t.Fatalf("a damaged HEAD is a repository that cannot be read, not a missing one: %v", err)
	}
	plain := t.TempDir()
	if _, err := ReadGitStatus(plain); !errors.Is(err, ErrGitUnavailable) || errors.Is(err, ErrGitUnknown) {
		t.Fatalf("a directory with no repository is unavailable, not unknown: %v", err)
	}
	if _, err := (Store{Root: plain}).ungeneratedChanges("alpha", "HEAD"); err == nil {
		t.Fatal("an execution snapshot needs Git")
	}
}

// V2: a path Git was told to ignore cannot hide an edit from the content comparison.
func TestAssumeUnchangedAndSkipWorktreeDoNotHideAnEdit(t *testing.T) {
	for _, flag := range []string{"--assume-unchanged", "--skip-worktree"} {
		root := snapshotRepo(t, map[string]string{"src/a.go": "package a\n", "src/b.go": "package b\n"})
		head := snapshotTestGit(t, root, "rev-parse", "HEAD")
		snapshotTestGit(t, root, "update-index", flag, "src/a.go")
		if err := os.WriteFile(filepath.Join(root, "src", "a.go"), []byte("package changed\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		entries, err := ReadGitStatus(root)
		if err != nil || len(entries) != 0 {
			t.Fatalf("%s: git itself reports nothing, which is the point: %v %v", flag, entries, err)
		}
		diverged, err := WorktreeDivergence(root, head, nil)
		if err != nil || len(diverged) != 1 || diverged[0] != "src/a.go" {
			t.Fatalf("%s: diverged = %v err=%v", flag, diverged, err)
		}
	}
}

func TestHiddenExecutableModeIsDivergence(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("executable mode is not available on Windows")
	}
	root := snapshotRepo(t, map[string]string{"run.sh": "echo\n"})
	head := snapshotTestGit(t, root, "rev-parse", "HEAD")
	snapshotTestGit(t, root, "update-index", "--assume-unchanged", "run.sh")
	if err := os.Chmod(filepath.Join(root, "run.sh"), 0o755); err != nil {
		t.Fatal(err)
	}
	if entries, err := ReadGitStatus(root); err != nil || len(entries) != 0 {
		t.Fatalf("the index hides the mode change: %v %v", entries, err)
	}
	if paths, err := WorktreeDivergence(root, head, nil); err != nil || !reflect.DeepEqual(paths, []string{"run.sh"}) {
		t.Fatalf("hidden executable mode: %v %v", paths, err)
	}
}

// V3: paths are exact; a replaced type, a link, a deletion and an executable bit are divergences.
func TestWorktreeDivergenceReadsPathsAndTypesExactly(t *testing.T) {
	odd := "dir with space/\"quoted\" é.txt"
	root := snapshotRepo(t, map[string]string{odd: "x", "run.sh": "echo\n", "plain.txt": "p", "gone.txt": "g", "kept/inside.txt": "k"})
	if err := os.Symlink("plain.txt", filepath.Join(root, "link.txt")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	snapshotTestGit(t, root, "add", "-A")
	snapshotTestGit(t, root, "commit", "-q", "-m", "link")
	head := snapshotTestGit(t, root, "rev-parse", "HEAD")
	if diverged, err := WorktreeDivergence(root, head, nil); err != nil || len(diverged) != 0 {
		t.Fatalf("the control is intact: %v %v", diverged, err)
	}
	snapshotTestGit(t, root, "mv", odd, "moved é.txt")
	entries, err := ReadGitStatus(root)
	if err != nil || len(entries) != 1 || entries[0].Path != "moved é.txt" || entries[0].OrigPath != odd {
		t.Fatalf("a rename must carry both exact paths: %+v %v", entries, err)
	}
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	// A committed link retargeted at another file is a divergence.
	must(os.Remove(filepath.Join(root, "link.txt")))
	must(os.Symlink("run.sh", filepath.Join(root, "link.txt")))
	must(os.Remove(filepath.Join(root, "gone.txt")))
	must(os.Remove(filepath.Join(root, "plain.txt")))
	if err := os.Symlink("run.sh", filepath.Join(root, "plain.txt")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	must(os.Chmod(filepath.Join(root, "run.sh"), 0o755))
	must(os.RemoveAll(filepath.Join(root, "kept")))
	must(os.WriteFile(filepath.Join(root, "kept"), []byte("now a file"), 0o644))
	diverged, err := WorktreeDivergence(root, head, nil)
	if err != nil {
		t.Fatal(err)
	}
	got := strings.Join(diverged, "|")
	for _, want := range []string{"gone.txt", "link.txt", "plain.txt", "run.sh", "kept/inside.txt", odd, "moved é.txt"} {
		if !strings.Contains(got, want) {
			t.Errorf("divergence lacks %q: %s", want, got)
		}
	}
}

// V4: derived regeneration is a list of what a producer writes, not a prefix.
func TestDerivedRegenerationIsAListNotAPrefix(t *testing.T) {
	for p, tracked := range map[string]bool{
		".pose/assessments/technical-debt.md": true, ".pose/state/technical-debt.json": true,
		".pose/indexes/spec-graph.json":          true,
		".pose/results/delivery-validation.json": false,
	} {
		if !IsDerivedRegeneration(p, tracked) {
			t.Errorf("%s (tracked=%v) is a producer's output", p, tracked)
		}
	}
	for p, tracked := range map[string]bool{
		".pose/indexes/validation-matrix.json": true, ".pose/indexes/notes.md": false, ".pose/policy/review.json": true,
		".pose/results/sealed-evidence.json": true, ".pose/review-bundles/rvb-1.json": true, "pose-mcp/lib.go": true,
		".pose/indexes/../policy/review.json": true, ".pose/indexes/sub/dir/x.json": false,
	} {
		if IsDerivedRegeneration(p, tracked) {
			t.Errorf("%s (tracked=%v) must not be authorised by a prefix", p, tracked)
		}
	}
	root := snapshotRepo(t, map[string]string{".pose/indexes/validation-matrix.json": "{}", ".pose/indexes/spec-graph.json": "{}", ".pose/policy/review.json": "{}"})
	head := snapshotTestGit(t, root, "rev-parse", "HEAD")
	for _, name := range []string{".pose/indexes/validation-matrix.json", ".pose/indexes/spec-graph.json", ".pose/policy/review.json"} {
		if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(name)), []byte(`{"changed":true}`), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	diverged, err := WorktreeDivergence(root, head, IsDerivedRegeneration)
	if err != nil || strings.Join(diverged, "|") != ".pose/indexes/validation-matrix.json|.pose/policy/review.json" {
		t.Fatalf("diverged = %v err=%v", diverged, err)
	}
}

// V5: a bundle superseded, changed or unreadable while a run was going no longer governs the scope.
func TestADispatchStopsGoverningWhenItsBundleIsSuperseded(t *testing.T) {
	store, bundle, now := retentionFixture(t)
	plan := DispatchPlan{Bundle: bundle, Brief: ReviewBrief{Scope: "spec:backend"}, PolicyDigest: directoryDigest(filepath.Join(store.Root, ".pose", "policy"))}
	if reason := store.dispatchNoLongerGoverns(plan); reason != "" {
		t.Fatalf("the control still governs: %s", reason)
	}
	addMatrixCheck(t, store)
	if _, err := store.SealReviewBundle("spec:backend", now.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if reason := store.dispatchNoLongerGoverns(plan); !strings.Contains(reason, "superseded") {
		t.Fatalf("a superseded bundle governs: %q", reason)
	}
	plan.PolicyDigest = "sha256:other"
	if reason := store.dispatchNoLongerGoverns(plan); reason == "" {
		t.Fatal("a changed policy left the run governing")
	}
}

func damagedIndexRepo(t *testing.T) string {
	t.Helper()
	root := snapshotRepo(t, map[string]string{"a.txt": "a", ".pose/project.json": `{"schema_version":1,"project_id":"proj.snapshot"}`})
	if err := os.WriteFile(filepath.Join(root, ".git", "index"), []byte("not an index"), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

// R1: every consumer of the working-tree status treats an unreadable one as unknown.
func TestAnUnreadableSnapshotIsNotClean(t *testing.T) {
	clean := snapshotRepo(t, map[string]string{"a.txt": "a"})
	if dirty, _ := takeWorkingTreeSnapshot(clean).change("a.txt"); dirty {
		t.Fatal("a clean repository reads dirty")
	}
	if dirty, detail := takeWorkingTreeSnapshot(damagedIndexRepo(t)).change("a.txt"); !dirty || !strings.Contains(detail, "unknown") {
		t.Fatalf("an unreadable status read as clean: %v %q", dirty, detail)
	}
	if dirty, _ := takeWorkingTreeSnapshot(t.TempDir()).change("a.txt"); dirty {
		t.Fatal("a directory with no repository has no changes to report")
	}
}

func TestAnUnreadableStatusIsNotACleanPath(t *testing.T) {
	if dirty, detail := reviewBundleWorkingTreeChange(damagedIndexRepo(t), "a.txt"); !dirty || !strings.Contains(detail, "unknown") {
		t.Fatalf("%v %q", dirty, detail)
	}
	if dirty, _ := reviewBundleWorkingTreeChange(snapshotRepo(t, map[string]string{"a.txt": "a"}), "a.txt"); dirty {
		t.Fatal("a clean path reads dirty")
	}
	if dirty, _ := reviewBundleWorkingTreeChange(t.TempDir(), "a.txt"); dirty {
		t.Fatal("no repository, no change")
	}
}

func TestTheProjectionIsNotCoherentWhenStatusCannotBeRead(t *testing.T) {
	snap := (Store{Root: damagedIndexRepo(t)}).CurrentObligationSnapshot()
	if snap.Coherent || !strings.Contains(strings.Join(snap.Limitations, " "), "git status could not be read") {
		t.Fatalf("snapshot = %+v", snap)
	}
}
