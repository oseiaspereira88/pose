package pose

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

// R1: a heartbeat held between its read and its publication while the run
// stops cannot leave the marker behind, and no further heartbeat starts once
// stop has asked, whichever case the watcher's select picks (review round 1,
// HB01): a stop waits for at most one write. R2: a stop after a cancel
// request ended the watcher returns (spec
// pose-delegation-marker-stop-joins-heartbeat).
func TestAStoppedRunsHeartbeatCannotBringItsMarkerBack(t *testing.T) {
	savedEvery, savedHook, savedSync, savedStop := delegationHeartbeatEvery, PublishBeforeReplace, PublishSyncHook, delegationStopRequested
	t.Cleanup(func() {
		delegationHeartbeatEvery, PublishBeforeReplace, PublishSyncHook, delegationStopRequested = savedEvery, savedHook, savedSync, savedStop
	})
	delegationHeartbeatEvery = time.Millisecond
	s := Store{Root: t.TempDir()}
	for trial := 0; trial < 30; trial++ {
		attempt := fmt.Sprintf("att-held-%d", trial)
		marker := attempt + ".json"
		entered, proceed, requested := make(chan struct{}), make(chan struct{}), make(chan struct{})
		var writes atomic.Int32
		PublishBeforeReplace = func(path string) {
			if filepath.Base(path) != marker {
				return
			}
			if writes.Add(1) == 1 {
				close(entered)
				<-proceed
			}
		}
		delegationStopRequested = func() { close(requested) }
		// The held heartbeat's rename is over when it syncs its directory.
		renamed := make(chan struct{}, 1)
		PublishSyncHook = func(kind, path string) error {
			if kind == "dir" && filepath.Base(path) == marker && writes.Load() > 0 {
				select {
				case renamed <- struct{}{}:
				default:
				}
			}
			return nil
		}
		stop, _, err := s.markDelegationActive(activeRun(attempt), func() {})
		if err != nil {
			t.Fatal(err)
		}
		stopped := make(chan struct{})
		release := func() {
			select {
			case <-proceed:
			default:
				close(proceed)
			}
		}
		select {
		case <-entered:
		case <-time.After(10 * time.Second):
			t.Fatal("no heartbeat started")
		}
		go func() { stop(); close(stopped) }()
		select {
		case <-requested:
		case <-time.After(10 * time.Second):
			release()
			t.Fatal("stop did not ask the watcher to end")
		}
		// stop has asked; the heartbeat ticker is ready again by now, and
		// the held write ends only after that.
		time.Sleep(3 * time.Millisecond)
		release()
		select {
		case <-renamed:
		case <-time.After(10 * time.Second):
			t.Fatal("the held heartbeat did not finish")
		}
		select {
		case <-stopped:
		case <-time.After(10 * time.Second):
			t.Fatal("stop did not return")
		}
		if n := writes.Load(); n != 1 {
			t.Fatalf("trial %d: %d heartbeats wrote during stop, want 1", trial, n)
		}
		dir, err := s.delegationActiveDir(false)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := os.Stat(filepath.Join(dir, marker)); !os.IsNotExist(err) {
			t.Fatalf("trial %d: the stopped run's marker is back: %v", trial, err)
		}
	}

	// R2: a cancel request ends the watcher before stop.
	PublishBeforeReplace, PublishSyncHook = nil, nil
	delegationStopRequested = func() {}
	ctx, cancel := context.WithCancel(context.Background())
	stop, cancelled, err := s.markDelegationActive(activeRun("att-cancel"), cancel)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.RequestDelegationCancel("att-cancel"); err != nil {
		t.Fatal(err)
	}
	select {
	case <-ctx.Done():
	case <-time.After(10 * time.Second):
		t.Fatal("the cancel request was not seen")
	}
	done := make(chan struct{})
	go func() { stop(); close(done) }()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("stop after a cancel blocked")
	}
	if !cancelled.Load() {
		t.Fatal("the run was not recorded cancelled")
	}
}
