package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	posemodel "github.com/harne8/pose-mcp/internal/pose"
)

// A project's first commit carries its first spec (LR-R4-03).
func TestChangeSetResolvesFromTheFirstCommit(t *testing.T) {
	repo := newRemovalRepo(t)
	repo.write("a.txt", "one\n")
	first := repo.commit("first\n\nPOSE-Spec: alpha")
	repo.write("a.txt", "two\n")
	second := repo.commit("second\n\nPOSE-Spec: alpha")
	set, err := resolveGitChangeSet(repo.root, "alpha", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if set.ResolvedBase != posemodel.GitEmptyTree(first) || set.ResolvedHead != second || len(set.Commits) != 2 {
		t.Fatalf("%+v", set)
	}
	if len(set.Paths) != 1 || set.Paths[0].Path != "a.txt" || set.Paths[0].Action != "created" {
		t.Fatalf("paths: %+v", set.Paths)
	}
	if set.DiffDigest == "" {
		t.Fatal("no diff")
	}
}

// A commit with a parent resolves as it always did.
func TestChangeSetFromACommitWithAParentKeepsItsBase(t *testing.T) {
	repo := newRemovalRepo(t)
	repo.write("a.txt", "one\n")
	base := repo.commit("first")
	repo.write("a.txt", "two\n")
	repo.commit("second\n\nPOSE-Spec: alpha")
	set, err := resolveGitChangeSet(repo.root, "alpha", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if set.ResolvedBase != base {
		t.Fatalf("%+v", set)
	}
}

// A shallow boundary has a parent the clone lacks: it is no root, and the
// change set stays unresolved rather than claiming the whole tree as created
// (RAW-R1-B1).
func TestShallowBoundaryIsNotARoot(t *testing.T) {
	for _, format := range []string{"sha1", "sha256"} {
		t.Run(format, func(t *testing.T) {
			origin := t.TempDir()
			artifactGit(t, origin, "init", "-q", "-b", "main", "--object-format="+format)
			artifactGit(t, origin, "config", "user.email", "pose@example.invalid")
			artifactGit(t, origin, "config", "user.name", "POSE Tests")
			writeArtifactTestFile(t, origin, "a.txt", "first\n")
			artifactGit(t, origin, "add", "-A")
			artifactGit(t, origin, "commit", "-qm", "first")
			writeArtifactTestFile(t, origin, "a.txt", "second\n")
			artifactGit(t, origin, "add", "-A")
			artifactGit(t, origin, "commit", "-qm", "second\n\nPOSE-Spec: alpha")
			shallow := filepath.Join(t.TempDir(), "shallow")
			artifactGit(t, origin, "clone", "-q", "--depth=1", "file://"+origin, shallow)
			if set, err := resolveGitChangeSet(shallow, "alpha", "", ""); err == nil {
				t.Fatalf("a shallow boundary resolved: %+v", set)
			}
			if set, err := resolveGitChangeSet(origin, "alpha", "", ""); err != nil || posemodel.IsGitEmptyTree(set.ResolvedBase) || set.Paths[0].Action != "modified" {
				t.Fatalf("the full history: %+v %v", set, err)
			}
		})
	}
}

func TestRootCommitIsReadFromItsObject(t *testing.T) {
	for _, format := range []string{"sha1", "sha256"} {
		repo := t.TempDir()
		artifactGit(t, repo, "init", "-q", "-b", "main", "--object-format="+format)
		artifactGit(t, repo, "config", "user.email", "pose@example.invalid")
		artifactGit(t, repo, "config", "user.name", "POSE Tests")
		writeArtifactTestFile(t, repo, "a.txt", "first\n")
		artifactGit(t, repo, "add", "-A")
		artifactGit(t, repo, "commit", "-qm", "first\n\nparent of nothing\n\nPOSE-Spec: alpha")
		first := artifactGit(t, repo, "rev-parse", "HEAD")
		set, err := resolveGitChangeSet(repo, "alpha", "", "")
		if err != nil || set.ResolvedBase != posemodel.GitEmptyTree(first) {
			t.Fatalf("%s: %+v %v", format, set, err)
		}
	}
}

// A persisted root base is a valid explicit base (RAW-R2-B1).
func TestExplicitEmptyTreeBaseResolves(t *testing.T) {
	for _, format := range []string{"sha1", "sha256"} {
		repo := t.TempDir()
		artifactGit(t, repo, "init", "-q", "-b", "main", "--object-format="+format)
		artifactGit(t, repo, "config", "user.email", "pose@example.invalid")
		artifactGit(t, repo, "config", "user.name", "POSE Tests")
		writeArtifactTestFile(t, repo, "a.txt", "first\n")
		artifactGit(t, repo, "add", "-A")
		artifactGit(t, repo, "commit", "-qm", "first")
		head := artifactGit(t, repo, "rev-parse", "HEAD")
		set, err := resolveGitChangeSet(repo, "alpha", posemodel.GitEmptyTree(head), head)
		if err != nil || set.ResolvedBase != posemodel.GitEmptyTree(head) || set.ResolvedHead != head || len(set.Commits) != 1 || len(set.Paths) != 1 {
			t.Fatalf("%s: %+v %v", format, set, err)
		}
	}
}

// A message's size says nothing about the header (RAW-R2-B2).
func TestRootWithALargeMessageResolves(t *testing.T) {
	repo := t.TempDir()
	artifactGit(t, repo, "init", "-q", "-b", "main")
	artifactGit(t, repo, "config", "user.email", "pose@example.invalid")
	artifactGit(t, repo, "config", "user.name", "POSE Tests")
	writeArtifactTestFile(t, repo, "a.txt", "first\n")
	artifactGit(t, repo, "add", "-A")
	message := filepath.Join(t.TempDir(), "message")
	if err := os.WriteFile(message, []byte("first\n\n"+strings.Repeat("x", 5*1024*1024)+"\n\nPOSE-Spec: alpha\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	artifactGit(t, repo, "commit", "-q", "-F", message)
	first := artifactGit(t, repo, "rev-parse", "HEAD")
	isRoot, err := commitHasNoParent(repo, first)
	if err != nil || !isRoot {
		t.Fatalf("%v %v", isRoot, err)
	}
	set, err := resolveGitChangeSet(repo, "alpha", "", "")
	if err != nil || set.ResolvedBase != posemodel.GitEmptyTree(first) {
		t.Fatalf("%+v %v", set, err)
	}
}

// A header that cannot be read says so, and refuses.
func TestUnreadableHeaderRefuses(t *testing.T) {
	repo := t.TempDir()
	artifactGit(t, repo, "init", "-q", "-b", "main")
	if isRoot, err := commitHasNoParent(repo, strings.Repeat("a", 40)); isRoot || err == nil {
		t.Fatalf("%v %v", isRoot, err)
	}
}

// An administrative-only commit moves the head but not the material digest; a
// source commit moves both (spec pose-validation-evidence-material-binding).
func TestChangeSetMaterialDigestIgnoresAdministrativeCommits(t *testing.T) {
	repo := newRemovalRepo(t)
	repo.write("a.go", "package a\n")
	repo.commit("code\n\nPOSE-Spec: alpha")
	before, err := resolveGitChangeSet(repo.root, "alpha", "", "")
	if err != nil || before.MaterialDigest == "" {
		t.Fatalf("%v %+v", err, before)
	}
	repo.write(".pose/indexes/spec-graph.json", "{}\n")
	repo.write(".pose/results/delivery-validation.json", "{}\n")
	repo.commit("index\n\nPOSE-Spec: alpha")
	admin, err := resolveGitChangeSet(repo.root, "alpha", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if admin.ResolvedHead == before.ResolvedHead || admin.DiffDigest == before.DiffDigest {
		t.Fatalf("head did not move: %+v", admin)
	}
	if admin.MaterialDigest != before.MaterialDigest {
		t.Fatalf("administrative commit changed the material digest")
	}
	repo.write("a.go", "package a\n\nvar x = 1\n")
	repo.commit("more code\n\nPOSE-Spec: alpha")
	material, err := resolveGitChangeSet(repo.root, "alpha", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if material.MaterialDigest == before.MaterialDigest {
		t.Fatal("material commit kept the digest")
	}
}

// A symlink replacing an administrative record is material, and an explicit
// range carries a digest too.
func TestChangeSetMaterialDigestSeesSymlinksAndRanges(t *testing.T) {
	repo := newRemovalRepo(t)
	repo.write("a.go", "package a\n")
	base := repo.commit("code\n\nPOSE-Spec: alpha")
	repo.write(".pose/indexes/x.json", "{}\n")
	repo.commit("index\n\nPOSE-Spec: alpha")
	before, err := resolveGitChangeSet(repo.root, "alpha", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(repo.root, ".pose/indexes/x.json")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("../../a.go", filepath.Join(repo.root, ".pose/indexes/x.json")); err != nil {
		t.Fatal(err)
	}
	repo.commit("link\n\nPOSE-Spec: alpha")
	after, err := resolveGitChangeSet(repo.root, "alpha", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if before.MaterialDigest == "" || after.MaterialDigest == before.MaterialDigest {
		t.Fatalf("symlink in an administrative class kept the digest: %q %q", before.MaterialDigest, after.MaterialDigest)
	}
	ranged, err := resolveGitChangeSet(repo.root, "alpha", base, "HEAD")
	if err != nil || ranged.MaterialDigest == "" {
		t.Fatalf("range set has no material digest: %v %+v", err, ranged)
	}
}

// A later commit without the trailer that changes an observed path moves the
// digest: the tree read is the checkout's, not the set's last commit.
func TestChangeSetMaterialDigestReadsTheCheckoutTree(t *testing.T) {
	repo := newRemovalRepo(t)
	repo.write("a.go", "package a\n")
	repo.commit("code\n\nPOSE-Spec: alpha")
	before, err := resolveGitChangeSet(repo.root, "alpha", "", "")
	if err != nil {
		t.Fatal(err)
	}
	repo.write("a.go", "package a\n\nvar sneaky = 1\n")
	repo.commit("someone else touches an observed path")
	after, err := resolveGitChangeSet(repo.root, "alpha", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if after.MaterialDigest == before.MaterialDigest {
		t.Fatal("an untrailered commit on an observed path kept the digest")
	}
	repo.write(".pose/indexes/validation-matrix.json", "{}\n")
	repo.commit("matrix changes without the trailer")
	matrix, err := resolveGitChangeSet(repo.root, "alpha", "", "")
	if err != nil || matrix.MaterialDigest == after.MaterialDigest {
		t.Fatalf("a matrix change kept the digest: %v", err)
	}
}

// A shallow checkout never binds evidence.
func TestChangeSetMaterialDigestIsUnreadableInAShallowCheckout(t *testing.T) {
	origin := t.TempDir()
	artifactGit(t, origin, "init", "-q", "-b", "main")
	artifactGit(t, origin, "config", "user.email", "pose@example.invalid")
	artifactGit(t, origin, "config", "user.name", "POSE Tests")
	writeArtifactTestFile(t, origin, "a.txt", "first\n")
	artifactGit(t, origin, "add", "-A")
	artifactGit(t, origin, "commit", "-qm", "first")
	writeArtifactTestFile(t, origin, "a.txt", "second\n")
	artifactGit(t, origin, "add", "-A")
	artifactGit(t, origin, "commit", "-qm", "second\n\nPOSE-Spec: alpha")
	shallow := filepath.Join(t.TempDir(), "shallow")
	artifactGit(t, origin, "clone", "-q", "--depth=1", "file://"+origin, shallow)
	git := func(limit int, args ...string) ([]byte, error) { return gitOutputBounded(shallow, limit, args...) }
	paths := []posemodel.ObservedPath{{Action: "modified", Path: "a.txt"}}
	if got := materialSetDigestAt(git, shallow, "HEAD", paths); got != posemodel.MaterialDigestUnreadable {
		t.Fatalf("shallow checkout: %q", got)
	}
	whole := func(limit int, args ...string) ([]byte, error) { return gitOutputBounded(origin, limit, args...) }
	if got := materialSetDigestAt(whole, origin, "HEAD", paths); got == "" || got == posemodel.MaterialDigestUnreadable {
		t.Fatalf("complete checkout: %q", got)
	}
}
