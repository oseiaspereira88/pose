//go:build linux

package pose

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
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
	if left, _ := filepath.Glob(filepath.Join(dir, a.AttemptID+"*")); len(left) != 1 || !strings.HasSuffix(left[0], ".release.lock") {
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
	if left, _ := filepath.Glob(filepath.Join(dir, marker.AttemptID+"*")); len(left) != 1 || !strings.HasSuffix(left[0], ".release.lock") {
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

// Review rounds 2 and 5 (N3, N1): one release runs at a time, under a lock
// the operating system releases when its holder ends.
func TestOneReleaseRunsAtATime(t *testing.T) {
	root := t.TempDir()
	s := Store{Root: root}
	attempt := "rat-" + strings.Repeat("9", 64)
	dir, _ := s.delegationActiveDir(true)
	lockPath := filepath.Join(dir, attempt+".release.lock")
	held, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	unlock, err := lockSpecTransferFile(held)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.ReleaseAbandonedAttempt(attempt, time.Now(), false); !errors.Is(err, ErrAttemptNotAbandoned) || !strings.Contains(err.Error(), "in progress") {
		t.Fatalf("a release ran while another held the lock: %v", err)
	}
	unlock()
	held.Close()
	// A holder that ends releases the lock with it: nothing is left to take over.
	if _, err := exec.LookPath("flock"); err == nil {
		holder := exec.Command("flock", lockPath, "sleep", "30")
		holder.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
		if err := holder.Start(); err != nil {
			t.Fatal(err)
		}
		time.Sleep(200 * time.Millisecond)
		if _, err := s.ReleaseAbandonedAttempt(attempt, time.Now(), false); err == nil || !strings.Contains(err.Error(), "in progress") {
			t.Fatalf("the other process's lock was not seen: %v", err)
		}
		_ = syscall.Kill(-holder.Process.Pid, syscall.SIGKILL)
		_ = holder.Wait()
	}
	if _, err := s.ReleaseAbandonedAttempt(attempt, time.Now(), false); err != nil && strings.Contains(err.Error(), "in progress") {
		t.Fatalf("an ended holder's lock still blocks the release: %v", err)
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
	// Review round 4, N1: a release stopped between its run and its receipt
	// leaves the attempt with a run; it is still released, with that run.
	if err := os.Remove(attemptReceiptPath(root, old)); err != nil {
		t.Fatal(err)
	}
	again, err := s.ReleaseAbandonedAttempt(old.AttemptID, time.Now(), true)
	if err != nil || again.RunID != run.RunID {
		t.Fatalf("the interrupted release did not complete with its run: %s then %s (%v)", run.RunID, again.RunID, err)
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

// Review round 4, N3: with the run history incomplete, no reservation is
// taken as runless, and the loss is reported.
func TestAnIncompleteRunHistoryRulesNothingOut(t *testing.T) {
	root := t.TempDir()
	s := Store{Root: root}
	a := DelegatedReviewAttempt{SchemaVersion: 1, BundleID: "rvb-00000000000000c1", BundleDigest: "sha256:" + strings.Repeat("a", 64), Scope: "spec:alpha", Adapter: "fake", Ordinal: 1,
		ReservedAt: time.Now().Add(-time.Hour).UTC().Format(time.RFC3339Nano), Nonce: strings.Repeat("3", 32)}
	a.AttemptID = attemptIdentity(a)
	rel, _ := filepath.Rel(root, reviewAttemptPath(root, a))
	if err := publishLedgerJSON(root, filepath.ToSlash(rel), a); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, reviewRunsDir), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, reviewRunsDir, "rrn-0000000000000bad.json"), []byte("{broken"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, runLost, err := s.ReadReviewRuns(""); err != nil || len(runLost) == 0 {
		t.Skipf("the fixture does not make the run history incomplete: %v %v", runLost, err)
	}
	orphans, lost, err := s.orphanReservations(time.Now())
	if err != nil || len(orphans) != 0 || !strings.Contains(strings.Join(lost, "\n"), "run history is incomplete") {
		t.Fatalf("an incomplete history ruled a run out: %v %v %v", orphans, lost, err)
	}
	if _, err := s.ReleaseAbandonedAttempt(a.AttemptID, time.Now(), true); err == nil {
		t.Fatal("a reservation was released over an incomplete run history")
	}
}

// Review round 5, N3: a dispatch that recorded its run and stopped before
// the receipt is completed with that run, not with a second one; N2: a
// release that stopped after the receipt only removes the marker.
func TestARecordedRunCompletesItsAttempt(t *testing.T) {
	f := provenanceRunFixture(t, "openai", "probe-model")
	plan, err := f.store.PlanReviewDispatch(f.bundle.BundleID, "review", "fixture labelled notes", "fake", "receipt lost")
	if err != nil {
		t.Fatal(err)
	}
	saved := beforeRunResultPublished
	t.Cleanup(func() { beforeRunResultPublished = saved })
	recorded := ""
	beforeRunResultPublished = func(r ReviewRun) { recorded = r.RunID; panic("the dispatching process ends before the receipt") }
	func() {
		defer func() { _ = recover() }()
		_, _ = f.store.RunReviewDispatchContext(context.Background(), plan, time.Now)
	}()
	beforeRunResultPublished = saved
	runs, _, _ := f.store.ReadReviewRuns(f.bundle.BundleID)
	if recorded == "" || len(runs) == 0 {
		t.Fatalf("the dispatch recorded no run: %q %d", recorded, len(runs))
	}
	attempt := runs[len(runs)-1].AttemptID
	writeDeadMarker(t, f.store, attempt)
	activeDir, _ := f.store.delegationActiveDir(false)
	markerBytes, _ := os.ReadFile(filepath.Join(activeDir, attempt+".json"))
	released, err := f.store.ReleaseAbandonedAttempt(attempt, time.Now(), false)
	if err != nil || released.RunID != recorded {
		t.Fatalf("the release did not complete with the recorded run %s: %s %v", recorded, released.RunID, err)
	}
	if after, _, _ := f.store.ReadReviewRuns(f.bundle.BundleID); len(after) != len(runs) {
		t.Fatalf("the release recorded another run: %d then %d", len(runs), len(after))
	}
	// The marker comes back, as a release stopped after the receipt leaves it.
	if err := os.WriteFile(filepath.Join(activeDir, attempt+".json"), markerBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	again, err := f.store.ReleaseAbandonedAttempt(attempt, time.Now(), false)
	if err != nil || again.RunID != recorded {
		t.Fatalf("a release after the receipt did not complete: %s %v", again.RunID, err)
	}
	if left, _ := filepath.Glob(filepath.Join(activeDir, attempt+".json")); len(left) != 0 {
		t.Fatal("the stalled marker stayed after the receipt")
	}
}

// Review round 6, N1: an unreadable active marker may be a live attempt's,
// so no reservation is taken as markerless.
func TestAnUnreadableMarkerRulesNothingOut(t *testing.T) {
	root := t.TempDir()
	s := Store{Root: root}
	a := DelegatedReviewAttempt{SchemaVersion: 1, BundleID: "rvb-00000000000000d1", BundleDigest: "sha256:" + strings.Repeat("a", 64), Scope: "spec:alpha", Adapter: "fake", Ordinal: 1,
		ReservedAt: time.Now().Add(-time.Hour).UTC().Format(time.RFC3339Nano), Nonce: strings.Repeat("4", 32)}
	a.AttemptID = attemptIdentity(a)
	rel, _ := filepath.Rel(root, reviewAttemptPath(root, a))
	if err := publishLedgerJSON(root, filepath.ToSlash(rel), a); err != nil {
		t.Fatal(err)
	}
	dir, _ := s.delegationActiveDir(true)
	if err := os.WriteFile(filepath.Join(dir, a.AttemptID+".json"), []byte("{damaged"), 0o600); err != nil {
		t.Fatal(err)
	}
	orphans, lost, err := s.orphanReservations(time.Now())
	if err != nil || len(orphans) != 0 || !strings.Contains(strings.Join(lost, "\n"), "active marker is unreadable") {
		t.Fatalf("an unreadable marker let a reservation pass as markerless: %v %v %v", orphans, lost, err)
	}
	if _, err := s.ReleaseAbandonedAttempt(a.AttemptID, time.Now(), true); err == nil {
		t.Fatal("a reservation with an unreadable marker was released")
	}
}

// Review round 10, N2: a directory listing of the orphan scan is bounded and
// says when it was cut.
func TestTheOrphanScanListsBoundedDirectories(t *testing.T) {
	dir := t.TempDir()
	for i := 0; i < 12; i++ {
		if err := os.WriteFile(filepath.Join(dir, fmt.Sprintf("%06d.json", i)), []byte("{}"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	entries, truncated, err := readDirBounded(dir, 10)
	if err != nil || len(entries) != 10 || !truncated {
		t.Fatalf("listing: %d entries, truncated=%t, %v", len(entries), truncated, err)
	}
	entries, truncated, err = readDirBounded(dir, 20)
	if err != nil || len(entries) != 12 || truncated {
		t.Fatalf("listing: %d entries, truncated=%t, %v", len(entries), truncated, err)
	}
}
