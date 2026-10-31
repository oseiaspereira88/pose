package pose

// Active delegated runs and cancellation (spec pose-delegation-surface R13).
//
// A run's record is written when it ends, so a running run is known only by a
// private marker its process writes when the attempt starts and removes when
// the record is published. The marker names the process the way a start lock
// does, so a reader can tell a running owner from a dead one. Cancelling
// writes a request beside the marker; the running process watches for it,
// stops the agent and records the run failed with the cause cancelled. The
// attempt stays consumed: cancelling never returns budget.

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const delegationActiveDir = reviewPrivateDir + "/active"

// DelegationActiveRun is the marker of a run in progress.
type DelegationActiveRun struct {
	SchemaVersion int    `json:"schema_version"`
	AttemptID     string `json:"attempt_id"`
	// RunToken names this run of the attempt: a cancellation request carries
	// it, so a request meant for one run never stops another.
	RunToken  string         `json:"run_token"`
	Resumes   string         `json:"resumes,omitempty"`
	Label     string         `json:"label,omitempty"`
	Purpose   string         `json:"purpose"`
	Agent     string         `json:"agent"`
	Scope     string         `json:"scope"`
	BundleID  string         `json:"bundle_id"`
	StartedAt string         `json:"started_at"`
	Owner     StartLockOwner `json:"owner"`
	// HeartbeatAt is rewritten every HeartbeatEvery seconds by the process
	// that runs the attempt (spec pose-delegated-review-completion-signal R4,
	// D2).
	HeartbeatAt    string `json:"heartbeat_at,omitempty"`
	HeartbeatEvery int    `json:"heartbeat_every,omitempty"`
	// StalledAt is computed by the reader from the stall claim the first
	// reader that found the run stalled published.
	StalledAt string `json:"stalled_at,omitempty"`
	// Liveness is computed by the reader: owner-active, owner-dead or owner-unknown.
	Liveness string `json:"liveness,omitempty"`
	Detail   string `json:"detail,omitempty"`
	// CancelRequested is computed by the reader.
	CancelRequested bool `json:"cancel_requested,omitempty"`
	// Round is the review round the run reviews (spec
	// pose-delegated-review-round-protocol R1).
	Round int `json:"round,omitempty"`
}

func (s Store) delegationActiveDir(create bool) (string, error) {
	if create {
		if _, err := ensureReviewArtifactDir(s.Root, reviewPrivateDir, true); err != nil {
			return "", err
		}
		dir := filepath.Join(s.Root, filepath.FromSlash(reviewPrivateDir))
		ignore := filepath.Join(dir, ".gitignore")
		if _, err := os.Lstat(ignore); errors.Is(err, os.ErrNotExist) {
			if err := PublishFile(ignore, []byte("*\n"), 0600, PublishAuthoritative, PublishExpectation{Mode: ExpectAbsent}); err != nil && !errors.Is(err, os.ErrExist) {
				return "", err
			}
		}
	}
	return ensureReviewArtifactDir(s.Root, delegationActiveDir, create)
}

func validAttemptFileName(attempt string) bool {
	return attempt != "" && len(attempt) <= 128 && !strings.ContainsAny(attempt, `/\.`+"\x00")
}

// markDelegationActive writes the marker and starts the cancellation watcher.
// The returned stop removes both; cancelled reports whether a cancellation
// request stopped the run.
//
// stop(true) stops the watcher and the heartbeat but keeps the marker: a run
// whose record was not published leaves it, and it turns stalled once its
// process is gone (spec pose-delegated-review-completion-signal R4, R5).
func (s Store) markDelegationActive(run ReviewRun, cancel context.CancelFunc) (stop func(keepMarker ...bool), cancelled *atomic.Bool, err error) {
	cancelled = &atomic.Bool{}
	if !validAttemptFileName(run.AttemptID) {
		return func(...bool) {}, cancelled, fmt.Errorf("pose: attempt %q cannot name an active marker", run.AttemptID)
	}
	dir, err := s.delegationActiveDir(true)
	if err != nil {
		return func(...bool) {}, cancelled, err
	}
	owner := StartLockOwner{SchemaVersion: StartLockOwnerSchemaVersion, Resource: "delegate:" + run.AttemptID, Worktree: s.Root, Operation: "pose delegate " + run.Purpose, PID: os.Getpid(), AcquiredAt: run.StartedAt}
	owner.Host, _ = os.Hostname()
	identifyStartLockProcess(&owner)
	token := make([]byte, 16)
	if _, err := rand.Read(token); err != nil {
		return func(...bool) {}, cancelled, err
	}
	every := delegationHeartbeatEvery
	marker := DelegationActiveRun{SchemaVersion: 1, AttemptID: run.AttemptID, RunToken: hex.EncodeToString(token), Resumes: run.Resumes, Label: run.Label, Purpose: run.Purpose, Agent: run.Adapter, Scope: run.Scope, BundleID: run.BundleID, StartedAt: run.StartedAt, Owner: owner, Round: run.Round,
		HeartbeatAt: time.Now().UTC().Format(time.RFC3339Nano), HeartbeatEvery: int(every / time.Second)}
	raw, _ := json.MarshalIndent(marker, "", "  ")
	path := filepath.Join(dir, run.AttemptID+".json")
	request := cancelRequestPath(dir, marker)
	if err := PublishFile(path, append(raw, '\n'), 0600, PublishAuthoritative, PublishExpectation{Mode: ExpectAbsent}); err != nil {
		return func(...bool) {}, cancelled, fmt.Errorf("pose: another run of attempt %s is active: %w", run.AttemptID, err)
	}
	done, exited := make(chan struct{}), make(chan struct{})
	go func() {
		defer close(exited)
		ticker := time.NewTicker(250 * time.Millisecond)
		defer ticker.Stop()
		heartbeat := time.NewTicker(every)
		defer heartbeat.Stop()
		for {
			select {
			case <-done:
				return
			case <-heartbeat.C:
				// Select picks among ready cases at random: once stop has
				// asked, no further heartbeat starts, so it waits for at
				// most the one in flight (review round 1, HB01).
				select {
				case <-done:
					return
				default:
				}
				beatDelegationMarker(path, marker.RunToken)
			case <-ticker.C:
				if _, err := os.Lstat(request); err == nil {
					cancelled.Store(true)
					cancel()
					return
				}
			}
		}
	}()
	var once sync.Once
	stop = func(keepMarker ...bool) {
		once.Do(func() {
			close(done)
			delegationStopRequested()
			// A heartbeat in flight finishes before the marker goes, so it
			// cannot rename a new one into place after the removal (spec
			// pose-delegation-marker-stop-joins-heartbeat R1).
			<-exited
			_ = os.Remove(request)
			if len(keepMarker) == 0 || !keepMarker[0] {
				_ = os.Remove(path)
			}
		})
	}
	return stop, cancelled, nil
}

// delegationStopRequested is a test seam called once stop has asked the
// watcher to end, before it waits for it.
var delegationStopRequested = func() {}

// delegationHeartbeatEvery is the interval a running attempt declares.
var delegationHeartbeatEvery = 10 * time.Second

// beatDelegationMarker rewrites the heartbeat of the marker run token names,
// by compare-and-swap: a marker already recorded stalled, or replaced by
// another run, is left as it is.
func beatDelegationMarker(path, token string) {
	raw, err := readLedgerRaw(path)
	if err != nil {
		return
	}
	var marker DelegationActiveRun
	if json.Unmarshal(raw, &marker) != nil || marker.RunToken != token {
		return
	}
	marker.HeartbeatAt = time.Now().UTC().Format(time.RFC3339Nano)
	next, _ := json.MarshalIndent(marker, "", "  ")
	_ = PublishFile(path, append(next, '\n'), 0600, PublishAuthoritative, ExpectBytes(raw, true))
}

// delegationMarkerSilent reports a heartbeat older than three intervals.
func delegationMarkerSilent(marker DelegationActiveRun, now time.Time) bool {
	every := time.Duration(marker.HeartbeatEvery) * time.Second
	if every <= 0 {
		every = delegationHeartbeatEvery
	}
	last := marker.HeartbeatAt
	if last == "" {
		last = marker.StartedAt
	}
	at, err := time.Parse(time.RFC3339Nano, last)
	return err == nil && now.Sub(at) > 3*every
}

// stallClaimPath names the claim that records a run stalled.
func stallClaimPath(dir string, marker DelegationActiveRun) string {
	return filepath.Join(dir, marker.AttemptID+"."+marker.RunToken+".stalled")
}

func readDelegationStall(dir string, marker DelegationActiveRun) string {
	raw, err := readLedgerRaw(stallClaimPath(dir, marker))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(raw))
}

// recordDelegationStall publishes the stall claim exclusively: the first
// reader's instant is the one every reader returns, so concurrent readers
// agree (R4).
func recordDelegationStall(dir string, marker DelegationActiveRun) string {
	at := time.Now().UTC().Format(time.RFC3339Nano)
	_ = PublishFile(stallClaimPath(dir, marker), []byte(at+"\n"), 0600, PublishAuthoritative, PublishExpectation{Mode: ExpectAbsent})
	return readDelegationStall(dir, marker)
}

func cancelRequestPath(dir string, marker DelegationActiveRun) string {
	return filepath.Join(dir, marker.AttemptID+"."+marker.RunToken+".cancel")
}

func validRunToken(token string) bool {
	if len(token) != 32 {
		return false
	}
	_, err := hex.DecodeString(token)
	return err == nil
}

// ListDelegationActive reads the markers with the liveness of their owners,
// and returns as losses the markers it cannot read: a run it cannot observe is
// reported, never listed as running nor left out in silence.
func (s Store) ListDelegationActive() ([]DelegationActiveRun, []string, error) {
	dir, err := s.delegationActiveDir(false)
	if errors.Is(err, os.ErrNotExist) {
		return []DelegationActiveRun{}, nil, nil
	}
	if err != nil {
		return nil, nil, err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, nil, err
	}
	out := []DelegationActiveRun{}
	var lost []string
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		raw, err := readLedgerRaw(filepath.Join(dir, e.Name()))
		if errors.Is(err, os.ErrNotExist) {
			continue // the run ended between the listing and the read
		}
		if err != nil {
			lost = append(lost, "the active run marker "+e.Name()+" could not be read: "+err.Error())
			continue
		}
		var marker DelegationActiveRun
		if json.Unmarshal(raw, &marker) != nil || marker.SchemaVersion != 1 || marker.AttemptID+".json" != e.Name() || !validRunToken(marker.RunToken) {
			lost = append(lost, "the active run marker "+e.Name()+" is not one this engine reads")
			continue
		}
		marker.Liveness, marker.Detail = probeStartLockOwner(marker.Owner)
		marker.StalledAt = readDelegationStall(dir, marker)
		if marker.StalledAt == "" && delegationMarkerSilent(marker, time.Now()) && marker.Liveness == LockOwnerDead {
			marker.StalledAt = recordDelegationStall(dir, marker)
		}
		if _, err := os.Lstat(cancelRequestPath(dir, marker)); err == nil {
			marker.CancelRequested = true
		}
		out = append(out, marker)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].StartedAt < out[j].StartedAt || out[i].StartedAt == out[j].StartedAt && out[i].AttemptID < out[j].AttemptID
	})
	return out, lost, nil
}

// ErrDelegationNotActive means no running run matches.
var ErrDelegationNotActive = errors.New("pose: the run is not running")

// RequestDelegationCancel asks the process running attempt to stop. A marker
// whose owner is proven dead is not cancelled: nothing runs to stop.
func (s Store) RequestDelegationCancel(attempt string) (DelegationActiveRun, error) {
	active, _, err := s.ListDelegationActive()
	if err != nil {
		return DelegationActiveRun{}, err
	}
	for _, marker := range active {
		if marker.AttemptID != attempt {
			continue
		}
		if marker.Liveness == LockOwnerDead {
			return marker, fmt.Errorf("pose: the process of attempt %s is gone (%s); nothing runs to cancel", attempt, marker.Detail)
		}
		dir, err := s.delegationActiveDir(false)
		if err != nil {
			return marker, err
		}
		request := cancelRequestPath(dir, marker)
		if err := PublishFile(request, []byte(time.Now().UTC().Format(time.RFC3339Nano)+"\n"), 0600, PublishAuthoritative, PublishExpectation{Mode: ExpectAbsent}); err != nil && !errors.Is(err, os.ErrExist) {
			if _, statErr := os.Lstat(request); statErr != nil {
				return marker, err
			}
		}
		// The run may have ended between the read and the request: a request
		// that names no running run is withdrawn, never left for a successor.
		if !s.DelegationRunActive(marker) {
			_ = os.Remove(request)
			return marker, ErrDelegationNotActive
		}
		marker.CancelRequested = true
		return marker, nil
	}
	return DelegationActiveRun{}, ErrDelegationNotActive
}

// DelegationRunActive reports whether the run marker names is still the
// running run of its attempt.
func (s Store) DelegationRunActive(marker DelegationActiveRun) bool {
	dir, err := s.delegationActiveDir(false)
	if err != nil {
		return false
	}
	raw, err := readLedgerRaw(filepath.Join(dir, marker.AttemptID+".json"))
	if err != nil {
		return false
	}
	var now DelegationActiveRun
	return json.Unmarshal(raw, &now) == nil && now.RunToken == marker.RunToken
}

// DelegationPrivateDir is a directory of the engine's private delegation
// state, created without following a link; Git ignores the whole private tree.
func (s Store) DelegationPrivateDir(name string) (string, error) {
	if _, err := s.delegationActiveDir(true); err != nil {
		return "", err
	}
	return ensureReviewArtifactDir(s.Root, reviewPrivateDir+"/"+name, true)
}

// DelegationActiveMarkerExists reports, without reading the run history,
// whether attempt still has a marker: a waiter checks this cheaply and reads
// the history only when it is gone (spec
// pose-delegated-review-completion-signal R3).
func (s Store) DelegationActiveMarkerExists(attempt string) bool {
	if !validAttemptFileName(attempt) {
		return false
	}
	dir, err := s.delegationActiveDir(false)
	if err != nil {
		return false
	}
	_, err = os.Lstat(filepath.Join(dir, attempt+".json"))
	return err == nil
}
