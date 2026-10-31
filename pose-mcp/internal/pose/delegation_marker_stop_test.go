package pose

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// R1: a heartbeat held between its read and its publication while the run
// stops cannot leave the marker behind; R2: a stop after a cancel request
// ended the watcher returns (spec
// pose-delegation-marker-stop-joins-heartbeat).
func TestAStoppedRunsHeartbeatCannotBringItsMarkerBack(t *testing.T) {
	savedEvery, savedHook, savedSync := delegationHeartbeatEvery, PublishBeforeReplace, PublishSyncHook
	t.Cleanup(func() {
		delegationHeartbeatEvery, PublishBeforeReplace, PublishSyncHook = savedEvery, savedHook, savedSync
	})
	delegationHeartbeatEvery = 20 * time.Millisecond
	entered, proceed := make(chan struct{}), make(chan struct{})
	held := false
	// Hold the first heartbeat after its byte check, before its rename.
	PublishBeforeReplace = func(path string) {
		if filepath.Base(path) == "att-held.json" && !held {
			held = true
			close(entered)
			<-proceed
		}
	}
	// The held heartbeat's rename is over when it syncs its directory.
	renamed := make(chan struct{}, 1)
	PublishSyncHook = func(kind, path string) error {
		if kind == "dir" && filepath.Base(path) == "att-held.json" && held {
			select {
			case renamed <- struct{}{}:
			default:
			}
		}
		return nil
	}
	s := Store{Root: t.TempDir()}
	stop, _, err := s.markDelegationActive(activeRun("att-held"), func() {})
	if err != nil {
		t.Fatal(err)
	}
	select {
	case <-entered:
	case <-time.After(10 * time.Second):
		t.Fatal("no heartbeat started")
	}
	stopped := make(chan struct{})
	go func() { stop(); close(stopped) }()
	// The heartbeat finishes its write only after stop has begun.
	time.Sleep(100 * time.Millisecond)
	close(proceed)
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
	dir, err := s.delegationActiveDir(false)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "att-held.json")); !os.IsNotExist(err) {
		t.Fatalf("the stopped run's marker is back: %v", err)
	}

	// R2: a cancel request ends the watcher before stop.
	PublishBeforeReplace, PublishSyncHook = nil, nil
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
