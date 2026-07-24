package pose

// Docs review-pending marks (spec pose-docs-assessment-followups): an
// append-only JSONL log at .pose/docs-review.jsonl, replayed to compute
// each doc's currently pending review triggers — same "marked, replayed
// to compute current state, never mutated in place" shape as the
// capability assessment's stale triggers, just JSONL instead of markdown
// bullets (this spec's own Restrições: a mark never touches the doc
// file itself).

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func (s Store) DocsReviewPath() string { return filepath.Join(s.Root, ".pose", "docs-review.jsonl") }

// DocsReviewTrigger is one pending "this doc may need review" demand.
type DocsReviewTrigger struct {
	Since   string   `json:"since"`
	Trigger string   `json:"trigger"`
	Hits    []string `json:"hits,omitempty"`
}

// DocsReviewEvent is one append-only JSONL line: either a mark ("marked")
// or a closure ("resolved") — the whole review lifecycle of one doc is
// the replay of these events, never an in-place edit.
type DocsReviewEvent struct {
	At      string   `json:"at"`
	Doc     string   `json:"doc"`
	Kind    string   `json:"kind"` // "marked" | "resolved"
	Trigger string   `json:"trigger,omitempty"`
	Hits    []string `json:"hits,omitempty"`
	Outcome string   `json:"outcome,omitempty"` // resolved only: "updated" | "no_change_needed"
	Commit  string   `json:"commit,omitempty"`  // resolved+updated only
	Reason  string   `json:"reason,omitempty"`  // resolved+no_change_needed (required), or a manual mark's rationale
}

// LoadDocsReviewEvents reads the append-only log. A missing file is not
// an error (no marks yet) — same posture as capability history.jsonl.
func LoadDocsReviewEvents(path string) ([]DocsReviewEvent, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	trimmed := strings.TrimRight(string(raw), "\n")
	if trimmed == "" {
		return nil, nil
	}
	lines := strings.Split(trimmed, "\n")
	events := make([]DocsReviewEvent, 0, len(lines))
	for i, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		var ev DocsReviewEvent
		if err := json.Unmarshal([]byte(line), &ev); err != nil {
			return nil, fmt.Errorf("pose: docs-review.jsonl line %d: %w", i+1, err)
		}
		events = append(events, ev)
	}
	return events, nil
}

// PendingDocsReviews replays events and returns, per doc, its currently
// open triggers. A "resolved" event clears every trigger pending for
// that doc at once — resolving is the deliberate act that closes
// everything currently open, same decision already made for `pose assess
// snapshot` clearing capability stale marks.
func PendingDocsReviews(events []DocsReviewEvent) map[string][]DocsReviewTrigger {
	pending := map[string][]DocsReviewTrigger{}
	for _, ev := range events {
		switch ev.Kind {
		case "marked":
			pending[ev.Doc] = appendUniqueDocsReviewTrigger(pending[ev.Doc], DocsReviewTrigger{
				Since: ev.At, Trigger: ev.Trigger, Hits: ev.Hits,
			})
		case "resolved":
			delete(pending, ev.Doc)
		}
	}
	return pending
}

// appendUniqueDocsReviewTrigger replaces an existing trigger with the
// same Trigger text in place rather than accumulating a duplicate (R3
// dedup: the same gatilho never opens a second mark on an already-marked
// doc).
func appendUniqueDocsReviewTrigger(triggers []DocsReviewTrigger, next DocsReviewTrigger) []DocsReviewTrigger {
	for i, existing := range triggers {
		if existing.Trigger == next.Trigger {
			triggers[i] = next
			return triggers
		}
	}
	return append(triggers, next)
}

// MatchesOwns reports whether file falls under one of owns's declared
// areas: an exact match, a directory prefix ("site" covers
// "site/README.md"), or a glob pattern — broader than the capability
// mechanism's pure-glob Paths because "owns:" is described by its own
// spec as "áreas de código/componentes", where whole-directory ownership
// is the common case, not the exception.
func MatchesOwns(file string, owns []string) bool {
	for _, o := range owns {
		o = strings.TrimSuffix(o, "/")
		if o == "" {
			continue
		}
		if file == o || strings.HasPrefix(file, o+"/") {
			return true
		}
		if matched, _ := filepath.Match(o, file); matched {
			return true
		}
	}
	return false
}
