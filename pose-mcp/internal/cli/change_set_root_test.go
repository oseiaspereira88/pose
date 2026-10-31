package cli

import (
	"path/filepath"
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
