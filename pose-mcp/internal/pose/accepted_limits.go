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
}

var acceptedLimitRE = regexp.MustCompile(`^-\s+(L[1-9][0-9]*):\s+(.+)$`)

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
		lower := strings.ToLower(m[2])
		if !strings.Contains(lower, "accepted because") && !strings.Contains(lower, "aceito porque") {
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
	var out []ReviewAcceptedLimit
	for _, spec := range specs {
		limits, _ := ParseAcceptedLimits(spec.Body)
		out = append(out, limits...)
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
