package cli

import (
	"testing"
)

// Validation records a material binding only when the working copy equals the
// committed material (spec pose-validation-evidence-material-binding R1, R3).
func TestMaterialWorktreeCleanSeesUncommittedMaterial(t *testing.T) {
	repo := newRemovalRepo(t)
	repo.write("a.go", "package a\n")
	repo.write(".pose/indexes/validation-matrix.json", "{}\n")
	repo.write(".pose/results/delivery-validation.json", "{}\n")
	repo.commit("base")
	if !materialWorktreeClean(repo.root) {
		t.Fatal("a clean checkout is not clean")
	}
	repo.write(".pose/results/delivery-validation.json", "{\"changed\":1}\n")
	if !materialWorktreeClean(repo.root) {
		t.Fatal("an administrative record in progress made the checkout unclean")
	}
	repo.write("a.go", "package a\n\nvar fixed = true\n")
	if materialWorktreeClean(repo.root) {
		t.Fatal("an uncommitted source change was ignored")
	}
	artifactGit(t, repo.root, "checkout", "--", "a.go")
	repo.write(".pose/indexes/validation-matrix.json", "{\"weakened\":true}\n")
	if materialWorktreeClean(repo.root) {
		t.Fatal("an uncommitted matrix change was ignored")
	}
	artifactGit(t, repo.root, "checkout", "--", ".pose/indexes/validation-matrix.json")
	repo.write("extra.go", "package a\n")
	if materialWorktreeClean(repo.root) {
		t.Fatal("an untracked source file was ignored")
	}
}
