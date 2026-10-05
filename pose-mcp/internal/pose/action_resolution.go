package pose

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ActionPolicy declares who may resolve requests addressed to a role, and
// what assurance a resolution needs (spec pose-action-request-resolution).
// It lives in .pose/policy/actions.json. Signing pins, the human-authority
// grant and the audience are the review policy's: one trust configuration.
type ActionPolicy struct {
	SchemaVersion int `json:"schema_version"`
	// Roles maps a role to the principals that hold it. A principal ending in
	// `*` matches by prefix (`agent:*`).
	Roles map[string][]string `json:"roles"`
	// IdentityAssurance is `declared` (default) or `verified`. Under
	// verified, a resolution must carry a signed claim from a trusted issuer
	// bound to the request digest; a human role also needs the human
	// authority grant.
	IdentityAssurance string `json:"identity_assurance,omitempty"`
}

// ActionResolution is a requested write to a request's journal.
type ActionResolution struct {
	RequestID        string
	Type             string // answered | cancelled | waived | invalidated
	Actor            string
	Execution        string
	Channel          string
	Role             string
	RequestDigest    string
	Answer           string
	Evidence         []string
	Reason           string
	PreparedBy       string
	AppliedBy        string
	ConfirmationMode string
	IdempotencyKey   string
	// ExpectedRevision is the journal length the writer read. A write
	// against an older revision is refused: two answers never both land.
	ExpectedRevision int
	Claim            *ActionAuthorityClaim
	Envelope         *ActionClaimEnvelope
}

// LoadActionPolicy reads the policy; an absent file is the empty declared
// policy, under which a role-addressed request has nobody to resolve it.
func LoadActionPolicy(root string) (ActionPolicy, error) {
	policy := ActionPolicy{SchemaVersion: 1, Roles: map[string][]string{}, IdentityAssurance: ReviewIdentityAssuranceDeclared}
	raw, err := os.ReadFile(filepath.Join(root, ".pose", "policy", "actions.json"))
	if os.IsNotExist(err) {
		return policy, nil
	}
	if err != nil {
		return policy, err
	}
	if err := json.Unmarshal(raw, &policy); err != nil {
		return policy, fmt.Errorf("pose: invalid .pose/policy/actions.json: %w", err)
	}
	if policy.SchemaVersion != 1 {
		return policy, fmt.Errorf("pose: unsupported action policy schema %d", policy.SchemaVersion)
	}
	if policy.IdentityAssurance == "" {
		policy.IdentityAssurance = ReviewIdentityAssuranceDeclared
	}
	if policy.IdentityAssurance != ReviewIdentityAssuranceDeclared && policy.IdentityAssurance != ReviewIdentityAssuranceVerified {
		return policy, fmt.Errorf("pose: action policy identity_assurance must be declared or verified")
	}
	if policy.Roles == nil {
		policy.Roles = map[string][]string{}
	}
	return policy, nil
}

func (p ActionPolicy) holds(role, principal string) bool {
	for _, candidate := range p.Roles[role] {
		if candidate == principal || strings.HasSuffix(candidate, "*") && strings.HasPrefix(principal, strings.TrimSuffix(candidate, "*")) {
			return true
		}
	}
	return false
}

// authorizeActionActor decides whether actor may resolve r. The recipient
// role, or the named recipient principal, is the authority; an unassigned
// request may be answered by its requester's counterpart only through a role.
func authorizeActionActor(policy ActionPolicy, r ActionRequest, actor, declaredRole string) (string, error) {
	switch {
	case r.Recipient.Role != "":
		if !policy.holds(r.Recipient.Role, actor) {
			return "", fmt.Errorf("%w: %s does not hold role %s in .pose/policy/actions.json", ErrActionNotAuthorized, actor, r.Recipient.Role)
		}
		return r.Recipient.Role, nil
	case r.Recipient.Principal != "":
		if actor != r.Recipient.Principal {
			return "", fmt.Errorf("%w: the request is addressed to %s", ErrActionNotAuthorized, r.Recipient.Principal)
		}
		return "", nil
	default:
		// Unassigned: an answer needs a role the policy grants, named by the
		// resolver, so authority is never assumed from the request alone.
		if declaredRole == "" || !policy.holds(declaredRole, actor) {
			return "", fmt.Errorf("%w: the request is unassigned; name a role %s holds with --role", ErrActionNotAuthorized, actor)
		}
		return declaredRole, nil
	}
}

// ResolveActionRequest appends an answer, a cancellation, a waiver or an
// invalidation, under the request's lock, against the revision the writer
// read. It returns the derived view after the write.
func (s Store) ResolveActionRequest(res ActionResolution, now time.Time) (ActionRequestView, error) {
	view, err := s.LoadActionRequest(res.RequestID)
	if err != nil {
		return ActionRequestView{}, err
	}
	r := view.Request
	project := s.CurrentObligationSnapshot().Project
	if r.Project != project {
		return ActionRequestView{}, fmt.Errorf("%w: request %s belongs to %s, this project is %s", ErrActionForeignProject, r.ID, r.Project, project)
	}
	if !validReviewPrincipal(res.Actor) {
		return ActionRequestView{}, fmt.Errorf("%w: actor must be an agent: or human: principal", ErrActionNotAuthorized)
	}
	if res.ExpectedRevision < 1 {
		return ActionRequestView{}, fmt.Errorf("%w: a resolution names the revision it was written against", ErrActionRevisionConflict)
	}
	if strings.TrimSpace(res.IdempotencyKey) == "" {
		return ActionRequestView{}, errors.New("pose: an idempotency key is required so a retry never records twice")
	}
	if res.RequestDigest != r.RequestDigest {
		return ActionRequestView{}, fmt.Errorf("%w: the answer was given for %s; the request is now %s", ErrActionStaleDigest, res.RequestDigest, r.RequestDigest)
	}
	policy, err := LoadActionPolicy(s.Root)
	if err != nil {
		return ActionRequestView{}, err
	}
	event := ActionEvent{Type: res.Type, At: now.UTC().Truncate(time.Second).Format(time.RFC3339), Actor: res.Actor, Execution: res.Execution, Channel: res.Channel,
		RequestDigest: res.RequestDigest, Answer: res.Answer, Evidence: res.Evidence, Reason: strings.TrimSpace(res.Reason), PreparedBy: res.PreparedBy, AppliedBy: res.AppliedBy,
		ConfirmationMode: res.ConfirmationMode, IdempotencyKey: res.IdempotencyKey, Claim: res.Claim, Envelope: res.Envelope, Assurance: ReviewIdentityAssuranceDeclared}
	for _, p := range []string{res.PreparedBy, res.AppliedBy} {
		if p != "" && !validReviewPrincipal(p) {
			return ActionRequestView{}, errors.New("pose: prepared_by and applied_by must be agent: or human: principals")
		}
	}
	switch res.ConfirmationMode {
	case "", ReviewConfirmationAdoptedConclusions, ReviewConfirmationAuthorizedOperation:
	default:
		return ActionRequestView{}, fmt.Errorf("pose: unknown confirmation_mode %q", res.ConfirmationMode)
	}
	switch res.Type {
	case ActionEventAnswered:
		if !actionAnswerAdmitted(r, res.Answer) {
			return ActionRequestView{}, fmt.Errorf("%w: %q is not an answer %s admits", ErrActionInvalidAnswer, res.Answer, r.ID)
		}
		if r.Kind == ActionExternalOperation && res.Answer == "done" && len(res.Evidence) == 0 {
			return ActionRequestView{}, fmt.Errorf("%w: an external operation reported done needs an evidence reference", ErrActionInvalidAnswer)
		}
		role, err := authorizeActionActor(policy, r, res.Actor, res.Role)
		if err != nil {
			return ActionRequestView{}, err
		}
		event.Role = role
		if res.Claim != nil || res.Envelope != nil || policy.IdentityAssurance == ReviewIdentityAssuranceVerified {
			if err := s.verifyActionClaim(r, res, role); err != nil {
				return ActionRequestView{}, err
			}
			event.Assurance = ReviewIdentityAssuranceVerified
		}
	case ActionEventCancelled, ActionEventWaived:
		if event.Reason == "" {
			return ActionRequestView{}, fmt.Errorf("pose: a %s needs a reason", res.Type)
		}
		// The requester may withdraw its own question; a waiver dispenses
		// with the condition, so it needs the request's authority.
		if !(res.Type == ActionEventCancelled && res.Actor == r.RequestedBy.Principal) {
			role, err := authorizeActionActor(policy, r, res.Actor, res.Role)
			if err != nil {
				return ActionRequestView{}, err
			}
			event.Role = role
		}
	case ActionEventInvalidated:
		if event.Reason == "" {
			return ActionRequestView{}, errors.New("pose: an invalidation needs a reason")
		}
	default:
		return ActionRequestView{}, fmt.Errorf("pose: unknown resolution type %q", res.Type)
	}
	return s.appendActionEvent(r.ID, event, res.ExpectedRevision, now)
}

func actionAnswerAdmitted(r ActionRequest, answer string) bool {
	if r.Kind == ActionInput {
		return strings.TrimSpace(answer) != ""
	}
	for _, option := range r.Options {
		if option.ID == answer {
			return true
		}
	}
	return false
}

// verifyActionClaim checks a signed resolution claim: a trusted issuer bound
// to this project, audience, request digest, principal, role and answer. A
// declared `human:` actor without such a claim never passes as verified.
func (s Store) verifyActionClaim(r ActionRequest, res ActionResolution, role string) error {
	fail := func(reason string) error { return fmt.Errorf("%w: %s", ErrActionVerificationFailed, reason) }
	claim, envelope := res.Claim, res.Envelope
	if claim == nil || envelope == nil {
		return fail("the action policy requires verified identity and the resolution carries no signed claim; a principal string is a declaration, not a proof")
	}
	review, _, err := s.loadReviewPolicy()
	if err != nil {
		return err
	}
	if envelope.Algorithm != "ed25519" || envelope.Issuer != claim.Issuer {
		return fail("the envelope must be ed25519 and name the claim's issuer")
	}
	if !reviewIssuerHoldsGrant(review.TrustedAttestationIssuers, envelope.Issuer, envelope.PublicKey) {
		return fail("issuer " + envelope.Issuer + " is not a trusted attestation issuer")
	}
	public, err := base64.StdEncoding.DecodeString(envelope.PublicKey)
	if err != nil || len(public) != ed25519.PublicKeySize {
		return fail("invalid issuer public key")
	}
	signature, err := base64.StdEncoding.DecodeString(envelope.Signature)
	if err != nil {
		return fail("invalid signature encoding")
	}
	canonical, _ := json.Marshal(claim)
	if !ed25519.Verify(ed25519.PublicKey(public), canonical, signature) {
		return fail("the signature does not cover this claim")
	}
	switch {
	case review.AuthorityAudience == "":
		return fail("the review policy declares no authority_audience, so a claim for any project would satisfy it")
	case review.AuthorityProject == "":
		return fail("the review policy declares no authority_project, so a claim for any project served by this verifier would satisfy it")
	case claim.Audience != review.AuthorityAudience:
		return fail("the claim is addressed to " + claim.Audience + " and this project answers to " + review.AuthorityAudience)
	case claim.Project != review.AuthorityProject:
		return fail("the claim names project " + claim.Project + " and this project is " + review.AuthorityProject)
	case claim.RequestID != r.ID || claim.RequestDigest != r.RequestDigest:
		return fail("the claim binds another request or another version of it")
	case claim.Principal != res.Actor:
		return fail("the claim principal is not the answering actor")
	case claim.Answer != res.Answer:
		return fail("the claim signs another answer")
	case claim.SchemaVersion != 1:
		return fail("unsupported claim schema")
	}
	// The claim's role is the kind of principal (human or agent), as in
	// review authority; the request's role is checked against the policy.
	if (claim.Role == "human") != strings.HasPrefix(claim.Principal, "human:") || (claim.Role != "human" && claim.Role != "agent") {
		return fail("the claim role " + claim.Role + " does not match its principal " + claim.Principal)
	}
	_ = role
	if strings.HasPrefix(claim.Principal, "human:") && !reviewIssuerHoldsGrant(review.HumanAuthorityIssuers, envelope.Issuer, envelope.PublicKey) {
		return fail("issuer " + envelope.Issuer + " is not authorised to assert a human principal")
	}
	if _, err := time.Parse(time.RFC3339, claim.IssuedAt); err != nil {
		return fail("the claim has no valid issued_at")
	}
	if claim.ExpiresAt != "" {
		expiry, err := time.Parse(time.RFC3339, claim.ExpiresAt)
		if err != nil || expiry.Before(time.Now().UTC()) {
			return fail("the claim is expired or its expiry is unreadable")
		}
	}
	return nil
}
