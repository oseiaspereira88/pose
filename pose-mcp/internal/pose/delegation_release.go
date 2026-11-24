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
// time per attempt under an exclusive claim that records its owner; a claim
// whose owner is gone is taken over, and a release interrupted after it
// recorded the run completes it instead of recording another (N1).

import (
	"bytes"
	"encoding/json"
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
	claim := filepath.Join(dir, attemptID+".release")
	if err := claimAttemptRelease(claim, now); err != nil {
		return ReviewRun{}, err
	}
	run, err := s.releaseAbandonedAttemptClaimed(attemptID, marker, now, operatorConfirmed)
	if rerr := os.Remove(claim); rerr != nil && !errors.Is(rerr, os.ErrNotExist) && err == nil {
		err = rerr
	}
	return run, err
}

// claimAttemptRelease publishes the release claim with its owner. A claim
// whose owner is proven gone is an interrupted release and is taken over.
func claimAttemptRelease(claim string, now time.Time) error {
	owner := StartLockOwner{SchemaVersion: StartLockOwnerSchemaVersion, Resource: "release:" + filepath.Base(claim), Operation: "pose delegate cancel", PID: os.Getpid(), AcquiredAt: now.UTC().Format(time.RFC3339Nano)}
	owner.Host, _ = os.Hostname()
	identifyStartLockProcess(&owner)
	raw, _ := json.Marshal(owner)
	publish := func() error {
		return PublishFile(claim, append(raw, '\n'), 0600, PublishAuthoritative, PublishExpectation{Mode: ExpectAbsent})
	}
	if publish() == nil {
		return nil
	}
	held, err := os.ReadFile(claim)
	if err != nil {
		return fmt.Errorf("%w: the release claim %s cannot be read: %v", ErrAttemptNotAbandoned, claim, err)
	}
	var previous StartLockOwner
	if json.Unmarshal(held, &previous) != nil {
		return fmt.Errorf("%w: the release claim %s is not one this engine reads", ErrAttemptNotAbandoned, claim)
	}
	if liveness, detail := probeStartLockOwner(previous); liveness != LockOwnerDead {
		return fmt.Errorf("%w: another release of it is in progress (%s)", ErrAttemptNotAbandoned, detail)
	}
	// Taking over is a rename of the stale claim to a name of this process,
	// which only one taker can win, then an exclusive publish: two takers
	// never both enter the release (review round 4, N2).
	stale := fmt.Sprintf("%s.stale.%d.%d", claim, os.Getpid(), now.UnixNano())
	if err := os.Rename(claim, stale); err != nil {
		return fmt.Errorf("%w: another release took over the claim %s", ErrAttemptNotAbandoned, claim)
	}
	if after, err := os.ReadFile(stale); err != nil || !bytes.Equal(after, held) {
		return fmt.Errorf("%w: the claim %s changed while it was taken over", ErrAttemptNotAbandoned, claim)
	}
	if err := publish(); err != nil {
		return fmt.Errorf("%w: another release took over the claim %s", ErrAttemptNotAbandoned, claim)
	}
	_ = os.Remove(stale)
	return nil
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
	// A release interrupted after recording the run completes with that run.
	recorded, found, err := s.recordedAbandonmentRun(*reserved)
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

// recordedAbandonmentRun finds the failed run an interrupted release of a
// recorded for it.
func (s Store) recordedAbandonmentRun(a DelegatedReviewAttempt) (ReviewRun, bool, error) {
	runs, lost, err := s.ReadReviewRuns(a.BundleID)
	if err != nil {
		return ReviewRun{}, false, err
	}
	if len(lost) > 0 {
		// An unreadable run may be this release's: the history is unknown,
		// and the release does not write another (review round 4, N3).
		return ReviewRun{}, false, fmt.Errorf("the run history of %s is incomplete, so an earlier release cannot be ruled out: %s", a.BundleID, strings.Join(lost, "; "))
	}
	for _, r := range runs {
		if r.AttemptID == a.AttemptID && r.Status == ReviewRunFailed && strings.HasPrefix(r.Failure, DelegationAbandonedCause) {
			return r, true, nil
		}
	}
	return ReviewRun{}, false, nil
}

// orphanReservations lists the reservations an earlier engine left without
// a receipt, a marker or a run, older than reviewOrphanReservationAge, and
// says which reservations or receipts it could not read (review round 3, N3).
func (s Store) orphanReservations(now time.Time) ([]DelegatedReviewAttempt, []string, error) {
	base := filepath.Join(s.Root, reviewAttemptDir)
	dirs, err := os.ReadDir(base)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, err
	}
	markers, _, err := s.ListDelegationActive()
	if err != nil {
		return nil, nil, err
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
	ran := map[string]bool{}
	for _, r := range runs {
		// The run an interrupted release recorded leaves the reservation
		// abandoned until its receipt is written (review round 4, N1).
		if r.Status == ReviewRunFailed && strings.HasPrefix(r.Failure, DelegationAbandonedCause) {
			continue
		}
		ran[r.AttemptID] = true
	}
	var out []DelegatedReviewAttempt
	var lost []string
	budget := reviewReadBudget(nil)
	for _, d := range dirs {
		if !d.IsDir() {
			continue
		}
		entries, err := os.ReadDir(filepath.Join(base, d.Name()))
		if err != nil {
			lost = append(lost, "the reservations in "+d.Name()+" could not be listed: "+err.Error())
			continue
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
			if marked[a.AttemptID] || ran[a.AttemptID] {
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
	return out, lost, nil
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
