package pose

// Accepted limits (spec pose-review-declared-threat-model): a spec declares
// the residual limits of a guarantee it makes, each with a stable id and why
// it is accepted. The review plan seals them; the brief lists them; a
// reviewer classifies a finding that falls inside one as within that limit,
// or disputes the limit as a finding of its own. A declared limit never
// hides a defect inside the declared scope: it is visible in the bundle, the
// brief and the attestation.

import (
	"fmt"
	"regexp"
	"strings"
)

// ReviewAcceptedLimit is one declared residual limit.
type ReviewAcceptedLimit struct {
	ID   string `json:"id"`
	Text string `json:"text"`
	// Spec names the spec that declared the limit when the scope holds more
	// than one; the id is then qualified as <spec>#L<N> (review round 1, N1).
	Spec string `json:"spec,omitempty"`
}

var acceptedLimitRE = regexp.MustCompile(`^[-*+]\s+(L[1-9][0-9]*):\s+(.+)$`)

// acceptedLimitMarkerRE finds the marker where a sentence opens, after the
// assumption: "<assumption>. Accepted because <reason>." A marker inside a
// sentence ("Not accepted because") is not one (review round 3, N1).
var acceptedLimitMarkerRE = regexp.MustCompile(`[.;:!?]\s+(?:Accepted because|Aceito porque)\s+`)

// acceptedLimitReasonWordRE is a word with at least one letter.
var acceptedLimitReasonWordRE = regexp.MustCompile(`\p{L}+`)

// hasAcceptedLimitReason reads an entry as written: an assumption of at
// least two words, then the marker opening a sentence, then a reason of at
// least two words in that same sentence. A marker alone, punctuation, a
// missing assumption or a reason borrowed from a later sentence does not say
// why a limit is accepted (review rounds 1–4).
func hasAcceptedLimitReason(text string) bool {
	loc := acceptedLimitMarkerRE.FindStringIndex(text)
	if loc == nil {
		return false
	}
	assumption, rest := text[:loc[0]], text[loc[1]:]
	if end := strings.IndexAny(rest, ".;!?"); end >= 0 {
		rest = rest[:end]
	}
	return len(acceptedLimitReasonWordRE.FindAllString(assumption, -1)) >= 2 && len(acceptedLimitReasonWordRE.FindAllString(rest, -1)) >= 2
}

// ParseAcceptedLimits reads the `### Accepted limits` section of a spec body.
// It returns the limits and the problems that keep the section from being
// read as written: a bullet that is not `- L<N>: ...`, a repeated id, or a
// limit that does not say why it is accepted ("Accepted because ..." or
// "Aceito porque ...").
func ParseAcceptedLimits(body string) ([]ReviewAcceptedLimit, []string) {
	var limits []ReviewAcceptedLimit
	var problems []string
	in := false
	seen := map[string]bool{}
	// The section is read fail-closed (review rounds 8 to 10): it holds only
	// limits, each one list item whose wrapped lines are indented deeper than
	// its marker (also after a blank line). Any other line in the section
	// (prose, a lazy unindented continuation, a nested list, a fence, another
	// kind of item) is reported, and the limit it belongs to or follows is
	// not sealed: the engine never seals a limit it may have read in part.
	// Fenced blocks and HTML comments outside the section are not the spec's
	// declarations, so examples there are never read.
	type entry struct {
		id, text string
		indent   int
		bad      bool
	}
	var entries []entry
	report := func(line, why string) {
		problems = append(problems, "accepted limits: "+quoteLedgerText(strings.TrimSpace(line))+" "+why)
		if len(entries) > 0 && !entries[len(entries)-1].bad {
			entries[len(entries)-1].bad = true
		}
	}
	// One pass over the raw text, so comments, fences and the section are
	// read from the same lines (spec pose-review-declared-limits-followups
	// D1; review round 1, N1–N4). Outside the section a comment hides what
	// it holds, as Markdown does, and a marker in inline code opens none.
	// Inside the section, and on its heading, any comment marker refuses the
	// whole section: the engine never seals a limit a comment may have cut.
	refused := false
	inComment := false
	fence := ""
	for _, raw := range strings.Split(body, "\n") {
		marker := strings.Contains(raw, "<!--") || strings.Contains(raw, "-->")
		if in && marker {
			// The next section's heading closes the section, whatever
			// comment it holds (spec pose-review-declared-limits-residuals R4).
			// Only a line that is an unindented heading as written, its '#'
			// first, is the next heading; one that reads as a heading only
			// once a comment is removed stays inside and refuses (review
			// round 1, N1).
			if vis, still := visibleOutsideComments(raw, inComment); fence == "" && !inComment && strings.HasPrefix(raw, "#") && headingLine(vis) && !isAcceptedLimitsHeading(vis) {
				in, inComment = false, still
				continue
			}
			if !refused {
				problems = append(problems, "accepted limits: the section holds an HTML comment marker; write the limits without comments")
			}
			refused = true
			_, inComment = visibleOutsideComments(raw, inComment)
			continue
		}
		line := raw
		if fence == "" {
			visible, still := visibleOutsideComments(raw, inComment)
			inComment = still
			line = visible
			if marker && isAcceptedLimitsHeading(line) {
				// A comment on the section's own heading line.
				problems = append(problems, "accepted limits: the section heading holds an HTML comment marker; write it without comments")
				refused = true
				in = true
				continue
			}
		}
		trimmed := strings.TrimSpace(line)
		indent := len(line) - len(strings.TrimLeft(line, " \t"))
		// Four spaces or more is indented code or a limit's continuation,
		// never a fence (R3).
		if m := fenceMarker(trimmed); m != "" && (indent <= 3 || fence != "") {
			if in {
				report(line, "is a code fence; examples belong outside the section")
			}
			switch {
			case fence == "":
				fence = m
			case m[0] == fence[0] && len(m) >= len(fence) && strings.TrimLeft(trimmed, m[:1]) == "":
				fence = ""
			}
			continue
		}
		if fence != "" {
			continue
		}
		if in && indent >= 1 && indent <= 3 && markdownHeadingRE.MatchString(trimmed) {
			// An indented heading-like line inside the section: read as a
			// heading it would end the section and cut a limit (R1; review
			// round 1, N5).
			report(line, "is an indented heading-like line in the accepted limits section; write headings unindented and each limit as one item")
			continue
		}
		if indent <= 3 && markdownHeadingRE.MatchString(trimmed) {
			in = isAcceptedLimitsHeading(trimmed)
			continue
		}
		if !in || trimmed == "" {
			continue
		}
		// A line indented under the current limit belongs to it: a list
		// there is nested, anything else continues the limit.
		under := len(entries) > 0 && indent > entries[len(entries)-1].indent
		if under && listItemRE.MatchString(trimmed) {
			report(line, "is a nested list; write the limit as one item")
			continue
		}
		if !under && indent <= 3 {
			if m := acceptedLimitRE.FindStringSubmatch(trimmed); m != nil {
				entries = append(entries, entry{id: m[1], text: strings.TrimSpace(m[2]), indent: indent})
				continue
			}
			if listItemRE.MatchString(trimmed) {
				report(line, "is not written as '- L<N>: <assumption>. Accepted because <reason>.'")
				entries = append(entries, entry{bad: true})
				continue
			}
		}
		if !under {
			report(line, "is neither a limit nor indented under one; the section holds only limits")
			continue
		}
		entries[len(entries)-1].text += " " + trimmed
	}
	if inComment {
		problems = append(problems, "accepted limits: an unterminated HTML comment hides the rest of the spec")
		return nil, boundProblems(problems)
	}
	if refused {
		return nil, boundProblems(problems)
	}
	for _, e := range entries {
		if e.id == "" {
			continue
		}
		if seen[e.id] {
			problems = append(problems, "accepted limit "+e.id+" is declared twice")
			continue
		}
		seen[e.id] = true
		if e.bad {
			problems = append(problems, "accepted limit "+e.id+" is not sealed: part of it could not be read")
			continue
		}
		if !hasAcceptedLimitReason(e.text) {
			problems = append(problems, "accepted limit "+e.id+" does not say why it is accepted ('Accepted because ...')")
			continue
		}
		limits = append(limits, ReviewAcceptedLimit{ID: e.id, Text: e.text})
	}
	return limits, boundProblems(problems)
}

// acceptedLimitProblemLimit bounds the problems one spec reports, and
// acceptedLimitProblemBytes the size of each.
const (
	acceptedLimitProblemLimit = 20
	acceptedLimitProblemBytes = 300
)

// boundProblems keeps the first problems and counts the rest (R4).
func boundProblems(problems []string) []string {
	// Each problem is bounded in size too (review round 1, N2).
	for i, p := range problems {
		problems[i] = TruncateUTF8(p, acceptedLimitProblemBytes)
	}
	if len(problems) <= acceptedLimitProblemLimit {
		return problems
	}
	rest := len(problems) - acceptedLimitProblemLimit
	return append(problems[:acceptedLimitProblemLimit:acceptedLimitProblemLimit], fmt.Sprintf("accepted limits: %d more problems not shown", rest))
}

// isAcceptedLimitsHeading reports whether line, its words read as Markdown
// shows them (a comment between them removed), is the section's heading (R1).
func isAcceptedLimitsHeading(line string) bool {
	return acceptedLimitsHeadingRE.MatchString(strings.Join(strings.Fields(line), " "))
}

// headingLine reports whether line is a Markdown heading (up to three spaces
// of indent).
func headingLine(line string) bool {
	indent := len(line) - len(strings.TrimLeft(line, " \t"))
	return indent <= 3 && markdownHeadingRE.MatchString(strings.TrimSpace(line))
}

var (
	// acceptedLimitsHeadingRE is the heading of the accepted limits section.
	acceptedLimitsHeadingRE = regexp.MustCompile(`(?i)^#{1,6}\s+(accepted limits|limites aceitos)\s*#*$`)
	markdownHeadingRE       = regexp.MustCompile(`^#{1,6}(\s|$)`)
	listItemRE              = regexp.MustCompile(`^([-*+]|[0-9]+[.)])(\s|$)`)
)

// fenceMarker returns the run of backticks or tildes that opens or closes a
// code fence on trimmed, or "". A backtick run followed by more backticks on
// the line is inline code, not a fence.
func fenceMarker(trimmed string) string {
	for _, c := range []string{"`", "~"} {
		n := len(trimmed) - len(strings.TrimLeft(trimmed, c))
		if n < 3 {
			continue
		}
		if c == "`" && strings.Contains(trimmed[n:], "`") {
			return ""
		}
		return trimmed[:n]
	}
	return ""
}

// acceptedLimitsAndProblemsOf collects the limits the scope's specs declare, with the problems that kept
// limits from being sealed, so the plan says so (review round 1, N1).
func (s Store) acceptedLimitsAndProblemsOf(scope ScopeRef) ([]ReviewAcceptedLimit, []string) {
	specs, err := s.reviewScopeSpecs(scope)
	if err != nil {
		return nil, nil
	}
	var problems []string
	for _, spec := range specs {
		if _, p := ParseAcceptedLimits(spec.Body); len(p) > 0 {
			problems = append(problems, "accepted limits of spec "+spec.Slug+" are not all sealed: "+strings.Join(p, "; "))
		}
	}
	return acceptedLimitsOfSpecs(specs), problems
}

// acceptedLimitsOfSpecs collects the limits specs declare, qualifying each id
// by its spec when there is more than one spec.
func acceptedLimitsOfSpecs(specs []Spec) []ReviewAcceptedLimit {
	var out []ReviewAcceptedLimit
	for _, spec := range specs {
		limits, _ := ParseAcceptedLimits(spec.Body)
		for _, l := range limits {
			// Ids are spec-local: across specs each is qualified by its spec,
			// so a finding names exactly one sealed limit.
			if len(specs) > 1 {
				l.ID, l.Spec = spec.Slug+"#"+l.ID, spec.Slug
			}
			out = append(out, l)
		}
	}
	return out
}

// sealedAcceptedLimit reports whether the bundle sealed limit id.
func sealedAcceptedLimit(bundle ReviewBundle, id string) bool {
	for _, l := range bundle.Payload.Plan.AcceptedLimits {
		if l.ID == id {
			return true
		}
	}
	return false
}

// htmlCommentStart returns the index in s, from from on, of the next "<!--"
// that opens an HTML comment, or -1. One inside an inline code span of its
// line opens nothing: Markdown shows it as text.
func htmlCommentStart(s string, from int) int {
	// Inline code is read as Markdown reads it: a run of n backticks opens a
	// span that only a run of exactly n backticks on the line closes; an
	// unclosed run is text (R2).
	for {
		k := strings.Index(s[from:], "<!--")
		if k < 0 {
			return -1
		}
		k += from
		lineStart := strings.LastIndex(s[:k], "\n") + 1
		if !insideCodeSpan(s[lineStart:], k-lineStart) {
			return k
		}
		from = k + 4
	}
}

// insideCodeSpan reports whether position pos of line falls inside an inline
// code span.
func insideCodeSpan(line string, pos int) bool {
	i := 0
	for i < len(line) {
		if line[i] != '`' {
			i++
			continue
		}
		n := 0
		for i+n < len(line) && line[i+n] == '`' {
			n++
		}
		open := i
		j := i + n
		closed := -1
		for j < len(line) {
			if line[j] != '`' {
				j++
				continue
			}
			m := 0
			for j+m < len(line) && line[j+m] == '`' {
				m++
			}
			if m == n {
				closed = j
				break
			}
			j += m
		}
		if closed < 0 {
			// An unclosed run is text; go on after it.
			i = open + n
			continue
		}
		if pos > open && pos < closed {
			return true
		}
		i = closed + n
	}
	return false
}

// visibleOutsideComments returns what of line Markdown shows outside HTML
// comments, and whether a comment is still open after it.
func visibleOutsideComments(line string, inComment bool) (string, bool) {
	var visible strings.Builder
	rest := line
	for rest != "" {
		if inComment {
			end := strings.Index(rest, "-->")
			if end < 0 {
				return visible.String(), true
			}
			rest = rest[end+3:]
			inComment = false
			continue
		}
		start := htmlCommentStart(rest, 0)
		if start < 0 {
			visible.WriteString(rest)
			break
		}
		visible.WriteString(rest[:start])
		rest = rest[start+4:]
		inComment = true
	}
	return visible.String(), inComment
}
