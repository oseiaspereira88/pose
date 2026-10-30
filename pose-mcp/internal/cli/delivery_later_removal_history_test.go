package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	posemodel "github.com/harne8/pose-mcp/internal/pose"
)

// removalRepo is an empty Git repository for history-shape cases (review
// round 1 of spec pose-delivery-later-removal).
type removalRepo struct {
	t    *testing.T
	root string
}

func newRemovalRepo(t *testing.T) *removalRepo {
	t.Helper()
	r := &removalRepo{t: t, root: t.TempDir()}
	artifactGit(t, r.root, "init", "-q", "-b", "main")
	artifactGit(t, r.root, "config", "user.email", "pose@example.invalid")
	artifactGit(t, r.root, "config", "user.name", "POSE Tests")
	return r
}

func (r *removalRepo) write(path, body string) { writeArtifactTestFile(r.t, r.root, path, body) }
func (r *removalRepo) rm(path string)          { artifactGit(r.t, r.root, "rm", "-q", "--", path) }
func (r *removalRepo) commit(msg string) string {
	artifactGit(r.t, r.root, "add", "-A", "--", ".")
	artifactGit(r.t, r.root, "commit", "-q", "--allow-empty", "-m", msg)
	return artifactGit(r.t, r.root, "rev-parse", "HEAD")
}
func (r *removalRepo) git(args ...string) string { return artifactGit(r.t, r.root, args...) }

const removalPath = "internal/x.go"

func removalClaims(path string) []posemodel.ArtifactClaim {
	return []posemodel.ArtifactClaim{{Spec: "a", Action: "created", Path: path}, {Spec: "b", Action: "removed", Path: path}}
}

func removalSets(a, b []string) []posemodel.ChangeSet {
	return []posemodel.ChangeSet{{ID: "cs-a", Spec: "a", Commits: a}, {ID: "cs-b", Spec: "b", Commits: b}}
}

// superseded reports whether the removal of path supersedes spec a, and why
// not.
func superseded(removals []posemodel.ArtifactRemoval) (bool, string) {
	for _, r := range removals {
		if r.Supersedes["a"] != "" {
			return true, ""
		}
		if why := r.Refusals["a"]; why != "" {
			return false, why
		}
	}
	return false, "no refusal recorded"
}

func TestLaterRemovalHistoryShapes(t *testing.T) {
	// LR-R1-01: the change sets' order is not history's: a later change by
	// a, after b's removal, is found in whichever set it is.
	t.Run("sets in any order", func(t *testing.T) {
		r := newRemovalRepo(t)
		r.write(removalPath, "1\n")
		a1 := r.commit("a creates")
		r.rm(removalPath)
		b := r.commit("b removes")
		r.write(removalPath, "2\n")
		a2 := r.commit("a adds it back")
		r.rm(removalPath)
		r.commit("unattributed removal")
		sets := []posemodel.ChangeSet{{ID: "cs-0-late", Spec: "a", Commits: []string{a2}}, {ID: "cs-1-early", Spec: "a", Commits: []string{a1}}, {ID: "cs-b", Spec: "b", Commits: []string{b}}}
		if ok, _ := superseded(artifactRemovals(r.root, removalClaims(removalPath), sets, nil)); ok {
			t.Fatal("a change after the removal was hidden by the order of the sets")
		}
	})
	// LR-R1-02: a root creation is a change; a merge that deletes the path
	// is a deletion; a merge that adds it back is a change after it.
	t.Run("root creation and merge deletion", func(t *testing.T) {
		r := newRemovalRepo(t)
		r.write(removalPath, "1\n")
		a := r.commit("a creates at the root")
		r.git("checkout", "-q", "-b", "side")
		r.rm(removalPath)
		r.commit("side removes")
		r.git("checkout", "-q", "main")
		r.write("other.txt", "o\n")
		r.commit("main moves on")
		r.git("merge", "-q", "--no-ff", "--no-edit", "side")
		b := r.git("rev-parse", "HEAD")
		if ok, why := superseded(artifactRemovals(r.root, removalClaims(removalPath), removalSets([]string{a}, []string{b}), nil)); !ok {
			t.Fatalf("a merge removal after a root creation: %s", why)
		}
	})
	t.Run("merge adding it back", func(t *testing.T) {
		r := newRemovalRepo(t)
		r.write("README", "r\n")
		r.commit("base")
		r.write(removalPath, "1\n")
		a1 := r.commit("a creates")
		r.rm(removalPath)
		b := r.commit("b removes")
		r.git("checkout", "-q", "-b", "side", a1)
		r.write(removalPath, "2\n")
		r.commit("side changes it")
		r.git("checkout", "-q", "main")
		// The merge conflicts (removed here, changed there); a resolves it
		// by keeping the change.
		_ = exec.Command("git", "-C", r.root, "merge", "-q", "--no-ff", "--no-edit", "side").Run()
		r.write(removalPath, "2\n")
		r.git("add", "--", removalPath)
		r.git("commit", "-q", "--no-edit")
		if parents := strings.Fields(r.git("rev-list", "--parents", "-n", "1", "HEAD")); len(parents) != 3 {
			t.Fatalf("the fixture's merge is not a merge: %v", parents)
		}
		a2 := r.git("rev-parse", "HEAD")
		r.rm(removalPath)
		r.commit("unattributed removal")
		if ok, _ := superseded(artifactRemovals(r.root, removalClaims(removalPath), removalSets([]string{a1, a2}, []string{b}), nil)); ok {
			t.Fatal("a merge that brought the path back after the removal was not a change")
		}
	})
	// A merge by a that keeps the path its branch has, against a parent where
	// b removed it, undoes the removal: a change against that parent, though
	// not against its first.
	t.Run("merge undoing the removal", func(t *testing.T) {
		r := newRemovalRepo(t)
		r.write("README", "r\n")
		r.commit("base")
		r.write(removalPath, "1\n")
		a1 := r.commit("a creates")
		r.git("checkout", "-q", "-b", "side")
		r.write("side.txt", "s\n")
		r.commit("side moves on")
		r.git("checkout", "-q", "main")
		r.rm(removalPath)
		b := r.commit("b removes")
		r.git("checkout", "-q", "side")
		r.git("merge", "-q", "--no-ff", "--no-commit", "main")
		r.write(removalPath, "1\n")
		r.git("add", "--", removalPath)
		r.git("commit", "-q", "-m", "a merges main and keeps x")
		a2 := r.git("rev-parse", "HEAD")
		if parents := strings.Fields(r.git("rev-list", "--parents", "-n", "1", "HEAD")); len(parents) != 3 {
			t.Fatalf("the fixture's merge is not a merge: %v", parents)
		}
		if ok, _ := superseded(artifactRemovals(r.root, removalClaims(removalPath), removalSets([]string{a1, a2}, []string{b}), nil)); ok {
			t.Fatal("a merge that undid the removal was not a change")
		}
	})
	// LR-R1-03: a type change and a deletion are changes too.
	t.Run("later type change and deletion", func(t *testing.T) {
		for _, later := range []string{"mode", "delete"} {
			r := newRemovalRepo(t)
			r.write(removalPath, "1\n")
			a1 := r.commit("a creates")
			r.rm(removalPath)
			b := r.commit("b removes")
			r.write(removalPath, "2\n")
			r.commit("an unattributed commit adds it back")
			if later == "mode" {
				if err := os.Chmod(filepath.Join(r.root, removalPath), 0o755); err != nil {
					t.Fatal(err)
				}
			} else {
				r.rm(removalPath)
			}
			a3 := r.commit("a " + later)
			if later == "mode" {
				r.rm(removalPath)
				r.commit("unattributed removal")
			}
			if ok, _ := superseded(artifactRemovals(r.root, removalClaims(removalPath), removalSets([]string{a1, a3}, []string{b}), nil)); ok {
				t.Fatalf("a later %s by a was not a change after the removal", later)
			}
		}
	})
	// LR-R1-04: history Git cannot read refuses, with its cause.
	t.Run("unreadable history", func(t *testing.T) {
		r := newRemovalRepo(t)
		r.write(removalPath, "1\n")
		a := r.commit("a creates")
		r.rm(removalPath)
		b := r.commit("b removes")
		missing := strings.Repeat("0", 39) + "1"
		ok, why := superseded(artifactRemovals(r.root, removalClaims(removalPath), removalSets([]string{a, missing}, []string{b}), nil))
		if ok || !strings.Contains(why, "could not be read") {
			t.Fatalf("an unreadable commit of a: %v %q", ok, why)
		}
	})
	// LR-R1-05: an explicit range replaces the claiming spec's sets.
	t.Run("explicit range", func(t *testing.T) {
		r := newRemovalRepo(t)
		r.write(removalPath, "1\n")
		a1 := r.commit("a creates")
		r.rm(removalPath)
		b := r.commit("b removes")
		r.write(removalPath, "2\n")
		a2 := r.commit("a adds it back, untagged")
		r.rm(removalPath)
		r.commit("unattributed removal")
		explicit := posemodel.ChangeSet{ID: "cs-range", Spec: "a", Commits: []string{a2}}
		if ok, _ := superseded(artifactRemovals(r.root, removalClaims(removalPath), removalSets([]string{a1}, []string{b}), nil, explicit)); ok {
			t.Fatal("the explicit range of a was not the one judged")
		}
		// And the other way: a range of only the earlier change is judged as
		// itself, not with a's other sets.
		earlier := posemodel.ChangeSet{ID: "cs-range-early", Spec: "a", Commits: []string{a1}}
		if ok, why := superseded(artifactRemovals(r.root, removalClaims(removalPath), removalSets([]string{a2}, []string{b}), nil, earlier)); !ok {
			t.Fatalf("a range before the removal was judged with a's other sets: %s", why)
		}
	})
	// LR-R1-06: one deletion that the change precedes is enough, whatever
	// other deletion the removing spec made.
	t.Run("several deletions", func(t *testing.T) {
		r := newRemovalRepo(t)
		r.write("README", "r\n")
		base := r.commit("base")
		r.write(removalPath, "1\n")
		r.git("checkout", "-q", "-b", "parallel")
		r.commit("parallel creates")
		r.rm(removalPath)
		bParallel := r.commit("b removes on a branch without a")
		r.git("checkout", "-q", "main")
		r.git("reset", "-q", "--hard", base)
		r.write(removalPath, "1\n")
		a := r.commit("a creates")
		r.rm(removalPath)
		bMain := r.commit("b removes after a")
		// The deletion that a's change does not precede sorts first, so a
		// check of only the first deletion would refuse.
		for i := 0; bMain < bParallel && i < 64; i++ {
			r.git("commit", "-q", "--amend", "-m", "b removes after a "+strings.Repeat(".", i+1))
			bMain = r.git("rev-parse", "HEAD")
		}
		if bMain < bParallel {
			t.Fatal("the fixture could not order the deletions")
		}
		if ok, why := superseded(artifactRemovals(r.root, removalClaims(removalPath), removalSets([]string{a}, []string{bParallel, bMain}), nil)); !ok {
			t.Fatalf("a deletion after a was hidden by another: %s", why)
		}
	})
	// LR-R1-07: a rename is not a removal.
	t.Run("rename away", func(t *testing.T) {
		r := newRemovalRepo(t)
		r.write(removalPath, "package internal\n\nfunc keep() {}\n")
		a := r.commit("a creates")
		r.git("mv", "--", removalPath, "internal/y.go")
		b := r.commit("b renames")
		if ok, _ := superseded(artifactRemovals(r.root, removalClaims(removalPath), removalSets([]string{a}, []string{b}), nil)); ok {
			t.Fatal("a rename was taken for the declared removal")
		}
	})
	// LR-R1-08: a claimed path is literal, never a pathspec.
	t.Run("pathspec magic", func(t *testing.T) {
		r := newRemovalRepo(t)
		r.write(removalPath, "1\n")
		a := r.commit("a creates")
		r.rm(removalPath)
		b := r.commit("b removes")
		magic := ":(top)" + removalPath
		if ok, _ := superseded(artifactRemovals(r.root, removalClaims(magic), removalSets([]string{a}, []string{b}), nil)); ok {
			t.Fatal("a pathspec spelling resolved to another path")
		}
	})
	// LR-R1-11: a change set's commit that is not an immutable object name
	// never reaches Git.
	t.Run("commit as an option", func(t *testing.T) {
		r := newRemovalRepo(t)
		r.write(removalPath, "1\n")
		a := r.commit("a creates")
		r.rm(removalPath)
		b := r.commit("b removes")
		victim := filepath.Join(t.TempDir(), "victim")
		if err := os.WriteFile(victim, []byte("keep me"), 0o600); err != nil {
			t.Fatal(err)
		}
		for _, sets := range [][]posemodel.ChangeSet{removalSets([]string{a}, []string{"--output=" + victim}), removalSets([]string{"--output=" + victim, a}, []string{b})} {
			ok, why := superseded(artifactRemovals(r.root, removalClaims(removalPath), sets, nil))
			if raw, _ := os.ReadFile(victim); string(raw) != "keep me" {
				t.Fatal("a recorded commit was passed to Git as an option")
			}
			if ok || !strings.Contains(why, "not an immutable commit") {
				t.Fatalf("an option as a commit: %v %q", ok, why)
			}
		}
	})
}
