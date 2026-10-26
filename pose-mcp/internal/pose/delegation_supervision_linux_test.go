package pose

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// writeActiveMarker writes the marker of a run whose process is this one, or,
// when dead, one whose recorded start no longer matches it.
func writeActiveMarker(t *testing.T, s Store, attempt string, heartbeat time.Time, dead bool) {
	t.Helper()
	dir, err := s.delegationActiveDir(true)
	if err != nil {
		t.Fatal(err)
	}
	owner := StartLockOwner{SchemaVersion: StartLockOwnerSchemaVersion, Resource: "delegate:" + attempt, Worktree: s.Root, PID: os.Getpid()}
	owner.Host, _ = os.Hostname()
	identifyStartLockProcess(&owner)
	if dead {
		owner.ProcessStart = "1"
	}
	marker := DelegationActiveRun{SchemaVersion: 1, AttemptID: attempt, RunToken: hex.EncodeToString([]byte("synthetic-run-16")), Purpose: "review", Agent: "a", Scope: "spec:x", BundleID: "rvb-1",
		StartedAt: heartbeat.UTC().Format(time.RFC3339Nano), Owner: owner, HeartbeatAt: heartbeat.UTC().Format(time.RFC3339Nano), HeartbeatEvery: 10}
	raw, _ := json.Marshal(marker)
	if err := os.WriteFile(filepath.Join(dir, attempt+".json"), raw, 0o600); err != nil {
		t.Fatal(err)
	}
}

// R4, D2: a run is stalled only when its heartbeat is older than three
// intervals and its process is gone; concurrent readers agree on the record.
func TestDelegationStallNeedsSilenceAndADeadOwner(t *testing.T) {
	s := Store{Root: t.TempDir()}
	old := time.Now().Add(-time.Minute)
	writeActiveMarker(t, s, "att-silent-alive", old, false)
	writeActiveMarker(t, s, "att-recent-dead", time.Now(), true)
	writeActiveMarker(t, s, "att-silent-dead", old, true)
	instants := make([]string, 8)
	var wg sync.WaitGroup
	for i := range instants {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			active, _, err := s.ListDelegationActive()
			if err != nil {
				t.Error(err)
				return
			}
			for _, m := range active {
				switch m.AttemptID {
				case "att-silent-dead":
					instants[i] = m.StalledAt
				default:
					if m.StalledAt != "" {
						t.Errorf("%s recorded stalled", m.AttemptID)
					}
				}
			}
		}(i)
	}
	wg.Wait()
	for _, at := range instants {
		if at == "" || at != instants[0] {
			t.Fatalf("readers disagree on the stall: %v", instants)
		}
	}
	attempts, _, err := s.ListDelegationAttempts()
	if err != nil {
		t.Fatal(err)
	}
	for _, a := range attempts {
		if a.AttemptID == "att-silent-dead" {
			if a.State != DelegationStalled || a.Active {
				t.Fatalf("stalled attempt listed as %+v", a)
			}
			if result, terminal := DelegationWaitOutcome(a); !terminal || result.Code != DelegationWaitStalled {
				t.Fatalf("wait outcome %+v %t", result, terminal)
			}
		}
	}
}

// R4: the process running an attempt beats at its declared interval.
func TestDelegationMarkerBeats(t *testing.T) {
	s := Store{Root: t.TempDir()}
	saved := delegationHeartbeatEvery
	delegationHeartbeatEvery = 50 * time.Millisecond
	defer func() { delegationHeartbeatEvery = saved }()
	_, cancel := context.WithCancel(context.Background())
	stop, _, err := s.markDelegationActive(activeRun("att-beat"), cancel)
	if err != nil {
		t.Fatal(err)
	}
	defer stop()
	first, _, _ := s.ListDelegationActive()
	time.Sleep(300 * time.Millisecond)
	later, _, _ := s.ListDelegationActive()
	if len(first) != 1 || len(later) != 1 || later[0].HeartbeatAt <= first[0].HeartbeatAt || later[0].StalledAt != "" {
		t.Fatalf("no heartbeat: %+v then %+v", first, later)
	}
}
