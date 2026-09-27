package pose

// Causality closeout (spec pose-abm-causality-attestation) completes the
// structural-causality contract. That contract makes a criterion answering
// for observed structure map each material fact to a decision basis that
// reaches a requirement. This one adds what a syntactically valid map can
// still hide: a basis that is no longer alive, a justification that was never
// given or was pasted across facts, a risk accepted without an owner or over
// an integrity fact, unknown coverage read as absence, and an elevated band
// with nobody answering for the structure at all.
//
// It governs only bundles stamped with it, and it is stamped only when the
// review policy adopts causality_closeout_version, so every bundle sealed
// without it keeps its verdict.

import (
	"regexp"
	"sort"
	"strconv"
	"strings"
)

const (
	CausalityCloseoutContract      = "causality-closeout"
	CausalityCloseoutPolicyVersion = 1
	// causalityProportionalityLimit is how many facts one pasted
	// justification may cover on the same basis before it stops reading as a
	// reason for each of them.
	causalityProportionalityLimit = 3
)

var (
	causalityOwnerRE = regexp.MustCompile(`^@[a-z0-9][a-z0-9._-]*$`)
	causalityDateRE  = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
	// integrityStructuralKinds are the facts a risk acceptance cannot waive:
	// they change what other parties rely on, not only this scope's cost.
	integrityStructuralKinds = map[string]bool{"public-contract": true, "governance-contract": true}
)

// CausalityCloseoutAdopted reads the review policy's capability.
func (s Store) CausalityCloseoutAdopted() (bool, error) {
	policy, err := s.GetReviewPolicy()
	if err != nil {
		return false, err
	}
	return policy.CausalityCloseoutVersion == CausalityCloseoutPolicyVersion, nil
}

// reviewScopeBasis is the sealed decision basis of a scope: the contract-node
// digest of each spec it covers.
func (s Store) reviewScopeBasis(scope ScopeRef) map[string]string {
	specs, err := s.reviewScopeSpecs(scope)
	if err != nil {
		return nil
	}
	basis := map[string]string{}
	for _, spec := range specs {
		basis[spec.Slug] = ProjectContractNodes(spec.Slug, spec.Body).Digest
	}
	return basis
}

// reviewScopeNodeStates indexes the live state of every R/A/D node the scope
// declares, keyed by the bare ID mappings use.
func (s Store) reviewScopeNodeStates(scope ScopeRef) map[string]ContractNode {
	out := map[string]ContractNode{}
	specs, err := s.reviewScopeSpecs(scope)
	if err != nil {
		return out
	}
	for _, spec := range specs {
		for _, node := range ProjectContractNodes(spec.Slug, spec.Body).Nodes {
			out[node.ID] = node
		}
	}
	return out
}

// reviewCausalityCloseoutBlockers applies the closeout rules to one attempt.
func (s Store) reviewCausalityCloseoutBlockers(scopeRef string, structure *ReviewPlanStructure, band string, required map[string]ReviewPlanCriterion, dispositions []ReviewCriterion) []string {
	owners := reviewStructuralMappingCriteria(required)
	blockers := []string{}
	material := 0
	if structure != nil {
		material = len(structure.Material)
	}
	if (band == "elevated" || band == "critical") && material > 0 && len(owners) == 0 {
		blockers = append(blockers, "the "+band+" band seals "+pluralizeStructuralFacts(material)+
			" and no required criterion answers for observed structure; add a profile criterion that maps it")
	}
	if len(owners) == 0 || structure == nil {
		return blockers
	}
	scope, err := ParseScopeRef(scopeRef)
	if err != nil {
		return append(blockers, "causality closeout cannot be checked for scope "+describeMappingDelta(scopeRef))
	}
	facts := map[string]ReviewStructuralFact{}
	for _, fact := range structure.Material {
		facts[fact.ID] = fact
	}
	nodes := s.reviewScopeNodeStates(scope)
	for _, id := range owners {
		disposition, found := ReviewCriterion{}, false
		for _, candidate := range dispositions {
			if candidate.ID == id {
				disposition, found = candidate, true
				break
			}
		}
		if !found {
			continue
		}
		if disposition.Disposition == "not-applicable" && material == 0 && len(structure.Unknown) > 0 {
			blockers = append(blockers, "criterion "+id+" is not-applicable while structural coverage is unknown ("+
				strings.Join(structure.Unknown, ", ")+"); unknown is a gap in the reading, not an absence of structure")
			continue
		}
		if disposition.Disposition != "passed" {
			continue
		}
		pasted := map[string][]string{}
		for _, mapping := range disposition.Mappings {
			fact, known := facts[mapping.Delta]
			if !known {
				continue // the structural-causality contract already reports it
			}
			state := mapping.Disposition
			if state == "" && strings.TrimSpace(mapping.Basis) != "" {
				state = ReviewMappingMapped
			}
			switch state {
			case ReviewMappingMapped:
				basis := strings.TrimSpace(mapping.Basis)
				if node, ok := nodes[basis]; ok {
					switch {
					case node.Kind == "assumption" && (node.State == "invalidated" || node.State == "withdrawn"):
						blockers = append(blockers, "criterion "+id+" maps "+mapping.Delta+" to assumption "+basis+", which is "+node.State+"; reconcile the basis before it justifies "+fact.Kind+" "+fact.Subject)
					case node.Kind == "decision" && node.State == "withdrawn":
						blockers = append(blockers, "criterion "+id+" maps "+mapping.Delta+" to withdrawn decision "+basis)
					}
				}
				rationale := strings.Join(strings.Fields(strings.ToLower(mapping.Rationale)), " ")
				if rationale == "" {
					blockers = append(blockers, "criterion "+id+" maps "+mapping.Delta+" to "+basis+" without saying why; a resolvable link is complete, not accepted")
					continue
				}
				key := basis + "\x00" + rationale
				pasted[key] = append(pasted[key], mapping.Delta)
			case ReviewMappingAcceptedRisk:
				if integrityStructuralKinds[fact.Kind] {
					blockers = append(blockers, "criterion "+id+" accepts "+mapping.Delta+" ("+fact.Kind+" "+fact.Subject+") as a risk; an integrity fact is mapped or raised as a finding, never waived")
				}
				if !causalityOwnerRE.MatchString(mapping.Owner) || !causalityDateRE.MatchString(mapping.ReviewBy) {
					blockers = append(blockers, "criterion "+id+" accepts "+mapping.Delta+" as a risk without an @owner and a review-by date")
				}
			}
		}
		for key, deltas := range pasted {
			if len(deltas) >= causalityProportionalityLimit {
				sort.Strings(deltas)
				basis := strings.SplitN(key, "\x00", 2)[0]
				blockers = append(blockers, "criterion "+id+" justifies "+strconv.Itoa(len(deltas))+" facts ("+strings.Join(deltas, ", ")+
					") on "+basis+" with one pasted reason; give each its own reason or record a proportionality finding")
			}
		}
	}
	sort.Strings(blockers)
	return blockers
}
