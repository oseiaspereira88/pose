package pose

import (
	"fmt"
	"sort"
	"strings"
)

// SpecFacts is the factual part of a spec's Final Report, derived from the
// delivery index instead of transcribed (spec pose-progressive-spec-surface):
// the paths the attributed commits changed and the checks that produced
// evidence. It never contains intent, rationale or accepted risk — those
// stay written by a person or an agent.
//
// Checks are the results the index ties to this spec (spec
// pose-spec-facts-evidence-binding): their scope provenance names it. Candidates are
// results the index holds that nothing ties to it; they are listed so the reader can see
// them, and are not evidence that the spec was validated.
type SpecFacts struct {
	Spec        string             `json:"spec"`
	Commits     int                `json:"commits"`
	Paths       []ObservedPath     `json:"paths"`
	Checks      []SpecFactCheck    `json:"checks"`
	Candidates  []SpecFactCheck    `json:"candidates"`
	Freshness   SpecFactsFreshness `json:"freshness"`
	Source      string             `json:"source"`
	Limitations []string           `json:"limitations,omitempty"`
}

// Classification of a result for a spec.
const (
	SpecFactAssociated = "associated"
	SpecFactCandidate  = "candidate"
)

// SpecFactCheck is one validation result the spec's evidence came from, or a candidate.
type SpecFactCheck struct {
	Module         string          `json:"module"`
	Check          string          `json:"check"`
	EvidenceClass  string          `json:"evidence_class"`
	Outcome        string          `json:"outcome"`
	Classification string          `json:"classification"`
	Basis          string          `json:"basis"`
	Binding        SpecFactBinding `json:"binding"`
	Limits         []string        `json:"limits,omitempty"`
}

// SpecFactBinding says what a result is bound to: the revision it ran against, the
// provenance it carries, and how that revision compares with the head of the spec's change
// sets. A result that ran against another head is reported as such, not dropped: it may be
// reusable, and whether it is is a judgment the facts do not make.
type SpecFactBinding struct {
	GitHead          string `json:"git_head,omitempty"`
	ProvenanceDigest string `json:"provenance_digest,omitempty"`
	GeneratedAt      string `json:"generated_at,omitempty"`
	Head             string `json:"head"`
}

// Freshness of the index the facts were read from.
const (
	SpecFactsFresh       = "fresh"
	SpecFactsStale       = "stale"
	SpecFactsUnknown     = "unknown"
	SpecFactsUnavailable = "unavailable"
)

// SpecFactsFreshness says whether the index can be trusted to describe the inputs now: a
// stale, unpublished or unreadable index is not a current, complete one.
type SpecFactsFreshness struct {
	Status      string   `json:"status"`
	Reason      string   `json:"reason,omitempty"`
	Generation  string   `json:"generation,omitempty"`
	StaleInputs []string `json:"stale_inputs,omitempty"`
}

func (s Store) specFactsFreshness() SpecFactsFreshness {
	view := s.ReadIndexGeneration()
	switch view.Status {
	case IndexGenerationUnavailable:
		return SpecFactsFreshness{Status: SpecFactsUnavailable, Reason: view.Reason}
	case IndexGenerationCurrent:
		stale := view.StaleInputs(func(name string) (string, bool) { return IndexInputBinding(s.Root, name) })
		if len(stale) > 0 {
			return SpecFactsFreshness{Status: SpecFactsStale, Generation: view.Descriptor.GenerationID, StaleInputs: stale,
				Reason: "an input of the index changed since it was published: " + strings.Join(stale, ", ")}
		}
		return SpecFactsFreshness{Status: SpecFactsFresh, Generation: view.Descriptor.GenerationID}
	}
	return SpecFactsFreshness{Status: SpecFactsUnknown, Reason: view.Reason}
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
	facts := SpecFacts{Spec: slug, Source: ".pose/indexes/delivery-integrity.json", Paths: []ObservedPath{}, Checks: []SpecFactCheck{}, Candidates: []SpecFactCheck{}}
	facts.Freshness = s.specFactsFreshness()
	seen := map[string]bool{}
	commits := map[string]bool{}
	heads := map[string]bool{}
	for _, cs := range graph.ChangeSets {
		if cs.Spec != slug {
			continue
		}
		if cs.ResolvedHead != "" {
			heads[cs.ResolvedHead] = true
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
	switch facts.Freshness.Status {
	case SpecFactsStale:
		facts.Limitations = append(facts.Limitations, "the index is stale ("+facts.Freshness.Reason+"): run `pose index` before relying on these facts")
	case SpecFactsUnknown:
		facts.Limitations = append(facts.Limitations, "the index was not published as a set, so its freshness is unknown: run `pose index`")
	case SpecFactsUnavailable:
		facts.Limitations = append(facts.Limitations, "the index is unavailable ("+facts.Freshness.Reason+"): run `pose index`")
	}
	sort.Slice(facts.Paths, func(i, j int) bool { return facts.Paths[i].Path < facts.Paths[j].Path })
	modules := map[string]bool{}
	for _, c := range spec.Components {
		modules[strings.TrimSpace(c)] = true
	}
	checkSeen := map[string]bool{}
	otherSpecs := 0
	for _, r := range graph.ValidationResults {
		_, scoped := r.ScopeProvenance[slug]
		if len(r.ScopeProvenance) > 0 && !scoped {
			// The index says this result was produced for other specs.
			otherSpecs++
			continue
		}
		key := r.Module + "|" + r.Check
		if checkSeen[key] {
			continue
		}
		checkSeen[key] = true
		fact := SpecFactCheck{Module: r.Module, Check: r.Check, EvidenceClass: r.EvidenceClass, Outcome: r.Outcome,
			Binding: SpecFactBinding{GitHead: r.GitHead, ProvenanceDigest: r.ProvenanceDigest, GeneratedAt: r.GeneratedAt, Head: bindingHead(r.GitHead, heads)}}
		switch {
		case scoped:
			fact.Classification, fact.Basis = SpecFactAssociated, "scope-provenance"
			if len(modules) > 0 && !modules[r.Module] {
				fact.Limits = append(fact.Limits, "the module is not among the spec's declared components; the association rests on the result's scope provenance")
			}
		default:
			// Nothing ties this result to the spec: a module that happens to be declared, or
			// a result the index holds, does not show that the check covered its delta.
			fact.Classification, fact.Basis = SpecFactCandidate, "none"
			if modules[r.Module] {
				fact.Basis = "components"
				fact.Limits = append(fact.Limits, "the module is declared by the spec, but the result carries no scope provenance: it is not shown to cover this spec's delta")
			} else {
				fact.Limits = append(fact.Limits, "the result carries no scope provenance and its module is not declared by the spec: it is not evidence of this spec")
			}
		}
		if fact.Binding.Head == "other-head" {
			fact.Limits = append(fact.Limits, "the result ran against a head other than the spec's change sets; whether it can be reused is a judgment these facts do not make")
		}
		if fact.Classification == SpecFactAssociated {
			facts.Checks = append(facts.Checks, fact)
		} else {
			facts.Candidates = append(facts.Candidates, fact)
		}
	}
	order := func(list []SpecFactCheck) {
		sort.Slice(list, func(i, j int) bool {
			if list[i].Module != list[j].Module {
				return list[i].Module < list[j].Module
			}
			return list[i].Check < list[j].Check
		})
	}
	order(facts.Checks)
	order(facts.Candidates)
	if len(facts.Checks) == 0 {
		facts.Limitations = append(facts.Limitations, "no indexed validation result is tied to this spec by scope provenance")
	}
	if len(facts.Candidates) > 0 {
		facts.Limitations = append(facts.Limitations, fmt.Sprintf("%d indexed result(s) are candidates only: nothing ties them to this spec", len(facts.Candidates)))
	}
	if otherSpecs > 0 {
		facts.Limitations = append(facts.Limitations, fmt.Sprintf("%d indexed result(s) carry scope provenance for other specs and are left out", otherSpecs))
	}
	return facts, nil
}

// bindingHead compares the head a result ran against with the heads of the spec's change sets.
func bindingHead(head string, heads map[string]bool) string {
	switch {
	case head == "":
		return "unknown-head"
	case len(heads) == 0:
		return "unknown-head"
	case heads[head]:
		return "same-head"
	}
	return "other-head"
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
