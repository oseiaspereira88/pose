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
// Proof of abandonment: a marker that is stalled with its owner proven dead.
// A reservation an earlier engine left without any marker (it reserved
// before marking) records no owner, so nothing proves it abandoned: it is
// listed once it is older than reviewOrphanReservationAge and has no
// receipt or run, and released only on the operator's explicit confirmation,
// which the recorded run states (review round 3, N2). One release runs at a
// time per attempt under a lock the operating system releases when its
// holder ends, so an interrupted release leaves nothing to take over (review
// round 5, N1). A release is idempotent: it completes with a run the
// attempt already recorded, whatever wrote it (N3), and with a receipt
// already published it only removes the stalled marker (N2).

import (
	"errors"
	"fmt"
	"io"
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
func (s Store) ReleaseAbandonedAttempt(attemptID string, now time.Time, operatorConfirmed bool) (ReviewRun, error) {
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
	lock, err := os.OpenFile(filepath.Join(dir, attemptID+".release.lock"), os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return ReviewRun{}, err
	}
	defer lock.Close()
	unlock, err := lockSpecTransferFile(lock)
	if err != nil {
		return ReviewRun{}, fmt.Errorf("%w: another release of it is in progress", ErrAttemptNotAbandoned)
	}
	defer unlock()
	return s.releaseAbandonedAttemptClaimed(attemptID, marker, now, operatorConfirmed)
}

func (s Store) releaseAbandonedAttemptClaimed(attemptID string, marker *DelegationActiveRun, now time.Time, operatorConfirmed bool) (ReviewRun, error) {
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
		if !operatorConfirmed {
			return ReviewRun{}, fmt.Errorf("%w: it has no marker, so nothing proves its dispatch ended; if no dispatch of it is running, confirm with --confirm-abandoned", ErrAttemptNotAbandoned)
		}
		orphans, _, err := s.orphanReservations(now)
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
		// A release that stopped after the receipt completes by removing
		// the stalled marker; with no marker there is nothing to release.
		if marker != nil {
			receipt, err := s.readAttemptReceipt(*reserved)
			if err != nil {
				return ReviewRun{}, err
			}
			run, err := s.LoadProvenanceReviewRun(receipt.RunID)
			if err != nil {
				return ReviewRun{}, err
			}
			return run, s.removeDelegationMarker(*marker)
		}
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
		run.Failure = DelegationAbandonedCause + " (reservation of " + reserved.ReservedAt + " with no marker, left by an earlier engine); released by pose delegate cancel on the operator's confirmation"
	}
	// A run the attempt already recorded, by its dispatch or by a release
	// interrupted before the receipt, completes the attempt; no second run
	// is written for it.
	recorded, found, err := s.recordedAttemptRun(*reserved)
	if err != nil {
		return ReviewRun{}, err
	}
	if !found {
		if recorded, err = s.recordReviewRun(run, nil); err != nil {
			return ReviewRun{}, err
		}
	}
	if err := s.finishDelegatedReviewAttempt(*reserved, recorded); err != nil {
		return recorded, err
	}
	if marker != nil {
		return recorded, s.removeDelegationMarker(*marker)
	}
	return recorded, nil
}

// recordedAttemptRun finds the latest run recorded for attempt a.
func (s Store) recordedAttemptRun(a DelegatedReviewAttempt) (ReviewRun, bool, error) {
	runs, lost, err := s.ReadReviewRuns(a.BundleID)
	if err != nil {
		return ReviewRun{}, false, err
	}
	if len(lost) > 0 {
		// An unreadable run may be this release's: the history is unknown,
		// and the release does not write another (review round 4, N3).
		return ReviewRun{}, false, fmt.Errorf("the run history of %s is incomplete, so an earlier release cannot be ruled out: %s", a.BundleID, strings.Join(lost, "; "))
	}
	var latest *ReviewRun
	for i := range runs {
		if runs[i].AttemptID == a.AttemptID && (latest == nil || runs[i].EndedAt > latest.EndedAt) {
			latest = &runs[i]
		}
	}
	if latest == nil {
		return ReviewRun{}, false, nil
	}
	return *latest, true, nil
}

// orphanReservations lists the reservations left without a receipt or a
// marker, older than reviewOrphanReservationAge, and
// says which reservations or receipts it could not read (review round 3, N3).
func (s Store) orphanReservations(now time.Time) ([]DelegatedReviewAttempt, []string, error) {
	base := filepath.Join(s.Root, reviewAttemptDir)
	dirs, dirsTruncated, err := readDirBounded(base, reviewOrphanScanMaxEntries)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, err
	}
	markers, markerLost, err := s.ListDelegationActive()
	if err != nil {
		return nil, nil, err
	}
	if len(markerLost) > 0 {
		// An unreadable marker may be a live attempt's: nothing is known to
		// have no marker (review round 6, N1).
		return nil, append([]string{"abandoned reservations cannot be determined while an active marker is unreadable"}, markerLost...), nil
	}
	marked := map[string]bool{}
	for _, m := range markers {
		marked[m.AttemptID] = true
	}
	runs, runLost, err := s.ReadReviewRuns("")
	if err != nil {
		return nil, nil, err
	}
	if len(runLost) > 0 {
		// Without the whole run history nothing is known to have no run.
		return nil, append([]string{"abandoned reservations cannot be determined while the run history is incomplete"}, runLost...), nil
	}
	// A recorded run does not finish an attempt, its receipt does: a
	// reservation whose run was recorded without a receipt still blocks and
	// is released with that run (review round 5, N3).
	_ = runs
	var out []DelegatedReviewAttempt
	var lost []string
	budget := reviewReadBudget(nil)
	for _, d := range dirs {
		if !d.IsDir() {
			continue
		}
		entries, truncated, err := readDirBounded(filepath.Join(base, d.Name()), reviewOrphanScanMaxEntries)
		if err != nil {
			lost = append(lost, "the reservations in "+d.Name()+" could not be listed: "+err.Error())
			continue
		}
		if truncated {
			lost = append(lost, fmt.Sprintf("the reservations in %s exceed %d entries; the rest were not read", d.Name(), reviewOrphanScanMaxEntries))
		}
		for _, e := range entries {
			if IsPublisherStaging(e.Name()) || !strings.HasSuffix(e.Name(), ".json") || strings.HasSuffix(e.Name(), ".result.json") {
				continue
			}
			var a DelegatedReviewAttempt
			if err := readLedgerJSONBudgetIn(s.fsys(), filepath.Join(base, d.Name(), e.Name()), &a, budget); err != nil {
				lost = append(lost, "the reservation "+d.Name()+"/"+e.Name()+" could not be read: "+err.Error())
				continue
			}
			if a.AttemptID != attemptIdentity(a) {
				lost = append(lost, "the reservation "+d.Name()+"/"+e.Name()+" does not match its identity")
				continue
			}
			if marked[a.AttemptID] {
				continue
			}
			if _, err := os.Lstat(attemptReceiptPath(s.Root, a)); err == nil {
				continue
			} else if !errors.Is(err, os.ErrNotExist) {
				lost = append(lost, "the receipt of "+a.AttemptID+" could not be inspected: "+err.Error())
				continue
			}
			at, err := time.Parse(time.RFC3339Nano, a.ReservedAt)
			if err != nil || now.Sub(at) < reviewOrphanReservationAge {
				continue
			}
			out = append(out, a)
		}
	}
	if dirsTruncated {
		lost = append(lost, fmt.Sprintf("the attempt directories exceed %d entries; the rest were not read", reviewOrphanScanMaxEntries))
	}
	return out, lost, nil
}

// reviewOrphanScanMaxEntries bounds each directory the orphan scan lists
// (review round 10, N2).
const reviewOrphanScanMaxEntries = 10000

// readDirBounded lists at most max entries of dir and says whether more exist.
func readDirBounded(dir string, max int) ([]os.DirEntry, bool, error) {
	f, err := os.Open(dir)
	if err != nil {
		return nil, false, err
	}
	defer f.Close()
	entries, err := f.ReadDir(max + 1)
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, false, err
	}
	if len(entries) > max {
		return entries[:max], true, nil
	}
	return entries, false, nil
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
