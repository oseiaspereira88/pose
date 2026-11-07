package cli

import (
	"os"
	"path/filepath"
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

func TestMaterialWorktreeCleanSeesAnEditedAdministrativeSymlink(t *testing.T) {
	repo := newRemovalRepo(t)
	repo.write("a.go", "package a\n")
	if err := os.MkdirAll(filepath.Join(repo.root, ".pose", "results"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("../../a.go", filepath.Join(repo.root, ".pose", "results", "x.json")); err != nil {
		t.Fatal(err)
	}
	repo.commit("a symlink inside a class")
	if !materialWorktreeClean(repo.root) {
		t.Fatal("a clean checkout is not clean")
	}
	link := filepath.Join(repo.root, ".pose", "results", "x.json")
	_ = os.Remove(link)
	if err := os.Symlink("../../b.go", link); err != nil {
		t.Fatal(err)
	}
	if materialWorktreeClean(repo.root) || materialTrackedUnchanged(repo.root) {
		t.Fatal("retargeting a committed symlink inside a class was ignored")
	}
}

func TestMaterialTrackedUnchangedIgnoresUntrackedBuildProducts(t *testing.T) {
	repo := newRemovalRepo(t)
	repo.write("a.go", "package a\n")
	repo.commit("base")
	repo.write("build/output.bin", "x")
	if !materialTrackedUnchanged(repo.root) {
		t.Fatal("an untracked build product counted as a material change")
	}
	repo.write("a.go", "package a\n\nvar changedByACheck = 1\n")
	if materialTrackedUnchanged(repo.root) {
		t.Fatal("a tracked file changed during the run went unseen")
	}
}

func TestMaterialWorktreeCleanSeesALinkReplacingAnAdministrativeRecord(t *testing.T) {
	repo := newRemovalRepo(t)
	repo.write("a.go", "package a\n")
	repo.write(".pose/results/x.json", "{}\n")
	repo.commit("a record")
	record := filepath.Join(repo.root, ".pose", "results", "x.json")
	if err := os.Remove(record); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("../../a.go", record); err != nil {
		t.Fatal(err)
	}
	if materialWorktreeClean(repo.root) || materialTrackedUnchanged(repo.root) {
		t.Fatal("a record replaced by a link was ignored")
	}
	artifactGit(t, repo.root, "checkout", "--", ".pose/results/x.json")
	if err := os.Symlink("../../a.go", filepath.Join(repo.root, ".pose", "results", "new.json")); err != nil {
		t.Fatal(err)
	}
	if materialWorktreeClean(repo.root) {
		t.Fatal("a new administrative link was ignored")
	}
}
