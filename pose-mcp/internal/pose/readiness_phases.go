package pose

import "sort"

// PhaseReadiness says, for one phase of one spec, what restricts it and how
// much the engine could see (spec pose-phase-scoped-readiness). It never
// changes the legacy Readiness.Ready, whose meaning consumers already rely on.
type PhaseReadiness struct {
	Phase string `json:"phase"`
	// State is clear, restricted, partially-restricted or unknown. Clear
	// needs every producer relevant to the phase to have been read; a phase
	// with no known restriction and a producer that was not read is unknown,
	// never clear.
	State       string             `json:"state"`
	Restricting []PhaseRestriction `json:"restricting,omitempty"`
	// Unread names the producers relevant to this phase that were not read.
	Unread []string `json:"unread,omitempty"`
	Note   string   `json:"note,omitempty"`
}

// PhaseRestriction is one obligation restricting the phase, over a scope.
// An empty scope is the whole spec.
type PhaseRestriction struct {
	Obligation string    `json:"obligation"`
	Category   string    `json:"category"`
	ReasonCode string    `json:"reason_code"`
	Scope      []NodeRef `json:"scope,omitempty"`
}

// SpecPhases is the per-phase projection of one spec.
type SpecPhases struct {
	SchemaVersion int                `json:"schema_version"`
	Spec          string             `json:"spec"`
	Status        string             `json:"status"`
	Phases        []PhaseReadiness   `json:"phases"`
	Snapshot      ObligationSnapshot `json:"snapshot"`
}

// Phase states.
const (
	PhaseClear               = "clear"
	PhaseRestricted          = "restricted"
	PhasePartiallyRestricted = "partially-restricted"
	PhaseUnknown             = "unknown"
)

// producerPhases says which producers can restrict which phase, so a phase
// is judged only by the producers that could have restricted it.
var producerPhases = map[string][]string{
	"readiness":          {PhaseStart, PhaseExecution},
	"closeout":           {PhaseCloseout},
	"review":             {PhaseReview, PhaseCloseout},
	"start":              {PhaseCloseout},
	"assessments":        {},
	"action-requests":    {PhaseStart, PhaseExecution, PhaseReview, PhaseCloseout, PhaseRelease},
	"release":            {PhaseRelease},
	"docs-review":        {PhaseCloseout},
	"capability-trigger": {PhaseCloseout},
	"findings":           {PhaseCloseout, PhaseRelease},
}

// SpecPhaseReadiness projects the phases of one spec from its obligations.
func (s Store) SpecPhaseReadiness(slug string) (SpecPhases, error) {
	spec, err := s.GetSpec(slug)
	if err != nil {
		return SpecPhases{}, err
	}
	report, err := s.ProjectObligations(ObligationQuery{Scope: "spec:" + slug})
	if err != nil {
		return SpecPhases{}, err
	}
	return PhasesFromReport(spec, report), nil
}

func PhasesFromReport(spec *Spec, report ObligationReport) SpecPhases {
	out := SpecPhases{SchemaVersion: 1, Spec: spec.Slug, Status: spec.Status, Snapshot: report.Snapshot}
	unread := map[string][]string{}
	for _, c := range report.Coverage {
		if c.State == CoverageStateCurrent {
			continue
		}
		// Not adopted is not unread: an unsupported start reconciliation
		// restricts nothing because there is nothing to reconcile.
		if c.Producer == "start" && c.State == CoverageStateUnsupported {
			continue
		}
		for _, phase := range producerPhases[c.Producer] {
			unread[phase] = append(unread[phase], c.Producer)
		}
	}
	for _, phase := range []string{PhaseStart, PhaseExecution, PhaseReview, PhaseCloseout, PhaseRelease} {
		p := PhaseReadiness{Phase: phase}
		wholeSpec := false
		for _, o := range report.Obligations {
			if !o.Restricts(phase) {
				continue
			}
			var scope []NodeRef
			for _, effect := range o.Effects {
				if effect.Phase == phase && effect.Mode == EffectBlock {
					scope = append(scope, effect.Scope...)
				}
			}
			if len(scope) == 0 {
				wholeSpec = true
			}
			p.Restricting = append(p.Restricting, PhaseRestriction{Obligation: o.ID, Category: o.Category, ReasonCode: o.ReasonCode, Scope: scope})
		}
		sort.Slice(p.Restricting, func(i, j int) bool { return p.Restricting[i].Obligation < p.Restricting[j].Obligation })
		p.Unread = uniqueSorted(unread[phase])
		switch {
		case wholeSpec:
			p.State = PhaseRestricted
		case len(p.Restricting) > 0:
			// Restricted on named nodes only. That is what is known; it is
			// not evidence that the rest is independent of those nodes.
			p.State = PhasePartiallyRestricted
			p.Note = "restricted on the named scope; no direct restriction was found on the rest of the spec in the consulted sources, and independence from the restricted scope is not demonstrated"
		case len(p.Unread) > 0:
			p.State = PhaseUnknown
			p.Note = "no known restriction, but a producer that could restrict this phase was not read"
		default:
			p.State = PhaseClear
			p.Note = "no restriction found and every producer relevant to this phase was read"
		}
		out.Phases = append(out.Phases, p)
	}
	return out
}

// PhaseRestricts reports whether the phase carries any restriction, whole or
// partial.
func (p SpecPhases) PhaseRestricts(phase string) (PhaseReadiness, bool) {
	for _, candidate := range p.Phases {
		if candidate.Phase == phase {
			return candidate, candidate.State == PhaseRestricted || candidate.State == PhasePartiallyRestricted
		}
	}
	return PhaseReadiness{}, false
}
