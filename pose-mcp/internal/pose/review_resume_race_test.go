package pose

import (
	"errors"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"
)

func resumableRunFixture(t *testing.T) runProvenanceFixture {
	t.Helper()
	if runtime.GOOS != "linux" {
		t.Skip("Unix fixture command")
	}
	return provenanceRunFixture(t, "fixture-vendor", "fixture-model", func(a *ReviewerAdapter) {
		a.Session = &ReviewerSession{Pattern: `session: (\S+)`}
		a.Limit = &ReviewerLimit{ExitCodes: []int{75}}
		a.Command = []string{"sh", "-c", `cat >/dev/null; printf 'session: fixture-private-reference\n'; exit 75`}
		a.Resume = &ReviewerResume{Command: []string{"sh", "-c", `cat >/dev/null; printf 'decision: approved\n'`, "sh", "{session}"}}
	})
}

// Spec pose-review-resume-residuals R1, R6 (review round 1, N2): a resume that
// loses its take to another resume that already ended returns that
// continuation as its result, and runs nothing.
func TestRunContinuityALosingResumeReturnsTheRecordedContinuation(t *testing.T) {
	f := resumableRunFixture(t)
	plan, err := f.store.PlanReviewResume(f.run.RunID)
	if err != nil {
		t.Fatal(err)
	}
	var winner ReviewRun
	defer func() { beforeContinuationTake = func() {} }()
	beforeContinuationTake = func() {
		beforeContinuationTake = func() {}
		other, err := f.store.PlanReviewResume(f.run.RunID)
		if err != nil {
			t.Fatal(err)
		}
		if winner, err = f.store.RunReviewDispatch(other, time.Now); err != nil {
			t.Fatal(err)
		}
		dir, _ := reviewRunRuntimeDirPath(runtimeSourceOf(f.run))
		if err := os.Mkdir(dir, 0o700); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { os.Remove(dir) })
	}
	got, err := f.store.RunReviewDispatch(plan, time.Now)
	if err != nil || winner.RunID == "" || got.RunID != winner.RunID {
		t.Fatalf("the losing resume: %s %v, winner %s", got.RunID, err, winner.RunID)
	}
}

// R6 (N2): a claim taken by another between the take and the claim removes
// the directory this continuation took and reports the other as pending.
func TestRunContinuityALostClaimReleasesTheDirectory(t *testing.T) {
	f := resumableRunFixture(t)
	plan, err := f.store.PlanReviewResume(f.run.RunID)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { beforeContinuationClaim = func() {} }()
	beforeContinuationClaim = func() {
		beforeContinuationClaim = func() {}
		if err := f.store.claimReviewContinuation(f.run, time.Now()); err != nil {
			t.Fatal(err)
		}
	}
	_, err = f.store.RunReviewDispatch(plan, time.Now)
	var pending *PendingReviewContinuation
	if !errors.As(err, &pending) {
		t.Fatalf("a lost claim: %v", err)
	}
	dir, _ := reviewRunRuntimeDirPath(runtimeSourceOf(f.run))
	if _, err := os.Lstat(dir); !os.IsNotExist(err) {
		t.Fatalf("a lost claim left the directory: %v", err)
	}
}

// R1 (N1): a recorded continuation of another kind is refused, as the plans
// refuse it.
func TestRunContinuityARecordedContinuationOfAnotherKindIsRefused(t *testing.T) {
	prior := ReviewRun{RunID: "rrn-p"}
	resume := DispatchPlan{predecessor: &prior}
	handoff := DispatchPlan{predecessor: &prior, handoff: true, Adapter: "b"}
	if err := continuationKindMismatch(resume, ReviewRun{RunID: "rrn-h", HandsOffFrom: "rrn-p", Adapter: "b"}); err == nil || !strings.Contains(err.Error(), "handed off") {
		t.Fatalf("a resume that met a handoff: %v", err)
	}
	if err := continuationKindMismatch(handoff, ReviewRun{RunID: "rrn-r", Adapter: "a"}); err == nil || !strings.Contains(err.Error(), "already continued") {
		t.Fatalf("a handoff that met a resume: %v", err)
	}
	if err := continuationKindMismatch(handoff, ReviewRun{RunID: "rrn-h", HandsOffFrom: "rrn-p", Adapter: "b"}); err != nil {
		t.Fatalf("a handoff that met its own kind: %v", err)
	}
}
