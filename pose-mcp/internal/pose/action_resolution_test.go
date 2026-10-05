package pose

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Spec pose-action-request-resolution. Negative paths first: an answer only
// counts when it comes from the right authority, for the current content,
// once.

func openedDecision(t *testing.T, s Store) ActionRequestView {
	t.Helper()
	view, err := s.OpenActionRequest(decisionRequest(), time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	return view
}

func answer(view ActionRequestView, actor, value, key string) ActionResolution {
	return ActionResolution{RequestID: view.Request.ID, Type: ActionEventAnswered, Actor: actor, Answer: value, RequestDigest: view.Request.RequestDigest, ExpectedRevision: view.Revision, IdempotencyKey: key}
}

func TestAnActorWithoutTheRoleDoesNotSatisfyTheRequest(t *testing.T) {
	s := actionFixture(t)
	view := openedDecision(t, s)
	_, err := s.ResolveActionRequest(answer(view, "agent:impl", "preserve-v1", "k1"), time.Now())
	if !errors.Is(err, ErrActionNotAuthorized) {
		t.Fatalf("the requesting agent answered its own decision: %v", err)
	}
	if after, _ := s.LoadActionRequest(view.Request.ID); after.Revision != 1 || after.State != ActionStateOpen {
		t.Fatalf("a refused answer was recorded: %+v", after)
	}
}

func TestDecliningAnApprovalIsAnAnswerThatAuthorizesNothing(t *testing.T) {
	s := actionFixture(t)
	r := decisionRequest()
	r.Kind, r.Options, r.Recommend = ActionApproval, nil, ""
	view, err := s.OpenActionRequest(r, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	after, err := s.ResolveActionRequest(answer(view, "human:maintainer", "decline", "k1"), time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if after.State != ActionStateAnswered || after.Satisfaction != SatisfactionPending {
		t.Fatalf("a declined approval: %+v", after)
	}
	report, _ := s.ProjectObligations(ObligationQuery{Category: ObligationActorAction})
	if len(report.Obligations) != 1 || !report.Obligations[0].Restricts(PhaseExecution) || !strings.Contains(report.Obligations[0].Message, "does not satisfy") {
		t.Fatalf("a declined approval released its phase: %+v", report.Obligations)
	}
}

func TestADeclaredHumanAnswerIsLabelledAndRefusedWhereVerifiedIsRequired(t *testing.T) {
	s := actionFixture(t)
	view := openedDecision(t, s)
	after, err := s.ResolveActionRequest(answer(view, "human:maintainer", "preserve-v1", "k1"), time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if after.Satisfaction != SatisfactionSatisfied || after.Assurance != ReviewIdentityAssuranceDeclared || len(after.Limitations) == 0 {
		t.Fatalf("declared answer not labelled as declared: %+v", after)
	}
	// The same declared answer under a verified policy is refused.
	s2 := actionFixture(t)
	_ = os.WriteFile(filepath.Join(s2.Root, ".pose/policy/actions.json"), []byte(`{"schema_version":1,"roles":{"maintainer":["human:maintainer"]},"identity_assurance":"verified"}`), 0o644)
	view2 := openedDecision(t, s2)
	if _, err := s2.ResolveActionRequest(answer(view2, "human:maintainer", "preserve-v1", "k1"), time.Now()); !errors.Is(err, ErrActionVerificationFailed) {
		t.Fatalf("an agent writing human:maintainer passed as verified: %v", err)
	}
}

func TestCancelAndWaiveNeedTheRightAuthority(t *testing.T) {
	s := actionFixture(t)
	view := openedDecision(t, s)
	cancel := ActionResolution{RequestID: view.Request.ID, Type: ActionEventCancelled, Actor: "agent:other", Reason: "not needed", RequestDigest: view.Request.RequestDigest, ExpectedRevision: 1, IdempotencyKey: "c1"}
	if _, err := s.ResolveActionRequest(cancel, time.Now()); !errors.Is(err, ErrActionNotAuthorized) {
		t.Fatalf("a third party cancelled the request: %v", err)
	}
	waive := cancel
	waive.Type, waive.Actor, waive.IdempotencyKey = ActionEventWaived, "agent:impl", "w1"
	if _, err := s.ResolveActionRequest(waive, time.Now()); !errors.Is(err, ErrActionNotAuthorized) {
		t.Fatalf("the requester waived a condition it does not own: %v", err)
	}
	waive.Actor, waive.Reason = "human:maintainer", ""
	if _, err := s.ResolveActionRequest(waive, time.Now()); err == nil {
		t.Fatal("a waiver without a reason was accepted")
	}
	waive.Reason = "accepted risk, recorded in the review"
	after, err := s.ResolveActionRequest(waive, time.Now())
	if err != nil || after.Satisfaction != SatisfactionWaived {
		t.Fatalf("an authorized waiver: %+v %v", after, err)
	}
	s2 := actionFixture(t)
	view2 := openedDecision(t, s2)
	ownCancel := ActionResolution{RequestID: view2.Request.ID, Type: ActionEventCancelled, Actor: "agent:impl", Reason: "the question was not material", RequestDigest: view2.Request.RequestDigest, ExpectedRevision: 1, IdempotencyKey: "c1"}
	if after, err := s2.ResolveActionRequest(ownCancel, time.Now()); err != nil || after.State != ActionStateCancelled {
		t.Fatalf("the requester withdrawing its own question: %+v %v", after, err)
	}
}

func TestReplayIsIdempotentAndConflictsAreRefused(t *testing.T) {
	s := actionFixture(t)
	view := openedDecision(t, s)
	res := answer(view, "human:maintainer", "preserve-v1", "same-key")
	first, err := s.ResolveActionRequest(res, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	replay, err := s.ResolveActionRequest(res, time.Now().Add(time.Minute))
	if err != nil || replay.Revision != first.Revision {
		t.Fatalf("a retry recorded twice: %d vs %d (%v)", replay.Revision, first.Revision, err)
	}
	conflicting := res
	conflicting.Answer = "break-v1"
	if _, err := s.ResolveActionRequest(conflicting, time.Now()); !errors.Is(err, ErrActionIdempotencyConflict) {
		t.Fatalf("the same key with another answer: %v", err)
	}
	second := answer(view, "human:maintainer", "break-v1", "other-key")
	if _, err := s.ResolveActionRequest(second, time.Now()); !errors.Is(err, ErrActionRevisionConflict) {
		t.Fatalf("a second answer against the same revision was accepted: %v", err)
	}
	if after, _ := s.LoadActionRequest(view.Request.ID); after.Answer != "preserve-v1" || after.Revision != 2 {
		t.Fatalf("history is not coherent: %+v", after)
	}
}

func TestAStaleDigestOrAChangedSubjectDoesNotSatisfy(t *testing.T) {
	s := actionFixture(t)
	r := decisionRequest()
	r.Subject = &ActionSubject{Node: NodeRef{Artifact: "self", Kind: "requirement", ID: "R4"}}
	view, err := s.OpenActionRequest(r, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	stale := answer(view, "human:maintainer", "preserve-v1", "k1")
	stale.RequestDigest = "sha256:0000"
	if _, err := s.ResolveActionRequest(stale, time.Now()); !errors.Is(err, ErrActionStaleDigest) {
		t.Fatalf("an answer for another version of the request: %v", err)
	}
	if _, err := s.ResolveActionRequest(answer(view, "human:maintainer", "preserve-v1", "k2"), time.Now()); err != nil {
		t.Fatal(err)
	}
	spec := filepath.Join(s.Root, ".pose/specs/2026-10-04-storage.md")
	raw, _ := os.ReadFile(spec)
	_ = os.WriteFile(spec, []byte(strings.Replace(string(raw), "Keep reading schema v1 records.", "Read v1 records only during migration.", 1)), 0o644)
	after, _ := s.LoadActionRequest(view.Request.ID)
	if after.State != ActionStateInvalidated || after.Satisfaction != SatisfactionInvalidated {
		t.Fatalf("an answer for content that changed still satisfies: %+v", after)
	}
}

func TestAResolutionFromAnotherProjectIsRefused(t *testing.T) {
	// Each fixture root is its own project, identified by its directory; a
	// declared identity in the environment would make them all one project.
	t.Setenv("POSE_DEFAULT_PROJECT_ID", "")
	t.Setenv("POSE_PROJECT_ROOTS", "")
	s := actionFixture(t)
	view := openedDecision(t, s)
	other := actionFixture(t)
	_ = os.MkdirAll(filepath.Join(other.Root, ".pose/actions"), 0o755)
	raw, _ := os.ReadFile(actionJournalPath(s.Root, view.Request.ID))
	_ = os.WriteFile(actionJournalPath(other.Root, view.Request.ID), raw, 0o644)
	if _, err := other.ResolveActionRequest(answer(view, "human:maintainer", "preserve-v1", "k1"), time.Now()); !errors.Is(err, ErrActionForeignProject) {
		t.Fatalf("a request copied from another project was resolved here: %v", err)
	}
}

func TestAVerifiedClaimBoundToTheRequestIsAccepted(t *testing.T) {
	s := actionFixture(t)
	seed := make([]byte, ed25519.SeedSize)
	for i := range seed {
		seed[i] = byte(i + 7)
	}
	private := ed25519.NewKeyFromSeed(seed)
	public := private.Public().(ed25519.PublicKey)
	pin := "harne8:confirm#" + digestBytes(public)
	_ = os.WriteFile(filepath.Join(s.Root, ".pose/policy/review.json"), []byte(`{"schema_version":2,"enabled":true,"profiles":{"spec":"spec-closeout@1"},"authority_audience":"proj.fixture","trusted_attestation_issuers":["`+pin+`"],"human_authority_issuers":["`+pin+`"]}`), 0o644)
	_ = os.WriteFile(filepath.Join(s.Root, ".pose/policy/actions.json"), []byte(`{"schema_version":1,"roles":{"maintainer":["human:maintainer"]},"identity_assurance":"verified"}`), 0o644)
	view := openedDecision(t, s)
	claim := ActionAuthorityClaim{SchemaVersion: 1, Project: "proj.fixture", Audience: "proj.fixture", RequestID: view.Request.ID, RequestDigest: view.Request.RequestDigest,
		Principal: "human:maintainer", Role: "human", Answer: "preserve-v1", Issuer: "harne8:confirm", IssuedAt: time.Now().UTC().Format(time.RFC3339)}
	canonical, _ := json.Marshal(claim)
	envelope := ActionClaimEnvelope{Issuer: "harne8:confirm", Algorithm: "ed25519", PublicKey: base64.StdEncoding.EncodeToString(public), Signature: base64.StdEncoding.EncodeToString(ed25519.Sign(private, canonical))}

	tampered := answer(view, "human:maintainer", "break-v1", "k0")
	tampered.Claim, tampered.Envelope = &claim, &envelope
	if _, err := s.ResolveActionRequest(tampered, time.Now()); !errors.Is(err, ErrActionVerificationFailed) {
		t.Fatalf("a claim for another answer verified: %v", err)
	}
	res := answer(view, "human:maintainer", "preserve-v1", "k1")
	res.Claim, res.Envelope = &claim, &envelope
	after, err := s.ResolveActionRequest(res, time.Now())
	if err != nil || after.Assurance != ReviewIdentityAssuranceVerified || after.Satisfaction != SatisfactionSatisfied || len(after.Limitations) != 0 {
		t.Fatalf("a verified, bound claim: %+v %v", after, err)
	}
}
