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

// A shallow clone that lost the side of a merge says nothing about it, and an
// empty answer from it does not count (WF-R1-01).
func TestShallowHistoryRefusesTheWitness(t *testing.T) {
	root, store, _ := acceptedRemoval(t)
	main := reviewFixtureGit(t, root, "branch", "--show-current")
	reviewFixtureGit(t, root, "checkout", "-qb", "side")
	touchAndDelete(t, root)
	for i := 0; i < 5; i++ {
		reviewFixtureGit(t, root, "commit", "--allow-empty", "-qm", "empty")
	}
	reviewFixtureGit(t, root, "checkout", "-q", main)
	reviewFixtureGit(t, root, "merge", "-q", "--no-ff", "-m", "merge", "side")
	if why := subjectRefusal(t, store); !strings.Contains(why, "no longer holds") {
		t.Fatalf("the full history: %q", why)
	}
	shallow := filepath.Join(t.TempDir(), "shallow")
	reviewFixtureGit(t, root, "clone", "-q", "--depth=3", "--no-local", "file://"+root, shallow)
	if why := subjectRefusal(t, Store{Root: shallow}); !strings.Contains(why, "shallow") {
		t.Fatalf("the shallow clone: %q", why)
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
	empty := write(`{"task":"ordinary"}` + "\n")
	if empty != "absent" {
		t.Fatalf("a history with no change set: %s", empty)
	}
	for name, body := range map[string]string{
		"null":   `{"change_set":null}`,
		"nested": `{"note":{"change_set":"not attribution"}}`,
	} {
		if got := write(body + "\n"); got != "absent" {
			t.Fatalf("%s reads as a change set: %s", name, got)
		}
	}
	one := `{"change_set":{"id":"1"}}` + "\n"
	first := write(one)
	if first == "absent" {
		t.Fatal("a change set is not bound")
	}
	if got := write(one + `{"note":"unrelated"}` + "\n"); got != first {
		t.Fatal("an unrelated report line staled the binding")
	}
	if got := write(one + `{"change_set":{"id":"2"}}`); got == first {
		t.Fatal("a new change set, on a last line without a newline, did not stale it")
	}
	if got := write(`{"change_set":{"id":"3"}}` + "\n"); got == first {
		t.Fatal("a changed change set did not stale it")
	}
	// The spellings the reader of the graph accepts.
	for _, field := range []string{`"change_\u0073et"`, `"CHANGE_SET"`} {
		a := write(`{` + field + `:{"id":"one"}}` + "\n")
		b := write(`{` + field + `:{"id":"two"}}` + "\n")
		if a == "absent" || a == b {
			t.Fatalf("%s is not bound: %s %s", field, a, b)
		}
	}
	if got := write("not json\n"); got == "absent" {
		t.Fatal("an unreadable record reads as nothing")
	}
	moved := write(`{"change_set":{"id":"3"}}` + "\n")
	if err := os.Rename(filepath.Join(dir, "a.jsonl"), filepath.Join(dir, "b.jsonl")); err != nil {
		t.Fatal(err)
	}
	if got, _ := IndexInputBindingIn(files, root, "recorded-change-sets"); got != moved {
		t.Fatal("the same change sets from another file staled it")
	}
	if err := os.Remove(filepath.Join(dir, "b.jsonl")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "ordinary.jsonl"), []byte(`{"task":"ordinary"}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got, _ := IndexInputBindingIn(files, root, "recorded-change-sets"); got != "absent" {
		t.Fatalf("a new report without a change set: %s", got)
	}
}

// The binding is the graph reader's result: each pair differs to the reader,
// and so to the binding (WF-R2-01, WF-R2-02).
func TestRecordedChangeSetsBindWhatTheReaderReads(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, ".pose", "reports", "history")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	read := func(body string) string {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, "a.jsonl"), []byte(body+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		got, _ := IndexInputBindingIn(NativeFiles(), root, "recorded-change-sets")
		return got
	}
	for name, pair := range map[string][2]string{
		"a duplicate merges":        {`{"change_set":{"id":"one","spec":"alpha"},"change_set":{"id":"two"}}`, `{"change_set":{"id":"one","spec":"beta"},"change_set":{"id":"two"}}`},
		"another field is invalid":  {`{"sequence":1,"change_set":{"id":"two","spec":"alpha"}}`, `{"sequence":"bad","change_set":{"id":"two","spec":"alpha"}}`},
		"a wrong type is shadowed":  {`{"change_set":7,"change_set":{"id":"two","spec":"alpha"}}`, `{"change_set":{"id":"two","spec":"alpha"}}`},
		"no change set, a bad type": {`{"sequence":"bad"}`, `{"sequence":1}`},
	} {
		if read(pair[0]) == read(pair[1]) {
			t.Errorf("%s: the binding did not follow the reader", name)
		}
	}
}

// The gap names no checkout: two clones of one history bind alike.
func TestRecordedChangeSetLossBindsAlikeAcrossClones(t *testing.T) {
	bindings := []string{}
	for range 2 {
		root := t.TempDir()
		dir := filepath.Join(root, ".pose", "reports", "history", "a.jsonl")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		got, _ := IndexInputBindingIn(NativeFiles(), root, "recorded-change-sets")
		bindings = append(bindings, got)
	}
	if bindings[0] == "absent" || bindings[0] != bindings[1] {
		t.Fatalf("%v", bindings)
	}
}

// A record over the limit is a loss, not a partial read (WF-R1-04).
func TestRecordedChangeSetRecordIsBounded(t *testing.T) {
	limit := RecordedChangeSetLineLimit
	RecordedChangeSetLineLimit = 64 * 1024
	t.Cleanup(func() { RecordedChangeSetLineLimit = limit })
	root := t.TempDir()
	dir := filepath.Join(root, ".pose", "reports", "history")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "a.jsonl"), []byte(`{"note":"`+strings.Repeat("x", 200*1024)+`"}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got, _ := IndexInputBindingIn(NativeFiles(), root, "recorded-change-sets"); got == "absent" {
		t.Fatal("an oversize record reads as nothing")
	}
}
