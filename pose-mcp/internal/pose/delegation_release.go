package pose

// Releasing an abandoned attempt (spec
// pose-review-run-reaps-lingering-reviewer-processes R5): a dispatch that
// ends after reserving its attempt and marking it active, but before
// recording its run, leaves a reservation without a receipt. Every later
// dispatch of the bundle refuses on it, and cancellation refused it because
// nothing runs. Once its owner is proven dead (the marker is stalled), the
// attempt is closed by the same records a run that failed writes: a failed
// run naming the abandonment, its receipt, and the marker's removal. Nothing
// is deleted from the ledger.

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// ErrAttemptNotAbandoned means the attempt is running, finished, or not
// proven abandoned.
var ErrAttemptNotAbandoned = errors.New("pose: the attempt is not an abandoned reservation")

// ReleaseAbandonedAttempt records the failed run of an attempt whose
// dispatch ended before recording it, and removes its stalled marker.
func (s Store) ReleaseAbandonedAttempt(attemptID string, now time.Time) (ReviewRun, error) {
	markers, _, err := s.ListDelegationActive()
	if err != nil {
		return ReviewRun{}, err
	}
	var marker *DelegationActiveRun
	for i := range markers {
		if markers[i].AttemptID == attemptID {
			marker = &markers[i]
		}
	}
	if marker == nil || marker.StalledAt == "" || marker.Liveness != LockOwnerDead {
		return ReviewRun{}, fmt.Errorf("%w: its owner is not proven gone", ErrAttemptNotAbandoned)
	}
	attempts, err := s.readDelegatedReviewAttempts(marker.BundleID)
	if err != nil {
		return ReviewRun{}, err
	}
	var reserved *DelegatedReviewAttempt
	for i := range attempts {
		if attempts[i].AttemptID == attemptID {
			reserved = &attempts[i]
		}
	}
	if reserved == nil {
		// The dispatch ended between its marker and its reservation: nothing
		// blocks the bundle, and the marker alone is removed.
		return ReviewRun{}, s.removeDelegationMarker(*marker)
	}
	if _, err := os.Lstat(attemptReceiptPath(s.Root, *reserved)); !errors.Is(err, os.ErrNotExist) {
		return ReviewRun{}, fmt.Errorf("%w: it already has a receipt", ErrAttemptNotAbandoned)
	}
	run := ReviewRun{SchemaVersion: 1, Scope: reserved.Scope, BundleID: reserved.BundleID, BundleDigest: reserved.BundleDigest,
		Kind: DelegationKindForPurpose(marker.Purpose), Adapter: reserved.Adapter, Reason: reserved.Reason, AttemptID: reserved.AttemptID,
		StartedAt: marker.StartedAt, ExitCode: -1, Label: marker.Label, Purpose: marker.Purpose, Round: marker.Round,
		Status: ReviewRunFailed, Failure: DelegationAbandonedCause + " (stalled at " + marker.StalledAt + "); released by pose delegate cancel",
		EndedAt: now.UTC().Format(time.RFC3339Nano)}
	recorded, err := s.recordReviewRun(run, nil)
	if err != nil {
		return ReviewRun{}, err
	}
	if err := s.finishDelegatedReviewAttempt(*reserved, recorded); err != nil {
		return recorded, err
	}
	return recorded, s.removeDelegationMarker(*marker)
}

func (s Store) removeDelegationMarker(marker DelegationActiveRun) error {
	dir, err := s.delegationActiveDir(false)
	if err != nil {
		return err
	}
	for _, p := range []string{filepath.Join(dir, marker.AttemptID+".json"), stallClaimPath(dir, marker), cancelRequestPath(dir, marker)} {
		if err := os.Remove(p); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	return nil
}
