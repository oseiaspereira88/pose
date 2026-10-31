package pose

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// R1: a run's attempt result is published while its active marker is still
// there, for a first attempt and for a continuation (a handoff); a waiter
// that finds the marker gone finds the receipt too (spec
// pose-delegation-result-before-marker).
func TestARunsResultIsPublishedBeforeItsMarkerGoes(t *testing.T) {
	var published []string
	saved := beforeRunResultPublished
	t.Cleanup(func() { beforeRunResultPublished = saved })
	var store Store
	beforeRunResultPublished = func(run ReviewRun) {
		if store.Root == "" {
			return
		}
		if !store.DelegationActiveMarkerExists(run.AttemptID) {
			t.Errorf("the marker of %s was gone before its result was published", run.RunID)
		}
		published = append(published, run.RunID)
	}
	f := provenanceRunFixture(t, "openai", "review-model", func(a *ReviewerAdapter) {
		a.Command = []string{"sh", "-c", "cat >/dev/null; echo 'session: sess-mark-1'; echo 'LIMIT reset: 2099-01-01T00:00:00Z'; exit 75"}
		a.Limit = &ReviewerLimit{ExitCodes: []int{75}, ResetPattern: `reset: (\S+)`}
		a.Session = &ReviewerSession{Pattern: `session: (\S+)`}
	})
	store = f.store
	if f.run.Status != ReviewRunInterrupted {
		t.Fatalf("first attempt: %+v", f.run)
	}
	path := filepath.Join(f.store.Root, ".pose", "policy", "reviewers.json")
	raw, _ := os.ReadFile(path)
	var reviewers ReviewersPolicy
	if err := json.Unmarshal(raw, &reviewers); err != nil {
		t.Fatal(err)
	}
	reviewers.Adapters["other"] = ReviewerAdapter{Executor: "agent:other-reviewer", Command: []string{"sh", "-c", "cat >/dev/null; echo decision: approved"}, Vendor: "google", Model: "other-model"}
	raw, _ = json.Marshal(reviewers)
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	first, err := f.store.PlanReviewDispatch(f.bundle.BundleID, "review", "", "other", "a first attempt")
	if err != nil {
		t.Fatal(err)
	}
	firstRun, err := f.store.RunReviewDispatch(first, time.Now)
	if err != nil || firstRun.Status != ReviewRunCompleted {
		t.Fatalf("first attempt: %+v %v", firstRun, err)
	}
	plan, err := f.store.PlanReviewHandoff(f.run.RunID, "other")
	if err != nil {
		t.Fatal(err)
	}
	run, err := f.store.RunReviewDispatch(plan, time.Now)
	if err != nil || run.Status != ReviewRunCompleted {
		t.Fatalf("handoff run: %+v %v", run, err)
	}
	// The fixture's own run is not observed (no store yet); the first
	// attempt and the handoff each published their result once, with their
	// marker present.
	if len(published) != 2 || published[0] != firstRun.RunID || published[1] != run.RunID {
		t.Fatalf("published results: %v, want %s then %s", published, firstRun.RunID, run.RunID)
	}
}
