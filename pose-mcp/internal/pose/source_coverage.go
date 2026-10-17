package pose

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// What a source read lost (spec pose-source-coverage-integrity, ADR 2026-10-17 "A
// source that lost something is incomplete for what depends on it").
//
// A reader that skips an unreadable file, a malformed record or a line its scanner
// cannot hold reports "nothing found" for the part it lost, and every consumer takes
// that as "nothing is owed" or "nothing was decided". A SourceCoverage travels with
// the result of a read and says what the read could not see, for which producer,
// artifact and scope, so a consumer asks whether the part it depends on was read
// instead of whether the list is empty.

// Kinds of loss. A loss is something that could not be read; an absent artifact is
// not a loss and a declared exclusion is not one either (SourceExclusion).
const (
	// LossReadError: the artifact exists but its bytes could not be read.
	LossReadError = "read-error"
	// LossParseError: the bytes were read and are not the format the producer reads.
	LossParseError = "parse-error"
	// LossInvalid: an element was read and refused by validation, so it is not in
	// the result although it is in the source.
	LossInvalid = "invalid"
	// LossScanIncomplete: a scan stopped before the end of the artifact (a line its
	// scanner cannot hold, a scanner error).
	LossScanIncomplete = "scan-incomplete"
	// LossUnsupported: a format this engine does not read, as opposed to corruption.
	LossUnsupported = "unsupported-format"
)

// SourceLoss is one thing a read could not see.
type SourceLoss struct {
	Producer string `json:"producer"`
	Artifact string `json:"artifact"`
	// Scope is the spec slug (or scope ref) the loss belongs to, when that can be
	// shown; empty means it could not be attributed, and a read of any scope may
	// have lost something to it.
	Scope string `json:"scope,omitempty"`
	// AttributedBy says how Scope was found ("path", "header"); empty when Scope is.
	AttributedBy string `json:"attributed_by,omitempty"`
	Kind         string `json:"kind"`
	Detail       string `json:"detail,omitempty"`
	// At is when the lost record was made, when its name says so (RFC 3339, UTC), so a
	// reader can tell a record older than the one it selected from one that may have
	// superseded it. Empty when nothing can be said.
	At string `json:"at,omitempty"`
	// ID is the record's name without its extension, the key records of the same
	// instant are ordered by.
	ID string `json:"id,omitempty"`
}

// SourceExclusion is a part of a source that a read leaves out on purpose. A read
// with exclusions is complete for the slice it declares, not for the whole.
type SourceExclusion struct {
	Producer string `json:"producer"`
	Pattern  string `json:"pattern"`
	Reason   string `json:"reason"`
	// Count is how many artifacts the exclusion left out; Examples names a few.
	Count    int      `json:"count,omitempty"`
	Examples []string `json:"examples,omitempty"`
}

// addExclusion records one artifact left out under a named exclusion, grouping by
// pattern so a thousand generated files are one entry that says so.
func (c *SourceCoverage) addExclusion(producer, pattern, reason, artifact string) {
	for i := range c.Exclusions {
		e := &c.Exclusions[i]
		if e.Producer == producer && e.Pattern == pattern {
			e.Count++
			if len(e.Examples) < 3 {
				e.Examples = append(e.Examples, artifact)
			}
			return
		}
	}
	c.Exclusions = append(c.Exclusions, SourceExclusion{Producer: producer, Pattern: pattern, Reason: reason, Count: 1, Examples: []string{artifact}})
}

// SourceCoverage is what one read could and could not see.
type SourceCoverage struct {
	Losses     []SourceLoss      `json:"losses"`
	Exclusions []SourceExclusion `json:"exclusions,omitempty"`
}

// Complete reports that nothing was lost anywhere in the source.
func (c SourceCoverage) Complete() bool { return len(c.Losses) == 0 }

// CompleteFor reports that nothing that could belong to scope was lost: no loss is
// attributed to it, and none is unattributed. A loss attributed to another scope does
// not count. An empty scope asks about the whole source.
func (c SourceCoverage) CompleteFor(scope string) bool {
	return len(c.LossesFor(scope)) == 0
}

// LossesFor lists the losses that can affect an answer about scope: those attributed
// to it and those that could not be attributed to anything. An empty scope asks about
// the whole source and so sees every loss.
func (c SourceCoverage) LossesFor(scope string) []SourceLoss {
	if scope == "" {
		return append([]SourceLoss{}, c.Losses...)
	}
	out := []SourceLoss{}
	for _, loss := range c.Losses {
		if loss.Scope == "" || loss.Scope == scope {
			out = append(out, loss)
		}
	}
	return out
}

// Add records a loss.
func (c *SourceCoverage) Add(loss SourceLoss) { c.Losses = append(c.Losses, loss) }

// Merge appends another coverage's losses and exclusions.
func (c *SourceCoverage) Merge(other SourceCoverage) {
	c.Losses = append(c.Losses, other.Losses...)
	c.Exclusions = append(c.Exclusions, other.Exclusions...)
}

// Normalize sorts the losses so the same read reports them in the same order, and
// makes the slices non-nil for JSON.
func (c *SourceCoverage) Normalize() {
	if c.Losses == nil {
		c.Losses = []SourceLoss{}
	}
	sort.SliceStable(c.Exclusions, func(i, j int) bool { return c.Exclusions[i].Pattern < c.Exclusions[j].Pattern })
	sort.SliceStable(c.Losses, func(i, j int) bool {
		a, b := c.Losses[i], c.Losses[j]
		if a.Artifact != b.Artifact {
			return a.Artifact < b.Artifact
		}
		return a.Kind < b.Kind
	})
}

// Describe is a one-line account of the losses a read of scope has, for places that
// carry text (a coverage detail, a limitation). It names no content: only where the
// read failed.
func (c SourceCoverage) Describe(scope string) string {
	losses := c.LossesFor(scope)
	if len(losses) == 0 {
		return ""
	}
	parts := make([]string, 0, len(losses))
	for i, loss := range losses {
		if i == 3 {
			parts = append(parts, fmt.Sprintf("and %d more", len(losses)-3))
			break
		}
		where := loss.Artifact
		if loss.Scope == "" {
			where += " (scope unknown)"
		}
		parts = append(parts, loss.Kind+" "+where)
	}
	return fmt.Sprintf("%d source(s) could not be read: %s", len(losses), strings.Join(parts, "; "))
}

var specFileDatePrefixRE = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}-`)

// specSlugFromPath is the slug a spec path names: the folder of a folder-layout
// spec, the file name without its date prefix for a flat one. It is a best effort used
// to attribute a loss to a scope when the header cannot be read; the frontmatter slug,
// when readable, wins.
func specSlugFromPath(path string) string {
	parts := strings.Split(strings.ReplaceAll(path, "\\", "/"), "/")
	base := parts[len(parts)-1]
	if base == "spec.md" && len(parts) >= 2 {
		return specFileDatePrefixRE.ReplaceAllString(parts[len(parts)-2], "")
	}
	return specFileDatePrefixRE.ReplaceAllString(strings.TrimSuffix(base, ".md"), "")
}

// sniffJSONString reads the readable prefix of a JSON record that may be truncated and
// returns the string at path (object keys from the top level), if the prefix has it
// exactly once. found is false when the prefix never reaches it and ambiguous is true
// when it appears more than once or the path crosses an array: a prefix that does not
// settle the answer attributes nothing. A key elsewhere in the record (a nested
// "bundle_id" in some metadata) is not at the path and proves nothing.
func sniffJSONString(raw []byte, path ...string) (value string, found, ambiguous bool) {
	type frame struct {
		object    bool
		key       string
		expectKey bool
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	var stack []frame
	at := func() bool {
		if len(stack) != len(path) {
			return false
		}
		for i, f := range stack {
			if !f.object || f.key != path[i] {
				return false
			}
		}
		return true
	}
	valueDone := func() {
		if n := len(stack); n > 0 && stack[n-1].object {
			stack[n-1].expectKey = true
		}
	}
	for {
		tok, err := dec.Token()
		if err != nil {
			break
		}
		switch t := tok.(type) {
		case json.Delim:
			switch t {
			case '{':
				stack = append(stack, frame{object: true, expectKey: true})
			case '[':
				stack = append(stack, frame{})
			default:
				stack = stack[:len(stack)-1]
				valueDone()
			}
		case string:
			if n := len(stack); n > 0 && stack[n-1].object && stack[n-1].expectKey {
				stack[n-1].key, stack[n-1].expectKey = t, false
				continue
			}
			// A string value. The path names the keys of the objects above it, and the
			// last key is the one this value belongs to.
			if len(stack) == len(path) && at() {
				if found {
					return "", false, true
				}
				value, found = t, true
			}
			valueDone()
		default:
			valueDone()
		}
	}
	return value, found, false
}
