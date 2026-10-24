package pose

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// ActionLifecycleNamespace prevents an answer signature from authorizing retirement.
const ActionLifecycleNamespace = "pose-action-lifecycle"

// ActionLifecycleStatement binds the exact effect and predecessor revision.
type ActionLifecycleStatement struct {
	SchemaVersion    int    `json:"schema_version"`
	Namespace        string `json:"namespace"`
	Project          string `json:"project"`
	RequestID        string `json:"request_id"`
	RequestDigest    string `json:"request_digest"`
	Operation        string `json:"operation"`
	Principal        string `json:"principal"`
	ExpectedRevision int    `json:"expected_revision"`
	IdempotencyKey   string `json:"idempotency_key"`
	Reason           string `json:"reason"`
	SuccessorID      string `json:"successor_id,omitempty"`
	SuccessorDigest  string `json:"successor_digest,omitempty"`
}

func (st ActionLifecycleStatement) Canonical() []byte {
	b, _ := json.Marshal(st)
	return append(b, '\n')
}
func LifecycleStatement(r ActionRequest, res ActionResolution) ActionLifecycleStatement {
	return ActionLifecycleStatement{1, ActionLifecycleNamespace, r.Project, r.ID, res.RequestDigest, res.Type, res.Actor,
		res.ExpectedRevision, res.IdempotencyKey, strings.TrimSpace(res.Reason), res.SuccessorID, res.SuccessorDigest}
}

func lifecycleEvent(res ActionResolution, now time.Time) ActionEvent {
	return ActionEvent{LifecycleVersion: 1, ExpectedRevision: res.ExpectedRevision, Type: res.Type,
		At: now.UTC().Format(time.RFC3339Nano), Actor: res.Actor, Execution: res.Execution, Channel: res.Channel,
		RequestDigest: res.RequestDigest, Reason: strings.TrimSpace(res.Reason), Role: res.Role,
		IdempotencyKey: res.IdempotencyKey, Claim: res.Claim, Envelope: res.Envelope,
		SupersededBy: res.SuccessorID, SuccessorDigest: res.SuccessorDigest,
		PreparedBy: res.PreparedBy, AppliedBy: res.AppliedBy, ConfirmationMode: res.ConfirmationMode}
}
func lifecycleResolution(e ActionEvent) ActionResolution {
	return ActionResolution{Type: e.Type, Actor: e.Actor, Channel: e.Channel, Role: e.Role, RequestDigest: e.RequestDigest,
		ExpectedRevision: e.ExpectedRevision, IdempotencyKey: e.IdempotencyKey, Reason: e.Reason,
		Claim: e.Claim, Envelope: e.Envelope, SuccessorID: e.SupersededBy, SuccessorDigest: e.SuccessorDigest}
}
func lifecycleAssurance(e ActionEvent) string {
	if e.LifecycleVersion == 1 && (e.SSHSignature != nil || e.Claim != nil || e.Envelope != nil) {
		return ReviewIdentityAssuranceVerified
	}
	return ReviewIdentityAssuranceDeclared
}
func lifecycleLimitations(e ActionEvent) []string {
	if e.LifecycleVersion == 0 {
		return []string{"legacy lifecycle event: operation authority was not recorded or verified"}
	}
	if lifecycleAssurance(e) != ReviewIdentityAssuranceVerified {
		return []string{"lifecycle identity and authority are declared, not verified"}
	}
	return nil
}

// admitLifecycle runs for both initial admission and the protected append.
func (s Store) admitLifecycle(r ActionRequest, e *ActionEvent, signature string, earliest, latest time.Time) error {
	return s.admitLifecycleMode(r, e, signature, earliest, latest, false)
}
func (s Store) admitLifecycleMode(r ActionRequest, e *ActionEvent, signature string, earliest, latest time.Time, allowMissingProof bool) error {
	fail := func(why string) error { return fmt.Errorf("%w: %s", ErrActionVerificationFailed, why) }
	if e.LifecycleVersion != 1 {
		return fail("unsupported lifecycle binding version")
	}
	if !validReviewPrincipal(e.Actor) {
		return ErrActionNotAuthorized
	}
	if e.RequestDigest != r.RequestDigest {
		return ErrActionStaleDigest
	}
	if e.ExpectedRevision < 1 || e.IdempotencyKey == "" || e.Reason == "" {
		return fail("lifecycle needs revision, idempotency key and reason")
	}
	policy, err := LoadActionPolicyIn(s.fsys(), s.Root)
	if err != nil {
		return err
	}
	switch e.Type {
	case ActionEventCancelled:
		if e.Actor != r.RequestedBy.Principal {
			return fmt.Errorf("%w: only the requester may withdraw its question; use waiver for a dispensation", ErrActionNotAuthorized)
		}
		e.Role = ""
	case ActionEventWaived:
		role, err := authorizeActionActor(policy, r, e.Actor, e.Role)
		if err != nil {
			return err
		}
		e.Role = role
	case ActionEventSuperseded:
		if e.SupersededBy == "" || e.SuccessorDigest == "" || e.SupersededBy == r.ID {
			return fail("supersession needs a distinct bound successor")
		}
		if e.Actor != r.RequestedBy.Principal {
			role, err := authorizeActionActor(policy, r, e.Actor, e.Role)
			if err != nil {
				return err
			}
			e.Role = role
		}
	default:
		return fmt.Errorf("%w: actor lifecycle operation %q is not admitted", ErrActionNotAuthorized, e.Type)
	}
	if e.Type != ActionEventSuperseded && (e.SupersededBy != "" || e.SuccessorDigest != "") {
		return fail("only supersession may bind a successor")
	}
	res := lifecycleResolution(*e)
	if signature != "" {
		sig, err := verifyActionSignature(policy, r, ActionResolution{Type: res.Type, Actor: res.Actor, RequestDigest: res.RequestDigest,
			ExpectedRevision: res.ExpectedRevision, IdempotencyKey: res.IdempotencyKey, Reason: res.Reason, SuccessorID: res.SuccessorID, SuccessorDigest: res.SuccessorDigest, Signature: signature})
		if err != nil {
			return err
		}
		e.SSHSignature = sig
	}
	if e.SSHSignature != nil {
		if err := VerifyRecordedActionSignature(r, *e); err != nil {
			return fail(err.Error())
		}
		registered := false
		for _, key := range policy.KeysFor(e.Actor) {
			if key.Fingerprint() == e.SSHSignature.Fingerprint {
				registered = true
			}
		}
		if !registered {
			return fail("lifecycle signing key is not currently registered to actor")
		}
	}
	if e.Claim != nil || e.Envelope != nil {
		if err := s.verifyActionClaimWithin(r, res, e.Role, earliest, latest); err != nil {
			return err
		}
	}
	proof := e.SSHSignature != nil || (e.Claim != nil && e.Envelope != nil)
	if err := policy.CheckPresenceMigration(e.Actor); err != nil {
		return err
	}
	if allowMissingProof && signature == "" && e.SSHSignature == nil {
		e.Assurance = ReviewIdentityAssuranceDeclared
		return nil
	}
	if (policy.IdentityAssurance == ReviewIdentityAssuranceVerified || e.Channel == "mcp") && !proof {
		return fail("this lifecycle operation needs proof, not a principal label")
	}
	if err := admitActionPresence(policy, e.Actor, e.SSHSignature, e.Claim != nil || e.Envelope != nil); err != nil {
		return err
	}
	e.Assurance = lifecycleAssurance(*e)
	return nil
}

// Reverify evidence historically without pretending current role/key membership
// establishes authorization at the recorded instant (same boundary as B01).
func (s Store) verifyLifecycleEvent(r ActionRequest, e ActionEvent) error {
	if e.Type != ActionEventCancelled && e.Type != ActionEventWaived && e.Type != ActionEventSuperseded {
		return nil
	}
	fail := func(why string) error {
		return fmt.Errorf("%w: lifecycle sequence %d: %s", ErrActionVerificationFailed, e.Seq, why)
	}
	if e.LifecycleVersion == 0 {
		policy, err := LoadActionPolicyIn(s.fsys(), s.Root)
		if err != nil {
			return err
		}
		if e.Assurance == ReviewIdentityAssuranceVerified || e.SSHSignature != nil || e.Claim != nil || e.Envelope != nil || policy.IdentityAssurance == ReviewIdentityAssuranceVerified {
			return fail("legacy lifecycle has no operation-bound proof; it cannot dispense a verified condition")
		}
		return nil
	}
	if e.LifecycleVersion != 1 || e.RequestDigest != r.RequestDigest || e.ExpectedRevision != e.Seq-1 || e.Reason == "" || e.IdempotencyKey == "" {
		return fail("invalid operation binding or predecessor revision")
	}
	if !validReviewPrincipal(e.Actor) {
		return fail("invalid actor")
	}
	if e.Type == ActionEventCancelled && e.Actor != r.RequestedBy.Principal {
		return fail("withdrawal was not by requester")
	}
	if e.Type == ActionEventSuperseded && (e.SupersededBy == "" || e.SuccessorDigest == "") {
		return fail("missing successor binding")
	}
	if e.Type != ActionEventSuperseded && (e.SupersededBy != "" || e.SuccessorDigest != "") {
		return fail("unexpected successor binding")
	}
	if e.SSHSignature != nil {
		if err := VerifyRecordedActionSignature(r, e); err != nil {
			return fail(err.Error())
		}
	}
	if e.Claim != nil || e.Envelope != nil {
		at, err := time.Parse(time.RFC3339Nano, e.At)
		if err != nil {
			return fail("invalid acceptance time")
		}
		latest := at
		if !strings.Contains(e.At, ".") {
			latest = at.Add(time.Second - time.Nanosecond)
		}
		if err := s.verifyActionClaimWithin(r, lifecycleResolution(e), e.Role, at, latest); err != nil {
			return fail(err.Error())
		}
	}
	if e.SSHSignature == nil && e.Claim == nil && e.Envelope == nil {
		policy, err := LoadActionPolicyIn(s.fsys(), s.Root)
		if err != nil {
			return err
		}
		if policy.IdentityAssurance == ReviewIdentityAssuranceVerified || e.Channel == "mcp" || e.Assurance == ReviewIdentityAssuranceVerified {
			return fail("lifecycle event carries no operation-bound proof")
		}
	}
	return nil
}
