package pose

// Delegated review as an action request (spec
// pose-delegated-review-action-fulfillment).
//
// A project asks for a delegated review, adjudication or smoke run as an
// action request of kind delegated-review (R1). The request binds its
// project, worktree, origin, scope, targets and purpose, the sealed bundle,
// the brief, the notes and the policies it was asked under, the agents it
// accepts and the destination its conclusion is recorded at (R4). It is
// satisfied by no actor's answer: only a fulfilment the engine verifies — a
// run of that brief whose signed proof verifies, whose attempt ledger is
// intact, whose containment completed and whose conclusion the engine
// recorded at the destination the request names (R2, R5). A local runner, a
// CI job and Harne8's Conductor submit the same envelope to the same
// verifier (R3, R9). Fulfilling the review is not approving: a valid
// rejecting run fulfils the request and stays visible, while approvals and
// gates stay where the attestation leaves them (R6).

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"
)

// ActionDelegatedReview is the kind of a request for a delegated run.
const ActionDelegatedReview = "delegated-review"

// ActionEventFulfilled records a verified fulfilment of a delegated-review
// request.
const ActionEventFulfilled = "fulfilled"

// ActionStateFulfilled is a delegated-review request whose run the engine
// verified.
const ActionStateFulfilled = "fulfilled"

// Delegated review destinations: where the conclusion of each purpose is
// recorded, which the request names and the runner does not choose (R9).
const (
	ReviewDestinationAttestation  = "attestation"
	ReviewDestinationAdjudication = "adjudication-record"
	ReviewDestinationSmoke        = "smoke-evidence"
)

// ActionReviewRequest is the material of a delegated-review request (R4).
type ActionReviewRequest struct {
	Purpose       string   `json:"purpose"`
	Scope         string   `json:"scope"`
	BundleID      string   `json:"bundle_id"`
	BundleDigest  string   `json:"bundle_digest"`
	BriefDigest   string   `json:"brief_digest"`
	BriefTemplate string   `json:"brief_template"`
	NotesDigest   string   `json:"notes_digest"`
	PolicyDigest  string   `json:"policy_digest"`
	Worktree      string   `json:"worktree"`
	Agents        []string `json:"agents,omitempty"`
	Destination   string   `json:"destination"`
}

// ActionFulfillment is what a verified fulfilment records: the run, its
// proof and conclusion, the decision it reached and where it was recorded,
// and who submitted it through which channel. Everything but the runner and
// the channel is the engine's reading of the run, never the runner's claim.
type ActionFulfillment struct {
	RunID       string `json:"run_id"`
	ProofID     string `json:"proof_id"`
	AttemptID   string `json:"attempt_id"`
	Decision    string `json:"decision"`
	Destination string `json:"destination"`
	Recorded    string `json:"recorded"`
	Runner      string `json:"runner"`
	Channel     string `json:"channel"`
	// Escalations are the requests the fulfilment opened for a person (R7).
	Escalations []string `json:"escalations,omitempty"`
}

// ActionFulfillmentEnvelope is what a runner submits, from any channel (R9).
type ActionFulfillmentEnvelope struct {
	RequestID        string `json:"request_id"`
	RequestDigest    string `json:"request_digest"`
	ExpectedRevision int    `json:"expected_revision"`
	IdempotencyKey   string `json:"idempotency_key"`
	RunID            string `json:"run_id"`
	Runner           string `json:"runner"`
	Channel          string `json:"channel"`
}

// Fulfilment channels.
var fulfillmentChannels = enumSet("cli", "mcp", "ci", "conductor")

// ErrActionFulfillmentInvalid is a fulfilment the engine could not verify.
var ErrActionFulfillmentInvalid = errors.New("action-fulfillment-invalid")

func reviewDestinationFor(purpose string) string {
	switch purpose {
	case DelegationPurposeAdjudicate:
		return ReviewDestinationAdjudication
	case DelegationPurposeSmoke:
		return ReviewDestinationSmoke
	}
	return ReviewDestinationAttestation
}

// delegatedReviewPolicyDigest is the digest of the policies a delegated run
// is governed by: review, reviewers and delegation.
func (s Store) delegatedReviewPolicyDigest() string {
	var material []byte
	for _, name := range []string{"review.json", "reviewers.json", "delegation.json"} {
		raw, err := os.ReadFile(filepath.Join(s.Root, ".pose", "policy", name))
		if err != nil && !os.IsNotExist(err) {
			return ""
		}
		material = append(material, []byte(name+"\x00")...)
		material = append(material, raw...)
		material = append(material, 0)
	}
	return digestBytes(material)
}

// PrepareDelegatedReviewMaterial is the material of a request for a run of
// purpose on target — a scope or a sealed bundle — with notes and the agents
// it accepts, read from the current bundle, its brief and the policies.
func (s Store) PrepareDelegatedReviewMaterial(target, purpose, notes string, agents []string) (ActionReviewRequest, error) {
	if purpose == "" {
		purpose = DelegationPurposeReview
	}
	if purpose != DelegationPurposeReview && purpose != DelegationPurposeAdjudicate && purpose != DelegationPurposeSmoke {
		return ActionReviewRequest{}, fmt.Errorf("pose: a delegated-review request is for review, adjudicate or smoke, not %s", purpose)
	}
	brief, err := s.RenderReviewBrief(target, DelegationKindForPurpose(purpose), notes)
	if err != nil {
		return ActionReviewRequest{}, err
	}
	worktree, err := dispatchWorktreeIdentity(s.Root)
	if err != nil {
		return ActionReviewRequest{}, err
	}
	accepted := slices.Clone(agents)
	slices.Sort(accepted)
	accepted = slices.Compact(accepted)
	for _, a := range accepted {
		if ValidateSlug(a) != nil {
			return ActionReviewRequest{}, fmt.Errorf("pose: agent %q is not an agent name", a)
		}
	}
	return ActionReviewRequest{Purpose: purpose, Scope: brief.Scope, BundleID: brief.BundleID, BundleDigest: brief.BundleDigest, BriefDigest: brief.Digest,
		BriefTemplate: brief.TemplateVersion, NotesDigest: NotesDigest(notes), PolicyDigest: s.delegatedReviewPolicyDigest(),
		Worktree: digestBytes([]byte(worktree)), Agents: accepted, Destination: reviewDestinationFor(purpose)}, nil
}

// NotesDigest is the digest a request binds of the implementer's notes.
func NotesDigest(notes string) string { return digestBytes([]byte(notes)) }

// delegatedReviewSubjectDigest is the digest a delegated-review request is
// current against: its scope's current bundle and the policies. A request
// whose bundle was superseded, or whose policies changed, is invalidated and
// must be asked again (R8).
func (s Store) delegatedReviewSubjectDigest(m ActionReviewRequest) string {
	current, err := s.CurrentReviewBundle(m.Scope)
	if err != nil {
		return ""
	}
	// A scope whose bundle no longer governs has none current: that is a
	// change of subject, which invalidates the request, not an unknown one.
	bundle := "none"
	if current != nil {
		bundle = current.BundleDigest
	}
	digest, _ := digestJSON(struct {
		Bundle string `json:"bundle"`
		Policy string `json:"policy"`
	}{bundle, s.delegatedReviewPolicyDigest()})
	return digest
}

func (s Store) prepareDelegatedReviewRequest(r *ActionRequest) error {
	if r.Review == nil {
		return errors.New("a delegated-review request carries its review material; open it with pose review request")
	}
	m := *r.Review
	switch {
	case m.Purpose != DelegationPurposeReview && m.Purpose != DelegationPurposeAdjudicate && m.Purpose != DelegationPurposeSmoke:
		return fmt.Errorf("purpose %q is not review, adjudicate or smoke", m.Purpose)
	case m.Destination != reviewDestinationFor(m.Purpose):
		return fmt.Errorf("a %s run is recorded as %s, not %s: the requester does not choose it either", m.Purpose, reviewDestinationFor(m.Purpose), m.Destination)
	case m.BundleID == "" || m.BundleDigest == "" || m.BriefDigest == "" || m.PolicyDigest == "" || m.Worktree == "":
		return errors.New("the review material is incomplete")
	}
	if len(r.Options) > 0 {
		return errors.New("a delegated-review request admits no answer: only a verified run fulfils it")
	}
	if !strings.HasPrefix(r.RequestedBy.Principal, "agent:") && !strings.HasPrefix(r.RequestedBy.Principal, "human:") {
		return errors.New("requested_by must be a principal")
	}
	digest := s.delegatedReviewSubjectDigest(m)
	current, _ := s.CurrentReviewBundle(m.Scope)
	if digest == "" || current == nil {
		return fmt.Errorf("%s has no current sealed bundle", m.Scope)
	}
	if current.BundleID != m.BundleID || current.BundleDigest != m.BundleDigest {
		return fmt.Errorf("bundle %s is not the current bundle of %s", m.BundleID, m.Scope)
	}
	if m.PolicyDigest != s.delegatedReviewPolicyDigest() {
		return errors.New("the policies changed since the material was read")
	}
	r.Subject = &ActionSubject{Digest: digest}
	return nil
}

var fulfillmentRunner = regexp.MustCompile(`^agent:[a-z0-9][a-z0-9._/-]{0,127}$`)

// verifyFulfillment is the engine's reading of a run submitted to fulfil
// request r: the run of the request's brief and bundle, of an accepted
// agent, whose proof verifies, whose attempt ledger is intact, whose
// containment completed and whose conclusion the engine recorded at the
// request's destination (R2, R5). It returns what the fulfilment records.
func (s Store) verifyFulfillment(r ActionRequest, runID, runner, channel string) (ActionFulfillment, error) {
	fail := func(format string, a ...any) (ActionFulfillment, error) {
		return ActionFulfillment{}, fmt.Errorf("%w: "+format, append([]any{ErrActionFulfillmentInvalid}, a...)...)
	}
	if r.Kind != ActionDelegatedReview || r.Review == nil {
		return fail("request %s is not a delegated-review request", r.ID)
	}
	// The runner is an agent: a fulfilment records no person (R9).
	if !fulfillmentRunner.MatchString(runner) {
		return fail("the runner must be an agent: principal; a fulfilment records no person")
	}
	if !fulfillmentChannels[channel] {
		return fail("channel %q is not cli, mcp, ci or conductor", channel)
	}
	m := *r.Review
	run, err := s.LoadProvenanceReviewRun(runID)
	if err != nil {
		return fail("the run cannot be read: %v", err)
	}
	switch {
	case run.Kind != DelegationKindForPurpose(m.Purpose):
		return fail("run %s is a %s run; the request asks for %s", run.RunID, run.Kind, m.Purpose)
	case run.BundleID != m.BundleID || run.BundleDigest != m.BundleDigest || run.Scope != m.Scope:
		return fail("run %s is of another bundle or scope", run.RunID)
	case run.BriefDigest != m.BriefDigest:
		return fail("run %s was sent another brief", run.RunID)
	case len(m.Agents) > 0 && !slices.Contains(m.Agents, run.Adapter):
		return fail("agent %s is not one the request accepts (%s)", run.Adapter, strings.Join(m.Agents, ", "))
	case run.Status != ReviewRunCompleted:
		// Partial, deadline and cleanup outcomes are not a rejection (R10).
		return fail("run %s did not complete: it is %s", run.RunID, run.Status)
	case run.Safety == nil || run.Safety.Operation.Outcome != OutcomeCompleted:
		return fail("run %s did not complete its containment", run.RunID)
	}
	if findings, err := s.ReviewRunLedgerFindings(run.BundleID); err != nil || len(findings) > 0 {
		return fail("the attempt ledger of %s is not intact: %v %s", run.BundleID, err, strings.Join(findings, "; "))
	}
	conclusion, err := s.ReadReviewConclusion(run.RunID)
	if err != nil || conclusion == nil {
		return fail("the engine recorded no conclusion of run %s; conclude it with pose review conclude", run.RunID)
	}
	if conclusion.State != ReviewConclusionRecorded {
		return fail("the conclusion of run %s is %s: %s", run.RunID, conclusion.State, strings.Join(conclusion.Reasons, "; "))
	}
	// The conclusion record says a conclusion was recorded; what was
	// concluded, under which proof and where, is read from the signed effect
	// and the run its proof binds, never from that record (review round 1,
	// F03).
	var proofID, recorded, decision string
	switch m.Destination {
	case ReviewDestinationAttestation:
		bundle, err := s.LoadReviewBundle(run.BundleID)
		if err != nil {
			return fail("the bundle cannot be read: %v", err)
		}
		read, err := s.ReadReviewAttestations(run.BundleID)
		if err != nil || !read.Coverage.Complete() {
			return fail("the attestations of %s cannot be read whole", run.BundleID)
		}
		for _, att := range read.Attestations {
			if att.Run != nil && att.Run.RunID == run.RunID && len(s.runBackedReviewBlockers(bundle, att)) == 0 {
				proofID, recorded, decision = att.Run.ProofID, att.Path, att.Decision
			}
		}
		if proofID == "" {
			return fail("no attestation verifies as the conclusion of run %s", run.RunID)
		}
	case ReviewDestinationAdjudication, ReviewDestinationSmoke:
		proof, why := s.verifiedVerdictProof(run.Kind, run.RunID)
		if proof == "" {
			return fail("the verdict of run %s does not verify: %s", run.RunID, why)
		}
		proofID, recorded = proof, AdjudicationsDir+"/"+run.RunID+".json"
		if m.Destination == ReviewDestinationSmoke {
			recorded = SmokeResultsDir + "/" + run.RunID + ".json"
		}
	}
	assurance, err := s.VerifyReviewRunProof(run.RunID, proofID)
	if err != nil || !assurance.CompleteReview {
		return fail("the proof of run %s does not verify a complete run: %v", run.RunID, err)
	}
	if draft := draftReviewDecision(run.Output); decision == "" {
		decision = draft
	} else if decision != draft && !(decision == "changes-requested" && draft == "rejected" || decision == "rejected" && draft == "changes-requested") {
		return fail("the recorded decision %s is not the run's %s", decision, draft)
	}
	return ActionFulfillment{RunID: run.RunID, ProofID: proofID, AttemptID: run.AttemptID, Decision: decision, Destination: m.Destination,
		Recorded: recorded, Runner: runner, Channel: channel}, nil
}

// verifyRecordedFulfillment verifies a fulfilment kept after its request's
// subject moved on: the run of the request's brief and bundle, of an
// accepted agent, completed, whose proof — signed by a pinned issuer, of
// this project — binds the exact record and output, whose decision the
// fulfilment states, and whose effect is where the fulfilment says. It does
// not ask whether the run still governs the scope: that is what moved on
// (review round 1, F01).
func (s Store) verifyRecordedFulfillment(r ActionRequest, f ActionFulfillment) error {
	if r.Kind != ActionDelegatedReview || r.Review == nil {
		return errors.New("a fulfilment is only of a delegated-review request")
	}
	m := *r.Review
	run, err := s.LoadProvenanceReviewRun(f.RunID)
	if err != nil {
		return err
	}
	switch {
	case run.Kind != DelegationKindForPurpose(m.Purpose) || run.BundleID != m.BundleID || run.BundleDigest != m.BundleDigest || run.BriefDigest != m.BriefDigest:
		return errors.New("the run is not of the request's brief and bundle")
	case len(m.Agents) > 0 && !slices.Contains(m.Agents, run.Adapter):
		return errors.New("the run's agent is not one the request accepts")
	case run.Status != ReviewRunCompleted:
		return errors.New("the run did not complete")
	case f.Destination != m.Destination:
		return errors.New("the fulfilment names another destination")
	}
	if _, err := s.VerifyReviewRunProofRecord(run.RunID, f.ProofID); err != nil {
		return err
	}
	draft := draftReviewDecision(run.Output)
	if f.Decision != draft && !(f.Decision == "changes-requested" && draft == "rejected" || f.Decision == "rejected" && draft == "changes-requested") {
		return errors.New("the fulfilment's decision is not the run's")
	}
	switch m.Destination {
	case ReviewDestinationAttestation:
		read, err := s.ReadReviewAttestations(run.BundleID)
		if err != nil {
			return err
		}
		for _, att := range read.Attestations {
			if att.Path == f.Recorded && att.Run != nil && att.Run.RunID == run.RunID && att.Run.ProofID == f.ProofID && att.Decision == f.Decision {
				return nil
			}
		}
		return errors.New("the attestation the fulfilment names is not the run's")
	default:
		want := AdjudicationsDir + "/" + run.RunID + ".json"
		if m.Destination == ReviewDestinationSmoke {
			want = SmokeResultsDir + "/" + run.RunID + ".json"
		}
		if f.Recorded != want {
			return errors.New("the fulfilment names another record")
		}
		if _, err := os.Stat(filepath.Join(s.Root, filepath.FromSlash(want))); err != nil {
			return errors.New("the verdict record the fulfilment names is gone")
		}
		return nil
	}
}

// FulfilActionRequest appends a verified fulfilment of request env names,
// under the request's lock, against the revision the runner read. A retry
// with the same key and the same run is a faithful no-op; the same key with
// another run, a stale digest or revision, or a request no longer open is
// refused (R8). A fulfilment that does not verify is refused with the
// engine's reason and leaves the request as it was (R5).
func (s Store) FulfilActionRequest(env ActionFulfillmentEnvelope, now time.Time) (ActionRequestView, error) {
	return s.fulfilActionRequestEscalating(env, now)
}

// PreflightFulfillment is FulfilActionRequest without writing.
func (s Store) PreflightFulfillment(env ActionFulfillmentEnvelope, now time.Time) (ActionRequestView, ActionEvent, error) {
	return s.prepareFulfillment(env, now)
}

func (s Store) prepareFulfillment(env ActionFulfillmentEnvelope, now time.Time) (ActionRequestView, ActionEvent, error) {
	view, err := s.LoadActionRequest(env.RequestID)
	if err != nil {
		return view, ActionEvent{}, err
	}
	r := view.Request
	project, err := s.WriteProjectID()
	if err != nil {
		return view, ActionEvent{}, err
	}
	switch {
	case r.Project != project:
		return view, ActionEvent{}, fmt.Errorf("%w: request %s belongs to %s, this project is %s", ErrActionForeignProject, r.ID, r.Project, project)
	case r.Kind != ActionDelegatedReview:
		return view, ActionEvent{}, fmt.Errorf("%w: request %s is a %s request; only a delegated-review request is fulfilled by a run", ErrActionFulfillmentInvalid, r.ID, r.Kind)
	case strings.TrimSpace(env.IdempotencyKey) == "":
		return view, ActionEvent{}, errors.New("pose: an idempotency key is required so a retry never records twice")
	case env.RequestDigest != r.RequestDigest:
		return view, ActionEvent{}, fmt.Errorf("%w: the fulfilment was prepared for %s; the request is now %s", ErrActionStaleDigest, env.RequestDigest, r.RequestDigest)
	case env.ExpectedRevision < 1:
		return view, ActionEvent{}, fmt.Errorf("%w: a fulfilment names the revision it was written against", ErrActionRevisionConflict)
	}
	f, err := s.verifyFulfillment(r, env.RunID, env.Runner, env.Channel)
	if err != nil {
		return view, ActionEvent{}, err
	}
	event := ActionEvent{Type: ActionEventFulfilled, Actor: env.Runner, Channel: env.Channel, RequestDigest: r.RequestDigest, IdempotencyKey: env.IdempotencyKey,
		Fulfillment: &f, Assurance: ReviewIdentityAssuranceVerified}
	// A retry is recognized before state and revision, as at the writer.
	for _, prior := range view.Events {
		if prior.IdempotencyKey != "" && prior.IdempotencyKey == env.IdempotencyKey {
			if prior.Type != ActionEventFulfilled || prior.Fulfillment == nil || prior.Fulfillment.RunID != f.RunID {
				return view, ActionEvent{}, ErrActionIdempotencyConflict
			}
			return view, *withFulfillmentReplay(prior, event), nil
		}
	}
	if env.ExpectedRevision != view.Revision {
		return view, ActionEvent{}, ErrActionRevisionConflict
	}
	if view.SubjectUnknown || view.State != ActionStateOpen {
		return view, ActionEvent{}, fmt.Errorf("%w: request %s is %s", ErrActionNotOpen, r.ID, view.State)
	}
	return view, event, nil
}

// fulfilActionRequestEscalating appends the fulfilment and asks a person
// what the run raised — reviewers in conflict, a business decision —
// recording the escalations with it (R7).
func (s Store) fulfilActionRequestEscalating(env ActionFulfillmentEnvelope, now time.Time) (ActionRequestView, error) {
	view, event, err := s.prepareFulfillment(env, now)
	if err != nil {
		return view, err
	}
	ids, err := s.fulfillmentEscalations(view.Request, *event.Fulfillment, now)
	if err != nil {
		return view, err
	}
	if len(ids) > 0 {
		f := *event.Fulfillment
		f.Escalations = ids
		event.Fulfillment = &f
	}
	return s.appendActionEvent(env.RequestID, event, env.ExpectedRevision, now)
}

// withFulfillmentReplay is the event a retry submits, made the same as the
// one recorded where the engine's later reading may differ only by time.
func withFulfillmentReplay(prior, event ActionEvent) *ActionEvent {
	out := event
	if prior.Fulfillment != nil {
		f := *prior.Fulfillment
		out.Fulfillment = &f
	}
	return &out
}

// fulfillmentEventVerifier is how the journal reader verifies a recorded
// fulfilment. It is assigned in init: the verification reads review state
// whose readers, in turn, read action requests.
var fulfillmentEventVerifier func(Store, ActionRequest, ActionEvent) error

// delegatedReviewSubjectReader is how the journal reader reads the current
// subject of a delegated-review request, assigned in init for the same
// reason.
var delegatedReviewSubjectReader func(Store, ActionReviewRequest) string

// recordedFulfillmentVerifier verifies a fulfilment kept after its subject
// moved on, assigned in init for the same reason.
var recordedFulfillmentVerifier func(Store, ActionRequest, ActionFulfillment) error

func init() {
	recordedFulfillmentVerifier = func(s Store, r ActionRequest, f ActionFulfillment) error { return s.verifyRecordedFulfillment(r, f) }
	fulfillmentEventVerifier = func(s Store, r ActionRequest, e ActionEvent) error { return s.verifyFulfillmentEvent(r, e) }
	delegatedReviewSubjectReader = func(s Store, m ActionReviewRequest) string { return s.delegatedReviewSubjectDigest(m) }
}

// verifyFulfillmentEvent re-verifies a recorded fulfilment on read: a
// fulfilment whose run no longer verifies satisfies nothing.
func (s Store) verifyFulfillmentEvent(r ActionRequest, e ActionEvent) error {
	if e.Fulfillment == nil {
		return fmt.Errorf("%w: fulfilment sequence %d carries no run", ErrActionVerificationFailed, e.Seq)
	}
	if e.RequestDigest != r.RequestDigest {
		return fmt.Errorf("%w: fulfilment sequence %d binds another request version", ErrActionVerificationFailed, e.Seq)
	}
	// The event's header says what its payload says: who submitted it and
	// through which channel (review round 2, R2-F02).
	if e.Actor != e.Fulfillment.Runner || e.Channel != e.Fulfillment.Channel || e.Assurance != ReviewIdentityAssuranceVerified {
		return fmt.Errorf("%w: fulfilment sequence %d names another submitter than its payload", ErrActionVerificationFailed, e.Seq)
	}
	f, err := s.verifyFulfillment(r, e.Fulfillment.RunID, e.Fulfillment.Runner, e.Fulfillment.Channel)
	if err != nil {
		return fmt.Errorf("%w: fulfilment sequence %d: %v", ErrActionVerificationFailed, e.Seq, err)
	}
	recorded := *e.Fulfillment
	recorded.Escalations = nil
	got, _ := json.Marshal(f)
	want, _ := json.Marshal(recorded)
	if string(got) != string(want) {
		return fmt.Errorf("%w: fulfilment sequence %d is not what the engine reads of run %s", ErrActionVerificationFailed, e.Seq, f.RunID)
	}
	return nil
}
