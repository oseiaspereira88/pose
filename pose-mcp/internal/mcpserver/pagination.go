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
	"sort"
	"strings"
	"time"

	"github.com/harne8/pose-mcp/internal/pose"
)

// Shared schema text for every paginated list tool (spec
// pose-mcp-protocol-completeness R1) — pinned by
// TestListToolsShareThePaginationSchema so the contract cannot drift tool by
// tool the way project_id once did.
const (
	sharedCursorDescription = "Opaque pagination cursor from a previous call's next_cursor; omit for the first page. Treat as opaque — do not parse or construct it. A cursor belongs to one query, project and collection and is signed: a changed collection, another query, an expired token or a server restart is refused with a cursor-* code and you restart from the first page; it is never continued over a changed collection."
	sharedLimitDescription  = "Optional page size (at most 500; a continuation with a cursor is always a bounded page); omit it together with the cursor to return every item in one page (default, unpaginated — preserves the pre-pagination response shape). Every page recomputes the whole collection and its exact total, so limit bounds the page returned, not the work done. Large unfiltered result sets return a `notice` field nudging you toward a narrower filter or a smaller limit — check for it before assuming the response is complete."
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
func pageList[T any](s *Server, store pose.Store, tool, filters string, before pageGen, items []T, keys []string, cursor string, limit int) ([]T, string, pose.GenerationPage, error) {
	// The generation is read before the collection (by the caller) and again now:
	// a publication that landed in between would label items read under one
	// generation with another, so the page is refused instead of mislabelled.
	after := pageGeneration(store)
	if after != before {
		return nil, "", pose.GenerationPage{}, fmt.Errorf("%s: the index generation moved while the collection was read; retry the call", tool)
	}
	sum := sha256.Sum256([]byte(store.Root))
	project := store.FederatedProjectID + "|" + hex.EncodeToString(sum[:8])
	page, next, meta, err := pose.PageByGeneration(s.pageKey(), pose.GenerationQuery{Tool: tool, Project: project, Filters: filters}, after.id, items, keys, cursor, limit, time.Now())
	meta.GenerationStatus = after.status
	if err != nil {
		return nil, "", meta, fmt.Errorf("%s: %w", tool, err)
	}
	return page, next, meta, nil
}

// pageGen is the index generation a collection was read under, with the status
// the index reader reported (an unavailable or diverged generation is disclosed,
// not hidden behind a successful page).
type pageGen struct{ id, status string }

func pageGeneration(store pose.Store) pageGen {
	view := store.ReadIndexGeneration()
	return pageGen{id: view.Descriptor.GenerationID, status: string(view.Status)}
}

// canonicalList normalizes a comma-separated filter: trimmed, lower-cased,
// de-duplicated and sorted, so equivalent filters bind the same cursor.
func canonicalList(value string) string {
	seen := map[string]bool{}
	var out []string
	for _, part := range strings.Split(value, ",") {
		part = strings.ToLower(strings.TrimSpace(part))
		if part != "" && !seen[part] {
			seen[part] = true
			out = append(out, part)
		}
	}
	sort.Strings(out)
	return strings.Join(out, ",")
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
