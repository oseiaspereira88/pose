package pose

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// Generation-bound query cursors (spec pose-generation-bound-query-cursors, ADR
// generation-bound-query-cursors). A cursor continues one query over one
// collection: it carries the tool, the project binding, the normalized filters,
// the digest of the ordered collection (with the index generation it was read
// from) and the position, and it is signed. A continuation recomputes the
// collection and refuses the cursor when the collection is no longer the one it
// was issued for; it never continues over a changed collection and never
// restarts silently at the first page. No resident state is kept.

const (
	GenerationCursorVersion  = 2
	GenerationCursorMaxBytes = 2048
	GenerationCursorTTL      = 15 * time.Minute
	GenerationPageMax        = 500
)

// Cursor refusal codes. Each one tells the caller what to do: restart from the
// first page (stale, expired, signature, unsupported) or correct the call.
const (
	CursorMalformed          = "cursor-malformed"
	CursorUnsupportedVersion = "cursor-unsupported-version"
	CursorSignature          = "cursor-signature"
	CursorBinding            = "cursor-binding"
	CursorExpired            = "cursor-expired"
	CursorStale              = "cursor-stale"
	CursorPosition           = "cursor-position"
)

// CursorError is a refused cursor with its code.
type CursorError struct {
	Code    string
	Message string
}

func (e *CursorError) Error() string { return e.Code + ": " + e.Message }

// CursorErrorCode returns the code of a refused cursor, or "".
func CursorErrorCode(err error) string {
	var c *CursorError
	if errors.As(err, &c) {
		return c.Code
	}
	return ""
}

// GenerationQuery names the query a cursor is bound to. Filters is a canonical
// string of the normalized filters and ordering.
type GenerationQuery struct {
	Tool    string
	Project string
	Filters string
}

// GenerationPage discloses what a page was read from. A page is a navigation
// view of a collection read now: it neither certifies the whole history nor
// authorizes a gate or an effect, which re-read current inputs.
type GenerationPage struct {
	Generation       string `json:"generation"`
	GenerationStatus string `json:"generation_status,omitempty"`
	Collection       string `json:"collection"`
	Position         int    `json:"position"`
	Total            int    `json:"total"`
	LimitApplied     int    `json:"limit_applied,omitempty"`
	Cost             string `json:"cost"`
	Disclosure       string `json:"disclosure"`
}

// GenerationPageCost states what a page costs: these list tools load their whole
// collection, so every page recomputes it and its exact total; the page bound
// limits what is returned, not what is read.
const GenerationPageCost = "full-scan: every page recomputes the whole collection and its exact total; limit bounds the page returned, not the work done"

const generationPageDisclosure = "page of a collection read for this call: an absent next_cursor means this collection was exhausted, not that nothing is pending elsewhere, and the page authorizes no gate or effect (those read current inputs)"

type cursorPayload struct {
	V    int    `json:"v"`
	Tool string `json:"tool"`
	// Bind is the digest of the project binding and the canonical filters, so the
	// token stays small whatever the filters are.
	Bind       string `json:"bind"`
	Collection string `json:"collection"`
	After      int    `json:"after"`
	AfterKey   string `json:"after_key"`
	Expires    int64  `json:"exp"`
}

// NewCursorKey returns a fresh signing key. A key that lives only in a server
// process makes a cursor stale when the process restarts: the caller restarts
// from the first page, and a one-shot CLI needs no key at all.
func NewCursorKey() ([]byte, error) {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return nil, err
	}
	return key, nil
}

func queryBinding(q GenerationQuery) string {
	sum := sha256.Sum256([]byte(q.Project + "\x00" + q.Filters))
	return hex.EncodeToString(sum[:16])
}

func sign(key, payload []byte) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write(payload)
	return mac.Sum(nil)
}

func sealCursor(key []byte, p cursorPayload) string {
	raw, _ := json.Marshal(p)
	return base64.RawURLEncoding.EncodeToString(raw) + "." + base64.RawURLEncoding.EncodeToString(sign(key, raw))
}

func openCursor(key []byte, token string, now time.Time) (cursorPayload, error) {
	var p cursorPayload
	if len(token) > GenerationCursorMaxBytes {
		return p, &CursorError{CursorMalformed, "the cursor exceeds the size bound"}
	}
	body, mac, ok := strings.Cut(token, ".")
	if !ok {
		// A version 1 token is an ordinal over a collection nobody identified: it is
		// not promoted to a generation.
		if raw, err := base64.RawURLEncoding.DecodeString(token); err == nil {
			var legacy struct {
				V int `json:"v"`
			}
			if json.Unmarshal(raw, &legacy) == nil && legacy.V == 1 {
				return p, &CursorError{CursorUnsupportedVersion, "this cursor is from an older version and names no collection; restart from the first page"}
			}
		}
		return p, &CursorError{CursorMalformed, "the cursor is not a cursor of this server"}
	}
	raw, err1 := base64.RawURLEncoding.DecodeString(body)
	sig, err2 := base64.RawURLEncoding.DecodeString(mac)
	if err1 != nil || err2 != nil {
		return p, &CursorError{CursorMalformed, "the cursor is not a cursor of this server"}
	}
	if !hmac.Equal(sig, sign(key, raw)) {
		return p, &CursorError{CursorSignature, "the cursor was not issued by this server process (it was altered, or the server restarted); restart from the first page"}
	}
	if err := json.Unmarshal(raw, &p); err != nil {
		return p, &CursorError{CursorMalformed, "the cursor payload is unreadable"}
	}
	if p.V != GenerationCursorVersion {
		return p, &CursorError{CursorUnsupportedVersion, fmt.Sprintf("cursor version %d is not supported; restart from the first page", p.V)}
	}
	if now.Unix() > p.Expires {
		return p, &CursorError{CursorExpired, "the cursor expired; restart from the first page"}
	}
	return p, nil
}

// CollectionDigest identifies an ordered collection read from an index
// generation: the generation, then every key with its position.
func CollectionDigest(generation string, keys []string) string {
	h := sha256.New()
	h.Write([]byte(generation + "\n"))
	for i, k := range keys {
		fmt.Fprintf(h, "%d\x00%s\n", i, k)
	}
	return "sha256:" + hex.EncodeToString(h.Sum(nil))
}

// PageByGeneration returns one page of an ordered collection and the cursor to
// continue it. keys are the total-order keys of items, in order. With no cursor
// and limit<=0 it returns every item and no cursor (the call that predates
// pagination). A continuation is refused unless it was issued by this key for
// this query over this very collection.
func PageByGeneration[T any](key []byte, q GenerationQuery, generation string, items []T, keys []string, cursor string, limit int, now time.Time) ([]T, string, GenerationPage, error) {
	meta := GenerationPage{Generation: generation, Collection: CollectionDigest(generation, keys), Total: len(items), Cost: GenerationPageCost, Disclosure: generationPageDisclosure}
	if limit > GenerationPageMax {
		limit, meta.LimitApplied = GenerationPageMax, GenerationPageMax
	}
	after := 0
	if cursor != "" {
		p, err := openCursor(key, cursor, now)
		if err != nil {
			return nil, "", meta, err
		}
		if p.Tool != q.Tool || p.Bind != queryBinding(q) {
			return nil, "", meta, &CursorError{CursorBinding, "the cursor belongs to another query, project or filter set"}
		}
		if p.Collection != meta.Collection {
			return nil, "", meta, &CursorError{CursorStale, "the collection changed since the cursor was issued; restart from the first page"}
		}
		if p.After < 1 || p.After > len(items) || keys[p.After-1] != p.AfterKey {
			return nil, "", meta, &CursorError{CursorPosition, "the cursor position is not in this collection"}
		}
		after = p.After
	}
	meta.Position = after
	// Only a call that names neither a cursor nor a limit is the call that predates
	// pagination and returns everything; a continuation is always a bounded page.
	if cursor != "" && limit <= 0 {
		limit = GenerationPageMax
	}
	rest := items[after:]
	if limit <= 0 || limit >= len(rest) {
		return rest, "", meta, nil
	}
	end := after + limit
	next := sealCursor(key, cursorPayload{V: GenerationCursorVersion, Tool: q.Tool, Bind: queryBinding(q), Collection: meta.Collection, After: end, AfterKey: keys[end-1], Expires: now.Add(GenerationCursorTTL).Unix()})
	if len(next) > GenerationCursorMaxBytes {
		return nil, "", meta, &CursorError{CursorMalformed, "the cursor would exceed the size bound; narrow the query"}
	}
	return rest[:limit], next, meta, nil
}
