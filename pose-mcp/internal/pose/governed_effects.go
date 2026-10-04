package pose

import "fmt"

// AgencyReadinessPolicyVersion adopts governed effects (spec
// pose-governed-effect-enforcement): once adopted, an unsatisfied action
// request that restricts start, closeout or release refuses that transition
// at the domain write point, whichever surface calls it. An instance that
// does not adopt keeps its documented behaviour; an engine update never adds
// the gate on its own.
const AgencyReadinessPolicyVersion = 1

// AgencyReadinessAdopted reads the review policy's capability.
func (s Store) AgencyReadinessAdopted() (bool, error) {
	policy, err := s.GetReviewPolicy()
	if err != nil {
		return false, err
	}
	return policy.AgencyReadinessVersion == AgencyReadinessPolicyVersion, nil
}

// GovernedEffectRestrictions returns the action-request obligations that
// restrict phase for the given specs (all specs when slugs is empty). It is
// recomputed at every call: a write gate never trusts an earlier read.
//
// Only action requests are enforced here. Dependencies, review and evidence
// already have their own gates at these points, and the projection is not a
// second policy engine on top of them.
func (s Store) GovernedEffectRestrictions(phase string, slugs []string) ([]Obligation, error) {
	if actionRequestObligationSource == nil {
		return nil, nil
	}
	all, err := s.ListSpecs("", "")
	if err != nil {
		return nil, err
	}
	want := map[string]bool{}
	for _, slug := range slugs {
		want[slug] = true
	}
	var specs []Spec
	for _, sp := range all {
		if len(slugs) == 0 || want[sp.Slug] {
			specs = append(specs, sp)
		}
	}
	project := s.CurrentObligationSnapshot().Project
	obligations, err := actionRequestObligationSource(s, project, specs)
	if err != nil {
		return nil, err
	}
	var out []Obligation
	for _, o := range obligations {
		if o.Restricts(phase) {
			out = append(out, o)
		}
	}
	return out, nil
}

// governedEffectDiagnostic renders one restriction as the typed cause the
// refusing surface reports.
func governedEffectDiagnostic(o Obligation, phase, ref string) Diagnostic {
	return NewDiagnostic("action-request-pending", fmt.Sprintf("%s restricts %s: %s", o.Source.Detail, phase, o.Message), ref, o.Source.Detail)
}

// ReleaseGovernedRefusal is the refusal a release write reports when an
// adopted, unsatisfied action request restricts the release of one of its
// specs. Empty when the capability is not adopted or nothing restricts.
func (s Store) ReleaseGovernedRefusal(specs []string) ([]Diagnostic, error) {
	adopted, err := s.AgencyReadinessAdopted()
	if err != nil || !adopted {
		return nil, err
	}
	restrictions, err := s.GovernedEffectRestrictions(PhaseRelease, specs)
	if err != nil {
		return nil, err
	}
	var out []Diagnostic
	for _, o := range restrictions {
		out = append(out, governedEffectDiagnostic(o, PhaseRelease, "release"))
	}
	return out, nil
}
