package mcpserver

// Opaque cursor pagination for POSE list tools (spec
// pose-mcp-protocol-completeness R1). The domain layer already returns each
// list deterministically ordered (spec slug, roadmap slug, knowledge slug,
// or generated_at); the cursor is a versioned, base64-opaque position token
// over that fixed order — clients must never parse it. Cursor and limit are
// both optional: omitting them returns every item in one page, byte-for-byte
// what the tool returned before pagination existed (compatibility).

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
)

const pageCursorVersion = 1

// Shared schema text for every paginated list tool (spec
// pose-mcp-protocol-completeness R1) — pinned by
// TestListToolsShareThePaginationSchema so the contract cannot drift tool by
// tool the way project_id once did.
const (
	sharedCursorDescription = "Opaque pagination cursor from a previous call's next_cursor; omit for the first page. Treat as opaque — do not parse or construct it."
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

type pageCursor struct {
	V     int `json:"v"`
	After int `json:"after"`
}

func encodePageCursor(after int) string {
	b, _ := json.Marshal(pageCursor{V: pageCursorVersion, After: after})
	return base64.RawURLEncoding.EncodeToString(b)
}

// decodePageCursor returns the starting offset for an opaque cursor. An
// empty token is offset 0 (first page); a malformed or wrong-version token
// is a client error, never silently coerced to 0 (that would skip items
// unpredictably in a way a client can't detect).
func decodePageCursor(token string) (int, error) {
	if token == "" {
		return 0, nil
	}
	raw, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return 0, fmt.Errorf("invalid cursor")
	}
	var c pageCursor
	if err := json.Unmarshal(raw, &c); err != nil || c.V != pageCursorVersion || c.After < 0 {
		return 0, fmt.Errorf("invalid cursor")
	}
	return c.After, nil
}

// paginatePage slices a deterministically ordered slice starting at the
// cursor's offset. limit<=0 means "no explicit page size" — the historical,
// still-supported unpaginated call — and returns everything from the
// cursor with no next_cursor. next is "" once the slice is exhausted.
func paginatePage[T any](items []T, after, limit int) (page []T, next string) {
	if after > len(items) {
		after = len(items)
	}
	rest := items[after:]
	if limit <= 0 || limit >= len(rest) {
		return rest, ""
	}
	return rest[:limit], encodePageCursor(after + limit)
}
