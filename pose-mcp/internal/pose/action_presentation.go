package pose

import "sort"

// ActionPresentation groups open requests for one conversation with a person
// (spec pose-action-request-presentation). Grouping is presentation only:
// every request keeps its id, its authority and its own answer, and the
// content shown is exactly what a resolution binds to.
type ActionPresentation struct {
	Groups []ActionGroup `json:"groups"`
	// InterruptNow counts requests that restrict start or execution; the
	// rest can wait until the work reaches the phase they restrict.
	InterruptNow int `json:"interrupt_now"`
	CanWait      int `json:"can_wait"`
}

// ActionGroup is the requests of one origin that first bite at the same
// phase.
type ActionGroup struct {
	Origin        string                   `json:"origin"`
	EarliestPhase string                   `json:"earliest_phase"`
	Interrupt     bool                     `json:"interrupt"`
	Requests      []ActionPresentationItem `json:"requests"`
}

// ActionPresentationItem is the bound content of one request.
type ActionPresentationItem struct {
	ID            string          `json:"id"`
	Kind          string          `json:"kind"`
	Question      string          `json:"question"`
	Context       string          `json:"context,omitempty"`
	Options       []ActionOption  `json:"options"`
	Recommend     string          `json:"recommendation,omitempty"`
	Recipient     ObligationActor `json:"recipient"`
	Restricts     []string        `json:"restricts"`
	RequestDigest string          `json:"request_digest"`
	Revision      int             `json:"revision"`
}

var phaseOrder = map[string]int{PhaseStart: 0, PhaseExecution: 1, PhaseReview: 2, PhaseCloseout: 3, PhaseRelease: 4}

// PresentActionRequests groups the requests still waiting for an answer.
// Answered, cancelled, waived, superseded and invalidated requests are not
// asked again; a request whose answer was invalidated by a changed subject
// is, because its condition is unmet again.
func PresentActionRequests(views []ActionRequestView, actor string) ActionPresentation {
	out := ActionPresentation{Groups: []ActionGroup{}}
	groups := map[string]*ActionGroup{}
	var keys []string
	for _, view := range views {
		r := view.Request
		waiting := view.State == ActionStateOpen || (view.State == ActionStateInvalidated && view.Satisfaction == SatisfactionInvalidated && view.Answer != "")
		if !waiting {
			continue
		}
		if actor != "" && r.Recipient.Principal != actor && r.Recipient.Role != actor {
			continue
		}
		earliest := ""
		var restricts []string
		for _, effect := range r.Effects {
			if effect.Mode != EffectBlock {
				continue
			}
			restricts = append(restricts, effect.Phase)
			if earliest == "" || phaseOrder[effect.Phase] < phaseOrder[earliest] {
				earliest = effect.Phase
			}
		}
		if earliest == "" {
			earliest = PhaseRelease
		}
		sort.Slice(restricts, func(i, j int) bool { return phaseOrder[restricts[i]] < phaseOrder[restricts[j]] })
		key := r.Origin + "|" + earliest
		group, ok := groups[key]
		if !ok {
			group = &ActionGroup{Origin: r.Origin, EarliestPhase: earliest, Interrupt: earliest == PhaseStart || earliest == PhaseExecution}
			groups[key] = group
			keys = append(keys, key)
		}
		group.Requests = append(group.Requests, ActionPresentationItem{ID: r.ID, Kind: r.Kind, Question: r.Question, Context: r.Context, Options: r.Options,
			Recommend: r.Recommend, Recipient: r.Recipient, Restricts: restricts, RequestDigest: r.RequestDigest, Revision: view.Revision})
		if group.Interrupt {
			out.InterruptNow++
		} else {
			out.CanWait++
		}
	}
	sort.Slice(keys, func(i, j int) bool {
		a, b := groups[keys[i]], groups[keys[j]]
		if a.Interrupt != b.Interrupt {
			return a.Interrupt
		}
		if phaseOrder[a.EarliestPhase] != phaseOrder[b.EarliestPhase] {
			return phaseOrder[a.EarliestPhase] < phaseOrder[b.EarliestPhase]
		}
		return a.Origin < b.Origin
	})
	for _, key := range keys {
		out.Groups = append(out.Groups, *groups[key])
	}
	return out
}
