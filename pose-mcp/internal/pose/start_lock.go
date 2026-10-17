package pose

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// StartLockOwner identifies the process that holds a start lock, with enough
// context to tell a live owner from a dead one on the same machine: host,
// machine and boot identity, PID namespace, PID and the process start time,
// so a reused PID is not mistaken for the owner (spec
// pose-journal-checkpoint-and-lock-recovery R6). Machine and boot are
// application-specific digests, never the raw IDs. Fields a platform cannot
// observe are empty, and an owner missing them is never proven dead.
type StartLockOwner struct {
	SchemaVersion int    `json:"schema_version"`
	Resource      string `json:"resource"`
	Worktree      string `json:"worktree"`
	Operation     string `json:"operation"`
	Host          string `json:"host"`
	MachineID     string `json:"machine_id,omitempty"`
	BootID        string `json:"boot_id,omitempty"`
	PIDNamespace  string `json:"pid_namespace,omitempty"`
	PID           int    `json:"pid"`
	ProcessStart  string `json:"process_start,omitempty"`
	AcquiredAt    string `json:"acquired_at"`
}

// StartLockOwnerSchemaVersion is the schema of StartLockOwner.
const StartLockOwnerSchemaVersion = 1

// States of a start lock held by another call.
const (
	LockOwnerActive  = "owner-active"  // the owner is running
	LockOwnerUnknown = "owner-unknown" // its liveness cannot be established
	LockOwnerDead    = "owner-dead"    // the owner is proven not to be running
)

// StartLockError is a start lock held by another call. Code is the start
// error code; State says whether its owner is active, unknown or proven dead;
// OwnerDigest is the SHA-256 of the lock bytes a recovery must name, so it
// removes exactly the lock that was inspected (R7).
type StartLockError struct {
	Code        string          `json:"code"`
	State       string          `json:"state"`
	Path        string          `json:"path"`
	Reason      string          `json:"reason"`
	Owner       *StartLockOwner `json:"owner,omitempty"`
	OwnerDigest string          `json:"owner_digest"`
}

func (e *StartLockError) Error() string { return "start: " + e.Code }

// Start lock error codes.
const (
	startLockHeld     = "start-in-progress"
	lifecycleLockHeld = "spec-lifecycle-in-progress"
	startLockUnknown  = "start-lock-owner-unknown"
	startLockOrphaned = "start-lock-orphaned"
)

func startLockPath(root, slug string) string {
	return filepath.Join(root, ".pose", "starts", "."+slug+".lock")
}

// lockStart takes the exclusive per-spec start lock for operation and
// records its owner in it. A lock held by another call is never broken here,
// whatever its age: it is reported with its owner's state, and only a lock
// whose owner is proven dead can be removed, by RecoverStartLock (R6/R7).
func lockStart(root, slug, operation string) (func(), error) {
	dir := filepath.Join(root, ".pose", "starts")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	path := startLockPath(root, slug)
	owner := currentStartLockOwner(root, "spec:"+slug, operation)
	raw, err := json.Marshal(owner)
	if err != nil {
		return nil, err
	}
	raw = append(raw, '\n')
	if err := publishStartLock(dir, path, raw); err != nil {
		if errors.Is(err, os.ErrExist) {
			return nil, inspectStartLock(root, slug)
		}
		return nil, err
	}
	return func() {
		// Release only our own lock: a lock whose bytes changed is not ours.
		if current, err := os.ReadFile(path); err == nil && bytes.Equal(current, raw) {
			os.Remove(path)
		}
	}, nil
}

// publishStartLock creates the lock with its owner already in it: the owner
// is written to an exclusive temporary file that is then hard-linked to the
// lock path, which fails if the lock exists. A concurrent call therefore
// never reads a lock without its owner. Where hard links are unavailable the
// lock is created exclusively and written after, and a concurrent call may
// briefly read it as owner-unknown, which is still a refusal.
func publishStartLock(dir, path string, raw []byte) error {
	tmp, err := os.CreateTemp(dir, ".lock-*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	_, err = tmp.Write(raw)
	if closeErr := tmp.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	linkErr := os.Link(tmp.Name(), path)
	if linkErr == nil || errors.Is(linkErr, os.ErrExist) {
		return linkErr
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	_, err = f.Write(raw)
	if closeErr := f.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		os.Remove(path)
	}
	return err
}

// LockSpecLifecycle takes the per-spec lifecycle lock that start, close and
// transfer share (spec pose-writer-coordination-and-atomic-publication R3).
// It never waits: a lock another call holds is a *StartLockError with its
// owner's state, so no two lifecycle writers interleave and none deadlocks.
func LockSpecLifecycle(root, slug, operation string) (func(), error) {
	if ValidateSlug(slug) != nil {
		return nil, startError("invalid-spec")
	}
	return lockStart(root, slug, operation)
}

// inspectStartLock classifies a start lock that exists.
func inspectStartLock(root, slug string) *StartLockError {
	path := startLockPath(root, slug)
	rel := filepath.ToSlash(filepath.Join(".pose", "starts", "."+slug+".lock"))
	e := &StartLockError{Code: startLockUnknown, State: LockOwnerUnknown, Path: rel}
	raw, err := os.ReadFile(path)
	if err != nil {
		e.Reason = "the lock cannot be read: " + err.Error()
		return e
	}
	sum := sha256.Sum256(raw)
	e.OwnerDigest = hex.EncodeToString(sum[:])
	var owner StartLockOwner
	if len(bytes.TrimSpace(raw)) == 0 || strictJSONBytes(raw, &owner) != nil || owner.SchemaVersion != StartLockOwnerSchemaVersion {
		e.Reason = "the lock records no owner this engine can read (an older engine, or a write that did not finish)"
		return e
	}
	e.Owner = &owner
	state, reason := probeStartLockOwner(owner)
	e.State, e.Reason = state, reason
	switch state {
	case LockOwnerActive:
		// A running start keeps its historical code; any other lifecycle
		// writer (a close, a transfer) is named for what it is.
		e.Code = startLockHeld
		if !strings.HasPrefix(owner.Operation, "start.") {
			e.Code = lifecycleLockHeld
		}
	case LockOwnerDead:
		e.Code = startLockOrphaned
	}
	return e
}

// StartLockState reports the start lock of a spec, if any.
func (s Store) StartLockState(slug string) (*StartLockError, error) {
	if ValidateSlug(slug) != nil {
		return nil, startError("invalid-spec")
	}
	if _, err := os.Lstat(startLockPath(s.Root, slug)); os.IsNotExist(err) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	return inspectStartLock(s.Root, slug), nil
}

// RecoverStartLock removes a start lock whose owner is proven dead. The
// caller names the owner digest it inspected; under an exclusive lock of the
// starts directory the lock is read again, and it is removed only if its
// bytes are still those and its owner is still proven dead. A lock with an
// active or unknown owner, or one that changed, is refused. The start record
// and the spec are not touched: the interrupted intention stays where it was
// recorded and resumes from there (R7/R8).
func (s Store) RecoverStartLock(slug, ownerDigest string) (*StartLockError, error) {
	if ValidateSlug(slug) != nil {
		return nil, startError("invalid-spec")
	}
	// Only a lock proven dead is worth taking the directory lock for; the
	// decision is made again under it.
	if _, err := os.Lstat(startLockPath(s.Root, slug)); os.IsNotExist(err) {
		return nil, startError("start-lock-absent")
	}
	if first := inspectStartLock(s.Root, slug); first.State != LockOwnerDead {
		return first, startError("start-lock-owner-not-dead")
	}
	dir, err := os.Open(filepath.Join(s.Root, ".pose", "starts"))
	if err != nil {
		return nil, err
	}
	defer dir.Close()
	unlock, err := lockSpecTransferFile(dir)
	if err != nil {
		return nil, fmt.Errorf("start: start-lock-recovery-busy (%v)", err)
	}
	defer unlock()
	if _, err := os.Lstat(startLockPath(s.Root, slug)); os.IsNotExist(err) {
		return nil, startError("start-lock-absent")
	}
	state := inspectStartLock(s.Root, slug)
	if state.OwnerDigest != ownerDigest {
		return state, startError("start-lock-changed")
	}
	if state.State != LockOwnerDead {
		return state, startError("start-lock-owner-not-dead")
	}
	if err := os.Remove(startLockPath(s.Root, slug)); err != nil {
		return state, err
	}
	return state, nil
}

// CurrentStartLockOwner is the owner a lock taken now by this process
// records.
func CurrentStartLockOwner(root, resource, operation string) StartLockOwner {
	return currentStartLockOwner(root, resource, operation)
}

func currentStartLockOwner(root, resource, operation string) StartLockOwner {
	host, _ := os.Hostname()
	worktree, err := filepath.Abs(root)
	if err != nil {
		worktree = root
	}
	owner := StartLockOwner{SchemaVersion: StartLockOwnerSchemaVersion, Resource: resource, Worktree: worktree, Operation: operation,
		Host: host, PID: os.Getpid(), AcquiredAt: time.Now().UTC().Format(time.RFC3339)}
	identifyStartLockProcess(&owner)
	return owner
}
