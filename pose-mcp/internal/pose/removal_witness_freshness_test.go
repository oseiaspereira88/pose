package pose

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func acceptedRemoval(t *testing.T) (string, Store, *ArtifactRemoval) {
	t.Helper()
	root, store := reviewBundleFixture(t)
	removal := &ArtifactRemoval{Spec: "later", Path: "api/server.go", Commit: "c1", Supersedes: map[string]string{"backend": "c1"}}
	withRemoval(t, root, removal)
	reviewFixtureGit(t, root, "add", "--all")
	reviewFixtureGit(t, root, "commit", "-qm", "index", "--allow-empty")
	return root, store, removal
}

func subjectRefusal(t *testing.T, store Store) string {
	t.Helper()
	bundle, err := store.PrepareReviewBundle("spec:backend")
	if err != nil {
		return err.Error()
	}
	for _, blocker := range bundle.Blockers {
		if strings.Contains(blocker, "api/server.go") {
			return blocker
		}
	}
	return ""
}

func touchAndDelete(t *testing.T, root string) string {
	t.Helper()
	writeReviewFixture(t, root, "api/server.go", "package api\n")
	reviewFixtureGit(t, root, "add", "--all")
	reviewFixtureGit(t, root, "commit", "-qm", "touch")
	touched := reviewFixtureGit(t, root, "rev-parse", "HEAD")
	reviewFixtureGit(t, root, "rm", "-q", "api/server.go")
	reviewFixtureGit(t, root, "commit", "-qm", "delete again")
	return touched
}

func TestRemovalWitnessHoldsOnCurrentHistory(t *testing.T) {
	_, store, _ := acceptedRemoval(t)
	if why := subjectRefusal(t, store); why != "" {
		t.Fatalf("an accepted removal was refused: %s", why)
	}
}

// R1-R2: the stale index still says accepted; Git does not.
func TestLaterTouchRefusesTheWitness(t *testing.T) {
	root, store, removal := acceptedRemoval(t)
	touchAndDelete(t, root)
	why := subjectRefusal(t, store)
	head := reviewFixtureGit(t, root, "rev-parse", "HEAD")
	for _, want := range []string{"api/server.go", "no longer holds", "run pose index", head, removal.Commit} {
		if !strings.Contains(why, want) {
			t.Fatalf("refusal %q lacks %q", why, want)
		}
	}
}

func TestTouchThroughAMergeRefusesTheWitness(t *testing.T) {
	root, store, _ := acceptedRemoval(t)
	branch := reviewFixtureGit(t, root, "rev-parse", "--abbrev-ref", "HEAD")
	reviewFixtureGit(t, root, "checkout", "-qb", "side")
	touchAndDelete(t, root)
	reviewFixtureGit(t, root, "checkout", "-q", branch)
	reviewFixtureGit(t, root, "merge", "-q", "--no-ff", "-m", "merge", "side")
	if why := subjectRefusal(t, store); !strings.Contains(why, "no longer holds") {
		t.Fatalf("refusal: %q", why)
	}
}

func TestDeletingCommitOffHeadRefusesTheWitness(t *testing.T) {
	root, store, removal := acceptedRemoval(t)
	graph, err := store.GetDeliveryIntegrity("")
	if err != nil {
		t.Fatal(err)
	}
	branch := reviewFixtureGit(t, root, "rev-parse", "--abbrev-ref", "HEAD")
	reviewFixtureGit(t, root, "checkout", "-qb", "rewritten", removal.Commit+"~1")
	reviewFixtureGit(t, root, "rm", "-q", "api/server.go")
	reviewFixtureGit(t, root, "commit", "-qm", "another deletion")
	reviewFixtureGit(t, root, "branch", "-f", branch, "HEAD")
	reviewFixtureGit(t, root, "checkout", "-q", branch)
	if why := store.removalWitnessStale("api/server.go", graph, []string{"backend"}); !strings.Contains(why, "is not on HEAD") {
		t.Fatalf("refusal: %q", why)
	}
}

func TestUnknownDeletingCommitRefusesTheWitness(t *testing.T) {
	root, store, _ := acceptedRemoval(t)
	unknown := strings.Repeat("a", 40)
	withRemoval(t, root, nil)
	graph, err := store.GetDeliveryIntegrity("")
	if err != nil {
		t.Fatal(err)
	}
	graph.Removals = []ArtifactRemoval{{Spec: "later", Path: "api/server.go", Commit: unknown, Supersedes: map[string]string{"backend": unknown}}}
	if why := store.removalWitnessStale("api/server.go", graph, []string{"backend"}); !strings.Contains(why, "cannot be read") {
		t.Fatalf("refusal: %q", why)
	}
}

// History is read raw, as the removal was computed: a replace ref hides nothing.
func TestReplaceRefDoesNotHideALaterTouch(t *testing.T) {
	root, store, removal := acceptedRemoval(t)
	touchAndDelete(t, root)
	head := reviewFixtureGit(t, root, "rev-parse", "HEAD")
	reviewFixtureGit(t, root, "replace", "--graft", head, removal.Commit)
	if why := subjectRefusal(t, store); !strings.Contains(why, "touches it after") {
		t.Fatalf("refusal: %q", why)
	}
}

func TestRecordedChangeSetsAreBound(t *testing.T) {
	root := t.TempDir()
	files := NativeFiles()
	if got := IndexInputBindingsIn(files, root)["recorded-change-sets"]; got != "absent" {
		t.Fatalf("a generation does not bind it, or no history reads as %q", got)
	}
	dir := filepath.Join(root, ".pose", "reports", "history")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	write := func(body string) string {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, "a.jsonl"), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		got, _ := IndexInputBindingIn(files, root, "recorded-change-sets")
		return got
	}
	one := `{"change_set":{"id":"1"}}` + "\n"
	first := write(one)
	if got := write(one + `{"note":"unrelated"}` + "\n"); got != first {
		t.Fatal("an unrelated report line staled the binding")
	}
	if got := write(one + `{"change_set":{"id":"2"}}`); got == first {
		t.Fatal("a new change set, on a last line without a newline, did not stale it")
	}
	if got := write(`{"change_set":{"id":"3"}}` + "\n"); got == first {
		t.Fatal("a changed change set did not stale it")
	}
	moved := write(`{"change_set":{"id":"3"}}` + "\n")
	if err := os.Rename(filepath.Join(dir, "a.jsonl"), filepath.Join(dir, "b.jsonl")); err != nil {
		t.Fatal(err)
	}
	if got, _ := IndexInputBindingIn(files, root, "recorded-change-sets"); got == moved {
		t.Fatal("a moved record did not stale it")
	}
}
