package pose

// Releasing an abandoned attempt (spec
// pose-review-run-reaps-lingering-reviewer-processes R5): a dispatch that
// ends after reserving its attempt but before recording its run leaves a
// reservation without a receipt. Every later dispatch of the bundle refuses
// on it, and cancellation refused it because nothing runs. Once it is proven
// abandoned, the attempt is closed by the same records a failed run writes:
// a failed run naming the abandonment and its receipt. Nothing is deleted
// from the ledger.
//
// Proof of abandonment: a marker that is stalled with its owner proven dead;
// or, for a reservation an earlier engine left without any marker (it
// reserved before marking), no marker, no receipt, no run of the attempt and
// a reservation older than reviewOrphanReservationAge, far beyond the
// instant an earlier engine took between reserving and marking (review
// round 2, N1). One release runs at a time per attempt: a claim published
// exclusively guards it (N3).

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ErrAttemptNotAbandoned means the attempt is running, finished, or not
// proven abandoned.
var ErrAttemptNotAbandoned = errors.New("pose: the attempt is not an abandoned reservation")

// reviewOrphanReservationAge is how old a reservation without any marker
// must be to count as abandoned.
const reviewOrphanReservationAge = 10 * time.Minute

// ReleaseAbandonedAttempt records the failed run of an attempt whose
// dispatch ended before recording it, and removes its stalled marker. A
// stalled marker whose reservation was never published is only removed; the
// returned run is then empty.
func (s Store) ReleaseAbandonedAttempt(attemptID string, now time.Time) (ReviewRun, error) {
	if !validAttemptFileName(attemptID) {
		return ReviewRun{}, fmt.Errorf("%w: %q names no attempt", ErrAttemptNotAbandoned, attemptID)
	}
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
	if marker != nil && (marker.StalledAt == "" || marker.Liveness != LockOwnerDead) {
		return ReviewRun{}, fmt.Errorf("%w: its owner is not proven gone", ErrAttemptNotAbandoned)
	}
	dir, err := s.delegationActiveDir(true)
	if err != nil {
		return ReviewRun{}, err
	}
	claim := filepath.Join(dir, attemptID+".release")
	if err := PublishFile(claim, []byte(now.UTC().Format(time.RFC3339Nano)+"\n"), 0600, PublishAuthoritative, PublishExpectation{Mode: ExpectAbsent}); err != nil {
		return ReviewRun{}, fmt.Errorf("%w: another release of it is in progress or was interrupted (%s)", ErrAttemptNotAbandoned, claim)
	}
	run, err := s.releaseAbandonedAttemptClaimed(attemptID, marker, now)
	if rerr := os.Remove(claim); rerr != nil && !errors.Is(rerr, os.ErrNotExist) && err == nil {
		err = rerr
	}
	return run, err
}

func (s Store) releaseAbandonedAttemptClaimed(attemptID string, marker *DelegationActiveRun, now time.Time) (ReviewRun, error) {
	var reserved *DelegatedReviewAttempt
	if marker != nil {
		attempts, err := s.readDelegatedReviewAttempts(marker.BundleID)
		if err != nil {
			return ReviewRun{}, err
		}
		for i := range attempts {
			if attempts[i].AttemptID == attemptID {
				reserved = &attempts[i]
			}
		}
		if reserved == nil {
			// The dispatch ended between its marker and its reservation:
			// nothing blocks the bundle, and the marker alone is removed.
			return ReviewRun{}, s.removeDelegationMarker(*marker)
		}
	} else {
		orphans, err := s.orphanReservations(now)
		if err != nil {
			return ReviewRun{}, err
		}
		for i := range orphans {
			if orphans[i].AttemptID == attemptID {
				reserved = &orphans[i]
			}
		}
		if reserved == nil {
			return ReviewRun{}, fmt.Errorf("%w: no stalled marker and no reservation older than %s without a receipt", ErrAttemptNotAbandoned, reviewOrphanReservationAge)
		}
	}
	if _, err := os.Lstat(attemptReceiptPath(s.Root, *reserved)); err == nil {
		return ReviewRun{}, fmt.Errorf("%w: it already has a receipt", ErrAttemptNotAbandoned)
	} else if !errors.Is(err, os.ErrNotExist) {
		return ReviewRun{}, fmt.Errorf("the receipt of %s cannot be inspected: %w", attemptID, err)
	}
	run := ReviewRun{SchemaVersion: 1, Scope: reserved.Scope, BundleID: reserved.BundleID, BundleDigest: reserved.BundleDigest,
		Kind: ReviewBriefKindReview, Adapter: reserved.Adapter, Reason: reserved.Reason, AttemptID: reserved.AttemptID,
		StartedAt: reserved.ReservedAt, ExitCode: -1, Purpose: DelegationPurposeReview,
		Status: ReviewRunFailed, EndedAt: now.UTC().Format(time.RFC3339Nano)}
	if marker != nil {
		run.Kind, run.Purpose, run.Label, run.Round, run.StartedAt = DelegationKindForPurpose(marker.Purpose), marker.Purpose, marker.Label, marker.Round, marker.StartedAt
		run.Failure = DelegationAbandonedCause + " (stalled at " + marker.StalledAt + "); released by pose delegate cancel"
	} else {
		run.Failure = DelegationAbandonedCause + " (reservation of " + reserved.ReservedAt + " with no marker, left by an earlier engine); released by pose delegate cancel"
	}
	recorded, err := s.recordReviewRun(run, nil)
	if err != nil {
		return ReviewRun{}, err
	}
	if err := s.finishDelegatedReviewAttempt(*reserved, recorded); err != nil {
		return recorded, err
	}
	if marker != nil {
		return recorded, s.removeDelegationMarker(*marker)
	}
	return recorded, nil
}

// orphanReservations lists the reservations an earlier engine left without
// a receipt, a marker or a run, older than reviewOrphanReservationAge.
func (s Store) orphanReservations(now time.Time) ([]DelegatedReviewAttempt, error) {
	base := filepath.Join(s.Root, reviewAttemptDir)
	dirs, err := os.ReadDir(base)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	markers, _, err := s.ListDelegationActive()
	if err != nil {
		return nil, err
	}
	marked := map[string]bool{}
	for _, m := range markers {
		marked[m.AttemptID] = true
	}
	runs, _, err := s.ReadReviewRuns("")
	if err != nil {
		return nil, err
	}
	ran := map[string]bool{}
	for _, r := range runs {
		ran[r.AttemptID] = true
	}
	var out []DelegatedReviewAttempt
	budget := reviewReadBudget(nil)
	for _, d := range dirs {
		if !d.IsDir() {
			continue
		}
		entries, err := os.ReadDir(filepath.Join(base, d.Name()))
		if err != nil {
			return nil, err
		}
		for _, e := range entries {
			if IsPublisherStaging(e.Name()) || !strings.HasSuffix(e.Name(), ".json") || strings.HasSuffix(e.Name(), ".result.json") {
				continue
			}
			var a DelegatedReviewAttempt
			if err := readLedgerJSONBudgetIn(s.fsys(), filepath.Join(base, d.Name(), e.Name()), &a, budget); err != nil || a.AttemptID != attemptIdentity(a) {
				continue
			}
			if marked[a.AttemptID] || ran[a.AttemptID] {
				continue
			}
			if _, err := os.Lstat(attemptReceiptPath(s.Root, a)); !errors.Is(err, os.ErrNotExist) {
				continue
			}
			at, err := time.Parse(time.RFC3339Nano, a.ReservedAt)
			if err != nil || now.Sub(at) < reviewOrphanReservationAge {
				continue
			}
			out = append(out, a)
		}
	}
	return out, nil
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
