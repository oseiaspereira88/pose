package pose

import (
	"sort"
	"strconv"
	"strings"
)

// What the subject was observed to do, as opposed to what the scope declared.
//
// Declared metadata is the author's account of a change. It is a good first
// trigger and a poor last one: an undeclared dependency is still a dependency,
// and a Markdown extension is not proof of low risk. This file adds the other
// half — facts read from both sides of the sealed subject — and keeps it bounded
// in two ways. Only a closed vocabulary counts as material, so nobody owes a
// decision for every file, and the observation is resolved only when an adopted
// profile actually selects on it, so a repository that did not opt in pays
// nothing, not even the Git reads.
//
// MaterialStructuralKinds is that vocabulary. It is the set a detector can
// observe *and* that carries a design cost worth a reviewer's sentence. A
// rename, a lock file and an unreadable manifest are all observed and all
// reported, and none of them is here: the first two are consequences rather than
// decisions, and the third is uncertainty, which is shown and never charged.
var MaterialStructuralKinds = map[string]bool{
	"dependency":          true,
	"component":           true,
	"delivery-metadata":   true,
	"governance-contract": true,
	"public-contract":     true,
	"submodule":           true,
}

// runtimeTransitive is the one runtime excluded from materiality. A lock file
// alone does not create dozens of mandatory decisions; the direct relationship
// the author declared is what does.
const runtimeTransitive = "runtime-transitive"

func sortedMaterialStructuralKinds() []string {
	kinds := []string{}
	for kind := range MaterialStructuralKinds {
		kinds = append(kinds, kind)
	}
	sort.Strings(kinds)
	return kinds
}

// StructuralDeltaIsMaterial reports whether one observed delta is a material
// structural fact. A delta whose state is not `observed` never is: an unknown or
// unsupported reading is a gap in coverage, and treating it as a fact would let
// a detector limitation manufacture an obligation.
func StructuralDeltaIsMaterial(delta StructuralDelta) bool {
	if delta.State != "observed" || !MaterialStructuralKinds[delta.Kind] {
		return false
	}
	if delta.Kind == "dependency" && delta.Runtime == runtimeTransitive {
		return false
	}
	// A `component` delta is emitted for every observed manifest, so `changed`
	// says only that a manifest in that component was edited — which the
	// dependency deltas beside it already say precisely. The boundary itself
	// moving is the design fact: a component appearing, disappearing or being
	// relocated. Counting `changed` would charge a second mapping for the same
	// edit and make a transitive-only bump look like a boundary change.
	if delta.Kind == "component" && delta.Action == "changed" {
		return false
	}
	return true
}

// ReviewStructuralFact is one material fact, identified by the content-derived
// display id the delta report assigned it. Values stay digests and names; no
// manifest content reaches a review artifact through here.
type ReviewStructuralFact struct {
	ID      string `json:"id"`
	Kind    string `json:"kind"`
	Action  string `json:"action"`
	Subject string `json:"subject"`
	Path    string `json:"path,omitempty"`
	Runtime string `json:"runtime,omitempty"`
}

// ReviewPlanStructure is the observed structural context of a plan.
//
// Unlike the band summary, this is not a projection of obligations already
// resolved: the material set is an obligation input, because a criterion that
// answers for observed structure owes one answer per fact. It therefore enters
// the plan digest, and a subject that gains a material fact makes a sealed review
// stale rather than silently owing more.
type ReviewPlanStructure struct {
	Observed      bool                   `json:"observed"`
	Status        string                 `json:"status,omitempty"`
	InputDigest   string                 `json:"input_digest,omitempty"`
	SubjectDigest string                 `json:"subject_digest,omitempty"`
	Kinds         []string               `json:"kinds,omitempty"`
	Material      []ReviewStructuralFact `json:"material,omitempty"`
	// Unknown is coverage the detectors could not resolve. It is reported so a
	// clean material set is never mistaken for a complete reading.
	Unknown []string `json:"unknown,omitempty"`
	Reason  string   `json:"reason,omitempty"`
}

// resolveReviewStructure observes the sealed subject. It is deliberately not
// called through PrepareReviewBundle: that builds a bundle *from* a plan, so a
// plan asking it for a subject would be circular. The subject builder underneath
// takes the scope, the mapped components and the integrity graph, and knows
// nothing about plans.
func (s Store) resolveReviewStructure(scope ScopeRef, components []ReviewPlanComponent) *ReviewPlanStructure {
	structure := &ReviewPlanStructure{}
	graph, err := s.GetDeliveryIntegrity("")
	if err != nil {
		structure.Reason = "delivery-integrity index is unavailable, so no subject is attributable; run `pose index`"
		return structure
	}
	subject, _, blockers, err := s.reviewBundleSubject(scope, components, graph, nil)
	if err != nil {
		structure.Reason = err.Error()
		return structure
	}
	if len(blockers) > 0 {
		structure.Reason = strings.Join(blockers, "; ")
		return structure
	}
	report, err := AssessDesignDelta(s.Root, subject, scope.String(), DesignDeltaOptions{})
	if err != nil {
		structure.Reason = err.Error()
		return structure
	}
	structure.Observed = true
	structure.Status = report.Status
	structure.InputDigest = report.InputDigest
	structure.SubjectDigest = subject.ImplementationDigest
	kinds := []string{}
	for _, delta := range report.Deltas {
		if !StructuralDeltaIsMaterial(delta) {
			if delta.State != "observed" {
				structure.Unknown = append(structure.Unknown, delta.Kind+":"+delta.Subject+":"+delta.State)
			}
			continue
		}
		kinds = append(kinds, delta.Kind)
		structure.Material = append(structure.Material, ReviewStructuralFact{
			ID: delta.DisplayID, Kind: delta.Kind, Action: delta.Action,
			Subject: delta.Subject, Path: delta.Path, Runtime: delta.Runtime,
		})
	}
	for _, detector := range report.Coverage.Detectors {
		if detector.State == "unknown" || detector.State == "unsupported" {
			structure.Unknown = append(structure.Unknown, "detector:"+detector.ID+":"+detector.State)
		}
	}
	if report.Coverage.Truncated {
		structure.Unknown = append(structure.Unknown, "coverage:truncated")
	}
	sort.Slice(structure.Material, func(i, j int) bool { return structure.Material[i].ID < structure.Material[j].ID })
	structure.Kinds = uniqueSorted(kinds)
	structure.Unknown = uniqueSorted(structure.Unknown)
	return structure
}

// reviewStructuralMappingCriteria returns the required criteria that declared
// themselves answerable for observed structure.
func reviewStructuralMappingCriteria(required map[string]ReviewPlanCriterion) []string {
	ids := []string{}
	for id, criterion := range required {
		if criterion.RequiresStructuralMapping {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	return ids
}

// reviewDesignBasisNamespace resolves what a mapping may point at, from the
// scope's own decision basis. The engine checks the namespace, the existence and
// the reach to a requirement or constraint — never the causality, which is the
// reviewer's judgment and not a parse.
type reviewDesignBasisNamespace struct {
	Requirements map[string]bool
	Reaches      map[string]bool
	Known        map[string]bool
	HasSection   bool
}

func (s Store) reviewDesignBasisNamespace(scope ScopeRef) reviewDesignBasisNamespace {
	ns := reviewDesignBasisNamespace{Requirements: map[string]bool{}, Reaches: map[string]bool{}, Known: map[string]bool{}}
	specs, err := s.reviewScopeSpecs(scope)
	if err != nil {
		return ns
	}
	for _, spec := range specs {
		requirements, constraints := declaredDesignRequirements(spec.Body)
		for id := range requirements {
			ns.Requirements[id], ns.Known[id], ns.Reaches[id] = true, true, true
		}
		for id := range constraints {
			ns.Requirements[id], ns.Known[id], ns.Reaches[id] = true, true, true
		}
		report := ValidateDesignBasis(spec.Body, s.Root)
		if !report.HasSection {
			continue
		}
		ns.HasSection = true
		assumptionReaches := map[string]bool{}
		for _, assumption := range report.Assumptions {
			ns.Known[assumption.ID] = true
			for _, affected := range assumption.Affects {
				if ns.Requirements[affected] {
					assumptionReaches[assumption.ID] = true
				}
			}
			if assumptionReaches[assumption.ID] {
				ns.Reaches[assumption.ID] = true
			}
		}
		for _, decision := range report.Decisions {
			ns.Known[decision.ID] = true
			for _, basis := range decision.Basis {
				// A decision reaches a requirement through its basis, or
				// directly when the basis names one. Pointing at a decision
				// that reaches nothing is exactly the case the design rejects:
				// it states a choice without the requirement that pays for it.
				if assumptionReaches[basis] || ns.Requirements[basis] {
					ns.Reaches[decision.ID] = true
				}
			}
		}
	}
	return ns
}

// reviewStructuralCausalityBlockers enforces the mapping obligation for one
// attempt. It runs only for a bundle sealed under the structural-causality
// contract, so the attestations recorded before it keep their verdict.
func (s Store) reviewStructuralCausalityBlockers(scopeRef string, structure *ReviewPlanStructure, required map[string]ReviewPlanCriterion, dispositions []ReviewCriterion) []string {
	if structure == nil || len(structure.Material) == 0 {
		return nil
	}
	scope, err := ParseScopeRef(scopeRef)
	if err != nil {
		return []string{"structural causality cannot be checked for scope " + describeMappingDelta(scopeRef)}
	}
	owners := reviewStructuralMappingCriteria(required)
	if len(owners) == 0 {
		return nil
	}
	facts := map[string]ReviewStructuralFact{}
	for _, fact := range structure.Material {
		facts[fact.ID] = fact
	}
	namespace := s.reviewDesignBasisNamespace(scope)
	blockers := []string{}
	for _, id := range owners {
		disposition := ReviewCriterion{}
		found := false
		for _, candidate := range dispositions {
			if candidate.ID == id {
				disposition, found = candidate, true
				break
			}
		}
		if !found {
			continue // the missing-criterion blocker is already raised upstream
		}
		switch disposition.Disposition {
		case "not-applicable":
			// The scope moved dependencies, contracts or governance, and the
			// criterion that answers for exactly that says nothing applies.
			blockers = append(blockers, "criterion "+id+" is not-applicable while the bundle seals "+
				pluralizeStructuralFacts(len(structure.Material))+" ("+strings.Join(structure.Kinds, ", ")+
				"); an observed structural change is not inapplicable, it is unmapped")
			continue
		case "passed":
		default:
			continue // finding and invalid dispositions are handled upstream
		}
		covered := map[string]bool{}
		for _, mapping := range disposition.Mappings {
			fact, known := facts[mapping.Delta]
			if !known {
				blockers = append(blockers, "criterion "+id+" maps "+describeMappingDelta(mapping.Delta)+", which this subject does not observe as material")
				continue
			}
			if covered[mapping.Delta] {
				blockers = append(blockers, "criterion "+id+" maps "+mapping.Delta+" twice")
				continue
			}
			covered[mapping.Delta] = true
			state := mapping.Disposition
			if state == "" && strings.TrimSpace(mapping.Basis) != "" {
				state = ReviewMappingMapped
			}
			switch state {
			case ReviewMappingMapped:
				basis := strings.TrimSpace(mapping.Basis)
				switch {
				case basis == "":
					blockers = append(blockers, "criterion "+id+" maps "+mapping.Delta+" as mapped and names no basis")
				case !designIDRE.MatchString(basis):
					blockers = append(blockers, "criterion "+id+" maps "+mapping.Delta+" to "+basis+", which is not a decision-basis ref")
				case !namespace.Known[basis]:
					blockers = append(blockers, "criterion "+id+" maps "+mapping.Delta+" to "+basis+", which this scope's decision basis does not declare")
				case !namespace.Reaches[basis]:
					blockers = append(blockers, "criterion "+id+" maps "+mapping.Delta+" to "+basis+", which reaches no requirement or constraint; a decision without the requirement that pays for it does not justify "+fact.Kind+" "+fact.Subject)
				}
			case ReviewMappingMissingEvidence, ReviewMappingNotApplicable, ReviewMappingAcceptedRisk:
				if strings.TrimSpace(mapping.Rationale) == "" {
					blockers = append(blockers, "criterion "+id+" disposes "+mapping.Delta+" as "+state+" with no rationale")
				}
				if strings.TrimSpace(mapping.Basis) != "" {
					blockers = append(blockers, "criterion "+id+" disposes "+mapping.Delta+" as "+state+" and still names basis "+mapping.Basis+"; state one or the other")
				}
			default:
				blockers = append(blockers, "criterion "+id+" disposes "+mapping.Delta+" as "+describeMappingDelta(state)+", which is not mapped, missing-evidence, not-applicable or accepted-risk")
			}
		}
		for _, fact := range structure.Material {
			if !covered[fact.ID] {
				blockers = append(blockers, "criterion "+id+" passes without answering for "+fact.ID+" ("+fact.Kind+" "+fact.Action+" "+fact.Subject+")")
			}
		}
	}
	sort.Strings(blockers)
	return blockers
}

func pluralizeStructuralFacts(count int) string {
	if count == 1 {
		return "1 material structural fact"
	}
	return strconv.Itoa(count) + " material structural facts"
}

// describeMappingDelta keeps a reviewer-supplied value quotable without letting
// a crafted string forge the rest of the message.
func describeMappingDelta(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "an empty id"
	}
	if len(value) > 64 {
		value = value[:64]
	}
	return strconv.Quote(value)
}
