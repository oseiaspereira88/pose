package mcpserver

import (
	"context"
	"encoding/json"
	"testing"
)

func TestMemoryCursorStore_AppendAssignsMonotonicIDs(t *testing.T) {
	store := newMemoryCursorStore()
	ctx := context.Background()
	first, err := store.Append(ctx, "s1", "ping", json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("Append: %v", err)
	}
	second, err := store.Append(ctx, "s1", "ping", json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("Append: %v", err)
	}
	if first.ID != 1 || second.ID != 2 {
		t.Fatalf("ids = %d, %d, want 1, 2", first.ID, second.ID)
	}
}

func TestMemoryCursorStore_SinceReturnsOnlyNewerEvents(t *testing.T) {
	store := newMemoryCursorStore()
	ctx := context.Background()
	for i := 0; i < 5; i++ {
		if _, err := store.Append(ctx, "s1", "ping", json.RawMessage(`{}`)); err != nil {
			t.Fatalf("Append: %v", err)
		}
	}
	missed, err := store.Since(ctx, "s1", 2)
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

func TestMemoryCursorStore_SinceUnknownSessionReturnsEmptyNotError(t *testing.T) {
	store := newMemoryCursorStore()
	missed, err := store.Since(context.Background(), "ghost", 0)
	if err != nil {
		t.Fatalf("Since: %v", err)
	}
	if len(missed) != 0 {
		t.Fatalf("len(missed) = %d, want 0", len(missed))
	}
}

func TestMemoryCursorStore_SessionsAreIndependent(t *testing.T) {
	store := newMemoryCursorStore()
	ctx := context.Background()
	if _, err := store.Append(ctx, "s1", "ping", json.RawMessage(`{}`)); err != nil {
		t.Fatal(err)
	}
	first, err := store.Append(ctx, "s2", "ping", json.RawMessage(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	if first.ID != 1 {
		t.Fatalf("s2's first event ID = %d, want 1 (independent sequence from s1)", first.ID)
	}
}

func TestMemoryCursorStore_EvictsOldestBeyondRetention(t *testing.T) {
	store := newMemoryCursorStore()
	ctx := context.Background()
	for i := 0; i < streamRetention+10; i++ {
		if _, err := store.Append(ctx, "s1", "ping", json.RawMessage(`{}`)); err != nil {
			t.Fatal(err)
		}
	}
	all, err := store.Since(ctx, "s1", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != streamRetention {
		t.Fatalf("len(all) = %d, want %d (retention bound)", len(all), streamRetention)
	}
	// The oldest streamRetention+10-streamRetention=10 events were evicted;
	// the surviving log starts right after them.
	if all[0].ID != 11 {
		t.Fatalf("all[0].ID = %d, want 11 (first 10 evicted)", all[0].ID)
	}
}
