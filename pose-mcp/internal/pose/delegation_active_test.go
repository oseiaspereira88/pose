package pose

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func activeRun(attempt string) ReviewRun {
	return ReviewRun{AttemptID: attempt, Purpose: "review", Adapter: "a", Scope: "spec:x", BundleID: "rvb-1", StartedAt: time.Now().UTC().Format(time.RFC3339Nano)}
}

func waitFor(cond func() bool) bool {
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return true
		}
		time.Sleep(50 * time.Millisecond)
	}
	return cond()
}

// B04 of the first review: a cancellation names one run of an attempt. A
// request meant for an earlier run never stops its successor, a refused
// second marker never erases a pending request, and a request for a run that
// ended is withdrawn.
func TestDelegationCancelNamesOneRun(t *testing.T) {
	s := Store{Root: t.TempDir()}
	_, cancelA := context.WithCancel(context.Background())
	stopA, _, err := s.markDelegationActive(activeRun("att-1"), cancelA)
	if err != nil {
		t.Fatal(err)
	}
	active, _, _ := s.ListDelegationActive()
	if len(active) != 1 || !validRunToken(active[0].RunToken) {
		t.Fatalf("marker: %+v", active)
	}
	markerA := active[0]
	stopA()

	ctxB, cancelB := context.WithCancel(context.Background())
	stopB, cancelledB, err := s.markDelegationActive(activeRun("att-1"), cancelB)
	if err != nil {
		t.Fatal(err)
	}
	defer stopB()
	dir, _ := s.delegationActiveDir(false)
	if err := os.WriteFile(cancelRequestPath(dir, markerA), []byte("late\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	time.Sleep(600 * time.Millisecond)
	if cancelledB.Load() || ctxB.Err() != nil {
		t.Fatal("a request meant for the earlier run stopped its successor")
	}
	if _, err := s.RequestDelegationCancel("att-1"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.markDelegationActive(activeRun("att-1"), func() {}); err == nil {
		t.Fatal("a second marker of a running attempt was accepted")
	}
	if !waitFor(func() bool { return cancelledB.Load() && ctxB.Err() != nil }) {
		t.Fatal("a refused second marker erased the pending request, or the request was not honoured")
	}

	stopB()
	if _, err := s.RequestDelegationCancel("att-1"); !errors.Is(err, ErrDelegationNotActive) {
		t.Fatalf("a run that ended was cancelled: %v", err)
	}
	// Only the stale request this test wrote by hand is left: the marker and
	// every request the engine wrote for the runs are gone.
	entries, _ := os.ReadDir(dir)
	stale := filepath.Base(cancelRequestPath(dir, markerA))
	for _, e := range entries {
		if e.Name() != stale {
			t.Fatalf("left in the active directory: %s", e.Name())
		}
	}
}

// N04 of the third review: a marker the listing cannot read is a loss, so ls
// never reports no running run when it could not observe one.
func TestDelegationActiveReportsUnreadableMarkers(t *testing.T) {
	s := Store{Root: t.TempDir()}
	_, cancel := context.WithCancel(context.Background())
	stop, _, err := s.markDelegationActive(activeRun("att-1"), cancel)
	if err != nil {
		t.Fatal(err)
	}
	defer stop()
	dir, err := s.delegationActiveDir(false)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "att-1.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, raw[:len(raw)/2], 0600); err != nil {
		t.Fatal(err)
	}
	attempts, lost, err := s.ListDelegationAttempts()
	if err != nil {
		t.Fatal(err)
	}
	if len(attempts) != 0 || len(lost) != 1 || !strings.Contains(lost[0], "att-1.json") {
		t.Fatalf("an unreadable marker went unreported: attempts=%+v lost=%v", attempts, lost)
	}
}
