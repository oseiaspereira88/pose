package pose

// Accepted limits (spec pose-review-declared-threat-model): a spec declares
// the residual limits of a guarantee it makes, each with a stable id and why
// it is accepted. The review plan seals them; the brief lists them; a
// reviewer classifies a finding that falls inside one as within that limit,
// or disputes the limit as a finding of its own. A declared limit never
// hides a defect inside the declared scope: it is visible in the bundle, the
// brief and the attestation.

import (
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
	// Any unterminated comment, not only the first, hides what follows it
	// (spec pose-review-declared-limits-followups R2).
	if unterminatedHTMLComment(body) {
		problems = append(problems, "accepted limits: an unterminated HTML comment hides the rest of the spec")
	}
	// A comment inside the section, also one written in inline code, would
	// change a limit's sealed text: the section is refused whole instead of
	// guessing which is meant (R2, D1).
	if commentInAcceptedLimitsSection(body) {
		problems = append(problems, "accepted limits: the section holds an HTML comment marker; write the limits without comments")
		return nil, problems
	}
	text := stripLimitComments(body)
	fence := ""
	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)
		indent := len(line) - len(strings.TrimLeft(line, " \t"))
		if marker := fenceMarker(trimmed); marker != "" {
			if in {
				report(line, "is a code fence; examples belong outside the section")
			}
			switch {
			case fence == "":
				fence = marker
			case marker[0] == fence[0] && len(marker) >= len(fence) && strings.TrimLeft(trimmed, marker[:1]) == "":
				fence = ""
			}
			continue
		}
		if fence != "" {
			continue
		}
		if in && indent > 0 && len(entries) > 0 && indent > entries[len(entries)-1].indent && indent <= 3 && markdownHeadingRE.MatchString(trimmed) {
			// A heading-like line indented under a limit: read as a heading it
			// would end the section and cut the limit (R1).
			report(line, "is a heading inside a limit; write the limit as one item")
			continue
		}
		if indent <= 3 && markdownHeadingRE.MatchString(trimmed) {
			heading := strings.TrimSpace(strings.Trim(trimmed, "# "))
			in = strings.EqualFold(heading, "Accepted limits") || strings.EqualFold(heading, "Limites aceitos")
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
	return limits, problems
}

var (
	markdownHeadingRE = regexp.MustCompile(`^#{1,6}(\s|$)`)
	listItemRE        = regexp.MustCompile(`^([-*+]|[0-9]+[.)])(\s|$)`)
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

// acceptedLimitsOf collects the limits the scope's specs declare.
func (s Store) acceptedLimitsOf(scope ScopeRef) []ReviewAcceptedLimit {
	specs, err := s.reviewScopeSpecs(scope)
	if err != nil {
		return nil
	}
	return acceptedLimitsOfSpecs(specs)
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
// that opens an HTML comment, or -1. One written in inline code (an odd
// number of backticks before it on its line) opens nothing: Markdown shows
// it as text, as a spec that names the marker does (review of
// pose-review-declared-limits-followups: its own R2 named it).
func htmlCommentStart(s string, from int) int {
	for {
		k := strings.Index(s[from:], "<!--")
		if k < 0 {
			return -1
		}
		k += from
		lineStart := strings.LastIndex(s[:k], "\n") + 1
		if strings.Count(s[lineStart:k], "`")%2 == 0 {
			return k
		}
		from = k + 4
	}
}

// stripLimitComments removes the HTML comments of body, as Markdown hides
// them; an unterminated one hides the rest.
func stripLimitComments(body string) string {
	var b strings.Builder
	from := 0
	for {
		start := htmlCommentStart(body, from)
		if start < 0 {
			b.WriteString(body[from:])
			return b.String()
		}
		b.WriteString(body[from:start])
		end := strings.Index(body[start+4:], "-->")
		if end < 0 {
			return b.String()
		}
		from = start + 4 + end + 3
	}
}

// unterminatedHTMLComment reports whether any comment body opens has no
// "-->" after it.
func unterminatedHTMLComment(body string) bool {
	from := 0
	for {
		start := htmlCommentStart(body, from)
		if start < 0 {
			return false
		}
		end := strings.Index(body[start+4:], "-->")
		if end < 0 {
			return true
		}
		from = start + 4 + end + 3
	}
}

// commentInAcceptedLimitsSection reports whether a line of the accepted
// limits section, read before comments are stripped, holds a comment marker.
// A comment outside the section, and a section heading inside one, are not
// the section.
func commentInAcceptedLimitsSection(body string) bool {
	in, inComment := false, false
	for _, line := range strings.Split(body, "\n") {
		if inComment {
			k := strings.Index(line, "-->")
			if k < 0 {
				continue
			}
			inComment = false
			line = line[k+3:]
		}
		if in && (strings.Contains(line, "<!--") || strings.Contains(line, "-->")) {
			return true
		}
		if k := htmlCommentStart(line, 0); k >= 0 {
			if !strings.Contains(line[k:], "-->") {
				inComment = true
			}
			continue
		}
		trimmed := strings.TrimSpace(line)
		indent := len(line) - len(strings.TrimLeft(line, " \t"))
		if indent <= 3 && markdownHeadingRE.MatchString(trimmed) {
			heading := strings.TrimSpace(strings.Trim(trimmed, "# "))
			in = strings.EqualFold(heading, "Accepted limits") || strings.EqualFold(heading, "Limites aceitos")
		}
	}
	return false
}
