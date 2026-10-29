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
	EscalationPersonRequired    = "person-required"
	EscalationBudgetExhausted   = "budget-exhausted"
	EscalationConflict          = "reviewers-in-conflict"
	EscalationBusiness          = "business-decision"
	EscalationAgentsUnavailable = "agents-unavailable"
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
	if cause == EscalationAgentsUnavailable && r.Review != nil {
		// No service request was persisted for an unavailable route. Its
		// temporary request ID includes the current timestamp, so use the
		// actual review material rather than asking again every second.
		return r.Review.Scope + "@" + r.Review.BundleID + "#" + r.Review.Purpose + "@" + r.Review.PolicyDigest
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
		options = defaultEscalationOptions()
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
	for _, v := range existing {
		if sameEscalation(v, marker, prepared) {
			return v.Request.ID, nil
		}
	}
	view, err := s.OpenActionRequest(asked, now)
	if err != nil {
		return "", fmt.Errorf("pose: the %s escalation of %s cannot be opened: %w", cause, key, err)
	}
	return view.Request.ID, nil
}

// PrepareDelegatedReviewReplacing is the decision request cause asks of a
// person as the successor of request supersedes, and the authority that
// retires the predecessor — the one given, or, for role, the declared
// authority bound to this successor — without writing: what a preview shows
// and a proof is signed over (review round 5, R5-F02).
func (s Store) PrepareDelegatedReviewReplacing(r ActionRequest, cause, question, context string, options []ActionOption, supersedes string, authority *ActionResolution, role string, now time.Time) (ActionRequest, *ActionResolution, error) {
	marker := escalationMarker(escalationKey(r, cause), cause)
	if len(options) < 2 {
		options = defaultEscalationOptions()
	}
	successor := ActionRequest{Origin: r.Origin, Kind: ActionDecision, Question: question, Context: strings.TrimSpace(marker + "\n" + context),
		Options: options, RequestedBy: r.RequestedBy, Recipient: ObligationActor{Role: "maintainer"}, Targets: r.Targets, Effects: r.Effects,
		Supersedes: supersedes, RequestedAt: r.RequestedAt}
	prepared, err := s.PrepareActionRequest(successor, now)
	if err != nil {
		return ActionRequest{}, nil, err
	}
	switch {
	case role != "" && authority == nil:
		if authority, err = s.RoleSupersessionAuthority(prepared, r.RequestedBy.Principal, role); err != nil {
			return ActionRequest{}, nil, err
		}
	case role != "":
		authority.Role = role
	}
	return prepared, authority, nil
}

// EscalateDelegatedReviewReplacing opens the decision request cause asks of
// a person as the successor of request supersedes, retiring it in the same
// step under the supersession authority rules (review round 4, R4-F02). An
// equivalent escalation already open is not asked twice: the call is
// refused, naming it (review round 5, R5-F04). It is EscalateDelegatedReview
// when there is nothing to replace.
func (s Store) EscalateDelegatedReviewReplacing(r ActionRequest, cause, question, context string, options []ActionOption, supersedes string, authority *ActionResolution, role string, now time.Time) (string, error) {
	if supersedes == "" {
		return s.EscalateDelegatedReview(r, cause, question, context, options, now)
	}
	marker := escalationMarker(escalationKey(r, cause), cause)
	unlock, err := s.lockPrivateName("escalation-locks", digestBytes([]byte(marker)))
	if err != nil {
		return "", err
	}
	defer unlock()
	prepared, authority, err := s.PrepareDelegatedReviewReplacing(r, cause, question, context, options, supersedes, authority, role, now)
	if err != nil {
		return "", fmt.Errorf("pose: the %s escalation replacing %s cannot be opened: %w", cause, supersedes, err)
	}
	// An inventory that cannot be read cannot show the escalation is not
	// already asked (review round 6, R6-F01).
	existing, err := s.ListActionRequests()
	if err != nil {
		return "", fmt.Errorf("pose: the %s escalation replacing %s cannot be opened: the action requests cannot be read: %w", cause, supersedes, err)
	}
	for _, v := range existing {
		if v.Request.ID != prepared.ID && sameEscalation(v, marker, prepared) {
			return "", fmt.Errorf("pose: escalation %s already asks this of a person; answer it, and waive %s with its authority (pose action waive %s)", v.Request.ID, supersedes, supersedes)
		}
	}
	view, err := s.OpenActionRequestAuthorized(prepared, authority, now)
	if err != nil {
		return "", fmt.Errorf("pose: the %s escalation replacing %s cannot be opened: %w", cause, supersedes, err)
	}
	return view.Request.ID, nil
}

// sameEscalation reports whether v asks what prepared asks: the same cause
// and key, question, authority, options, targets and effects, and is still
// open or answered.
func sameEscalation(v ActionRequestView, marker string, prepared ActionRequest) bool {
	if v.Request.Kind != ActionDecision || !strings.HasPrefix(v.Request.Context, marker) || strings.TrimSpace(v.Request.Question) != strings.TrimSpace(prepared.Question) {
		return false
	}
	if v.State == ActionStateCancelled || v.State == ActionStateSuperseded || v.State == ActionStateInvalidated {
		return false
	}
	x, _ := json.Marshal([]any{v.Request.Recipient, v.Request.Options, v.Request.Targets, v.Request.Effects})
	y, _ := json.Marshal([]any{prepared.Recipient, prepared.Options, prepared.Targets, prepared.Effects})
	return string(x) == string(y)
}

func defaultEscalationOptions() []ActionOption {
	return []ActionOption{
		{ID: "person-reviews", Consequence: "a person reviews the scope and records the attestation; the delegated request is waived"},
		{ID: "keep-blocked", Consequence: "the scope stays blocked until the bundle changes"},
	}
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

// RoleSupersessionAuthority is the declared authority with which actor, in
// role, retires the predecessor of prepared: bound to the predecessor's
// digest and revision and to the successor, as a supersession requires.
// Proof, when the policy asks for it, is still asked by the domain.
func (s Store) RoleSupersessionAuthority(prepared ActionRequest, actor, role string) (*ActionResolution, error) {
	prior, err := s.LoadActionRequest(prepared.Supersedes)
	if err != nil {
		return nil, err
	}
	// Retired by this very successor already: a replay presents the
	// authority recorded with the retirement (review round 5, R5-F01).
	if prior.State == ActionStateSuperseded && len(prior.Events) > 0 {
		if last := prior.Events[len(prior.Events)-1]; last.SupersededBy == prepared.ID && last.SuccessorDigest == prepared.RequestDigest {
			res := lifecycleResolution(last)
			res.RequestID = prepared.Supersedes
			return &res, nil
		}
	}
	return &ActionResolution{Type: ActionEventSuperseded, RequestID: prepared.Supersedes, Actor: actor, Role: role, Channel: "cli",
		RequestDigest: prior.Request.RequestDigest, ExpectedRevision: prior.Revision, IdempotencyKey: "supersede:" + prepared.ID,
		Reason: "superseded by " + prepared.ID, SuccessorID: prepared.ID, SuccessorDigest: prepared.RequestDigest}, nil
}
