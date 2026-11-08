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
	"os"
	"path/filepath"
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
	after := s.pageGeneration(store)
	if after != before {
		return nil, "", pose.GenerationPage{}, fmt.Errorf("%s: the index generation moved while the collection was read; retry the call", tool)
	}
	sum := sha256.Sum256([]byte(store.Root))
	project := store.FederatedProjectID + "|" + hex.EncodeToString(sum[:8])
	page, next, meta, err := pose.PageByGeneration(s.pageKey(), pose.GenerationQuery{Tool: tool, Project: project, Filters: filters}, after.id, items, keys, cursor, limit, time.Now())
	meta.GenerationStatus, meta.GenerationReason = after.status, after.reason
	if after.diverged != "" {
		meta.GenerationDiverged = strings.Split(after.diverged, ",")
	}
	if err != nil {
		return nil, "", meta, fmt.Errorf("%s: %w", tool, err)
	}
	return page, next, meta, nil
}

// pageGen is the index generation a collection was read under, with the status
// the index reader reported (an unavailable or diverged generation is disclosed,
// not hidden behind a successful page).
type pageGen struct{ id, status, reason, diverged string }

func (s *Server) pageGeneration(store pose.Store) pageGen {
	sig := generationSignature(store.Root)
	s.pageGenMu.Lock()
	defer s.pageGenMu.Unlock()
	if entry, ok := s.pageGenMemo[store.Root]; ok && sig != "" && entry.sig == sig {
		return entry.gen
	}
	view := store.ReadIndexGeneration()
	diverged := append([]string(nil), view.Diverged...)
	sort.Strings(diverged)
	gen := pageGen{id: view.Descriptor.GenerationID, status: string(view.Status), reason: view.Reason, diverged: strings.Join(diverged, ",")}
	if s.pageGenMemo == nil {
		s.pageGenMemo = map[string]pageGenEntry{}
	}
	// A generation is verified once and reused while its descriptor and the files it
	// names (and their legacy counterparts) are unchanged: verifying it reads and
	// hashes every index file, which a page must not repeat.
	s.pageGenMemo[store.Root] = pageGenEntry{sig: sig, gen: gen}
	return gen
}

type pageGenEntry struct {
	sig string
	gen pageGen
}

// generationSignature identifies the state of the index set by what is cheap to
// read: the descriptor's bytes and the size, mode and modification time of the
// files it names and of their legacy counterparts. Empty when it cannot be built,
// which means the generation is verified in full.
func generationSignature(root string) string {
	dir := filepath.Join(root, ".pose", "indexes")
	raw, err := os.ReadFile(filepath.Join(dir, "generation.json"))
	if err != nil {
		return ""
	}
	var desc struct {
		GenerationID string `json:"generation_id"`
		Files        []struct {
			Name string `json:"name"`
		} `json:"files"`
	}
	if json.Unmarshal(raw, &desc) != nil || desc.GenerationID == "" {
		return ""
	}
	sum := sha256.Sum256(raw)
	var b strings.Builder
	b.WriteString(hex.EncodeToString(sum[:8]))
	stat := func(path string) {
		info, err := os.Lstat(path)
		if err != nil {
			b.WriteString("|absent")
			return
		}
		fmt.Fprintf(&b, "|%d:%o:%d", info.Size(), info.Mode(), info.ModTime().UnixNano())
		// The reader follows links: a link's target string and the target's own
		// metadata are what the verification saw.
		if info.Mode()&os.ModeSymlink != 0 {
			target, _ := os.Readlink(path)
			fmt.Fprintf(&b, ">%s", target)
			if resolved, err := os.Stat(path); err == nil {
				fmt.Fprintf(&b, ":%d:%o:%d", resolved.Size(), resolved.Mode(), resolved.ModTime().UnixNano())
			} else {
				b.WriteString(":dangling")
			}
		}
	}
	for _, f := range desc.Files {
		stat(filepath.Join(dir, "generations", desc.GenerationID, f.Name))
		stat(filepath.Join(dir, f.Name))
	}
	return b.String()
}

// requirePositiveLimit rejects an explicit null, zero or negative limit: the call
// that returns everything is the call that names neither cursor nor limit, and a
// supplied value never reaches that branch (spec pose-generation-bound-query-cursors R6, R7).
func requirePositiveLimit(tool string, args json.RawMessage) error {
	var present map[string]json.RawMessage
	_ = json.Unmarshal(args, &present)
	// The decoder matches argument names case-insensitively: a differently-cased
	// alias would be the value the tool reads while the check looked at another.
	for key := range present {
		for _, name := range []string{"limit", "cursor"} {
			if key != name && strings.EqualFold(key, name) {
				return fmt.Errorf("%s: argument %q is ambiguous; write it as %q", tool, key, name)
			}
		}
	}
	raw, given := present["limit"]
	if !given {
		return nil
	}
	var n *int
	if err := json.Unmarshal(raw, &n); err != nil || n == nil || *n <= 0 {
		return fmt.Errorf("%s: limit must be a positive number; omit it to return everything", tool)
	}
	return nil
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
