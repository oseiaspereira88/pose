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

var acceptedLimitRE = regexp.MustCompile(`^-\s+(L[1-9][0-9]*):\s+(.+)$`)

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
	for _, line := range strings.Split(body, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") {
			in = strings.EqualFold(strings.TrimSpace(strings.TrimLeft(trimmed, "#")), "Accepted limits") || strings.EqualFold(strings.TrimSpace(strings.TrimLeft(trimmed, "#")), "Limites aceitos")
			continue
		}
		if !in || !strings.HasPrefix(trimmed, "- ") {
			continue
		}
		m := acceptedLimitRE.FindStringSubmatch(trimmed)
		if m == nil {
			problems = append(problems, "accepted limit "+quoteLedgerText(trimmed)+" is not written as '- L<N>: <assumption>. Accepted because <reason>.'")
			continue
		}
		if seen[m[1]] {
			problems = append(problems, "accepted limit "+m[1]+" is declared twice")
			continue
		}
		seen[m[1]] = true
		if !hasAcceptedLimitReason(m[2]) {
			problems = append(problems, "accepted limit "+m[1]+" does not say why it is accepted ('Accepted because ...')")
		}
		limits = append(limits, ReviewAcceptedLimit{ID: m[1], Text: strings.TrimSpace(m[2])})
	}
	return limits, problems
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
