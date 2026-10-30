package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	posemodel "github.com/harne8/pose-mcp/internal/pose"
)

// gammaKept reports whether gamma's existence finding for internal/x.go
// remains in the delivery graph of the later-removal fixture.
func gammaKept(t *testing.T, root string) bool {
	t.Helper()
	graph, err := buildCurrentDeliveryGraph(root)
	if err != nil {
		t.Fatal(err)
	}
	_, kept := existenceFindings(graph)["gamma internal/x.go"]
	return kept
}

func TestLaterRemovalReviewRoundThree(t *testing.T) {
	// LR-R3-01: a replace ref that drops gamma's trailer, or a graft that
	// cuts gamma out of the history, changes no change set.
	t.Run("replace ref drops a trailer", func(t *testing.T) {
		root := laterRemovalFixture(t)
		gamma := artifactGit(t, root, "log", "--format=%H", "-n", "1", "--grep", "^POSE-Spec: gamma$")
		tree := artifactGit(t, root, "rev-parse", gamma+"^{tree}")
		parent := artifactGit(t, root, "rev-parse", gamma+"^")
		fake := artifactGit(t, root, "commit-tree", tree, "-p", parent, "-m", "gamma, without its trailer")
		artifactGit(t, root, "replace", gamma, fake)
		if !gammaKept(t, root) {
			t.Fatal("a replace ref hid gamma's change after the removal")
		}
	})
	t.Run("graft cuts a change out", func(t *testing.T) {
		for _, where := range []string{"repository", "environment"} {
			root := laterRemovalFixture(t)
			beta := artifactGit(t, root, "log", "--format=%H", "-n", "1", "--grep", "^POSE-Spec: beta$")
			after := artifactGit(t, root, "log", "--format=%H", "-n", "1", "--grep", "^an unattributed deletion$")
			graft := filepath.Join(root, ".git", "info", "grafts")
			if where == "environment" {
				graft = filepath.Join(t.TempDir(), "grafts")
				t.Setenv("GIT_GRAFT_FILE", graft)
			}
			if err := os.MkdirAll(filepath.Dir(graft), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(graft, []byte(after+" "+beta+"\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			if !gammaKept(t, root) {
				t.Fatalf("a graft in the %s hid gamma's change after the removal", where)
			}
		}
	})
	// LR-R3-02: a shallow repository refuses every removal.
	t.Run("shallow repository", func(t *testing.T) {
		root := laterRemovalFixture(t)
		first := artifactGit(t, root, "rev-list", "--max-parents=0", "HEAD")
		if err := os.WriteFile(filepath.Join(root, ".git", "shallow"), []byte(first+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		_, claims, sets, tracked, _, gaps, err := collectArtifactGraphInputsWithGapsIn(posemodel.NativeFiles(), root)
		if err != nil || !strings.Contains(gaps[""], "shallow") {
			t.Fatalf("a shallow repository: %v %v", gaps, err)
		}
		for _, removal := range artifactRemovals(root, claims, sets, tracked, gaps) {
			if len(removal.Supersedes) > 0 {
				t.Fatalf("a shallow repository satisfied a claim: %+v", removal)
			}
		}
	})
	// LR-R3-03: a change set names commits, not tags of them.
	t.Run("annotated tag", func(t *testing.T) {
		r := newRemovalRepo(t)
		r.write(removalPath, "1\n")
		a := r.commit("a creates")
		r.rm(removalPath)
		r.commit("b removes")
		r.git("tag", "-a", "-m", "b", "b-tag")
		tag := r.git("rev-parse", "b-tag")
		ok, why := superseded(artifactRemovals(r.root, removalClaims(removalPath), removalSets([]string{a}, []string{tag}), nil, nil))
		if ok || !strings.Contains(why, "not a commit") {
			t.Fatalf("a tag as a deleting commit: %v %q", ok, why)
		}
	})
	// LR-R3-04: the rename stream is read whole and strictly.
	t.Run("rename stream", func(t *testing.T) {
		for _, bad := range []string{"Dgarbage\x00other\x00", "D\x00other", "D\x00a\x00D", "\x00R100\x00a\x00b\x00", "D\x00\x00", "R100\x00a\x00"} {
			if _, err := renamedAwayPaths([]byte(bad)); err == nil {
				t.Errorf("malformed stream %q was accepted", bad)
			}
		}
		set, err := renamedAwayPaths([]byte("D\x00Rx\x00R100\x00a\x00b\x00C075\x00c\x00d\x00"))
		if err != nil || !set["a"] || set["c"] || set["Rx"] || len(set) != 1 {
			t.Fatalf("a valid stream: %v %v", set, err)
		}
		if set, err := renamedAwayPaths(nil); err != nil || len(set) != 0 {
			t.Fatalf("an empty diff: %v %v", set, err)
		}
	})
}
