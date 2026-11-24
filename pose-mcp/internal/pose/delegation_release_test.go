//go:build linux

package pose

import (
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

	run, err := s.ReleaseAbandonedAttempt(a.AttemptID, time.Now())
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
	if _, err := s.ReleaseAbandonedAttempt(a.AttemptID, time.Now()); !errors.Is(err, ErrAttemptNotAbandoned) {
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
	if _, err := s.ReleaseAbandonedAttempt(marker.AttemptID, time.Now()); !errors.Is(err, ErrAttemptNotAbandoned) {
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
	run, err := s.ReleaseAbandonedAttempt(marker.AttemptID, time.Now())
	if err != nil || run.RunID != "" {
		t.Fatalf("release: %+v %v", run, err)
	}
	if left, _ := filepath.Glob(filepath.Join(dir, marker.AttemptID+"*")); len(left) != 0 {
		t.Fatalf("the marker stayed: %v", left)
	}
}
