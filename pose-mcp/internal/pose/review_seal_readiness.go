package pose

// Seal readiness (spec pose-review-readiness-before-seal R1, R2): a new seal
// refuses what `pose close` and the conclusion preflight would refuse only
// after a full review round, using the same rules. A bundle already sealed
// is returned as it was and never re-judged here.

import "sort"

func (s Store) sealReadinessBlockers(bundle ReviewBundle) []string {
	scope, err := ParseScopeRef(bundle.Payload.Scope.Ref)
	if err != nil || scope.Kind != "spec" {
		return nil
	}
	var blockers []string
	// R1: the delivery declaration `pose close` requires, read from the same
	// index findings the delivery assurance gate reports.
	if policy, err := LoadDeliveryPolicyIn(s.fsys(), s.Root); err == nil && policy.Enabled {
		if graph, err := s.GetDeliveryIntegrity(""); err == nil {
			for _, finding := range graph.Findings {
				if finding.Code != "undeclared-delivery" || finding.Spec != scope.Slug || finding.Severity != "error" && finding.Severity != "critical" {
					continue
				}
				msg := "undeclared-delivery [" + finding.Path + "]: " + finding.Message
				if finding.Remediation != "" {
					msg += " (" + finding.Remediation + ")"
				}
				blockers = append(blockers, msg+"; pose close refuses it after the review")
			}
		}
	}
	// R2: a structural fact is mapped to a decision basis, and the preflight
	// refuses a basis that reaches no requirement or constraint. When the plan
	// asks for mappings, every declared decision must be usable as one.
	if structure := bundle.Payload.Plan.Structure; structure != nil && len(structure.Material) > 0 {
		ns := s.reviewDesignBasisNamespace(scope)
		var unreached []string
		for id := range ns.Known {
			if !ns.Reaches[id] {
				unreached = append(unreached, id)
			}
		}
		sort.Strings(unreached)
		for _, id := range unreached {
			blockers = append(blockers, "decision basis "+id+" reaches no requirement or constraint: give it a `- Basis:` line naming one, or the conclusion preflight refuses a mapping to it")
		}
	}
	return uniqueSorted(blockers)
}
