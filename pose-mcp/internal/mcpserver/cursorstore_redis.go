package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

// redisCursorStore is the durable CursorStore backing (pose-mcp-enterprise-
// hardening: "cursor deve sobreviver a restart quando Redis estiver
// habilitado"), used across replicas: a client reconnecting with
// Last-Event-ID can land on any replica and still replay what it missed,
// because the log lives in Redis, not in one replica's process memory.
//
// A Sorted Set per session (member = JSON-encoded StreamEvent, score = its
// ID) fits the access pattern exactly: Append is O(log N), Since(afterID) is
// a native ZRANGEBYSCORE "(afterID +inf" — no client-side filtering needed.
type redisCursorStore struct {
	client *goredis.Client
	// ttl bounds how long an abandoned session's keys survive in Redis —
	// sessions don't have an explicit close signal today (handleSSE just
	// stops on client disconnect), so a TTL is the only cleanup mechanism.
	ttl time.Duration
}

// NewRedisCursorStore connects to addr (host:port, matching graphforge's
// REDIS_ADDR convention — the same Redis instance the dev/prod compose
// stacks already run, no new infra). Does not itself verify connectivity;
// the caller decides whether a failed Ping should be fatal or degrade to
// the in-memory default.
func NewRedisCursorStore(addr, password string, db int) *redisCursorStore {
	client := goredis.NewClient(&goredis.Options{Addr: addr, Password: password, DB: db})
	return &redisCursorStore{client: client, ttl: time.Hour}
}

func (r *redisCursorStore) Ping(ctx context.Context) error {
	return r.client.Ping(ctx).Err()
}

func (r *redisCursorStore) streamKey(sessionID string) string {
	return "pose-mcp:stream:" + sessionID
}

func (r *redisCursorStore) seqKey(sessionID string) string {
	return "pose-mcp:stream-seq:" + sessionID
}

func (r *redisCursorStore) Append(ctx context.Context, sessionID, name string, data json.RawMessage) (StreamEvent, error) {
	seqKey := r.seqKey(sessionID)
	id, err := r.client.Incr(ctx, seqKey).Result()
	if err != nil {
		return StreamEvent{}, fmt.Errorf("cursor store: incr seq: %w", err)
	}
	ev := StreamEvent{ID: id, Name: name, Data: data}
	member, err := json.Marshal(ev)
	if err != nil {
		return StreamEvent{}, fmt.Errorf("cursor store: marshal event: %w", err)
	}
	key := r.streamKey(sessionID)
	pipe := r.client.TxPipeline()
	pipe.ZAdd(ctx, key, goredis.Z{Score: float64(id), Member: member})
	// Keep only the most recent streamRetention entries — same bound as
	// memoryCursorStore, so switching backends doesn't change replay depth.
	pipe.ZRemRangeByRank(ctx, key, 0, -(streamRetention + 1))
	pipe.Expire(ctx, key, r.ttl)
	pipe.Expire(ctx, seqKey, r.ttl)
	if _, err := pipe.Exec(ctx); err != nil {
		return StreamEvent{}, fmt.Errorf("cursor store: append: %w", err)
	}
	return ev, nil
}

func (r *redisCursorStore) Since(ctx context.Context, sessionID string, afterID int64) ([]StreamEvent, error) {
	members, err := r.client.ZRangeByScore(ctx, r.streamKey(sessionID), &goredis.ZRangeBy{
		Min: fmt.Sprintf("(%d", afterID),
		Max: "+inf",
	}).Result()
	if err != nil {
		return nil, fmt.Errorf("cursor store: since: %w", err)
	}
	out := make([]StreamEvent, 0, len(members))
	for _, m := range members {
		var ev StreamEvent
		if err := json.Unmarshal([]byte(m), &ev); err != nil {
			continue // a malformed entry must not sink the whole replay
		}
		out = append(out, ev)
	}
	return out, nil
}
