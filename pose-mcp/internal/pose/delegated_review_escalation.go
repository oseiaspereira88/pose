package pose

// People by exception (spec pose-delegated-review-action-fulfillment R7).
//
// A delegated run never satisfies what policy reserves for a person, and
// what a run cannot settle goes to a person as a question ready to answer:
// a scope whose plan requires a person's review, an attempt budget spent
// without the review, reviewers in conflict on one bundle, and a business
// decision a reviewer raised. Each opens one decision request — with the
// question, its options and their consequences, its context and the
// authority that answers it — addressed to the maintainer role, restricting
// what the delegated-review request restricts. Opening it again returns the
// one already open, so a retry never asks twice.

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// Escalation causes.
const (
	EscalationPersonRequired  = "person-required"
	EscalationBudgetExhausted = "budget-exhausted"
	EscalationConflict        = "reviewers-in-conflict"
	EscalationBusiness        = "business-decision"
)

// escalationMarker opens the context of an escalation, so the engine finds
// the one it opened: the request, or for a review reserved to a person the
// scope and bundle, which no request exists for yet, and the cause.
func escalationMarker(key, cause string) string {
	return "escalates " + key + " (" + cause + ")"
}

func escalationKey(r ActionRequest, cause string) string {
	if cause == EscalationPersonRequired && r.Review != nil {
		return r.Review.Scope + "@" + r.Review.BundleID
	}
	return r.ID
}

// EscalateDelegatedReview opens the decision request cause asks of a person
// about delegated-review request r, or returns the one that already asks
// the same question — open or answered — under a lock, so neither a retry
// nor a concurrent call asks twice; a different question is a new one
// (review round 1, F04, F10).
func (s Store) EscalateDelegatedReview(r ActionRequest, cause, question, context string, options []ActionOption, now time.Time) (string, error) {
	key := escalationKey(r, cause)
	marker := escalationMarker(key, cause)
	unlock, err := s.lockPrivateName("escalation-locks", digestBytes([]byte(marker)))
	if err != nil {
		return "", err
	}
	defer unlock()
	existing, err := s.ListActionRequests()
	if err != nil {
		return "", err
	}
	if len(options) < 2 {
		options = []ActionOption{
			{ID: "person-reviews", Consequence: "a person reviews the scope and records the attestation; the delegated request is waived"},
			{ID: "keep-blocked", Consequence: "the scope stays blocked until the bundle changes"},
		}
	}
	asked := ActionRequest{Origin: r.Origin, Kind: ActionDecision, Question: question, Context: strings.TrimSpace(marker + "\n" + context),
		Options: options, RequestedBy: ActionPrincipal{Principal: "agent:pose-engine"}, Recipient: ObligationActor{Role: "maintainer"},
		Targets: r.Targets, Effects: r.Effects}
	prepared, err := s.PrepareActionRequest(asked, now)
	if err != nil {
		return "", fmt.Errorf("pose: the %s escalation of %s cannot be opened: %w", cause, key, err)
	}
	// The same escalation asks the same question of the same authority with
	// the same options, about the same targets, with the same effects; any
	// other is a new one, so nothing asked is dropped (review round 2,
	// R2-F03).
	same := func(a, b ActionRequest) bool {
		x, _ := json.Marshal([]any{a.Recipient, a.Options, a.Targets, a.Effects})
		y, _ := json.Marshal([]any{b.Recipient, b.Options, b.Targets, b.Effects})
		return string(x) == string(y)
	}
	for _, v := range existing {
		if v.Request.Kind == ActionDecision && strings.HasPrefix(v.Request.Context, marker) && strings.TrimSpace(v.Request.Question) == strings.TrimSpace(question) &&
			same(v.Request, prepared) && v.State != ActionStateCancelled && v.State != ActionStateSuperseded && v.State != ActionStateInvalidated {
			return v.Request.ID, nil
		}
	}
	view, err := s.OpenActionRequest(asked, now)
	if err != nil {
		return "", fmt.Errorf("pose: the %s escalation of %s cannot be opened: %w", cause, key, err)
	}
	return view.Request.ID, nil
}

// PersonRequiredByPlan reports whether the plan of the request's bundle
// reserves the review for a person: no delegated run satisfies it.
func (s Store) PersonRequiredByPlan(m ActionReviewRequest) bool {
	bundle, err := s.LoadReviewBundle(m.BundleID)
	return err == nil && bundle.Payload.Plan.Independence == "mandatory-human"
}

// fulfillmentEscalations are the questions a verified fulfilment raises for
// a person: reviewers whose conclusions on the bundle disagree, and a
// business decision the run's reviewer asked for.
func (s Store) fulfillmentEscalations(r ActionRequest, f ActionFulfillment, now time.Time) ([]string, error) {
	var ids []string
	run, err := s.LoadProvenanceReviewRun(f.RunID)
	if err != nil {
		return nil, err
	}
	if draft, _ := ParseReviewDraft(run.Output); draft != nil && draft.BusinessDecision != nil {
		id, err := s.EscalateDelegatedReview(r, EscalationBusiness, draft.BusinessDecision.Question,
			"Raised by the reviewer of run "+run.RunID+" ("+run.Adapter+"); the run concluded "+f.Decision+".", draft.BusinessDecision.Options, now)
		if err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	if conflict := s.conflictingConclusions(run, f.Decision); conflict != "" {
		id, err := s.EscalateDelegatedReview(r, EscalationConflict, "Delegated reviewers disagree on "+r.Review.Scope+": which conclusion stands?",
			conflict, []ActionOption{
				{ID: "uphold-rejection", Consequence: "the rejection stands and the scope stays blocked until the bundle changes"},
				{ID: "person-reviews", Consequence: "a person reviews the scope and records the attestation that stands"},
			}, now)
		if err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}

// conflictingConclusions names the recorded conclusions of other runs on
// run's bundle whose decision differs from decision, or "".
func (s Store) conflictingConclusions(run ReviewRun, decision string) string {
	runs, _, err := s.ReadReviewRuns("")
	if err != nil {
		return ""
	}
	approving := decision == "approved"
	var others []string
	for _, other := range runs {
		if other.RunID == run.RunID || other.BundleID != run.BundleID || other.Kind != run.Kind {
			continue
		}
		c, err := s.ReadReviewConclusion(other.RunID)
		if err != nil || c == nil || c.State != ReviewConclusionRecorded {
			continue
		}
		if (c.Decision == "approved") != approving {
			others = append(others, other.RunID+" ("+other.Adapter+") concluded "+c.Decision)
		}
	}
	if len(others) == 0 {
		return ""
	}
	return "Run " + run.RunID + " (" + run.Adapter + ") concluded " + decision + "; " + strings.Join(others, "; ") + "."
}
