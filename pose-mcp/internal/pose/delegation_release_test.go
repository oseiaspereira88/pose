//go:build linux

package pose

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Spec pose-review-run-reaps-lingering-reviewer-processes R5, R6: a dispatch
// that ended between its reservation and its run record leaves an attempt
// that blocks the bundle; once its owner is proven gone, cancel releases it
// with a failed run and its receipt, and the ledger reads again.
func TestAnAbandonedAttemptIsReleasedWithARecord(t *testing.T) {
	root := t.TempDir()
	s := Store{Root: root}
	bundle := "rvb-00000000000000a1"
	a := DelegatedReviewAttempt{SchemaVersion: 1, BundleID: bundle, BundleDigest: "sha256:" + strings.Repeat("a", 64), Scope: "spec:alpha", Adapter: "fake", Ordinal: 1,
		ReservedAt: time.Now().Add(-time.Hour).UTC().Format(time.RFC3339Nano), Nonce: strings.Repeat("0", 32)}
	a.AttemptID = attemptIdentity(a)
	rel, _ := filepath.Rel(root, reviewAttemptPath(root, a))
	if err := publishLedgerJSON(root, filepath.ToSlash(rel), a); err != nil {
		t.Fatal(err)
	}
	// The ledger refuses the bundle while the reservation has no receipt.
	if _, err := s.readAttemptReceipt(a); err == nil || !strings.Contains(err.Error(), "incomplete or unreadable") {
		t.Fatalf("an unfinished reservation read as finished: %v", err)
	}

	// The marker the interrupted dispatch left: its owner is a process that
	// has exited.
	dead := exec.Command("true")
	if err := dead.Run(); err != nil {
		t.Fatal(err)
	}
	owner := StartLockOwner{SchemaVersion: StartLockOwnerSchemaVersion, Resource: "delegate:" + a.AttemptID, Worktree: root, Operation: "pose delegate review", PID: os.Getpid()}
	owner.Host, _ = os.Hostname()
	identifyStartLockProcess(&owner)
	owner.PID = dead.Process.Pid
	marker := DelegationActiveRun{SchemaVersion: 1, AttemptID: a.AttemptID, RunToken: strings.Repeat("b", 32), Purpose: DelegationPurposeReview, Agent: "fake", Scope: a.Scope, BundleID: bundle,
		StartedAt: a.ReservedAt, Owner: owner, HeartbeatAt: a.ReservedAt, HeartbeatEvery: 10}
	dir, err := s.delegationActiveDir(true)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(marker)
	if err := os.WriteFile(filepath.Join(dir, a.AttemptID+".json"), raw, 0o600); err != nil {
		t.Fatal(err)
	}

	run, err := s.ReleaseAbandonedAttempt(a.AttemptID, time.Now(), false)
	if err != nil {
		t.Fatal(err)
	}
	if run.Status != ReviewRunFailed || !strings.Contains(run.Failure, "abandoned") {
		t.Fatalf("the release recorded %s: %s", run.Status, run.Failure)
	}
	if _, err := s.readAttemptReceipt(a); err != nil {
		t.Fatalf("the released attempt still blocks the bundle: %v", err)
	}
	if left, _ := filepath.Glob(filepath.Join(dir, a.AttemptID+"*")); len(left) != 0 {
		t.Fatalf("the marker or its stall claim stayed: %v", left)
	}
	// Released once: a second release finds nothing abandoned.
	if _, err := s.ReleaseAbandonedAttempt(a.AttemptID, time.Now(), false); !errors.Is(err, ErrAttemptNotAbandoned) {
		t.Fatalf("a released attempt was released again: %v", err)
	}
}

// An attempt whose owner is alive is never released.
func TestARunningAttemptIsNotReleased(t *testing.T) {
	root := t.TempDir()
	s := Store{Root: root}
	bundle := "rvb-00000000000000a2"
	// Silent for an hour, so only the owner's liveness keeps it running.
	old := time.Now().Add(-time.Hour).UTC().Format(time.RFC3339Nano)
	a := DelegatedReviewAttempt{SchemaVersion: 1, BundleID: bundle, BundleDigest: "sha256:" + strings.Repeat("a", 64), Scope: "spec:alpha", Adapter: "fake", Ordinal: 1, ReservedAt: old, Nonce: strings.Repeat("0", 32)}
	a.AttemptID = attemptIdentity(a)
	rel, _ := filepath.Rel(root, reviewAttemptPath(root, a))
	if err := publishLedgerJSON(root, filepath.ToSlash(rel), a); err != nil {
		t.Fatal(err)
	}
	owner := StartLockOwner{SchemaVersion: StartLockOwnerSchemaVersion, Resource: "delegate:" + a.AttemptID, Worktree: root, Operation: "pose delegate review", PID: os.Getpid()}
	owner.Host, _ = os.Hostname()
	identifyStartLockProcess(&owner)
	marker := DelegationActiveRun{SchemaVersion: 1, AttemptID: a.AttemptID, RunToken: strings.Repeat("d", 32), Purpose: DelegationPurposeReview, Agent: "fake", Scope: "spec:alpha", BundleID: bundle, StartedAt: old, Owner: owner, HeartbeatAt: old, HeartbeatEvery: 10}
	dir, _ := s.delegationActiveDir(true)
	raw, _ := json.Marshal(marker)
	if err := os.WriteFile(filepath.Join(dir, marker.AttemptID+".json"), raw, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := s.ReleaseAbandonedAttempt(marker.AttemptID, time.Now(), false); !errors.Is(err, ErrAttemptNotAbandoned) {
		t.Fatalf("a running attempt was released: %v", err)
	}
}

// Review round 1, N1: the dispatch marks the attempt active before it
// publishes the reservation, so no instant leaves a reservation without a
// marker that turns stalled.
func TestTheAttemptIsMarkedActiveBeforeItIsReserved(t *testing.T) {
	var checked bool
	beforeAttemptPublished = func(store Store, a DelegatedReviewAttempt) {
		checked = true
		markers, _, err := store.ListDelegationActive()
		found := false
		for _, m := range markers {
			found = found || m.AttemptID == a.AttemptID
		}
		if err != nil || !found {
			t.Errorf("the reservation of %s is published with no active marker: %v", a.AttemptID, err)
		}
		if _, err := os.Lstat(reviewAttemptPath(store.Root, a)); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("the reservation was published before the marker: %v", err)
		}
	}
	t.Cleanup(func() { beforeAttemptPublished = func(Store, DelegatedReviewAttempt) {} })
	_ = provenanceRunFixture(t, "fixture-vendor", "fixture-model")
	if !checked {
		t.Fatal("the dispatch never reached the reservation")
	}
}

// A marker whose dispatch ended before publishing the reservation is removed
// by the release; ls names the cause and the release command (N2).
func TestAStalledMarkerWithoutReservationIsListedAndRemoved(t *testing.T) {
	root := t.TempDir()
	s := Store{Root: root}
	dead := exec.Command("true")
	if err := dead.Run(); err != nil {
		t.Fatal(err)
	}
	owner := StartLockOwner{SchemaVersion: StartLockOwnerSchemaVersion, Resource: "delegate:x", Worktree: root, Operation: "pose delegate review", PID: os.Getpid()}
	owner.Host, _ = os.Hostname()
	identifyStartLockProcess(&owner)
	owner.PID = dead.Process.Pid
	old := time.Now().Add(-time.Hour).UTC().Format(time.RFC3339Nano)
	marker := DelegationActiveRun{SchemaVersion: 1, AttemptID: "rat-" + strings.Repeat("e", 64), RunToken: strings.Repeat("f", 32), Purpose: DelegationPurposeReview, Agent: "fake", Scope: "spec:alpha", BundleID: "rvb-00000000000000a3", StartedAt: old, Owner: owner, HeartbeatAt: old, HeartbeatEvery: 10}
	dir, _ := s.delegationActiveDir(true)
	raw, _ := json.Marshal(marker)
	if err := os.WriteFile(filepath.Join(dir, marker.AttemptID+".json"), raw, 0o600); err != nil {
		t.Fatal(err)
	}
	attempts, _, err := s.ListDelegationAttempts()
	if err != nil || len(attempts) != 1 || attempts[0].State != DelegationStalled || !strings.Contains(attempts[0].Failure, DelegationAbandonedCause) || !strings.Contains(attempts[0].Failure, "pose delegate cancel "+marker.AttemptID) {
		t.Fatalf("the abandoned attempt is not listed with its cause: %+v %v", attempts, err)
	}
	run, err := s.ReleaseAbandonedAttempt(marker.AttemptID, time.Now(), false)
	if err != nil || run.RunID != "" {
		t.Fatalf("release: %+v %v", run, err)
	}
	if left, _ := filepath.Glob(filepath.Join(dir, marker.AttemptID+"*")); len(left) != 0 {
		t.Fatalf("the marker stayed: %v", left)
	}
}

// writeDeadMarker rewrites the marker of attempt so its owner is a process
// that has exited and its heartbeat is an hour old, as a dispatch killed
// mid-run leaves it.
func writeDeadMarker(t *testing.T, s Store, attempt string) {
	t.Helper()
	dir, err := s.delegationActiveDir(false)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, attempt+".json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var marker DelegationActiveRun
	if err := json.Unmarshal(raw, &marker); err != nil {
		t.Fatal(err)
	}
	dead := exec.Command("true")
	if err := dead.Run(); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-time.Hour).UTC().Format(time.RFC3339Nano)
	marker.Owner.PID, marker.HeartbeatAt, marker.StartedAt = dead.Process.Pid, old, old
	raw, _ = json.Marshal(marker)
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
}

// Review round 2, N4 (R6): a real dispatch interrupted after its reservation
// and before its run record blocks the next dispatch; once released, the
// next dispatch runs.
func TestAnInterruptedDispatchIsReleasedAndTheNextOneRuns(t *testing.T) {
	f := provenanceRunFixture(t, "openai", "probe-model")
	plan, err := f.store.PlanReviewDispatch(f.bundle.BundleID, "review", "fixture labelled notes", "fake", "interrupted")
	if err != nil {
		t.Fatal(err)
	}
	attempt := ""
	plan.OnStart = func(a string) error { attempt = a; panic("the dispatching process ends here") }
	func() {
		defer func() { _ = recover() }()
		_, _ = f.store.RunReviewDispatchContext(context.Background(), plan, time.Now)
	}()
	if attempt == "" {
		t.Fatal("the dispatch never reached its start")
	}
	writeDeadMarker(t, f.store, attempt)
	if _, err := f.store.PlanReviewDispatch(f.bundle.BundleID, "review", "fixture labelled notes", "fake", "after interruption"); err == nil || !strings.Contains(err.Error(), "pose delegate cancel "+attempt) {
		t.Fatalf("the interrupted attempt did not block the bundle with its release command: %v", err)
	}
	if _, err := f.store.ReleaseAbandonedAttempt(attempt, time.Now(), false); err != nil {
		t.Fatal(err)
	}
	next, err := f.store.PlanReviewDispatch(f.bundle.BundleID, "review", "fixture labelled notes", "fake", "after release")
	if err != nil {
		t.Fatalf("the released attempt still blocks the bundle: %v", err)
	}
	if run, err := f.store.RunReviewDispatchContext(context.Background(), next, time.Now); err != nil || run.Status != ReviewRunCompleted {
		t.Fatalf("the next dispatch did not run: %s %v", run.Status, err)
	}
}

// Review round 2, N3: a release already claimed is not run again.
func TestAClaimedReleaseIsNotRunTwice(t *testing.T) {
	root := t.TempDir()
	s := Store{Root: root}
	attempt := "rat-" + strings.Repeat("9", 64)
	dir, _ := s.delegationActiveDir(true)
	claim := filepath.Join(dir, attempt+".release")
	live := StartLockOwner{SchemaVersion: StartLockOwnerSchemaVersion, Resource: "release:x", Operation: "pose delegate cancel", PID: os.Getpid()}
	live.Host, _ = os.Hostname()
	identifyStartLockProcess(&live)
	raw, _ := json.Marshal(live)
	if err := os.WriteFile(claim, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := s.ReleaseAbandonedAttempt(attempt, time.Now(), false); !errors.Is(err, ErrAttemptNotAbandoned) || !strings.Contains(err.Error(), "in progress") {
		t.Fatalf("a claimed release ran again: %v", err)
	}
	// Review round 3, N1: a claim whose owner is gone is an interrupted
	// release, taken over.
	dead := exec.Command("true")
	if err := dead.Run(); err != nil {
		t.Fatal(err)
	}
	gone := live
	gone.PID = dead.Process.Pid
	raw, _ = json.Marshal(gone)
	if err := os.WriteFile(claim, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := claimAttemptRelease(claim, time.Now()); err != nil {
		t.Fatalf("an interrupted release's claim was not taken over: %v", err)
	}
}

// Review round 3, N1: a release interrupted after it recorded the run, and
// before the receipt, completes with that run instead of recording another.
func TestAnInterruptedReleaseCompletesWithItsRun(t *testing.T) {
	f := provenanceRunFixture(t, "openai", "probe-model")
	plan, err := f.store.PlanReviewDispatch(f.bundle.BundleID, "review", "fixture labelled notes", "fake", "interrupted")
	if err != nil {
		t.Fatal(err)
	}
	attempt := ""
	plan.OnStart = func(a string) error { attempt = a; panic("the dispatching process ends here") }
	func() {
		defer func() { _ = recover() }()
		_, _ = f.store.RunReviewDispatchContext(context.Background(), plan, time.Now)
	}()
	writeDeadMarker(t, f.store, attempt)
	activeDir, _ := f.store.delegationActiveDir(false)
	markerPath := filepath.Join(activeDir, attempt+".json")
	markerBytes, err := os.ReadFile(markerPath)
	if err != nil {
		t.Fatal(err)
	}
	first, err := f.store.ReleaseAbandonedAttempt(attempt, time.Now(), false)
	if err != nil {
		t.Fatal(err)
	}
	// Undo the receipt and keep the marker, as a release stopped between
	// the run and the receipt leaves them.
	attempts, _ := f.store.readDelegatedReviewAttempts(f.bundle.BundleID)
	for _, a := range attempts {
		if a.AttemptID == attempt {
			if err := os.Remove(attemptReceiptPath(f.store.Root, a)); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := os.WriteFile(markerPath, markerBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	second, err := f.store.ReleaseAbandonedAttempt(attempt, time.Now(), false)
	if err != nil || second.RunID != first.RunID {
		t.Fatalf("the interrupted release recorded another run: %s then %s (%v)", first.RunID, second.RunID, err)
	}
}

// Review round 2, N1: a reservation an earlier engine left with no marker is
// listed and released once it is old enough, never while it is young.
func TestAMarkerlessReservationOfAnEarlierEngineIsReleased(t *testing.T) {
	root := t.TempDir()
	s := Store{Root: root}
	reserve := func(age time.Duration, nonce string) DelegatedReviewAttempt {
		a := DelegatedReviewAttempt{SchemaVersion: 1, BundleID: "rvb-00000000000000b" + nonce[:1], BundleDigest: "sha256:" + strings.Repeat("a", 64), Scope: "spec:alpha", Adapter: "fake", Ordinal: 1,
			ReservedAt: time.Now().Add(-age).UTC().Format(time.RFC3339Nano), Nonce: strings.Repeat(nonce[:1], 32)}
		a.AttemptID = attemptIdentity(a)
		rel, _ := filepath.Rel(root, reviewAttemptPath(root, a))
		if err := publishLedgerJSON(root, filepath.ToSlash(rel), a); err != nil {
			t.Fatal(err)
		}
		return a
	}
	old, young := reserve(time.Hour, "1"), reserve(time.Minute, "2")
	attempts, _, err := s.ListDelegationAttempts()
	if err != nil {
		t.Fatal(err)
	}
	listed := map[string]DelegationAttempt{}
	for _, a := range attempts {
		listed[a.AttemptID] = a
	}
	if a, ok := listed[old.AttemptID]; !ok || a.State != DelegationStalled || !strings.Contains(a.Failure, "pose delegate cancel "+old.AttemptID+" --confirm-abandoned") {
		t.Fatalf("the old markerless reservation is not listed with its release: %+v", attempts)
	}
	if _, ok := listed[young.AttemptID]; ok {
		t.Fatal("a young reservation was listed as abandoned")
	}
	if _, err := s.ReleaseAbandonedAttempt(young.AttemptID, time.Now(), true); !errors.Is(err, ErrAttemptNotAbandoned) {
		t.Fatalf("a young reservation was released: %v", err)
	}
	// Review round 3, N2: nothing proves a markerless reservation abandoned,
	// so it is released only on the operator's confirmation.
	if _, err := s.ReleaseAbandonedAttempt(old.AttemptID, time.Now(), false); !errors.Is(err, ErrAttemptNotAbandoned) || !strings.Contains(err.Error(), "--confirm-abandoned") {
		t.Fatalf("a markerless reservation was released without confirmation: %v", err)
	}
	run, err := s.ReleaseAbandonedAttempt(old.AttemptID, time.Now(), true)
	if err != nil || run.Status != ReviewRunFailed || !strings.Contains(run.Failure, "operator's confirmation") {
		t.Fatalf("release: %+v %v", run, err)
	}
	if _, err := s.readAttemptReceipt(old); err != nil {
		t.Fatalf("the released reservation still blocks its bundle: %v", err)
	}
}

// Review round 3, N3: an unreadable reservation is reported, not skipped.
func TestAnUnreadableReservationIsReportedAsALoss(t *testing.T) {
	root := t.TempDir()
	s := Store{Root: root}
	dir := filepath.Join(root, reviewAttemptDir, "broken")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "000001.json"), []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, lost, err := s.ListDelegationAttempts()
	if err != nil || !strings.Contains(strings.Join(lost, "\n"), "broken/000001.json could not be read") {
		t.Fatalf("an unreadable reservation was skipped silently: %v %v", lost, err)
	}
}
