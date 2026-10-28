package pose

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"time"
)

// ActionRequest is a material request to an actor — a decision, an approval,
// an input, an external operation or an acceptance — that no other source
// records (spec pose-action-requests, ADR
// obligations-are-projected-action-requests-are-persisted).
//
// A request exists only when a different answer would materially change
// execution, scope, authority, risk acceptance, closeout or publication, and
// no authorization already given covers it. An ordinary question is a
// conversation, not a request.
type ActionRequest struct {
	SchemaVersion int                `json:"schema_version"`
	ID            string             `json:"id"`
	Project       string             `json:"project_id"`
	Origin        string             `json:"origin"`
	RequestedBy   ActionPrincipal    `json:"requested_by"`
	Recipient     ObligationActor    `json:"recipient"`
	Kind          string             `json:"kind"`
	Question      string             `json:"question"`
	Context       string             `json:"context,omitempty"`
	Options       []ActionOption     `json:"options"`
	Recommend     string             `json:"recommendation,omitempty"`
	Targets       []NodeRef          `json:"targets"`
	Effects       []ObligationEffect `json:"effects"`
	Subject       *ActionSubject     `json:"subject,omitempty"`
	Supersedes    string             `json:"supersedes,omitempty"`
	RequestedAt   string             `json:"requested_at"`
	RequestDigest string             `json:"request_digest"`
	// Review is the material of a delegated-review request (spec
	// pose-delegated-review-action-fulfillment R4); absent on every other
	// kind, so their digests are unchanged.
	Review *ActionReviewRequest `json:"review,omitempty"`
}

// ActionPrincipal is who asked, and from which execution.
type ActionPrincipal struct {
	Principal string `json:"principal"`
	Execution string `json:"execution,omitempty"`
}

// ActionOption is one answer the request admits, with what it entails.
type ActionOption struct {
	ID          string `json:"id"`
	Consequence string `json:"consequence"`
}

// ActionSubject binds the request to the content it is about. When that
// content changes, an answer recorded for the old content no longer
// satisfies the request.
type ActionSubject struct {
	Node   NodeRef `json:"node,omitempty"`
	Path   string  `json:"path,omitempty"`
	Digest string  `json:"digest"`
}

// Request kinds.
const (
	ActionDecision          = "decision"
	ActionApproval          = "approval"
	ActionInput             = "input"
	ActionExternalOperation = "external-operation"
	ActionAcceptance        = "acceptance"
)

// Derived request states.
const (
	ActionStateOpen        = "open"
	ActionStateAnswered    = "answered"
	ActionStateCancelled   = "cancelled"
	ActionStateWaived      = "waived"
	ActionStateSuperseded  = "superseded"
	ActionStateInvalidated = "invalidated"
)

const ActionRequestSchemaVersion = 1

var actionKinds = enumSet(ActionDecision, ActionApproval, ActionInput, ActionExternalOperation, ActionAcceptance, ActionDelegatedReview)

// fixedActionOptions are the answers kinds with a closed vocabulary admit;
// the first one is the satisfying answer.
var fixedActionOptions = map[string][]string{
	ActionApproval:          {"approve", "decline"},
	ActionAcceptance:        {"accept", "reject"},
	ActionExternalOperation: {"done", "failed"},
}

const actionDir = ".pose/actions"

// ActionRequestDigest is the digest of everything an answer is about. Two
// requests that ask the same thing of the same actor about the same targets
// share it; any material change produces a new one.
func ActionRequestDigest(r ActionRequest) string {
	material := struct {
		Project   string               `json:"project_id"`
		Origin    string               `json:"origin"`
		Recipient ObligationActor      `json:"recipient"`
		Kind      string               `json:"kind"`
		Question  string               `json:"question"`
		Context   string               `json:"context,omitempty"`
		Options   []ActionOption       `json:"options"`
		Targets   []NodeRef            `json:"targets"`
		Effects   []ObligationEffect   `json:"effects"`
		Subject   *ActionSubject       `json:"subject,omitempty"`
		Review    *ActionReviewRequest `json:"review,omitempty"`
	}{r.Project, r.Origin, r.Recipient, r.Kind, strings.TrimSpace(r.Question), strings.TrimSpace(r.Context), r.Options, r.Targets, r.Effects, r.Subject, r.Review}
	digest, _ := digestJSON(material)
	return digest
}

// PrepareActionRequest validates and completes a request without writing.
// project is this store's project id; local target refs (`requirement:R4`)
// are qualified against the origin.
func (s Store) PrepareActionRequest(r ActionRequest, now time.Time) (ActionRequest, error) {
	// Preparation is readonly with respect to caller-owned slices, including
	// concurrent attempts derived from the same prepared request.
	r.Options = slices.Clone(r.Options)
	r.Targets = slices.Clone(r.Targets)
	r.Effects = slices.Clone(r.Effects)
	for n := range r.Effects {
		r.Effects[n].Scope = slices.Clone(r.Effects[n].Scope)
	}
	if r.Subject != nil {
		subject := *r.Subject
		r.Subject = &subject
	}
	if r.Review != nil {
		review := *r.Review
		review.Agents = slices.Clone(r.Review.Agents)
		r.Review = &review
	}

	fail := func(format string, a ...any) (ActionRequest, error) {
		return ActionRequest{}, fmt.Errorf("pose: action request: "+format, a...)
	}
	r.SchemaVersion = ActionRequestSchemaVersion
	if r.Project == "" {
		project, err := s.WriteProjectID()
		if err != nil {
			return ActionRequest{}, err
		}
		r.Project = project
	}
	if ValidateSlug(r.Project) != nil {
		return fail("invalid project id")
	}
	origin, err := ParseArtifactRef(r.Origin)
	if err != nil {
		return fail("origin %q is not an artifact reference", r.Origin)
	}
	if origin.Project != "" && origin.Project != r.Project {
		return fail("origin %s belongs to project %s, not %s", r.Origin, origin.Project, r.Project)
	}
	origin.Project = r.Project
	if err := s.actionOriginExists(origin); err != nil {
		return fail("%v", err)
	}
	r.Origin = origin.String()
	if !actionKinds[r.Kind] {
		return fail("unknown kind %q (decision, approval, input, external-operation, acceptance, delegated-review)", r.Kind)
	}
	if r.Review != nil && r.Kind != ActionDelegatedReview {
		return fail("only a delegated-review request carries review material")
	}
	if strings.TrimSpace(r.Question) == "" {
		return fail("a question is required: the person must see what they decide")
	}
	if !validReviewPrincipal(r.RequestedBy.Principal) {
		return fail("requested_by must be an agent: or human: principal")
	}
	switch {
	case r.Recipient.Unassigned && (r.Recipient.Principal != "" || r.Recipient.Role != ""):
		return fail("an unassigned recipient names nobody")
	case r.Recipient.Principal == "" && r.Recipient.Role == "":
		r.Recipient = ObligationActor{Unassigned: true}
	case r.Recipient.Principal != "" && !validReviewPrincipal(r.Recipient.Principal):
		return fail("recipient principal must be an agent: or human: principal")
	}
	if fixed, ok := fixedActionOptions[r.Kind]; ok {
		if len(r.Options) == 0 {
			for _, id := range fixed {
				r.Options = append(r.Options, ActionOption{ID: id, Consequence: id})
			}
		}
		for _, option := range r.Options {
			if !containsString(fixed, option.ID) {
				return fail("%s admits only %s", r.Kind, strings.Join(fixed, "|"))
			}
		}
	}
	if r.Kind == ActionDecision && len(r.Options) < 2 {
		return fail("a decision needs at least two options, each with its consequence")
	}
	seen := map[string]bool{}
	for _, option := range r.Options {
		if ValidateSlug(option.ID) != nil || seen[option.ID] {
			return fail("option id %q must be a unique slug", option.ID)
		}
		if strings.TrimSpace(option.Consequence) == "" {
			return fail("option %s needs its consequence", option.ID)
		}
		seen[option.ID] = true
	}
	if r.Recommend != "" && !seen[r.Recommend] {
		return fail("recommendation %q is not one of the options", r.Recommend)
	}
	if len(r.Targets) == 0 {
		return fail("a request needs at least one target it restricts")
	}
	for i, target := range r.Targets {
		qualified, err := qualifyActionTarget(origin, target)
		if err != nil {
			return fail("target %q: %v", target.String(), err)
		}
		if _, _, _, err := s.actionMaterial(r.Project, qualified); err != nil {
			return fail("target: %v", err)
		}
		r.Targets[i] = qualified
	}
	if len(r.Effects) == 0 {
		return fail("a request needs at least one per-phase effect")
	}
	for i, effect := range r.Effects {
		if !obligationPhases[effect.Phase] || !obligationModes[effect.Mode] {
			return fail("effect %s/%s is not a known phase and mode", effect.Phase, effect.Mode)
		}
		for j, scope := range effect.Scope {
			qualified, err := qualifyActionTarget(origin, scope)
			if err != nil {
				return fail("effect scope %q: %v", scope.String(), err)
			}
			if _, _, _, err := s.actionMaterial(r.Project, qualified); err != nil {
				return fail("effect scope: %v", err)
			}
			r.Effects[i].Scope[j] = qualified
		}
	}
	if r.Subject == nil && (r.Kind == ActionApproval || r.Kind == ActionAcceptance) {
		adopted, err := s.AgencyReadinessAdopted()
		if err != nil {
			return fail("subject policy: %v", err)
		}
		if adopted {
			return fail("material approval or acceptance requires a subject under adopted agency readiness")
		}
	}
	if r.Kind == ActionDelegatedReview {
		// The subject of a delegated review is its scope's current bundle and
		// the policies, which the engine reads; it is never the caller's.
		if err := s.prepareDelegatedReviewRequest(&r); err != nil {
			return fail("%v", err)
		}
	} else if r.Subject != nil {
		subject, err := s.resolveActionSubject(origin, *r.Subject)
		if err != nil {
			return fail("subject: %v", err)
		}
		r.Subject = &subject
	}
	if r.Supersedes != "" {
		prior, err := s.LoadActionRequest(r.Supersedes)
		if err != nil {
			return fail("supersedes %s: %v", r.Supersedes, err)
		}
		if prior.Request.Project != r.Project {
			return fail("cannot supersede a request of another project")
		}
	}
	if r.RequestedAt == "" {
		r.RequestedAt = now.UTC().Truncate(time.Second).Format(time.RFC3339)
	}
	if _, err := time.Parse(time.RFC3339Nano, r.RequestedAt); err != nil {
		return fail("requested_at must be an RFC3339 timestamp from the prepared request")
	}
	r.RequestDigest = ActionRequestDigest(r)
	identity, _ := digestJSON([]string{"action-request/v1", r.RequestDigest, r.RequestedAt, r.RequestedBy.Principal, r.Supersedes})
	r.ID = "act-" + strings.TrimPrefix(identity, "sha256:")[:16]
	return r, nil
}

func qualifyActionTarget(origin ArtifactRef, target NodeRef) (NodeRef, error) {
	if strings.HasPrefix(target.Artifact, "xref:") {
		parsed, err := ParseNodeRef(target.String())
		if err != nil {
			return NodeRef{}, err
		}
		return parsed, nil
	}
	// A local form is `kind:ID` inside the origin, or the origin itself.
	if target.Artifact == "" || target.Artifact == "self" {
		return QualifyNodeRef(origin.Project, origin, target.Kind, target.ID), validateLocalNode(target.Kind, target.ID)
	}
	return NodeRef{}, errors.New("an artifact other than the origin must be qualified with xref:")
}

func validateLocalNode(kind, id string) error {
	if kind == "" && id == "" {
		return nil
	}
	if !nodeKinds[kind] || ValidateSlug(strings.ToLower(id)) != nil {
		return errors.New("invalid node kind or id")
	}
	return nil
}

// ParseActionTarget reads a CLI target: `requirement:R4` (inside the origin),
// `self` (the origin), or a qualified `xref:...#kind:ID`.
func ParseActionTarget(raw string) (NodeRef, error) {
	raw = strings.TrimSpace(raw)
	if raw == "self" || raw == "" {
		return NodeRef{Artifact: "self"}, nil
	}
	if strings.HasPrefix(raw, "xref:") {
		return ParseNodeRef(raw)
	}
	kind, id, ok := strings.Cut(raw, ":")
	if !ok {
		return NodeRef{}, errors.New("invalid-node-reference")
	}
	return NodeRef{Artifact: "self", Kind: kind, ID: id}, nil
}

func (s Store) actionOriginExists(origin ArtifactRef) error {
	switch origin.Kind {
	case "spec":
		if _, err := s.GetSpec(origin.Slug); err != nil {
			return fmt.Errorf("origin spec %s not found", origin.Slug)
		}
	case "roadmap", "milestone":
		rm, err := s.GetRoadmap(origin.Slug)
		if err != nil {
			return fmt.Errorf("origin roadmap %s not found", origin.Slug)
		}
		if origin.Kind == "milestone" {
			for _, m := range rm.Milestones {
				if m.ID == origin.Milestone {
					return nil
				}
			}
			return fmt.Errorf("origin milestone %s/%s not found", origin.Slug, origin.Milestone)
		}
	default:
		return fmt.Errorf("origin kind %s cannot carry an action request", origin.Kind)
	}
	return nil
}

// resolveActionSubject computes the current digest of the content a request
// is about: an exact qualified spec/node, or a project file.
func (s Store) resolveActionSubject(origin ArtifactRef, subject ActionSubject) (ActionSubject, error) {
	if subject.Path != "" && (subject.Node.Artifact != "" || subject.Node.Kind != "" || subject.Node.ID != "") {
		return ActionSubject{}, errors.New("subject names either a path or a node, never both")
	}
	switch {
	case subject.Path != "":
		rel := filepath.Clean(filepath.FromSlash(subject.Path))
		if !filepath.IsLocal(rel) {
			return ActionSubject{}, errors.New("subject path escapes project root")
		}
		if err := ValidateArtifactPathIn(s.fsys(), s.Root, subject.Path, false); err != nil {
			return ActionSubject{}, err
		}
		raw, err := s.fsys().ReadFile(filepath.Join(s.Root, rel))
		if err != nil {
			return ActionSubject{}, err
		}
		subject.Digest = digestBytes(raw)
	case subject.Node.Artifact != "" || subject.Node.ID != "":
		node, err := qualifyActionTarget(origin, subject.Node)
		if err != nil {
			return ActionSubject{}, err
		}
		_, _, digest, err := s.actionMaterial(origin.Project, node)
		if err != nil {
			return ActionSubject{}, err
		}
		if digest == "" {
			return ActionSubject{}, errors.New("subject must name material spec content or a node")
		}
		subject.Digest = digest
		subject.Node = node
	default:
		return ActionSubject{}, errors.New("a subject names a node or a path")
	}
	return subject, nil
}

// currentSubjectDigest recomputes a recorded subject; "" when it no longer
// resolves.
func (s Store) currentSubjectDigest(r ActionRequest) string {
	if r.Subject == nil {
		return ""
	}
	if r.Kind == ActionDelegatedReview && r.Review != nil {
		return delegatedReviewSubjectReader(s, *r.Review)
	}
	origin, err := ParseArtifactRef(r.Origin)
	if err != nil {
		return ""
	}
	probe := *r.Subject
	probe.Digest = ""
	current, err := s.resolveActionSubject(origin, probe)
	if err != nil {
		return ""
	}
	return current.Digest
}

// ActionAuthorityClaim is what an issuer signs for a resolution: this
// principal, in this role, answered this request with this answer.
type ActionAuthorityClaim struct {
	SchemaVersion int    `json:"schema_version"`
	Project       string `json:"project"`
	Audience      string `json:"audience"`
	RequestID     string `json:"request_id"`
	RequestDigest string `json:"request_digest"`
	Principal     string `json:"principal"`
	Role          string `json:"role"`
	Execution     string `json:"execution,omitempty"`
	Answer        string `json:"answer"`
	// Reason, when the answer has one, is signed with it (spec
	// pose-adopt-request-keeps-the-reason); omitted when empty.
	Reason           string `json:"reason,omitempty"`
	Issuer           string `json:"issuer"`
	IssuedAt         string `json:"issued_at"`
	ExpiresAt        string `json:"expires_at,omitempty"`
	Operation        string `json:"operation,omitempty"`
	ExpectedRevision int    `json:"expected_revision,omitempty"`
	IdempotencyKey   string `json:"idempotency_key,omitempty"`
	SuccessorID      string `json:"successor_id,omitempty"`
	SuccessorDigest  string `json:"successor_digest,omitempty"`
}

// ActionClaimEnvelope carries the Ed25519 signature over the canonical claim.
type ActionClaimEnvelope struct {
	Issuer    string `json:"issuer"`
	Algorithm string `json:"algorithm"`
	PublicKey string `json:"public_key"`
	Signature string `json:"signature"`
}

// ActionEvent is one append-only entry of a request's journal.
type ActionEvent struct {
	LifecycleVersion int                   `json:"lifecycle_version,omitempty"`
	ExpectedRevision int                   `json:"expected_revision,omitempty"`
	SuccessorDigest  string                `json:"successor_digest,omitempty"`
	SchemaVersion    int                   `json:"schema_version"`
	Seq              int                   `json:"seq"`
	Type             string                `json:"type"`
	At               string                `json:"at"`
	Request          *ActionRequest        `json:"request,omitempty"`
	Actor            string                `json:"actor,omitempty"`
	Execution        string                `json:"execution,omitempty"`
	Channel          string                `json:"channel,omitempty"`
	RequestDigest    string                `json:"request_digest,omitempty"`
	Answer           string                `json:"answer,omitempty"`
	Evidence         []string              `json:"evidence,omitempty"`
	Reason           string                `json:"reason,omitempty"`
	PreparedBy       string                `json:"prepared_by,omitempty"`
	AppliedBy        string                `json:"applied_by,omitempty"`
	ConfirmationMode string                `json:"confirmation_mode,omitempty"`
	Assurance        string                `json:"assurance,omitempty"`
	Role             string                `json:"role,omitempty"`
	Claim            *ActionAuthorityClaim `json:"claim,omitempty"`
	Envelope         *ActionClaimEnvelope  `json:"envelope,omitempty"`
	SSHSignature     *ActionSSHSignature   `json:"ssh_signature,omitempty"`
	IdempotencyKey   string                `json:"idempotency_key,omitempty"`
	SupersededBy     string                `json:"superseded_by,omitempty"`
	// Fulfillment is the engine's reading of a run that fulfilled a
	// delegated-review request (spec pose-delegated-review-action-fulfillment).
	Fulfillment *ActionFulfillment `json:"fulfillment,omitempty"`
}

// Event types.
const (
	ActionEventOpened      = "opened"
	ActionEventAnswered    = "answered"
	ActionEventRefused     = "refused"
	ActionEventCancelled   = "cancelled"
	ActionEventWaived      = "waived"
	ActionEventSuperseded  = "superseded"
	ActionEventInvalidated = "invalidated"
)

// ActionRequestView is a request with its derived state.
type ActionRequestView struct {
	Recovery     *ActionRecovery            `json:"recovery,omitempty"`
	NextStep     *ActionNextStep            `json:"next_step,omitempty"`
	Request      ActionRequest              `json:"request"`
	State        string                     `json:"state"`
	Answer       string                     `json:"answer,omitempty"`
	AnsweredBy   string                     `json:"answered_by,omitempty"`
	Assurance    string                     `json:"assurance,omitempty"`
	Presence     *ActionPresenceObservation `json:"presence,omitempty"`
	Satisfaction string                     `json:"satisfaction"`
	Revision     int                        `json:"revision"`
	Events       []ActionEvent              `json:"events"`
	Limitations  []string                   `json:"limitations,omitempty"`
	// SubjectUnknown is true when the subject no longer resolves: the engine
	// cannot tell whether the answer still applies, so it is not satisfied.
	SubjectUnknown bool `json:"subject_unknown,omitempty"`
}

func actionJournalPath(root, id string) string {
	if !filepath.IsLocal(id) {
		return ""
	}
	return filepath.Join(root, filepath.FromSlash(actionDir), id+".jsonl")
}

func (s Store) checkActionJournal(id string) error {
	if !filepath.IsLocal(id) || !strings.HasPrefix(id, "act-") || len(id) != 20 || ValidateSlug(id) != nil {
		return fmt.Errorf("pose: invalid action request id %q", id)
	}
	return validateProjectPathIn(s.fsys(), s.Root, filepath.Join(filepath.FromSlash(actionDir), id+".jsonl"))
}

// OpenActionRequest persists a prepared request as the first journal event.
// Opening the same prepared request again is idempotent.
func (s Store) OpenActionRequest(r ActionRequest, now time.Time) (ActionRequestView, error) {
	return s.OpenActionRequestAuthorized(r, nil, now)
}

// OpenActionRequestAuthorized accepts proof for retirement of the predecessor.
// Creation of a question alone does not grant authority over its predecessor.
func (s Store) OpenActionRequestAuthorized(r ActionRequest, authority *ActionResolution, now time.Time) (ActionRequestView, error) {
	prepared, err := s.PrepareActionRequest(r, now)
	if err != nil {
		return ActionRequestView{}, err
	}
	if prepared.Supersedes != "" {
		return s.openSupersession(prepared, authority, now)
	}
	return s.openPreparedAction(prepared)
}

func (s Store) openPreparedAction(prepared ActionRequest) (ActionRequestView, error) {
	if existing, err := s.loadActionRequestBase(prepared.ID); err == nil {
		if existing.Request.RequestDigest != prepared.RequestDigest {
			return ActionRequestView{}, fmt.Errorf("%w: request identity collision", ErrActionIdempotencyConflict)
		}
		return s.withActionRecovery(existing), nil
	} else if !os.IsNotExist(err) {
		return ActionRequestView{}, err
	}
	if _, err := ensureReviewArtifactDirIn(s.fsys(), s.Root, actionDir, true); err != nil {
		return ActionRequestView{}, err
	}
	event := ActionEvent{SchemaVersion: ActionRequestSchemaVersion, Seq: 1, Type: ActionEventOpened, At: prepared.RequestedAt, Request: &prepared, Actor: prepared.RequestedBy.Principal, Execution: prepared.RequestedBy.Execution, RequestDigest: prepared.RequestDigest}
	line, err := json.Marshal(event)
	if err != nil {
		return ActionRequestView{}, err
	}
	if err := s.fsys().CreateExclusive(actionJournalPath(s.Root, prepared.ID), append(line, '\n'), 0644, true); err != nil {
		return ActionRequestView{}, err
	}
	return s.LoadActionRequest(prepared.ID)
}

func (s Store) writeActionEvent(file *os.File, event ActionEvent) error {
	raw, err := json.Marshal(event)
	if err != nil {
		return err
	}
	if _, err := file.Write(append(raw, '\n')); err != nil {
		return err
	}
	if syncer, ok := s.fsys().(interface{ syncActionJournal(*os.File) error }); ok {
		return syncer.syncActionJournal(file)
	}
	return file.Sync()
}

// LoadActionRequest reads one journal and derives the request's state.
func (s Store) LoadActionRequest(id string) (ActionRequestView, error) {
	view, err := s.loadActionRequestBase(id)
	if err != nil {
		return view, err
	}
	return s.withActionRecovery(view), nil
}

func (s Store) loadActionRequestBase(id string) (ActionRequestView, error) {
	if err := s.checkActionJournal(id); err != nil {
		return ActionRequestView{}, err
	}
	raw, err := s.fsys().ReadFile(actionJournalPath(s.Root, id))
	if err != nil {
		return ActionRequestView{}, err
	}
	events, err := parseActionJournal(raw)
	if err != nil {
		var invalid *ActionJournalError
		if errors.As(err, &invalid) {
			invalid.Request, invalid.Path = id, filepath.ToSlash(filepath.Join(actionDir, id+".jsonl"))
		}
		return ActionRequestView{}, fmt.Errorf("pose: action request %s: %w", id, err)
	}
	if len(events) == 0 || events[0].Type != ActionEventOpened || events[0].Request == nil || events[0].Request.ID != id {
		return ActionRequestView{}, fmt.Errorf("pose: action request %s: journal does not start with its opening", id)
	}
	if ActionRequestDigest(*events[0].Request) != events[0].Request.RequestDigest {
		return ActionRequestView{}, fmt.Errorf("pose: action request %s: request digest does not match its content", id)
	}
	if err := s.verifyActionAnswers(events); err != nil {
		return ActionRequestView{}, fmt.Errorf("pose: action request %s: %w", id, err)
	}
	return s.deriveActionView(events), nil
}

// Labels do not establish proof. Validate persisted answers in the shared
// reader, so JSON, MCP, projections and gates cannot bypass verification.
func (s Store) verifyActionAnswers(events []ActionEvent) error {
	if len(events) == 0 || events[0].Request == nil {
		return fmt.Errorf("%w: journal has no request", ErrActionVerificationFailed)
	}
	r := *events[0].Request
	for _, event := range events[1:] {
		if event.Type == ActionEventFulfilled {
			// A fulfilment is judged when the view is derived: one that no
			// longer verifies satisfies nothing, but never makes the journal
			// unreadable.
			continue
		}
		if event.Type != ActionEventAnswered {
			if err := s.verifyLifecycleEvent(r, event); err != nil {
				return err
			}
			continue
		}
		fail := func(err error) error {
			return fmt.Errorf("%w: answer sequence %d: %v", ErrActionVerificationFailed, event.Seq, err)
		}
		proof := event.SSHSignature != nil || event.Claim != nil || event.Envelope != nil
		if !proof {
			if event.Assurance == ReviewIdentityAssuranceVerified {
				return fail(errors.New("verified answer carries no proof"))
			}
			continue // Declared legacy answers are not retroactively upgraded.
		}
		if event.RequestDigest != r.RequestDigest {
			return fail(errors.New("the answer binds another request version"))
		}
		if event.SSHSignature != nil {
			if err := VerifyRecordedActionSignature(r, event); err != nil {
				return fail(err)
			}
		}
		if event.Claim != nil || event.Envelope != nil {
			at, err := time.Parse(time.RFC3339, event.At)
			if err != nil {
				return fail(errors.New("issuer answer has no valid acceptance time"))
			}
			latest := at
			if !strings.Contains(event.At, ".") {
				latest = at.Add(time.Second - time.Nanosecond)
			}
			res := ActionResolution{Actor: event.Actor, Answer: event.Answer, Reason: event.Reason,
				RequestDigest: event.RequestDigest, Claim: event.Claim, Envelope: event.Envelope}
			if err := s.verifyActionClaimWithin(r, res, event.Role, at, latest); err != nil {
				return fail(err)
			}
		}
	}
	return nil
}

// deriveActionView folds the journal. Answered is not satisfied: a declined
// approval, a rejected acceptance or a failed operation is an answer that
// leaves the request's condition unmet.
func (s Store) deriveActionView(events []ActionEvent) ActionRequestView {
	request := *events[0].Request
	view := ActionRequestView{Request: request, State: ActionStateOpen, Satisfaction: SatisfactionPending, Revision: len(events), Events: events}
	for _, event := range events[1:] {
		switch event.Type {
		case ActionEventAnswered:
			view.State, view.Answer, view.AnsweredBy = ActionStateAnswered, event.Answer, event.Actor
			view.Presence = observeActionPresence(request, event)
			view.Assurance = ReviewIdentityAssuranceDeclared
			if event.SSHSignature != nil || event.Claim != nil || event.Envelope != nil {
				view.Assurance = ReviewIdentityAssuranceVerified
			}
			view.Satisfaction = SatisfactionPending
			if actionAnswerSatisfies(request, event.Answer) {
				view.Satisfaction = SatisfactionSatisfied
			}
		case ActionEventFulfilled:
			// The requested run was performed and verified: the request is
			// satisfied whatever the run concluded, and a rejection stays
			// visible as the answer (R6). While the request's subject is
			// current the fulfilment is verified again; once the bundle or the
			// policies moved on it stands as recorded.
			if request.Review != nil && request.Subject != nil && delegatedReviewSubjectReader(s, *request.Review) == request.Subject.Digest {
				if err := fulfillmentEventVerifier(s, request, event); err != nil {
					view.Limitations = append(view.Limitations, "a recorded fulfilment does not verify, so it satisfies nothing: "+err.Error())
					continue
				}
			} else {
				view.Limitations = append(view.Limitations, "the fulfilment was verified against bundle "+request.Review.BundleID+" when it was recorded; the scope or the policies have changed since")
			}
			view.State, view.Satisfaction, view.Assurance = ActionStateFulfilled, SatisfactionSatisfied, ReviewIdentityAssuranceVerified
			view.AnsweredBy = event.Actor
			if event.Fulfillment != nil {
				view.Answer = event.Fulfillment.Decision
				if event.Fulfillment.Decision != "approved" {
					view.Limitations = append(view.Limitations, "the delegated run concluded "+event.Fulfillment.Decision+": the review was performed, nothing is approved")
				}
			}
		case ActionEventCancelled:
			view.Assurance = lifecycleAssurance(event)
			view.Presence = observeActionPresence(request, event)
			view.Limitations = append(view.Limitations, lifecycleLimitations(event)...)
			view.State, view.Satisfaction = ActionStateCancelled, SatisfactionCancelled
		case ActionEventWaived:
			view.Assurance = lifecycleAssurance(event)
			view.Presence = observeActionPresence(request, event)
			view.Limitations = append(view.Limitations, lifecycleLimitations(event)...)
			view.State, view.Satisfaction = ActionStateWaived, SatisfactionWaived
		case ActionEventSuperseded:
			view.Assurance = lifecycleAssurance(event)
			view.Presence = observeActionPresence(request, event)
			view.Limitations = append(view.Limitations, lifecycleLimitations(event)...)
			view.State, view.Satisfaction = ActionStateSuperseded, SatisfactionCancelled
		case ActionEventInvalidated:
			view.State, view.Satisfaction = ActionStateInvalidated, SatisfactionInvalidated
		}
	}
	// An answer recorded for content that has since changed no longer
	// satisfies the request.
	if request.Subject == nil {
		view.Limitations = append(view.Limitations, "request has no material subject; its answer provides no content-change detection")
	}
	if (view.State == ActionStateAnswered || view.State == ActionStateOpen) && request.Subject != nil {
		current := s.currentSubjectDigest(request)
		switch {
		case current == "":
			view.SubjectUnknown, view.Satisfaction = true, SatisfactionPending
			view.Limitations = append(view.Limitations, "the request's subject no longer resolves; the answer cannot be checked against it")
		case current != request.Subject.Digest:
			view.State, view.Satisfaction = ActionStateInvalidated, SatisfactionInvalidated
			view.Limitations = append(view.Limitations, "the material subject changed since the request ("+request.Subject.Digest+" → "+current+")")
		}
	}
	if view.State == ActionStateAnswered && view.Assurance != ReviewIdentityAssuranceVerified {
		view.Limitations = append(view.Limitations, "the answering identity is declared, not verified")
	}
	return view
}

func actionAnswerSatisfies(r ActionRequest, answer string) bool {
	if fixed, ok := fixedActionOptions[r.Kind]; ok {
		return answer == fixed[0]
	}
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

// ListActionRequests reads every journal, oldest first. A journal that fails
// to parse is returned as an error naming it, never skipped.
func (s Store) ListActionRequests() ([]ActionRequestView, error) {
	entries, err := s.fsys().ReadDir(filepath.Join(s.Root, filepath.FromSlash(actionDir)))
	if os.IsNotExist(err) {
		return []ActionRequestView{}, nil
	}
	if err != nil {
		return nil, err
	}
	var out []ActionRequestView
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".jsonl") {
			continue
		}
		view, err := s.LoadActionRequest(strings.TrimSuffix(entry.Name(), ".jsonl"))
		if err != nil {
			return nil, err
		}
		out = append(out, view)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Request.RequestedAt != out[j].Request.RequestedAt {
			return out[i].Request.RequestedAt < out[j].Request.RequestedAt
		}
		return out[i].Request.ID < out[j].Request.ID
	})
	return out, nil
}

func actionObligation(project string, view ActionRequestView) Obligation {
	r := view.Request
	origin, _ := ParseArtifactRef(r.Origin)
	node := QualifyNodeRef(project, origin, "action", r.ID)
	o := newObligation(project, "action-requests", node, "action-request", r.ID)
	o.Category = ObligationActorAction
	o.ReasonCode = "action-" + view.State
	o.Recipient = r.Recipient
	o.Targets = slices.Clone(r.Targets)
	o.Effects = slices.Clone(r.Effects)
	o.Satisfaction = view.Satisfaction
	if view.SubjectUnknown {
		o.Knowledge = KnowledgeUnknown
	}
	o.Waiting = WaitingActor
	if r.Kind == ActionExternalOperation {
		o.Waiting = WaitingExternal
	}
	who := "an authorized actor"
	switch {
	case r.Recipient.Principal != "":
		who = r.Recipient.Principal
	case r.Recipient.Role != "":
		who = "role " + r.Recipient.Role
	}
	o.NextStep = view.NextStep
	if view.NextStep != nil && view.NextStep.Kind != "answer" {
		o.Condition = view.NextStep.Reason
	} else {
		o.Condition = fmt.Sprintf("%s answers %s %s with a satisfying answer bound to %s", who, r.Kind, r.ID, r.RequestDigest)
	}
	o.Observation.Limitations = append(o.Observation.Limitations, view.Limitations...)
	o.Message = fmt.Sprintf("%s (%s, %s): %s", r.ID, r.Kind, view.State, r.Question)
	if view.State == ActionStateAnswered && view.Satisfaction != SatisfactionSatisfied {
		o.Message += " — answered " + view.Answer + ", which does not satisfy it"
	}
	return o
}

// Resolution errors carry stable codes.
var (
	ErrActionRevisionConflict    = errors.New("action-revision-conflict")
	ErrActionIdempotencyConflict = errors.New("action-idempotency-conflict")
	ErrActionStaleDigest         = errors.New("action-request-digest-stale")
	ErrActionNotAuthorized       = errors.New("action-actor-not-authorized")
	ErrActionNotOpen             = errors.New("action-request-not-open")
	ErrActionForeignProject      = errors.New("action-foreign-project")
	ErrActionVerificationFailed  = errors.New("action-claim-not-verified")
	ErrActionInvalidAnswer       = errors.New("action-invalid-answer")
)

// appendActionEvent writes one event under an exclusive lock. expected < 0
// skips the revision check (used only by the engine's own supersession).
// A retry with the same idempotency key and the same content is a no-op; the
// same key with different content is refused.
func (s Store) lockedActionJournal(id string) (*os.File, []ActionEvent, func(), error) {
	if err := s.checkActionJournal(id); err != nil {
		return nil, nil, nil, err
	}
	if p := s.fsys().Preview(); p != nil {
		return nil, nil, nil, p.Unsupported("locking action journal " + id)
	}
	file, err := os.OpenFile(actionJournalPath(s.Root, id), os.O_RDWR|os.O_APPEND, 0644)
	if err != nil {
		return nil, nil, nil, err
	}
	unlock, err := lockSpecTransferFile(file)
	if err != nil {
		file.Close()
		return nil, nil, nil, fmt.Errorf("%w: another writer holds %s", ErrActionRevisionConflict, id)
	}
	release := func() { unlock(); file.Close() }
	raw, err := s.fsys().ReadFile(actionJournalPath(s.Root, id))
	if err != nil {
		release()
		return nil, nil, nil, err
	}
	events, err := parseActionJournal(raw)
	if err != nil {
		release()
		return nil, nil, nil, err
	}
	if len(events) == 0 || events[0].Request == nil || events[0].Request.ID != id || ActionRequestDigest(*events[0].Request) != events[0].Request.RequestDigest {
		release()
		return nil, nil, nil, fmt.Errorf("%w: invalid journal opening", ErrActionVerificationFailed)
	}
	if err := s.verifyActionAnswers(events); err != nil {
		release()
		return nil, nil, nil, err
	}
	return file, events, release, nil
}
func (s Store) appendActionEvent(id string, event ActionEvent, expected int, now time.Time) (ActionRequestView, error) {
	file, events, release, err := s.lockedActionJournal(id)
	if err != nil {
		return ActionRequestView{}, err
	}
	defer release()
	return s.appendActionEventLocked(file, events, id, event, expected, now)
}
func (s Store) appendActionEventLocked(file *os.File, events []ActionEvent, id string, event ActionEvent, expected int, now time.Time) (ActionRequestView, error) {
	for _, prior := range events {
		if prior.IdempotencyKey != "" && prior.IdempotencyKey == event.IdempotencyKey {
			if sameActionIntent(prior, event) {
				return s.withActionRecovery(s.deriveActionView(events)), nil
			}
			return ActionRequestView{}, fmt.Errorf("%w: key %s already recorded different content", ErrActionIdempotencyConflict, event.IdempotencyKey)
		}
	}
	if expected >= 0 && expected != len(events) {
		return ActionRequestView{}, fmt.Errorf("%w: written against revision %d, the journal is at %d", ErrActionRevisionConflict, expected, len(events))
	}
	current := s.deriveActionView(events)
	if event.Type == ActionEventSuperseded && current.State == ActionStateSuperseded {
		return ActionRequestView{}, ErrActionIntentConflict
	}
	if expected >= 0 && !actionResolutionStateAdmitted(current, event.Type) && event.Type != ActionEventInvalidated && event.Type != ActionEventSuperseded {
		return ActionRequestView{}, fmt.Errorf("%w: request %s is %s; supersede it with a new request instead", ErrActionNotOpen, id, current.State)
	}
	// A fulfilment is verified again inside the lock: the run must verify at
	// the write, not only when it was prepared.
	if event.Type == ActionEventFulfilled {
		if current.Request.Kind != ActionDelegatedReview || event.Fulfillment == nil {
			return ActionRequestView{}, ErrActionFulfillmentInvalid
		}
		if _, err := s.verifyFulfillment(current.Request, event.Fulfillment.RunID, event.Fulfillment.Runner, event.Fulfillment.Channel); err != nil {
			return ActionRequestView{}, err
		}
	}
	if event.Type == ActionEventAnswered || event.LifecycleVersion != 0 && event.Type != ActionEventSuperseded {
		res := resolutionFromEvent(event, expected)
		if _, _, _, err := s.prepareActionResolutionAgainst(s.withActionRecovery(current), res, now, false); err != nil {
			return ActionRequestView{}, err
		}
	}
	// Recompose authority, current policy, proof and revision inside the lock.
	if event.LifecycleVersion != 0 {
		if event.ExpectedRevision != len(events) {
			return ActionRequestView{}, ErrActionRevisionConflict
		}
		if event.Type == ActionEventSuperseded {
			successor, err := s.LoadActionRequest(event.SupersededBy)
			if err != nil || successor.Request.Supersedes != id || successor.Request.RequestDigest != event.SuccessorDigest {
				return ActionRequestView{}, fmt.Errorf("%w: successor no longer matches retirement", ErrActionVerificationFailed)
			}
		}
		if err := s.admitLifecycle(*events[0].Request, &event, "", now, now); err != nil {
			return ActionRequestView{}, err
		}
	}
	event.SchemaVersion = ActionRequestSchemaVersion
	event.Seq = len(events) + 1
	if event.At == "" {
		event.At = now.UTC().Truncate(time.Second).Format(time.RFC3339)
	}
	if err := s.writeActionEvent(file, event); err != nil {
		return ActionRequestView{}, err
	}
	return s.withActionRecovery(s.deriveActionView(append(events, event))), nil
}
