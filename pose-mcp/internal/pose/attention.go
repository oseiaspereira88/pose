package pose

import "sort"

// Attention groups an obligation report for a reader (spec
// pose-state-attention). Grouping is presentation: it never changes an
// obligation, merges ids or decides what blocks — the effects already say
// that. Ordering inside a group is by id, not by urgency.
type Attention struct {
	// Incomplete is true when any producer was not current; the reader sees
	// this before anything that suggests continuing.
	Incomplete bool               `json:"incomplete"`
	Coverage   []ProducerCoverage `json:"coverage_limits,omitempty"`
	// NotUsed names the sources this project does not have, which no producer
	// needs to read (spec pose-fresh-install-doctor-is-clean).
	NotUsed []string `json:"not_used,omitempty"`
	// ForActor holds what requires the queried actor or role, or, without a
	// query, what waits on any person or role.
	ForActor []string `json:"for_actor"`
	// Blocking maps each phase to the obligations that restrict it.
	Blocking map[string][]string `json:"blocking"`
	// Gates are mandatory, non-actor obligations (evidence, dependency,
	// reconciliation) that restrict a phase.
	Gates []string `json:"gates"`
	// Residual is advisory debt; it never restricts a phase by default.
	Residual []string `json:"residual"`
}

// BuildAttention groups report for actor ("" for anyone).
func BuildAttention(report ObligationReport, actor string) Attention {
	a := Attention{Blocking: map[string][]string{}, ForActor: []string{}, Gates: []string{}, Residual: []string{}}
	for _, c := range report.Coverage {
		if c.NotUsed {
			// A source this project does not have owes nothing; it is listed
			// apart, never as a coverage gap.
			a.NotUsed = append(a.NotUsed, c.Producer)
			continue
		}
		if c.State != CoverageStateCurrent {
			a.Incomplete = true
			a.Coverage = append(a.Coverage, c)
		}
	}
	if !report.Snapshot.Coherent {
		a.Incomplete = true
	}
	for _, o := range report.Obligations {
		if o.Category == ObligationResidualDebt {
			a.Residual = append(a.Residual, o.ID)
			continue
		}
		forActor := false
		if actor == "" {
			forActor = o.Waiting == WaitingActor || o.Recipient.Principal != "" || o.Recipient.Role != ""
		} else {
			forActor = o.Recipient.Principal == actor || o.Recipient.Role == actor
		}
		if forActor {
			a.ForActor = append(a.ForActor, o.ID)
		} else {
			a.Gates = append(a.Gates, o.ID)
		}
		for _, phase := range []string{PhaseStart, PhaseExecution, PhaseReview, PhaseCloseout, PhaseRelease} {
			if o.Restricts(phase) {
				a.Blocking[phase] = append(a.Blocking[phase], o.ID)
			}
		}
	}
	for _, list := range [][]string{a.ForActor, a.Gates, a.Residual} {
		sort.Strings(list)
	}
	for phase := range a.Blocking {
		sort.Strings(a.Blocking[phase])
	}
	return a
}
