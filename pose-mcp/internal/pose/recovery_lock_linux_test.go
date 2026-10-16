//go:build linux

package pose

import (
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
)

// Spec pose-journal-checkpoint-and-lock-recovery V5/V6/V7: a start lock is
// judged by its owner's identity, and only a proven-dead one is removed.

func lockFixture(t *testing.T) Store {
	t.Helper()
	store, _ := startFixture(t, true, "---\nslug: work\nstatus: draft\ncreated_at: 2026-09-27\n---\n\n# Spec: work\n\n## 2. Requirements\n\n- R1: One.\n")
	if err := os.MkdirAll(filepath.Join(store.Root, ".pose", "starts"), 0o755); err != nil {
		t.Fatal(err)
	}
	return store
}

func writeLockOwner(t *testing.T, store Store, edit func(*StartLockOwner)) {
	t.Helper()
	owner := currentStartLockOwner(store.Root, "spec:work", "start.apply")
	edit(&owner)
	raw, _ := json.Marshal(owner)
	if err := os.WriteFile(startLockPath(store.Root, "work"), append(raw, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
}

// exitedPID is the PID of a process that has finished.
func exitedPID(t *testing.T) int {
	t.Helper()
	cmd := exec.Command("true")
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}
	return cmd.Process.Pid
}

func TestAStartLockIsJudgedByItsOwnersIdentity(t *testing.T) {
	gone := exitedPID(t)
	for name, c := range map[string]struct {
		edit  func(*StartLockOwner)
		state string
	}{
		"this process":        {func(*StartLockOwner) {}, LockOwnerActive},
		"process gone":        {func(o *StartLockOwner) { o.PID, o.ProcessStart = gone, "1" }, LockOwnerDead},
		"pid reused":          {func(o *StartLockOwner) { o.ProcessStart = "1" }, LockOwnerDead},
		"earlier boot":        {func(o *StartLockOwner) { o.BootID = "0000" }, LockOwnerDead},
		"another host":        {func(o *StartLockOwner) { o.Host = "elsewhere.invalid" }, LockOwnerUnknown},
		"another machine":     {func(o *StartLockOwner) { o.MachineID = "0000" }, LockOwnerUnknown},
		"no machine id":       {func(o *StartLockOwner) { o.MachineID = "" }, LockOwnerUnknown},
		"another namespace":   {func(o *StartLockOwner) { o.PIDNamespace = "pid:[1]" }, LockOwnerUnknown},
		"no process start":    {func(o *StartLockOwner) { o.ProcessStart = "" }, LockOwnerUnknown},
		"an old age is moot":  {func(o *StartLockOwner) { o.AcquiredAt = "2001-01-01T00:00:00Z" }, LockOwnerActive},
		"gone but other host": {func(o *StartLockOwner) { o.PID, o.Host = gone, "elsewhere.invalid" }, LockOwnerUnknown},
	} {
		store := lockFixture(t)
		writeLockOwner(t, store, c.edit)
		_, err := lockStart(store.Root, "work", "start.apply")
		var held *StartLockError
		if !errors.As(err, &held) || held.State != c.state || held.OwnerDigest == "" || held.Owner == nil {
			t.Fatalf("%s: %+v %v", name, held, err)
		}
		if _, err := os.Stat(startLockPath(store.Root, "work")); err != nil {
			t.Fatalf("%s: the lock was removed by its inspection", name)
		}
	}
}

func TestALockWithoutAReadableOwnerIsUnknown(t *testing.T) {
	for name, content := range map[string]string{"empty, as older engines wrote it": "", "partial": `{"schema_version":1,"pid":`, "another schema": `{"schema_version":2}`} {
		store := lockFixture(t)
		if err := os.WriteFile(startLockPath(store.Root, "work"), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		state, err := store.StartLockState("work")
		if err != nil || state == nil || state.State != LockOwnerUnknown || state.Code != startLockUnknown {
			t.Fatalf("%s: %+v %v", name, state, err)
		}
		if _, err := store.RecoverStartLock("work", state.OwnerDigest); err == nil {
			t.Fatalf("%s: a lock with no readable owner was recovered", name)
		}
	}
}

func TestOnlyAProvenDeadLockIsRecoveredAndOnlyTheOneInspected(t *testing.T) {
	store := lockFixture(t)
	writeLockOwner(t, store, func(o *StartLockOwner) {})
	live, _ := store.StartLockState("work")
	if _, err := store.RecoverStartLock("work", live.OwnerDigest); err == nil || err.Error() != "start: start-lock-owner-not-dead" {
		t.Fatalf("a live owner's lock: %v", err)
	}
	gone := exitedPID(t)
	writeLockOwner(t, store, func(o *StartLockOwner) { o.PID, o.ProcessStart = gone, "1" })
	dead, _ := store.StartLockState("work")
	if _, err := store.RecoverStartLock("work", live.OwnerDigest); err == nil || err.Error() != "start: start-lock-changed" {
		t.Fatalf("a digest of another lock: %v", err)
	}
	if _, err := store.RecoverStartLock("work", dead.OwnerDigest); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(startLockPath(store.Root, "work")); !os.IsNotExist(err) {
		t.Fatal("the dead lock is still there")
	}
	if _, err := store.RecoverStartLock("work", dead.OwnerDigest); err == nil || err.Error() != "start: start-lock-absent" {
		t.Fatalf("a second recovery: %v", err)
	}
	unlock, err := lockStart(store.Root, "work", "start.apply")
	if err != nil {
		t.Fatalf("the lock cannot be taken after recovery: %v", err)
	}
	unlock()
}

// V6 — two recoveries of the same dead lock race: exactly one removes it,
// and a lock taken after it is never removed by the other.
func TestConcurrentRecoveriesRemoveTheDeadLockOnce(t *testing.T) {
	gone := exitedPID(t)
	for round := 0; round < 20; round++ {
		store := lockFixture(t)
		writeLockOwner(t, store, func(o *StartLockOwner) { o.PID, o.ProcessStart = gone, "1" })
		dead, _ := store.StartLockState("work")
		var wg sync.WaitGroup
		results := make([]error, 2)
		for i := range results {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				_, results[i] = store.RecoverStartLock("work", dead.OwnerDigest)
			}(i)
		}
		wg.Wait()
		succeeded := 0
		for _, err := range results {
			if err == nil {
				succeeded++
			}
		}
		if succeeded != 1 {
			t.Fatalf("round %d: %d recoveries succeeded: %v", round, succeeded, results)
		}
		unlock, err := lockStart(store.Root, "work", "start.apply")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := store.RecoverStartLock("work", dead.OwnerDigest); err == nil {
			t.Fatal("a recovery removed a lock taken after it")
		}
		unlock()
	}
}

// R8 — a release removes only its own lock, and a recovered lock leaves the
// start record and the spec where the interrupted run left them.
func TestAReleaseRemovesOnlyItsOwnLock(t *testing.T) {
	store := lockFixture(t)
	unlock, err := lockStart(store.Root, "work", "start.apply")
	if err != nil {
		t.Fatal(err)
	}
	writeLockOwner(t, store, func(o *StartLockOwner) { o.Operation = "start.cancel" })
	unlock()
	if _, err := os.Stat(startLockPath(store.Root, "work")); err != nil {
		t.Fatal("a release removed a lock that was not its own")
	}
}

func TestARecoveredLockLeavesTheInterruptedStartToResume(t *testing.T) {
	store := lockFixture(t)
	plan, err := store.PreviewStart("work")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.ApplyStart(plan, plan.Digest, func(phase string) error {
		if phase == "baseline-recorded" {
			return errors.New("crash")
		}
		return nil
	}); err == nil {
		t.Fatal("the fixture did not interrupt")
	}
	gone := exitedPID(t)
	writeLockOwner(t, store, func(o *StartLockOwner) { o.PID, o.ProcessStart = gone, "1" })
	record, _, _ := store.GetStartRecord("work")
	dead, _ := store.StartLockState("work")
	if _, err := store.RecoverStartLock("work", dead.OwnerDigest); err != nil {
		t.Fatal(err)
	}
	if after, _, _ := store.GetStartRecord("work"); after.PlanDigest != record.PlanDigest || after.Phase != "baseline-recorded" {
		t.Fatalf("the recovery touched the start record: %+v", after)
	}
	resumed, err := store.ApplyStart(plan, plan.Digest, nil)
	if err != nil || resumed.Phase != "started" {
		t.Fatalf("the interrupted start did not resume: %+v %v", resumed, err)
	}
}
