package mcpserver

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"
)

// redisTestAddr points at a real Redis for integration coverage. Set
// POSE_MCP_TEST_REDIS_ADDR (host:port) to run these; they skip otherwise —
// the same "skip, don't fake" pattern as any test needing infra this
// package can't spin up itself. Not a substitute for the (already-passing,
// infra-free) memoryCursorStore unit tests: this file exists specifically
// to prove the Redis-backed implementation satisfies the exact same
// CursorStore contract, including the parts memory can't demonstrate
// (surviving process restart, sharing state across two independent clients
// — standing in for "two replicas").
func redisTestAddr(t *testing.T) string {
	t.Helper()
	addr := os.Getenv("POSE_MCP_TEST_REDIS_ADDR")
	if addr == "" {
		t.Skip("POSE_MCP_TEST_REDIS_ADDR not set; skipping Redis-backed cursor store tests")
	}
	store := NewRedisCursorStore(addr, "", 0)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := store.Ping(ctx); err != nil {
		t.Skipf("Redis at %s not reachable: %v", addr, err)
	}
	return addr
}

// cleanupSession deletes a session's keys so repeated test runs against the
// same long-lived Redis instance never see a previous run's leftover
// events — the TTL alone isn't enough for that (it's an hour by default,
// happily longer than a fast local test loop).
func cleanupSession(t *testing.T, store *redisCursorStore, sessionID string) {
	t.Helper()
	t.Cleanup(func() {
		store.client.Del(context.Background(), store.streamKey(sessionID), store.seqKey(sessionID))
	})
}

func newTestRedisCursorStore(t *testing.T) *redisCursorStore {
	addr := redisTestAddr(t)
	store := NewRedisCursorStore(addr, "", 0)
	// Isolate each test: sessions are unique per t.Name(), so no explicit
	// flush is needed, but a short TTL keeps a failed run's keys from
	// accumulating across the whole test binary's lifetime.
	store.ttl = 30 * time.Second
	return store
}

func TestRedisCursorStore_AppendAssignsMonotonicIDs(t *testing.T) {
	store := newTestRedisCursorStore(t)
	ctx := context.Background()
	session := "redis-" + t.Name()
	cleanupSession(t, store, session)
	first, err := store.Append(ctx, session, "ping", json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("Append: %v", err)
	}
	second, err := store.Append(ctx, session, "ping", json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("Append: %v", err)
	}
	if first.ID != 1 || second.ID != 2 {
		t.Fatalf("ids = %d, %d, want 1, 2", first.ID, second.ID)
	}
}

func TestRedisCursorStore_SinceReturnsOnlyNewerEvents(t *testing.T) {
	store := newTestRedisCursorStore(t)
	ctx := context.Background()
	session := "redis-" + t.Name()
	cleanupSession(t, store, session)
	for i := 0; i < 5; i++ {
		if _, err := store.Append(ctx, session, "ping", json.RawMessage(`{}`)); err != nil {
			t.Fatalf("Append: %v", err)
		}
	}
	missed, err := store.Since(ctx, session, 2)
	if err != nil {
		t.Fatalf("Since: %v", err)
	}
	if len(missed) != 3 {
		t.Fatalf("len(missed) = %d, want 3", len(missed))
	}
	for i, ev := range missed {
		if ev.ID != int64(3+i) {
			t.Fatalf("missed[%d].ID = %d, want %d", i, ev.ID, 3+i)
		}
	}
}

// TestRedisCursorStore_SurvivesAcrossIndependentClients is the property
// that actually justifies Redis over the in-process default: two unrelated
// *redisCursorStore values (standing in for two replicas that never share
// memory) see the same log, because the log lives in Redis, not in either
// process.
func TestRedisCursorStore_SurvivesAcrossIndependentClients(t *testing.T) {
	addr := redisTestAddr(t)
	session := "redis-" + t.Name()
	ctx := context.Background()

	replicaA := NewRedisCursorStore(addr, "", 0)
	cleanupSession(t, replicaA, session)
	if _, err := replicaA.Append(ctx, session, "from-a", json.RawMessage(`{}`)); err != nil {
		t.Fatalf("Append via replica A: %v", err)
	}

	replicaB := NewRedisCursorStore(addr, "", 0)
	missed, err := replicaB.Since(ctx, session, 0)
	if err != nil {
		t.Fatalf("Since via replica B: %v", err)
	}
	if len(missed) != 1 || missed[0].Name != "from-a" {
		t.Fatalf("replica B saw %+v, want the event replica A appended", missed)
	}
}

func TestRedisCursorStore_EvictsOldestBeyondRetention(t *testing.T) {
	store := newTestRedisCursorStore(t)
	ctx := context.Background()
	session := "redis-" + t.Name()
	cleanupSession(t, store, session)
	for i := 0; i < streamRetention+10; i++ {
		if _, err := store.Append(ctx, session, "ping", json.RawMessage(`{}`)); err != nil {
			t.Fatal(err)
		}
	}
	all, err := store.Since(ctx, session, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != streamRetention {
		t.Fatalf("len(all) = %d, want %d (retention bound)", len(all), streamRetention)
	}
}
