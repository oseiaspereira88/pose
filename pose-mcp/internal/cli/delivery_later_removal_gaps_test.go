package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	posemodel "github.com/harne8/pose-mcp/internal/pose"
)

// failingGit puts a git on PATH that fails a command naming commit and verb
// with a recognisable cause, and runs the real git otherwise.
func failingGit(t *testing.T, verb, commit string) {
	t.Helper()
	real, err := exec.LookPath("git")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	script := "#!/bin/sh\nverb=no\nhit=no\nfor a in \"$@\"; do\n  [ \"$a\" = '" + verb + "' ] && verb=yes\n  [ \"$a\" = '" + commit + "' ] && hit=yes\ndone\nif [ $verb = yes ] && [ $hit = yes ]; then\n  printf '%s\\n' 'synthetic-history-failure' >&2\n  exit 128\nfi\nexec '" + real + "' \"$@\"\n"
	if err := os.WriteFile(filepath.Join(dir, "git"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func TestLaterRemovalReviewRoundTwo(t *testing.T) {
	// LR-R2-01: a path after a one-path record is never read as a status.
	t.Run("rename after a path beginning with R", func(t *testing.T) {
		r := newRemovalRepo(t)
		r.write("README.extra", "r\n")
		r.write(removalPath, "package internal\n\nfunc keep() {}\n")
		a := r.commit("a creates")
		r.rm("README.extra")
		r.git("mv", "--", removalPath, "internal/y.go")
		b := r.commit("b deletes README.extra and renames x")
		if ok, _ := superseded(artifactRemovals(r.root, removalClaims(removalPath), removalSets([]string{a}, []string{b}), nil, nil)); ok {
			t.Fatal("a rename after a deleted R-named path was taken for a removal")
		}
		// And a real deletion beside renames is still one.
		r2 := newRemovalRepo(t)
		r2.write("Rfirst.go", "package internal\n\nfunc first() {}\n")
		r2.write(removalPath, "1\n")
		a2 := r2.commit("a creates")
		r2.git("mv", "--", "Rfirst.go", "Rsecond.go")
		r2.rm(removalPath)
		b2 := r2.commit("b renames an R-named file and deletes x")
		if ok, why := superseded(artifactRemovals(r2.root, removalClaims(removalPath), removalSets([]string{a2}, []string{b2}), nil, nil)); !ok {
			t.Fatalf("a deletion beside a rename was refused: %s", why)
		}
	})
	// LR-R2-02: a shallow boundary is not a root.
	t.Run("shallow boundary", func(t *testing.T) {
		r := newRemovalRepo(t)
		r.write(removalPath, "1\n")
		a1 := r.commit("a creates")
		r.rm(removalPath)
		b := r.commit("b removes")
		r.write(removalPath, "2\n")
		r.commit("an unattributed commit adds it back")
		r.rm(removalPath)
		a2 := r.commit("a deletes it")
		if err := os.WriteFile(filepath.Join(r.root, ".git", "shallow"), []byte(a2+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		ok, why := superseded(artifactRemovals(r.root, removalClaims(removalPath), removalSets([]string{a1, a2}, []string{b}), nil, nil))
		if ok || !strings.Contains(why, "shallow") {
			t.Fatalf("a change at a shallow boundary: %v %q", ok, why)
		}
	})
	// LR-R1-10/LR-R2-06: replace refs do not change what is judged.
	t.Run("replace refs", func(t *testing.T) {
		r := newRemovalRepo(t)
		r.write(removalPath, "1\n")
		a1 := r.commit("a creates")
		r.rm(removalPath)
		b := r.commit("b removes")
		r.write(removalPath, "2\n")
		a2 := r.commit("a adds it back")
		r.rm(removalPath)
		r.commit("unattributed removal")
		tree := r.git("rev-parse", b+"^{tree}")
		fake := r.git("commit-tree", tree, "-p", b, "-m", "a adds nothing")
		r.git("replace", a2, fake)
		if ok, _ := superseded(artifactRemovals(r.root, removalClaims(removalPath), removalSets([]string{a1, a2}, []string{b}), nil, nil)); ok {
			t.Fatal("a replace ref hid a change after the removal")
		}
	})
	// LR-R2-03: what the collection could not read refuses.
	t.Run("collection gaps", func(t *testing.T) {
		r := newRemovalRepo(t)
		r.write(removalPath, "1\n")
		a := r.commit("a creates")
		r.rm(removalPath)
		b := r.commit("b removes")
		for _, gaps := range []map[string]string{{"a": "its trailer change set could not be read"}, {"": "the tracked paths could not be read"}} {
			ok, why := superseded(artifactRemovals(r.root, removalClaims(removalPath), removalSets([]string{a}, []string{b}), nil, gaps))
			if ok || !strings.Contains(why, "could not be read") {
				t.Fatalf("a gap %v: %v %q", gaps, ok, why)
			}
		}
		root := laterRemovalFixture(t)
		gamma := artifactGit(t, root, "log", "--format=%H", "-n", "1", "--grep", "^POSE-Spec: gamma$")
		failingGit(t, "diff-tree", gamma)
		_, claims, sets, tracked, _, collected, err := collectArtifactGraphInputsWithGapsIn(posemodel.NativeFiles(), root)
		if err != nil || !strings.Contains(collected["gamma"], "synthetic-history-failure") {
			t.Fatalf("the collection hid a failed change set: %v %v", collected, err)
		}
		for _, removal := range artifactRemovals(root, claims, sets, tracked, collected) {
			if removal.Path != removalPath {
				continue
			}
			if removal.Supersedes["gamma"] != "" || !strings.Contains(removal.Refusals["gamma"], "synthetic-history-failure") {
				t.Fatalf("a removal judged gamma on what was read: %+v", removal)
			}
		}
	})
	// LR-R2-04: an explicit range replaces every set of its spec, whatever
	// their IDs.
	t.Run("explicit range with a recorded ID", func(t *testing.T) {
		r := newRemovalRepo(t)
		r.write(removalPath, "1\n")
		a1 := r.commit("a creates")
		r.rm(removalPath)
		b := r.commit("b removes")
		r.write(removalPath, "2\n")
		a2 := r.commit("a adds it back")
		r.rm(removalPath)
		r.commit("unattributed removal")
		sets := []posemodel.ChangeSet{{ID: "cs-range", Spec: "a", Commits: []string{a2}}, {ID: "cs-b", Spec: "b", Commits: []string{b}}}
		explicit := posemodel.ChangeSet{ID: "cs-range", Spec: "a", Commits: []string{a1}}
		if ok, why := superseded(artifactRemovals(r.root, removalClaims(removalPath), sets, nil, nil, explicit)); !ok {
			t.Fatalf("a recorded set with the range's ID joined it: %s", why)
		}
	})
	// LR-R2-05: an ancestry failure says why.
	t.Run("ancestry failure cause", func(t *testing.T) {
		r := newRemovalRepo(t)
		r.write(removalPath, "1\n")
		a := r.commit("a creates")
		r.rm(removalPath)
		b := r.commit("b removes")
		failingGit(t, "merge-base", a)
		ok, why := superseded(artifactRemovals(r.root, removalClaims(removalPath), removalSets([]string{a}, []string{b}), nil, nil))
		if ok || !strings.Contains(why, "synthetic-history-failure") {
			t.Fatalf("an ancestry failure: %v %q", ok, why)
		}
	})
}
