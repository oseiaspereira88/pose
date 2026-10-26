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
	// Liveness is computed by the reader: owner-active, owner-dead or owner-unknown.
	Liveness string `json:"liveness,omitempty"`
	Detail   string `json:"detail,omitempty"`
	// CancelRequested is computed by the reader.
	CancelRequested bool `json:"cancel_requested,omitempty"`
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
func (s Store) markDelegationActive(run ReviewRun, cancel context.CancelFunc) (stop func(), cancelled *atomic.Bool, err error) {
	cancelled = &atomic.Bool{}
	if !validAttemptFileName(run.AttemptID) {
		return func() {}, cancelled, fmt.Errorf("pose: attempt %q cannot name an active marker", run.AttemptID)
	}
	dir, err := s.delegationActiveDir(true)
	if err != nil {
		return func() {}, cancelled, err
	}
	owner := StartLockOwner{SchemaVersion: StartLockOwnerSchemaVersion, Resource: "delegate:" + run.AttemptID, Worktree: s.Root, Operation: "pose delegate " + run.Purpose, PID: os.Getpid(), AcquiredAt: run.StartedAt}
	owner.Host, _ = os.Hostname()
	identifyStartLockProcess(&owner)
	token := make([]byte, 16)
	if _, err := rand.Read(token); err != nil {
		return func() {}, cancelled, err
	}
	marker := DelegationActiveRun{SchemaVersion: 1, AttemptID: run.AttemptID, RunToken: hex.EncodeToString(token), Resumes: run.Resumes, Label: run.Label, Purpose: run.Purpose, Agent: run.Adapter, Scope: run.Scope, BundleID: run.BundleID, StartedAt: run.StartedAt, Owner: owner}
	raw, _ := json.MarshalIndent(marker, "", "  ")
	path := filepath.Join(dir, run.AttemptID+".json")
	request := cancelRequestPath(dir, marker)
	if err := PublishFile(path, append(raw, '\n'), 0600, PublishAuthoritative, PublishExpectation{Mode: ExpectAbsent}); err != nil {
		return func() {}, cancelled, fmt.Errorf("pose: another run of attempt %s is active: %w", run.AttemptID, err)
	}
	done := make(chan struct{})
	go func() {
		ticker := time.NewTicker(250 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
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
	stop = func() {
		once.Do(func() {
			close(done)
			_ = os.Remove(request)
			_ = os.Remove(path)
		})
	}
	return stop, cancelled, nil
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

// ListDelegationActive reads the markers with the liveness of their owners.
func (s Store) ListDelegationActive() ([]DelegationActiveRun, error) {
	dir, err := s.delegationActiveDir(false)
	if errors.Is(err, os.ErrNotExist) {
		return []DelegationActiveRun{}, nil
	}
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	out := []DelegationActiveRun{}
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".json") || !e.Type().IsRegular() {
			continue
		}
		raw, err := readLedgerRaw(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		var marker DelegationActiveRun
		if json.Unmarshal(raw, &marker) != nil || marker.SchemaVersion != 1 || marker.AttemptID+".json" != e.Name() || !validRunToken(marker.RunToken) {
			continue
		}
		marker.Liveness, marker.Detail = probeStartLockOwner(marker.Owner)
		if _, err := os.Lstat(cancelRequestPath(dir, marker)); err == nil {
			marker.CancelRequested = true
		}
		out = append(out, marker)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].StartedAt < out[j].StartedAt || out[i].StartedAt == out[j].StartedAt && out[i].AttemptID < out[j].AttemptID
	})
	return out, nil
}

// ErrDelegationNotActive means no running run matches.
var ErrDelegationNotActive = errors.New("pose: the run is not running")

// RequestDelegationCancel asks the process running attempt to stop. A marker
// whose owner is proven dead is not cancelled: nothing runs to stop.
func (s Store) RequestDelegationCancel(attempt string) (DelegationActiveRun, error) {
	active, err := s.ListDelegationActive()
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
