package pose

import (
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// openReviewRequest opens a delegated-review request of purpose on the
// fixture's spec, for runs briefed with the fixture's notes.
func openReviewRequest(t *testing.T, f runProvenanceFixture, purpose string, agents ...string) ActionRequestView {
	t.Helper()
	m, err := f.store.PrepareDelegatedReviewMaterial("spec:backend", purpose, "fixture labelled notes", agents)
	if err != nil {
		t.Fatal(err)
	}
	view, err := f.store.OpenActionRequest(ActionRequest{Origin: "spec:backend", Kind: ActionDelegatedReview, Question: "Review spec:backend by a delegated agent",
		RequestedBy: ActionPrincipal{Principal: "agent:implementer"}, Targets: []NodeRef{{Kind: "requirement", ID: "R1"}},
		Effects: []ObligationEffect{{Phase: PhaseCloseout, Mode: EffectBlock}}, Review: &m}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	return view
}

func fulfil(f runProvenanceFixture, view ActionRequestView, run, key string) (ActionRequestView, error) {
	return f.store.FulfilActionRequest(ActionFulfillmentEnvelope{RequestID: view.Request.ID, RequestDigest: view.Request.RequestDigest, ExpectedRevision: view.Revision,
		IdempotencyKey: key, RunID: run, Runner: "agent:runner", Channel: "cli"}, time.Now())
}

// R1, R2, R5, R6, R8: a delegated-review request binds its material, admits
// no answer, and is fulfilled only by a verified run of its brief whose
// conclusion the engine recorded; a rejecting run fulfils it without
// approving; a retry is a no-op and a changed one is refused.
func TestDelegatedReviewRequestIsFulfilledByAVerifiedRun(t *testing.T) {
	f, draftPath := conclusionFixture(t)
	view := openReviewRequest(t, f, DelegationPurposeReview)
	if view.State != ActionStateOpen || view.Request.Review.Destination != ReviewDestinationAttestation || view.NextStep == nil || view.NextStep.Kind != "fulfil" {
		t.Fatalf("opened: %+v", view)
	}
	if _, err := f.store.ResolveActionRequest(ActionResolution{RequestID: view.Request.ID, Type: ActionEventAnswered, Actor: "human:ada", Answer: "approve",
		RequestDigest: view.Request.RequestDigest, ExpectedRevision: view.Revision, IdempotencyKey: "answer"}, time.Now()); err == nil {
		t.Fatal("an actor answered a delegated-review request")
	}
	att := approvedBundleAttestation(f.bundle, "agent:reviewer")
	att.Decision = "changes-requested"
	findings := []ReviewDraftFinding{{ID: "B1", Severity: "high", Disposition: "blocker", File: "x.go", Line: 3, Summary: "a defect"}}
	if err := os.WriteFile(draftPath, []byte(draftFrom(t, att, findings)), 0o600); err != nil {
		t.Fatal(err)
	}
	run := dispatchConclusion(t, f, "the requested review")
	if _, err := fulfil(f, view, run.RunID, "k1"); !errors.Is(err, ErrActionFulfillmentInvalid) || !strings.Contains(err.Error(), "no conclusion") {
		t.Fatalf("a run with no conclusion fulfilled: %v", err)
	}
	if c, err := f.store.ConcludeReviewRun(run.RunID, f.key.Name, true, time.Now()); err != nil || c.State != ReviewConclusionRecorded {
		t.Fatalf("conclude: %+v %v", c, err)
	}
	done, err := fulfil(f, view, run.RunID, "k1")
	if err != nil || done.State != ActionStateFulfilled || done.Satisfaction != SatisfactionSatisfied || done.Answer != "changes-requested" {
		t.Fatalf("fulfil: %+v %v", done, err)
	}
	if verification, _ := f.store.VerifyReviewBundle(f.bundle.Payload.Scope.Ref); verification.Approved {
		t.Fatal("a fulfilled rejection approved the scope")
	}
	// A fulfilled request is not waiting (review round 3, R3-F02).
	if waits, err := f.store.GovernanceWaits(time.Now()); err != nil || waits.Open != 0 || waits.Resolved != 1 {
		t.Fatalf("waits: %+v %v", waits, err)
	}
	again, err := fulfil(f, view, run.RunID, "k1")
	if err != nil || again.Revision != done.Revision {
		t.Fatalf("a retry was not a no-op: %+v %v", again, err)
	}
	if _, err := fulfil(f, done, run.RunID, "k2"); !errors.Is(err, ErrActionNotOpen) {
		t.Fatalf("a fulfilled request was fulfilled again: %v", err)
	}
	reread, err := f.store.LoadActionRequest(view.Request.ID)
	if err != nil || reread.State != ActionStateFulfilled {
		t.Fatalf("read back: %+v %v", reread, err)
	}
}

// R2, R5: a run of another brief, of an agent the request does not accept,
// or whose conclusion was refused, does not fulfil it.
func TestDelegatedReviewRequestRefusesOtherRuns(t *testing.T) {
	f, draftPath := conclusionFixture(t)
	view := openReviewRequest(t, f, DelegationPurposeReview, "someone-else")
	if err := os.WriteFile(draftPath, []byte(draftFrom(t, approvedBundleAttestation(f.bundle, "agent:reviewer"), nil)), 0o600); err != nil {
		t.Fatal(err)
	}
	run := dispatchConclusion(t, f, "another agent")
	if _, err := f.store.ConcludeReviewRun(run.RunID, f.key.Name, true, time.Now()); err != nil {
		t.Fatal(err)
	}
	if _, err := fulfil(f, view, run.RunID, "k"); err == nil || !strings.Contains(err.Error(), "not one the request accepts") {
		t.Fatalf("an unaccepted agent fulfilled: %v", err)
	}
	other := openReviewRequest(t, f, DelegationPurposeAdjudicate)
	if _, err := fulfil(f, other, run.RunID, "k"); err == nil || !strings.Contains(err.Error(), "asks for adjudicate") {
		t.Fatalf("a review fulfilled an adjudication request: %v", err)
	}
	if _, err := f.store.FulfilActionRequest(ActionFulfillmentEnvelope{RequestID: other.Request.ID, RequestDigest: other.Request.RequestDigest, ExpectedRevision: other.Revision,
		IdempotencyKey: "h", RunID: run.RunID, Runner: "human:ada", Channel: "cli"}, time.Now()); err == nil || !strings.Contains(err.Error(), "records no person") {
		t.Fatalf("a person was recorded as a runner: %v", err)
	}
}

// copyPoseRecords copies what a runner's checkout wrote under .pose — runs,
// proofs, ledger, conclusions, attestations — into root, as a CI job's
// commit lands them; the request journals stay root's own.
func copyPoseRecords(t *testing.T, from, root string) {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(from, ".pose"))
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if e.Name() == "actions" || !e.IsDir() {
			continue
		}
		if out, err := exec.Command("cp", "-a", filepath.Join(from, ".pose", e.Name()), filepath.Join(root, ".pose")).CombinedOutput(); err != nil {
			t.Fatalf("%v %s", err, out)
		}
	}
}

// R3, R9: a run a CI job performs in its own checkout fulfils the request
// in the repository, through the same envelope and verifier, and the
// fulfilment verifies in any other checkout of the project.
func TestDelegatedReviewRequestIsFulfilledByACIRunner(t *testing.T) {
	f, draftPath := conclusionFixture(t)
	view := openReviewRequest(t, f, DelegationPurposeReview)
	if err := os.WriteFile(draftPath, []byte(draftFrom(t, approvedBundleAttestation(f.bundle, "agent:reviewer"), nil)), 0o600); err != nil {
		t.Fatal(err)
	}
	ci := filepath.Join(t.TempDir(), "ci-checkout")
	if out, err := exec.Command("cp", "-a", f.store.Root, ci).CombinedOutput(); err != nil {
		t.Fatalf("%v %s", err, out)
	}
	runner := f
	runner.store = Store{Root: ci}
	run := dispatchConclusion(t, runner, "the CI job")
	if c, err := runner.store.ConcludeReviewRun(run.RunID, f.key.Name, true, time.Now()); err != nil || c.State != ReviewConclusionRecorded {
		t.Fatalf("the CI conclusion: %+v %v", c, err)
	}
	copyPoseRecords(t, ci, f.store.Root)
	done, err := f.store.FulfilActionRequest(ActionFulfillmentEnvelope{RequestID: view.Request.ID, RequestDigest: view.Request.RequestDigest, ExpectedRevision: view.Revision,
		IdempotencyKey: "ci-" + run.RunID, RunID: run.RunID, Runner: "agent:ci-runner", Channel: "ci"}, time.Now())
	if err != nil || done.State != ActionStateFulfilled || done.Answer != "approved" {
		t.Fatalf("a CI run did not fulfil: %+v %v", done, err)
	}
	other := filepath.Join(t.TempDir(), "another-clone")
	if out, err := exec.Command("cp", "-a", f.store.Root, other).CombinedOutput(); err != nil {
		t.Fatalf("%v %s", err, out)
	}
	if read, err := (Store{Root: other}).LoadActionRequest(view.Request.ID); err != nil || read.State != ActionStateFulfilled {
		t.Fatalf("the fulfilment does not verify in another checkout: %+v %v", read, err)
	}
	if verification, err := (Store{Root: other}).VerifyReviewBundle(f.bundle.Payload.Scope.Ref); err != nil || !verification.Approved {
		t.Fatalf("the run-backed attestation does not verify in another checkout: %v %v", verification.Blockers, err)
	}
}

// R8, R10: a request whose policies changed is invalidated and cannot be
// fulfilled; a run that did not complete is not a rejection and fulfils
// nothing.
func TestDelegatedReviewRequestRevalidatesAndRefusesIncompleteRuns(t *testing.T) {
	f, draftPath := conclusionFixture(t)
	view := openReviewRequest(t, f, DelegationPurposeReview)
	if _, err := fulfil(f, view, f.run.RunID, "k"); err == nil || !strings.Contains(err.Error(), "another brief") && !strings.Contains(err.Error(), "did not complete") && !strings.Contains(err.Error(), "no conclusion") {
		t.Fatalf("the fixture's run fulfilled: %v", err)
	}
	// The adapter prints a draft file that is gone: the run fails.
	if err := os.Remove(draftPath); err != nil {
		t.Fatal(err)
	}
	plan, err := f.store.PlanReviewDispatch(f.bundle.BundleID, "review", "fixture labelled notes", "fake", "a failing run")
	if err != nil {
		t.Fatal(err)
	}
	run, _ := f.store.RunReviewDispatch(plan, time.Now)
	if run.RunID == "" || run.Status == ReviewRunCompleted {
		t.Fatalf("the run did not fail: %+v", run)
	}
	if _, err := fulfil(f, view, run.RunID, "failed"); err == nil || !strings.Contains(err.Error(), "did not complete") {
		t.Fatalf("an incomplete run fulfilled: %s %v", run.Status, err)
	}
	raw, _ := os.ReadFile(filepath.Join(f.store.Root, ".pose", "policy", "review.json"))
	if err := os.WriteFile(filepath.Join(f.store.Root, ".pose", "policy", "review.json"), append(raw, ' '), 0o644); err != nil {
		t.Fatal(err)
	}
	stale, err := f.store.LoadActionRequest(view.Request.ID)
	if err != nil || stale.State != ActionStateInvalidated {
		t.Fatalf("a request under changed policies stayed %s: %v", stale.State, err)
	}
	if _, err := fulfil(f, stale, f.run.RunID, "stale"); err == nil {
		t.Fatal("an invalidated request was fulfilled")
	}
}

// R7: what a run cannot settle goes to a person as a decision request —
// a business decision the reviewer raised, reviewers in conflict on the
// bundle — once, restricting what the delegated request restricts.
func TestDelegatedReviewRequestEscalatesToAPerson(t *testing.T) {
	f, draftPath := conclusionFixture(t)
	view := openReviewRequest(t, f, DelegationPurposeReview)
	// An earlier run rejected the bundle.
	rejecting := approvedBundleAttestation(f.bundle, "agent:reviewer")
	rejecting.Decision = "changes-requested"
	findings := []ReviewDraftFinding{{ID: "B1", Severity: "high", Disposition: "blocker", File: "x.go", Line: 3, Summary: "a defect"}}
	if err := os.WriteFile(draftPath, []byte(draftFrom(t, rejecting, findings)), 0o600); err != nil {
		t.Fatal(err)
	}
	first := dispatchConclusion(t, f, "the first reviewer")
	if c, err := f.store.ConcludeReviewRun(first.RunID, f.key.Name, true, time.Now()); err != nil || c.State != ReviewConclusionRecorded {
		t.Fatalf("first: %+v %v", c, err)
	}
	// The requested run approves, and asks a business question.
	approving := approvedBundleAttestation(f.bundle, "agent:reviewer")
	text := draftFrom(t, approving, nil)
	text = strings.Replace(text, `"decision":"approved"`, `"decision":"approved","business_decision":{"question":"Ship to every tenant now?","options":[{"id":"ship","consequence":"every tenant gets it"},{"id":"pilot","consequence":"one tenant first"}]}`, 1)
	if err := os.WriteFile(draftPath, []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}
	second := dispatchConclusion(t, f, "the requested reviewer")
	if c, err := f.store.ConcludeReviewRun(second.RunID, f.key.Name, true, time.Now()); err != nil || c.State != ReviewConclusionRecorded {
		t.Fatalf("second: %+v %v", c, err)
	}
	done, err := fulfil(f, view, second.RunID, "k")
	if err != nil || done.State != ActionStateFulfilled {
		t.Fatalf("fulfil: %+v %v", done, err)
	}
	escalations := done.Events[len(done.Events)-1].Fulfillment.Escalations
	if len(escalations) != 2 {
		t.Fatalf("escalations: %v", escalations)
	}
	questions := []string{}
	for _, id := range escalations {
		e, err := f.store.LoadActionRequest(id)
		if err != nil || e.Request.Kind != ActionDecision || e.Request.Recipient.Role != "maintainer" || len(e.Request.Options) != 2 || e.State != ActionStateOpen {
			t.Fatalf("escalation %s: %+v %v", id, e.Request, err)
		}
		questions = append(questions, e.Request.Question)
	}
	if !strings.Contains(strings.Join(questions, "|"), "Ship to every tenant now?") || !strings.Contains(strings.Join(questions, "|"), "disagree") {
		t.Fatalf("questions: %v", questions)
	}
	// Asking again returns the one already open.
	var conflict ActionRequest
	for _, id := range escalations {
		if e, _ := f.store.LoadActionRequest(id); strings.Contains(e.Request.Question, "disagree") {
			conflict = e.Request
		}
	}
	again, err := f.store.EscalateDelegatedReview(view.Request, EscalationConflict, conflict.Question, "", conflict.Options, time.Now())
	if err != nil || !containsString(escalations, again) {
		t.Fatalf("a retry asked again: %s %v", again, err)
	}
}

// R11: the journey per kind — an adjudication request is fulfilled by an
// adjudication run whose verdict the engine recorded on the spec, a smoke
// request by a smoke run whose observations it recorded as evidence.
func TestDelegatedReviewRequestPerKind(t *testing.T) {
	for _, c := range []struct {
		purpose, kind string
		draft         ReviewDraft
		recorded      string
	}{
		{DelegationPurposeAdjudicate, ReviewBriefKindAdjudication, ReviewDraft{Decision: "changes-requested", Questions: []ReviewDraftQuestion{{ID: "R1", Verdict: AdjudicationDoesNotHold, Rationale: "no"}}}, AdjudicationsDir},
		{DelegationPurposeSmoke, ReviewBriefKindSmoke, ReviewDraft{Decision: "approved", Surfaces: []ReviewDraftSurface{{Ref: "contract:backend-api", Outcome: SmokeObserved, Commands: []string{"go run ./api"}, Observed: "ok"}}}, SmokeResultsDir},
	} {
		t.Run(c.purpose, func(t *testing.T) {
			f, draftPath := conclusionFixture(t)
			view := openReviewRequest(t, f, c.purpose)
			if err := os.WriteFile(draftPath, []byte(verdictDraft(t, c.draft)), 0o600); err != nil {
				t.Fatal(err)
			}
			plan, err := f.store.PlanReviewDispatch(f.bundle.BundleID, c.kind, "fixture labelled notes", "fake", c.purpose)
			if err != nil {
				t.Fatal(err)
			}
			run, err := f.store.RunReviewDispatch(plan, time.Now)
			if err != nil || run.Status != ReviewRunCompleted {
				t.Fatalf("dispatch: %+v %v", run, err)
			}
			if concluded, err := f.store.ConcludeReviewRun(run.RunID, f.key.Name, true, time.Now()); err != nil || concluded.State != ReviewConclusionRecorded {
				t.Fatalf("conclude: %+v %v", concluded, err)
			}
			done, err := fulfil(f, view, run.RunID, "k")
			if err != nil || done.State != ActionStateFulfilled || done.Answer != c.draft.Decision || !strings.HasPrefix(done.Events[len(done.Events)-1].Fulfillment.Recorded, c.recorded) {
				t.Fatalf("fulfil: %+v %v", done, err)
			}
		})
	}
}

// A fulfilled request stays readable and fulfilled once the scope moves on,
// and a fulfilment that no longer verifies while the request is current
// satisfies nothing without making the journal unreadable.
func TestDelegatedReviewFulfillmentOnRead(t *testing.T) {
	f, draftPath := conclusionFixture(t)
	view := openReviewRequest(t, f, DelegationPurposeReview)
	if err := os.WriteFile(draftPath, []byte(draftFrom(t, approvedBundleAttestation(f.bundle, "agent:reviewer"), nil)), 0o600); err != nil {
		t.Fatal(err)
	}
	run := dispatchConclusion(t, f, "read")
	if _, err := f.store.ConcludeReviewRun(run.RunID, f.key.Name, true, time.Now()); err != nil {
		t.Fatal(err)
	}
	if _, err := fulfil(f, view, run.RunID, "k"); err != nil {
		t.Fatal(err)
	}
	conclusion := filepath.Join(f.store.Root, filepath.FromSlash(ReviewConclusionsDir), run.RunID+".json")
	raw, _ := os.ReadFile(conclusion)
	if err := os.Remove(conclusion); err != nil {
		t.Fatal(err)
	}
	if v, err := f.store.LoadActionRequest(view.Request.ID); err != nil || v.State == ActionStateFulfilled || v.Satisfaction == SatisfactionSatisfied || !strings.Contains(strings.Join(v.Limitations, ";"), "does not verify") {
		t.Fatalf("an unverifiable fulfilment: %+v %v", v, err)
	}
	if err := os.WriteFile(conclusion, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	policy := filepath.Join(f.store.Root, ".pose", "policy", "review.json")
	praw, _ := os.ReadFile(policy)
	if err := os.WriteFile(policy, append(praw, ' '), 0o644); err != nil {
		t.Fatal(err)
	}
	if v, err := f.store.LoadActionRequest(view.Request.ID); err != nil || v.State != ActionStateFulfilled || v.Satisfaction == SatisfactionSatisfied || !strings.Contains(strings.Join(v.Limitations, ";"), "changed since") {
		t.Fatalf("a fulfilled request after the scope moved on: %+v %v", v, err)
	}
	if _, err := f.store.ListActionRequests(); err != nil {
		t.Fatalf("the journals became unreadable: %v", err)
	}
}

// R2, R5: a run sent another brief — other notes — and a run whose
// conclusion the engine refused do not fulfil the request.
func TestDelegatedReviewRequestRefusesOtherBriefsAndRefusedConclusions(t *testing.T) {
	f, draftPath := conclusionFixture(t)
	view := openReviewRequest(t, f, DelegationPurposeReview)
	if err := os.WriteFile(draftPath, []byte(draftFrom(t, approvedBundleAttestation(f.bundle, "agent:reviewer"), nil)), 0o600); err != nil {
		t.Fatal(err)
	}
	plan, err := f.store.PlanReviewDispatch(f.bundle.BundleID, "review", "other notes", "fake", "other notes")
	if err != nil {
		t.Fatal(err)
	}
	other, err := f.store.RunReviewDispatch(plan, time.Now)
	if err != nil || other.Status != ReviewRunCompleted {
		t.Fatalf("dispatch: %+v %v", other, err)
	}
	if _, err := f.store.ConcludeReviewRun(other.RunID, f.key.Name, true, time.Now()); err != nil {
		t.Fatal(err)
	}
	if _, err := fulfil(f, view, other.RunID, "brief"); err == nil || !strings.Contains(err.Error(), "another brief") {
		t.Fatalf("a run of another brief fulfilled: %v", err)
	}
	// A draft with no typed conclusion: the engine refuses to conclude it.
	if err := os.WriteFile(draftPath, []byte("decision: approved\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	refusedRun := dispatchConclusion(t, f, "refused")
	if c, err := f.store.ConcludeReviewRun(refusedRun.RunID, f.key.Name, true, time.Now()); err != nil || c.State != ReviewConclusionRefused {
		t.Fatalf("conclude: %+v %v", c, err)
	}
	if _, err := fulfil(f, view, refusedRun.RunID, "refused"); err == nil || !strings.Contains(err.Error(), "refused") {
		t.Fatalf("a refused conclusion fulfilled: %v", err)
	}
}

// appendForgedEvent writes event at the end of a request's journal, as a
// writer that bypasses the engine would.
func appendForgedEvent(t *testing.T, s Store, id string, event ActionEvent) {
	t.Helper()
	view, err := s.LoadActionRequest(id)
	if err != nil {
		t.Fatal(err)
	}
	event.SchemaVersion, event.Seq = ActionRequestSchemaVersion, view.Revision+1
	if event.At == "" {
		event.At = time.Now().UTC().Format(time.RFC3339)
	}
	raw, _ := json.Marshal(event)
	file, err := os.OpenFile(actionJournalPath(s.Root, id), os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if _, err := file.Write(append(raw, '\n')); err != nil {
		t.Fatal(err)
	}
}

// Review round 1, F01 and F02: a fulfilled event no engine wrote satisfies
// nothing — neither while the request is current, nor once the scope moved
// on — and on a request of another kind it is ignored without breaking the
// read.
func TestForgedFulfillmentsSatisfyNothing(t *testing.T) {
	f, _ := conclusionFixture(t)
	view := openReviewRequest(t, f, DelegationPurposeReview)
	forged := ActionEvent{Type: ActionEventFulfilled, Actor: "agent:forger", Channel: "cli", RequestDigest: view.Request.RequestDigest, IdempotencyKey: "forged",
		Fulfillment: &ActionFulfillment{RunID: "rrn-0000000000000000", ProofID: "rrp-" + strings.Repeat("0", 64), Decision: "approved", Destination: ReviewDestinationAttestation,
			Recorded: ".pose/review-attestations/rva-0000000000000000.json", Runner: "agent:forger", Channel: "cli"}}
	appendForgedEvent(t, f.store, view.Request.ID, forged)
	if v, err := f.store.LoadActionRequest(view.Request.ID); err != nil || v.State == ActionStateFulfilled || v.Satisfaction == SatisfactionSatisfied {
		t.Fatalf("a forged fulfilment counted while current: %+v %v", v.State, err)
	}
	policy := filepath.Join(f.store.Root, ".pose", "policy", "review.json")
	raw, _ := os.ReadFile(policy)
	if err := os.WriteFile(policy, append(raw, ' '), 0o644); err != nil {
		t.Fatal(err)
	}
	if v, err := f.store.LoadActionRequest(view.Request.ID); err != nil || v.State == ActionStateFulfilled || v.Satisfaction == SatisfactionSatisfied {
		t.Fatalf("a forged fulfilment counted once the scope moved on: %+v %v", v.State, err)
	}
	approval, err := f.store.OpenActionRequest(ActionRequest{Origin: "spec:backend", Kind: ActionApproval, Question: "Approve?", RequestedBy: ActionPrincipal{Principal: "agent:implementer"},
		Recipient: ObligationActor{Role: "maintainer"}, Targets: []NodeRef{{}}, Effects: []ObligationEffect{{Phase: PhaseCloseout, Mode: EffectBlock}}}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	forged.RequestDigest = approval.Request.RequestDigest
	appendForgedEvent(t, f.store, approval.Request.ID, forged)
	if v, err := f.store.LoadActionRequest(approval.Request.ID); err != nil || v.State != ActionStateOpen {
		t.Fatalf("a fulfilment on an approval: %+v %v", v.State, err)
	}
	if _, err := f.store.ListActionRequests(); err != nil {
		t.Fatalf("the journals became unreadable: %v", err)
	}
}

// Review round 1, F03: what a fulfilment records is read from the signed
// effect and the run, not from the conclusion record, which is not signed.
func TestFulfillmentReadsTheSignedDecision(t *testing.T) {
	f, draftPath := conclusionFixture(t)
	view := openReviewRequest(t, f, DelegationPurposeReview)
	att := approvedBundleAttestation(f.bundle, "agent:reviewer")
	att.Decision = "changes-requested"
	findings := []ReviewDraftFinding{{ID: "B1", Severity: "high", Disposition: "blocker", File: "x.go", Line: 3, Summary: "a defect"}}
	if err := os.WriteFile(draftPath, []byte(draftFrom(t, att, findings)), 0o600); err != nil {
		t.Fatal(err)
	}
	run := dispatchConclusion(t, f, "rewritten")
	if _, err := f.store.ConcludeReviewRun(run.RunID, f.key.Name, true, time.Now()); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(f.store.Root, filepath.FromSlash(ReviewConclusionsDir), run.RunID+".json")
	raw, _ := os.ReadFile(path)
	if err := os.WriteFile(path, []byte(strings.Replace(string(raw), `"decision": "changes-requested"`, `"decision": "approved"`, 1)), 0o644); err != nil {
		t.Fatal(err)
	}
	done, err := fulfil(f, view, run.RunID, "k")
	if err != nil || done.Answer != "changes-requested" {
		t.Fatalf("the rewritten conclusion decided: %+v %v", done.Answer, err)
	}
}

// Review round 1, F06: every field of the review material is in the request
// digest.
func TestDelegatedReviewMaterialIsInTheDigest(t *testing.T) {
	base := ActionRequest{Project: "p", Origin: "spec:x", Kind: ActionDelegatedReview, Question: "q", Review: &ActionReviewRequest{Purpose: "review", Scope: "spec:x",
		BundleID: "b", BundleDigest: "d", BriefDigest: "f", BriefTemplate: "t", NotesDigest: "n", PolicyDigest: "p", Worktree: "w", Agents: []string{"a"}, Destination: "attestation"}}
	digest := ActionRequestDigest(base)
	for _, change := range []func(*ActionReviewRequest){
		func(m *ActionReviewRequest) { m.Purpose = "smoke" }, func(m *ActionReviewRequest) { m.Scope = "spec:y" },
		func(m *ActionReviewRequest) { m.BundleID = "c" }, func(m *ActionReviewRequest) { m.BundleDigest = "e" },
		func(m *ActionReviewRequest) { m.BriefDigest = "g" }, func(m *ActionReviewRequest) { m.BriefTemplate = "u" },
		func(m *ActionReviewRequest) { m.NotesDigest = "o" }, func(m *ActionReviewRequest) { m.PolicyDigest = "q" },
		func(m *ActionReviewRequest) { m.Worktree = "x" }, func(m *ActionReviewRequest) { m.Agents = []string{"b"} },
		func(m *ActionReviewRequest) { m.Destination = "smoke-evidence" },
	} {
		m := *base.Review
		change(&m)
		changed := base
		changed.Review = &m
		if ActionRequestDigest(changed) == digest {
			t.Fatalf("a change of the material kept the digest: %+v", m)
		}
	}
	// The digest of every other kind is the one computed before the review
	// material existed (review round 2, R2-F04).
	plain := ActionRequest{Project: "p", Origin: "spec:x", Kind: ActionApproval, Question: "q", Targets: []NodeRef{{}}, Effects: []ObligationEffect{{Phase: PhaseCloseout, Mode: EffectBlock}}}
	legacy, _ := digestJSON(struct {
		Project   string             `json:"project_id"`
		Origin    string             `json:"origin"`
		Recipient ObligationActor    `json:"recipient"`
		Kind      string             `json:"kind"`
		Question  string             `json:"question"`
		Context   string             `json:"context,omitempty"`
		Options   []ActionOption     `json:"options"`
		Targets   []NodeRef          `json:"targets"`
		Effects   []ObligationEffect `json:"effects"`
		Subject   *ActionSubject     `json:"subject,omitempty"`
	}{plain.Project, plain.Origin, plain.Recipient, plain.Kind, plain.Question, plain.Context, plain.Options, plain.Targets, plain.Effects, plain.Subject})
	if ActionRequestDigest(plain) != legacy {
		t.Fatal("the digest of another kind changed")
	}
}

// Review round 1, F10: a different question for the same request and cause
// is a new escalation; the same question returns the one already asked.
func TestEscalationsDistinguishQuestions(t *testing.T) {
	f, _ := conclusionFixture(t)
	view := openReviewRequest(t, f, DelegationPurposeReview)
	first, err := f.store.EscalateDelegatedReview(view.Request, EscalationBusiness, "Ship now?", "", nil, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	again, _ := f.store.EscalateDelegatedReview(view.Request, EscalationBusiness, "Ship now?", "", nil, time.Now())
	other, _ := f.store.EscalateDelegatedReview(view.Request, EscalationBusiness, "Price it?", "", nil, time.Now())
	if again != first || other == first || other == "" {
		t.Fatalf("first %s, again %s, other %s", first, again, other)
	}
}

// Review round 2, R2-F02: a fulfilment whose header names another submitter
// than its payload satisfies nothing, and the view names the runner.
func TestFulfillmentHeaderIsBound(t *testing.T) {
	f, draftPath := conclusionFixture(t)
	view := openReviewRequest(t, f, DelegationPurposeReview)
	if err := os.WriteFile(draftPath, []byte(draftFrom(t, approvedBundleAttestation(f.bundle, "agent:reviewer"), nil)), 0o600); err != nil {
		t.Fatal(err)
	}
	run := dispatchConclusion(t, f, "header")
	if _, err := f.store.ConcludeReviewRun(run.RunID, f.key.Name, true, time.Now()); err != nil {
		t.Fatal(err)
	}
	done, err := fulfil(f, view, run.RunID, "k")
	if err != nil || done.AnsweredBy != "agent:runner" {
		t.Fatalf("fulfil: %+v %v", done.AnsweredBy, err)
	}
	path := actionJournalPath(f.store.Root, view.Request.ID)
	raw, _ := os.ReadFile(path)
	if err := os.WriteFile(path, []byte(strings.Replace(string(raw), `"actor":"agent:runner"`, `"actor":"human:invented"`, 1)), 0o644); err != nil {
		t.Fatal(err)
	}
	if v, err := f.store.LoadActionRequest(view.Request.ID); err != nil || v.Satisfaction == SatisfactionSatisfied || v.AnsweredBy == "human:invented" {
		t.Fatalf("a rewritten header counted: %+v %v", v.AnsweredBy, err)
	}
}

// Review round 2, R2-F03: an escalation with other targets or effects is a
// new one; nothing asked is dropped.
func TestEscalationsKeepTheirTargetsAndEffects(t *testing.T) {
	f, _ := conclusionFixture(t)
	view := openReviewRequest(t, f, DelegationPurposeReview)
	first, err := f.store.EscalateDelegatedReview(view.Request, EscalationBusiness, "Ship now?", "", nil, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	other := view.Request
	other.Effects = []ObligationEffect{{Phase: PhaseExecution, Mode: EffectBlock}}
	second, err := f.store.EscalateDelegatedReview(other, EscalationBusiness, "Ship now?", "", nil, time.Now())
	if err != nil || second == first {
		t.Fatalf("other effects reused %s: %s %v", first, second, err)
	}
	e, _ := f.store.LoadActionRequest(second)
	if len(e.Request.Effects) != 1 || e.Request.Effects[0].Phase != PhaseExecution {
		t.Fatalf("the effects were dropped: %+v", e.Request.Effects)
	}
}

// resealAfter commits what changed in the fixture's checkout, aligns the
// delivery index to the new head and seals a new bundle of the scope.
func resealAfter(t *testing.T, f runProvenanceFixture) ReviewBundle {
	t.Helper()
	git := func(args ...string) string {
		t.Helper()
		out, err := exec.Command("git", append([]string{"-C", f.store.Root, "-c", "user.name=fixture", "-c", "user.email=fixture@example.invalid", "-c", "commit.gpgsign=false"}, args...)...).CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v %s", args, err, out)
		}
		return strings.TrimSpace(string(out))
	}
	git("add", "-A")
	git("commit", "-q", "-m", "change")
	head := git("rev-parse", "HEAD")
	path := filepath.Join(f.store.Root, ".pose/indexes/delivery-integrity.json")
	raw, _ := os.ReadFile(path)
	var graph DeliveryIntegrityGraph
	_ = json.Unmarshal(raw, &graph)
	for i := range graph.ChangeSets {
		graph.ChangeSets[i].Base, graph.ChangeSets[i].Head, graph.ChangeSets[i].ResolvedBase, graph.ChangeSets[i].ResolvedHead = head, head, head, head
		graph.ChangeSets[i].Selector = "range:" + head + ".." + head
	}
	for i := range graph.ValidationResults {
		graph.ValidationResults[i].GitHead = head
	}
	raw, _ = json.Marshal(graph)
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	bundle, err := f.store.sealReviewBundle("spec:backend", nil, false, "", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	return bundle
}

// Review round 3, R3-F01: a request fulfilled for a bundle the scope no
// longer has keeps restricting until it is asked again; its next step says
// how, and the successor — opened by its requester, linked to it — retires
// it, so once the successor is fulfilled nothing restricts.
func TestAStaleFulfilledRequestIsReplaced(t *testing.T) {
	f, draftPath := conclusionFixture(t)
	view := openReviewRequest(t, f, DelegationPurposeReview)
	if err := os.WriteFile(draftPath, []byte(draftFrom(t, approvedBundleAttestation(f.bundle, "agent:reviewer"), nil)), 0o600); err != nil {
		t.Fatal(err)
	}
	run := dispatchConclusion(t, f, "first")
	if _, err := f.store.ConcludeReviewRun(run.RunID, f.key.Name, true, time.Now()); err != nil {
		t.Fatal(err)
	}
	if _, err := fulfil(f, view, run.RunID, "k"); err != nil {
		t.Fatal(err)
	}
	policy := filepath.Join(f.store.Root, ".pose", "policy", "review.json")
	raw, _ := os.ReadFile(policy)
	if err := os.WriteFile(policy, append(raw, ' '), 0o644); err != nil {
		t.Fatal(err)
	}
	stale, _ := f.store.LoadActionRequest(view.Request.ID)
	if stale.NextStep == nil || stale.NextStep.Kind != "replace-request" || !strings.Contains(stale.NextStep.Reason, "--supersedes "+view.Request.ID) {
		t.Fatalf("the stale request has no way forward: %+v", stale.NextStep)
	}
	f.bundle = resealAfter(t, f)
	m, err := f.store.PrepareDelegatedReviewMaterial("spec:backend", DelegationPurposeReview, "fixture labelled notes", nil)
	if err != nil {
		t.Fatal(err)
	}
	successor, err := f.store.OpenActionRequestAuthorized(ActionRequest{Origin: "spec:backend", Kind: ActionDelegatedReview, Question: "Review spec:backend again",
		RequestedBy: ActionPrincipal{Principal: "agent:implementer"}, Targets: []NodeRef{{Kind: "requirement", ID: "R1"}},
		Effects: []ObligationEffect{{Phase: PhaseCloseout, Mode: EffectBlock}}, Review: &m, Supersedes: view.Request.ID}, nil, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if old, _ := f.store.LoadActionRequest(view.Request.ID); old.State != ActionStateSuperseded {
		t.Fatalf("the predecessor was not retired: %s", old.State)
	}
	// The second review is the bundle's second round: it states its sweep.
	text := strings.Replace(draftFrom(t, approvedBundleAttestation(f.bundle, "agent:reviewer"), nil), `"decision":"approved"`, `"decision":"approved","sweep":"the whole change again"`, 1)
	if err := os.WriteFile(draftPath, []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}
	second := dispatchConclusion(t, f, "second")
	if c, err := f.store.ConcludeReviewRun(second.RunID, f.key.Name, true, time.Now()); err != nil || c.State != ReviewConclusionRecorded {
		t.Fatalf("second conclusion: %+v %v", c, err)
	}
	done, err := fulfil(f, successor, second.RunID, "k2")
	if err != nil || done.State != ActionStateFulfilled || done.Satisfaction != SatisfactionSatisfied {
		t.Fatalf("the successor: %+v %v", done, err)
	}
	if restrictions, err := f.store.GovernedEffectRestrictions(PhaseCloseout, []string{"backend"}); err != nil || len(restrictions) != 0 {
		t.Fatalf("still restricted: %+v %v", restrictions, err)
	}
}
