package pose

import (
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"time"
)

var (
	ErrActionRecoveryPending   = errors.New("action-recovery-pending")
	ErrActionIntentConflict    = errors.New("action-intent-conflict")
	ErrActionRecoveryUnknown   = errors.New("action-recovery-unknown")
	ErrActionMaterialUnchanged = errors.New("action-material-unchanged")
)

type ActionNextStep struct {
	Kind      string   `json:"kind"`
	Operation string   `json:"operation,omitempty"`
	RequestID string   `json:"request_id"`
	Reason    string   `json:"reason"`
	Required  []string `json:"required,omitempty"`
}

type ActionRefusal struct {
	Code   string `json:"code"`
	Reason string `json:"reason"`
}
type ActionPreflight struct {
	Preview        bool              `json:"preview"`
	Admissible     bool              `json:"admissible"`
	Acceptance     string            `json:"acceptance"`
	ProofPending   bool              `json:"proof_pending"`
	AuditPersisted bool              `json:"audit_persisted"`
	Request        ActionRequestView `json:"request"`
	Refusal        *ActionRefusal    `json:"refusal,omitempty"`
}

func actionRefusal(err error) *ActionRefusal {
	if err == nil {
		return nil
	}
	code := "action-resolution-invalid"
	for _, candidate := range []error{ErrActionNotAuthorized, ErrActionInvalidAnswer, ErrActionStaleDigest, ErrActionRevisionConflict, ErrActionIdempotencyConflict, ErrActionForeignProject, ErrActionNotOpen, ErrActionVerificationFailed, ErrActionRecoveryPending, ErrActionIntentConflict, ErrActionRecoveryUnknown, ErrActionMaterialUnchanged} {
		if errors.Is(err, candidate) {
			code = candidate.Error()
			break
		}
	}
	return &ActionRefusal{Code: code, Reason: err.Error()}
}
func actionProofPending(p ActionPolicy, r ActionResolution) bool {
	none := r.Signature == "" && r.Claim == nil && r.Envelope == nil
	return none && (p.IdentityAssurance == ReviewIdentityAssuranceVerified || r.Channel == "mcp") || r.Signature == "" && p.RequirePresence && strings.HasPrefix(r.Actor, "human:") && p.PresenceScope == ActionPresenceUniversal
}
func actionResolutionStateAdmitted(v ActionRequestView, typ string) bool {
	if v.State == ActionStateOpen {
		return true
	}
	if typ == ActionEventWaived && v.State == ActionStateFulfilled && v.Satisfaction == SatisfactionInvalidated {
		// Like an answer that no longer satisfies, a fulfilment for a bundle
		// the scope no longer has can be waived by its authority: the
		// retirement for when a person already answers the new bundle
		// (review round 6, R6-F02).
		return true
	}
	return typ == ActionEventWaived && v.State == ActionStateAnswered && v.Satisfaction == SatisfactionPending && !v.SubjectUnknown
}

// PreflightActionResolution reads the same domain preparation as apply. It
// never calls a signer, takes a lock or records a refusal event.
func (s Store) PreflightActionResolution(res ActionResolution, now time.Time) (ActionPreflight, error) {
	out := ActionPreflight{Preview: true, Acceptance: "not-certified", AuditPersisted: false}
	v, err := s.LoadActionRequest(res.RequestID)
	out.Request = v
	if err == nil {
		_, _, out.ProofPending, err = s.prepareActionResolutionAgainst(v, res, now, true)
	}
	out.Admissible = err == nil
	out.Refusal = actionRefusal(err)
	return out, err
}

func resolutionFromEvent(e ActionEvent, revision int) ActionResolution {
	r := ActionResolution{RequestID: "", Type: e.Type, Actor: e.Actor, Execution: e.Execution, Channel: e.Channel, Role: e.Role, RequestDigest: e.RequestDigest, ExpectedRevision: revision, Answer: e.Answer, Evidence: e.Evidence, Reason: e.Reason, PreparedBy: e.PreparedBy, AppliedBy: e.AppliedBy, ConfirmationMode: e.ConfirmationMode, IdempotencyKey: e.IdempotencyKey, Claim: e.Claim, Envelope: e.Envelope}
	if e.SSHSignature != nil {
		r.Signature = e.SSHSignature.Signature
	}
	return r
}

// Proof renewal does not create a new semantic decision. All effect and actor
// fields remain part of the idempotency identity.
func sameActionIntent(a, b ActionEvent) bool {
	strip := func(e ActionEvent) ActionEvent {
		e.Seq, e.SchemaVersion = 0, 0
		e.At, e.Assurance = "", ""
		e.Claim, e.Envelope, e.SSHSignature = nil, nil, nil
		return e
	}
	l, _ := json.Marshal(strip(a))
	r, _ := json.Marshal(strip(b))
	return string(l) == string(r)
}

func nextActionStep(v ActionRequestView) *ActionNextStep {
	step := &ActionNextStep{RequestID: v.Request.ID}
	switch {
	case v.Recovery != nil && v.Recovery.Phase != "complete":
		step.Kind, step.Operation, step.Reason = "resume-supersession", "supersede", "resume the same prepared successor with current predecessor authority"
		if v.Recovery.Phase == "repair-required" {
			step.Kind, step.Operation, step.Reason = "repair", "", v.Recovery.Reason
		}
		step.Required = []string{"prepared-successor-identity", "predecessor-revision-and-digest", "current-authority-and-proof"}
	case v.SubjectUnknown:
		step.Kind, step.Reason = "repair-subject", "restore or explicitly replace the missing material subject before resolving the condition"
		step.Required = []string{"readable-material-subject", "replacement-material-intent"}
	case v.Request.Kind == ActionDelegatedReview && v.State == ActionStateFulfilled && v.Satisfaction == SatisfactionInvalidated:
		// A request fulfilled for a bundle the scope no longer has is asked
		// again for the new one, linked to it (spec
		// pose-delegated-review-action-fulfillment R8; review round 3).
		step.Kind, step.Operation = "replace-request", "supersede"
		step.Reason = "the scope or the policies changed since this request was fulfilled; ask the new bundle with pose review request " + v.Request.Review.Scope + " --kind delegated-review --supersedes " + v.Request.ID +
			", or, when a person's escalation already asks it, waive this one with its authority (pose action waive " + v.Request.ID + ")"
		step.Required = []string{"new-current-bundle", "same-requester-or-authority"}
	case v.State == ActionStateInvalidated:
		step.Kind, step.Operation, step.Reason = "replace-request", "supersede", "open a materially updated request linked to this predecessor; do not answer its old digest"
		step.Required = []string{"new-material-question", "current-subject", "predecessor-authority"}
	case v.State == ActionStateAnswered && v.Satisfaction == SatisfactionPending:
		step.Kind, step.Operation, step.Reason = "remediate-or-waive", "supersede", "remediate or materially replace the declined condition, or obtain an authorized waiver; do not repeat it to seek approval"
		step.Required = []string{"material-remediation-or-waiver-reason", "authorized-recipient-or-role"}
	case v.State == ActionStateOpen && v.Request.Kind == ActionDelegatedReview:
		step.Kind, step.Operation, step.Reason = "fulfil", ActionEventFulfilled, "run the requested brief with an accepted agent, have the engine record its conclusion, and submit the run with pose action fulfil"
		step.Required = []string{"completed-run-of-the-brief", "recorded-conclusion", "request-digest-and-revision", "agent-runner"}
	case v.State == ActionStateOpen:
		step.Kind, step.Operation, step.Reason = "answer", "answered", "obtain an admitted answer from the authorized recipient for this request digest and revision"
		step.Required = []string{"authorized-actor", "admitted-answer", "request-digest-and-revision", "proof-if-required"}
	default:
		return nil
	}
	return step
}

func cloneActionNextStep(step *ActionNextStep) *ActionNextStep {
	if step == nil {
		return nil
	}
	out := *step
	out.Required = slices.Clone(step.Required)
	return &out
}
