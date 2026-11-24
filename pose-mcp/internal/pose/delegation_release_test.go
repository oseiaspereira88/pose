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
