package pose

import (
	"sort"
	"strings"
)

// SpecFacts is the factual part of a spec's Final Report, derived from the
// delivery index instead of transcribed (spec pose-progressive-spec-surface):
// the paths the attributed commits changed and the checks that produced
// evidence. It never contains intent, rationale or accepted risk — those
// stay written by a person or an agent.
type SpecFacts struct {
	Spec        string          `json:"spec"`
	Commits     int             `json:"commits"`
	Paths       []ObservedPath  `json:"paths"`
	Checks      []SpecFactCheck `json:"checks"`
	Source      string          `json:"source"`
	Limitations []string        `json:"limitations,omitempty"`
}

// SpecFactCheck is one validation result the spec's evidence came from.
type SpecFactCheck struct {
	Module        string `json:"module"`
	Check         string `json:"check"`
	EvidenceClass string `json:"evidence_class"`
	Outcome       string `json:"outcome"`
}

// DeriveSpecFacts reads the delivery index. A spec with no attributed change
// set says so instead of reporting an empty delivery.
func (s Store) DeriveSpecFacts(slug string) (SpecFacts, error) {
	spec, err := s.GetSpec(slug)
	if err != nil {
		return SpecFacts{}, err
	}
	graph, err := s.GetDeliveryIntegrity("")
	if err != nil {
		return SpecFacts{}, err
	}
	facts := SpecFacts{Spec: slug, Source: ".pose/indexes/delivery-integrity.json", Paths: []ObservedPath{}, Checks: []SpecFactCheck{}}
	seen := map[string]bool{}
	commits := map[string]bool{}
	for _, cs := range graph.ChangeSets {
		if cs.Spec != slug {
			continue
		}
		for _, c := range cs.Commits {
			commits[c] = true
		}
		for _, p := range cs.Paths {
			key := p.Action + "\x00" + p.Path
			if !seen[key] {
				seen[key] = true
				facts.Paths = append(facts.Paths, p)
			}
		}
	}
	facts.Commits = len(commits)
	if len(facts.Paths) == 0 {
		facts.Limitations = append(facts.Limitations, "no attributed change set is indexed for this spec; run `pose index` after committing with its POSE-Spec trailer")
	}
	sort.Slice(facts.Paths, func(i, j int) bool { return facts.Paths[i].Path < facts.Paths[j].Path })
	modules := map[string]bool{}
	for _, c := range spec.Components {
		modules[strings.TrimSpace(c)] = true
	}
	checkSeen := map[string]bool{}
	for _, r := range graph.ValidationResults {
		if len(modules) > 0 && !modules[r.Module] {
			continue
		}
		if _, scoped := r.ScopeProvenance[slug]; len(r.ScopeProvenance) > 0 && !scoped {
			continue
		}
		key := r.Module + "|" + r.Check
		if checkSeen[key] {
			continue
		}
		checkSeen[key] = true
		facts.Checks = append(facts.Checks, SpecFactCheck{Module: r.Module, Check: r.Check, EvidenceClass: r.EvidenceClass, Outcome: r.Outcome})
	}
	sort.Slice(facts.Checks, func(i, j int) bool {
		if facts.Checks[i].Module != facts.Checks[j].Module {
			return facts.Checks[i].Module < facts.Checks[j].Module
		}
		return facts.Checks[i].Check < facts.Checks[j].Check
	})
	if len(facts.Checks) == 0 {
		facts.Limitations = append(facts.Limitations, "no indexed validation result is scoped to this spec")
	}
	return facts, nil
}

// SpecSurface names how much of the template a spec carries. Full is the
// whole template; minimal keeps what governance needs from a small change:
// intent, requirements, the declared artifacts, validation with the
// requirement trace, and the final report with its follow-ups.
const (
	SpecSurfaceFull     = "full"
	SpecSurfaceStandard = "standard"
	SpecSurfaceMinimal  = "minimal"
)

// RequiredSpecSections returns the sections a surface requires.
func RequiredSpecSections(surface string) []string {
	switch surface {
	case SpecSurfaceMinimal:
		return []string{"Intent", "Requirements", "Technical Plan", "Validation", "Final Report"}
	default:
		return []string{"Intent", "Requirements", "Technical Plan", "Tasks", "Validation", "Final Report"}
	}
}
