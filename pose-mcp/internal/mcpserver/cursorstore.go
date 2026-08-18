package mcpserver

import (
	"context"
	"encoding/json"
	"sync"
)

// StreamEvent is a single resumable event on the GET /mcp SSE channel. ID is
// monotonic within a session — a client that reconnects with a Last-Event-ID
// header can ask for everything strictly after it.
type StreamEvent struct {
	ID   int64
	Name string
	Data json.RawMessage
}

// CursorStore persists a bounded, ordered log of recent stream events per
// session (pose-mcp-enterprise-hardening: "cursor durável" for horizontal
// scaling — a reconnect may land on a different replica than the one that
// sent the original events, so the log can't live only in that replica's
// process memory once there's more than one).
//
// The default (memoryCursorStore, wired by New/NewWithRoots with no further
// configuration) is process-local and lost on restart or across replicas —
// correct and sufficient for the single-node dev mode the spec's Restrições
// explicitly protect ("não tornar Redis obrigatório em modo single-node
// dev"). WithCursorStore swaps in a durable implementation (redisCursorStore)
// only when the operator configures one.
type CursorStore interface {
	// Append adds an event to session's log, assigns it the next ID in that
	// session's sequence, and evicts the oldest entry once the log exceeds
	// its retention bound. Returns the recorded event.
	Append(ctx context.Context, sessionID, name string, data json.RawMessage) (StreamEvent, error)
	// Since returns buffered events with ID > afterID, oldest first. A
	// missing/expired/unknown session returns an empty slice, never an
	// error — the caller (handleSSE) treats "nothing to replay" as normal,
	// not a failure (the alternative, refusing the reconnect outright,
	// would be worse than silently resuming from "now").
	Since(ctx context.Context, sessionID string, afterID int64) ([]StreamEvent, error)
}

// streamRetention bounds memory/storage per session: pings fire every 15s
// (see handleSSE), so this covers several minutes of disconnection without
// unbounded growth for long-lived sessions.
const streamRetention = 64

// memoryCursorStore is the zero-configuration default: an in-process ring
// buffer per session, guarded by a single mutex (stream volume here is a
// handful of events per session per minute — nowhere near where per-session
// locking would matter).
type memoryCursorStore struct {
	mu        sync.Mutex
	bySession map[string][]StreamEvent
	nextID    map[string]int64
}

func newMemoryCursorStore() *memoryCursorStore {
	return &memoryCursorStore{bySession: make(map[string][]StreamEvent), nextID: make(map[string]int64)}
}

func (m *memoryCursorStore) Append(_ context.Context, sessionID, name string, data json.RawMessage) (StreamEvent, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.nextID[sessionID]++
	ev := StreamEvent{ID: m.nextID[sessionID], Name: name, Data: data}
	log := append(m.bySession[sessionID], ev)
	if len(log) > streamRetention {
		log = log[len(log)-streamRetention:]
	}
	m.bySession[sessionID] = log
	return ev, nil
}

func (m *memoryCursorStore) Since(_ context.Context, sessionID string, afterID int64) ([]StreamEvent, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	log := m.bySession[sessionID]
	out := make([]StreamEvent, 0, len(log))
	for _, ev := range log {
		if ev.ID > afterID {
			out = append(out, ev)
		}
	}
	return out, nil
}
