package mcpserver

// Opaque cursor pagination for POSE list tools (spec
// pose-mcp-protocol-completeness R1). The domain layer already returns each
// list deterministically ordered (spec slug, roadmap slug, knowledge slug,
// or generated_at); the cursor is a versioned, base64-opaque position token
// over that fixed order — clients must never parse it. Cursor and limit are
// both optional: omitting them returns every item in one page, byte-for-byte
// what the tool returned before pagination existed (compatibility).

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/harne8/pose-mcp/internal/pose"
)

// Shared schema text for every paginated list tool (spec
// pose-mcp-protocol-completeness R1) — pinned by
// TestListToolsShareThePaginationSchema so the contract cannot drift tool by
// tool the way project_id once did.
const (
	sharedCursorDescription = "Opaque pagination cursor from a previous call's next_cursor; omit for the first page. Treat as opaque — do not parse or construct it. A cursor belongs to one query, project and collection and is signed: a changed collection, another query, an expired token or a server restart is refused with a cursor-* code and you restart from the first page; it is never continued over a changed collection."
	sharedLimitDescription  = "Optional page size; omit to return every item in one page (default, unpaginated — preserves the pre-pagination response shape). Large unfiltered result sets return a `notice` field nudging you toward a narrower filter or a smaller limit — check for it before assuming the response is complete."
)

// largeResultThreshold is the total-match count above which list responses
// attach a structured `notice` (spec pose-mcp-query-ergonomics R4). The goal
// is to surface the "this is a lot, consider filtering/paging" signal in the
// same response that would otherwise silently grow past a client's token
// budget — not only after the caller has already hit that wall once.
const largeResultThreshold = 20

// listNotice returns a hint string when total exceeds largeResultThreshold,
// or "" when the result is small enough that no nudge is warranted. hint
// names the tool-specific filter/paging levers available to narrow it.
func listNotice(total int, hint string) string {
	if total <= largeResultThreshold {
		return ""
	}
	return fmt.Sprintf("%d items matched. Response may be large — %s.", total, hint)
}

// pageKey returns the key that signs this server's cursors, created on first
// use. It lives only in the process: a cursor from another process, or from
// before a restart, is refused as stale and the caller restarts from the first
// page (spec pose-generation-bound-query-cursors R6).
func (s *Server) pageKey() []byte {
	s.pageKeyOnce.Do(func() {
		key, err := pose.NewCursorKey()
		if err != nil {
			// Without randomness no cursor can be trusted: an empty key still signs
			// consistently but a caller can no longer forge across processes.
			key = []byte("pose-cursor-key-unavailable")
		}
		s.pageKeyValue = key
	})
	return s.pageKeyValue
}

// pageList pages one ordered collection of a list tool: filters is the canonical
// string of the normalized filters and ordering, keys the total-order keys of
// items. The response carries the page's generation metadata (R4).
func pageList[T any](s *Server, store pose.Store, tool, filters string, items []T, keys []string, cursor string, limit int) ([]T, string, pose.GenerationPage, error) {
	generation := store.ReadIndexGeneration().Descriptor.GenerationID
	sum := sha256.Sum256([]byte(store.Root))
	project := store.FederatedProjectID + "|" + hex.EncodeToString(sum[:8])
	page, next, meta, err := pose.PageByGeneration(s.pageKey(), pose.GenerationQuery{Tool: tool, Project: project, Filters: filters}, generation, items, keys, cursor, limit, time.Now())
	if err != nil {
		return nil, "", meta, fmt.Errorf("%s: %w", tool, err)
	}
	return page, next, meta, nil
}

// contentKeys keys every item of an ordered collection by the digest of its JSON
// form. The domain layer orders each collection deterministically; a changed,
// inserted, removed or reordered item changes the collection digest.
func contentKeys[T any](items []T) []string {
	keys := make([]string, len(items))
	for i, item := range items {
		raw, _ := json.Marshal(item)
		sum := sha256.Sum256(raw)
		keys[i] = hex.EncodeToString(sum[:8])
	}
	return keys
}

// specKeys keys specs by slug, the unique total-order key of the listing.
func specKeys(specs []pose.Spec) []string {
	keys := make([]string, len(specs))
	for i, spec := range specs {
		sum := sha256.Sum256([]byte(spec.Slug + "\x00" + spec.Status + "\x00" + spec.CreatedAt))
		keys[i] = spec.Slug + ":" + hex.EncodeToString(sum[:4])
	}
	return keys
}
