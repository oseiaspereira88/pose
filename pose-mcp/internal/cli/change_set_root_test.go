package cli

import (
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
