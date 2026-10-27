package pose

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// conclusionFixture is a provenance fixture whose agent prints the draft
// written to draftPath, so a test dispatches the conclusion it means.
func conclusionFixture(t *testing.T) (runProvenanceFixture, string) {
	t.Helper()
	draftPath := filepath.Join(t.TempDir(), "draft.txt")
	if err := os.WriteFile(draftPath, []byte("decision: changes-requested\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	f := provenanceRunFixtureAs(t, nil, "openai", "review-model", func(a *ReviewerAdapter) {
		a.Command = []string{"sh", "-c", "cat >/dev/null; cat '" + draftPath + "'"}
	})
	return f, draftPath
}

// draftFrom renders att as the pose-draft block a reviewer returns.
func draftFrom(t *testing.T, att ReviewAttestation, findings []ReviewDraftFinding) string {
	t.Helper()
	draft := ReviewDraft{Schema: ReviewDraftSchema, Decision: att.Decision, Findings: findings}
	if draft.Findings == nil {
		draft.Findings = []ReviewDraftFinding{}
	}
	for _, c := range att.Criteria {
		draft.Criteria = append(draft.Criteria, ReviewDraftCriterion{ID: c.ID, Disposition: c.Disposition, Evidence: c.Evidence, Rationale: c.Rationale})
	}
	for _, tool := range att.Tools {
		draft.Tools = append(draft.Tools, ReviewDraftTool{ID: tool.ID, Component: tool.Component, Disposition: tool.Disposition, Evidence: tool.Evidence, Rationale: tool.Rationale})
	}
	raw, _ := json.Marshal(draft)
	return "decision: " + att.Decision + "\n```pose-draft\n" + string(raw) + "\n```\n"
}

func dispatchConclusion(t *testing.T, f runProvenanceFixture, reason string) ReviewRun {
	t.Helper()
	plan, err := f.store.PlanReviewDispatch(f.bundle.BundleID, "review", "fixture labelled notes", "fake", reason)
	if err != nil {
		t.Fatal(err)
	}
	run, err := f.store.RunReviewDispatch(plan, time.Now)
	if err != nil || run.Status != ReviewRunCompleted {
		t.Fatalf("dispatch: %+v %v", run, err)
	}
	return run
}

// R1, R4: a run whose typed conclusion approves is recorded by the engine as
// the run-backed attestation, signed by the issuer, and verify approves the
// bundle; concluding the same run again records nothing new.
func TestConcludeReviewRunRecordsTheAttestation(t *testing.T) {
	f, draftPath := conclusionFixture(t)
	att := approvedBundleAttestation(f.bundle, "agent:reviewer")
	if err := os.WriteFile(draftPath, []byte(draftFrom(t, att, nil)), 0o600); err != nil {
		t.Fatal(err)
	}
	run := dispatchConclusion(t, f, "the typed conclusion")
	preview, err := f.store.ConcludeReviewRun(run.RunID, f.key.Name, false, time.Now())
	if err != nil || preview.State != ReviewConclusionRecorded || preview.Attestation != "" {
		t.Fatalf("preview: %+v %v", preview, err)
	}
	if existing, _ := f.store.ReadReviewConclusion(run.RunID); existing != nil {
		t.Fatal("a preview wrote a conclusion")
	}
	conclusion, err := f.store.ConcludeReviewRun(run.RunID, f.key.Name, true, time.Now())
	if err != nil || conclusion.State != ReviewConclusionRecorded || conclusion.Attestation == "" || conclusion.ProofID == "" {
		t.Fatalf("conclusion: %+v %v", conclusion, err)
	}
	verification, err := f.store.VerifyReviewBundle(f.bundle.Payload.Scope.Ref)
	if err != nil || !verification.Approved {
		t.Fatalf("verify after the conclusion: %+v %v", verification.Blockers, err)
	}
	again, err := f.store.ConcludeReviewRun(run.RunID, f.key.Name, true, time.Now())
	if err != nil || !again.Replayed || again.Attestation != conclusion.Attestation {
		t.Fatalf("a replay: %+v %v", again, err)
	}
}

// R3: a run whose final message carries no typed conclusion, or whose bundle
// no longer governs, records no attestation and the reason is kept.
func TestConcludeReviewRunRefusesWhatIsNotAConclusion(t *testing.T) {
	f, _ := conclusionFixture(t)
	conclusion, err := f.store.ConcludeReviewRun(f.run.RunID, f.key.Name, true, time.Now())
	if err != nil || conclusion.State != ReviewConclusionRefused || !strings.Contains(strings.Join(conclusion.Reasons, ";"), "pose-draft") {
		t.Fatalf("a run without a draft: %+v %v", conclusion, err)
	}
	recorded, _ := f.store.ReadReviewConclusion(f.run.RunID)
	if recorded == nil || recorded.State != ReviewConclusionRefused {
		t.Fatalf("the refusal was not kept: %+v", recorded)
	}
	attestations, _ := f.store.ListReviewAttestations(f.bundle.Payload.Scope.Ref)
	if len(attestations) != 0 {
		t.Fatalf("a refused conclusion recorded an attestation: %+v", attestations)
	}
	if _, err := f.store.ConcludeReviewRun("trn-0123456789abcdef", f.key.Name, true, time.Now()); err == nil {
		t.Fatal("a task run concluded")
	}
}

// R1: a rejecting conclusion is recorded too, and the scope stays blocked
// until the bundle changes.
func TestConcludeReviewRunRecordsARejection(t *testing.T) {
	f, draftPath := conclusionFixture(t)
	att := approvedBundleAttestation(f.bundle, "agent:reviewer")
	att.Decision = "changes-requested"
	findings := []ReviewDraftFinding{{ID: "B1", Severity: "high", Disposition: "blocker", File: "x.go", Line: 3, Summary: "a defect"}}
	if err := os.WriteFile(draftPath, []byte(draftFrom(t, att, findings)), 0o600); err != nil {
		t.Fatal(err)
	}
	run := dispatchConclusion(t, f, "a rejecting conclusion")
	conclusion, err := f.store.ConcludeReviewRun(run.RunID, f.key.Name, true, time.Now())
	if err != nil || conclusion.State != ReviewConclusionRecorded || conclusion.Decision != "changes-requested" || conclusion.Attestation == "" {
		t.Fatalf("a rejection was not recorded: %+v %v", conclusion, err)
	}
	if verification, err := f.store.VerifyReviewBundle(f.bundle.Payload.Scope.Ref); err != nil || verification.Approved {
		t.Fatalf("a rejected scope verified approved: %v", err)
	}
}

// Review round 1, B1: whoever signs it, an attestation of a run briefed for
// a typed conclusion states that conclusion; criteria, tools or findings the
// draft did not declare are refused, and an approval verifies only as the
// draft concluded it.
func TestRunBackedAttestationStatesTheRunsConclusion(t *testing.T) {
	f, draftPath := conclusionFixture(t)
	if err := os.WriteFile(draftPath, []byte(draftFrom(t, ReviewAttestation{Decision: "approved"}, nil)), 0o600); err != nil {
		t.Fatal(err)
	}
	run := dispatchConclusion(t, f, "an empty conclusion")
	proof, err := f.store.SignReviewRunProof(run.RunID, f.key, time.Hour, time.Now(), true)
	if err != nil {
		t.Fatal(err)
	}
	att := approvedBundleAttestation(f.bundle, "agent:reviewer")
	att.Run = &ReviewRunReference{RunID: run.RunID, ProofID: proof.ProofID}
	envelope, err := f.store.SignReviewAttestation(att, f.key, nil, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	verified, err := f.store.VerifyReviewAttestationEnvelope(envelope)
	if err != nil {
		t.Fatal(err)
	}
	if blockers := f.store.runBackedReviewBlockers(f.bundle, verified); !strings.Contains(strings.Join(blockers, ";"), "does not state the run's conclusion") {
		t.Fatalf("criteria the draft did not declare were accepted: %v", blockers)
	}
}

// Review round 1, B2: two conclusions of one run at once sign and record
// one proof and one attestation; the other returns the record.
func TestConcludeReviewRunConcurrently(t *testing.T) {
	f, draftPath := conclusionFixture(t)
	if err := os.WriteFile(draftPath, []byte(draftFrom(t, approvedBundleAttestation(f.bundle, "agent:reviewer"), nil)), 0o600); err != nil {
		t.Fatal(err)
	}
	run := dispatchConclusion(t, f, "two at once")
	results := make(chan ReviewConclusion, 2)
	for i := 0; i < 2; i++ {
		go func(i int) {
			c, _ := f.store.ConcludeReviewRun(run.RunID, f.key.Name, true, time.Now().Add(time.Duration(i)*time.Millisecond))
			results <- c
		}(i)
	}
	first, second := <-results, <-results
	proofs, _ := os.ReadDir(filepath.Join(f.store.Root, filepath.FromSlash(reviewRunProofDir), run.RunID))
	attestations, _ := f.store.ListReviewAttestations(f.bundle.BundleID)
	if len(proofs) != 1 || len(attestations) != 1 || first.Replayed == second.Replayed {
		t.Fatalf("%d proofs, %d attestations; %+v / %+v", len(proofs), len(attestations), first, second)
	}
}

// Review round 1, B2: a reservation an interrupted conclusion left is
// completed from the attestation it recorded, or taken over once stale; a
// fresh one answers that a conclusion is in progress.
func TestConcludeReviewRunRecoversAReservation(t *testing.T) {
	f, draftPath := conclusionFixture(t)
	if err := os.WriteFile(draftPath, []byte(draftFrom(t, approvedBundleAttestation(f.bundle, "agent:reviewer"), nil)), 0o600); err != nil {
		t.Fatal(err)
	}
	run := dispatchConclusion(t, f, "interrupted")
	now := time.Now()
	pending := ReviewConclusion{SchemaVersion: 1, RunID: run.RunID, BundleID: run.BundleID, Scope: run.Scope, State: ReviewConclusionPending, ConcludedAt: now.UTC().Format(time.RFC3339Nano)}
	if _, err := f.store.writeReviewConclusion(pending, nil); err != nil {
		t.Fatal(err)
	}
	if busy, err := f.store.ConcludeReviewRun(run.RunID, f.key.Name, true, now); err != nil || busy.State != ReviewConclusionPending || !busy.Replayed {
		t.Fatalf("a fresh reservation: %+v %v", busy, err)
	}
	_, fresh, _ := f.store.readReviewConclusionRaw(run.RunID)
	stale := pending
	stale.ConcludedAt = now.Add(-reviewConclusionStale - time.Minute).UTC().Format(time.RFC3339Nano)
	if _, err := f.store.writeReviewConclusion(stale, fresh); err != nil {
		t.Fatal(err)
	}
	taken, err := f.store.ConcludeReviewRun(run.RunID, f.key.Name, true, time.Now())
	if err != nil || taken.State != ReviewConclusionRecorded || taken.Attestation == "" {
		t.Fatalf("a stale reservation was not taken over: %+v %v", taken, err)
	}
	// The record lost after the attestation was written is completed from it.
	_, raw, _ := f.store.readReviewConclusionRaw(run.RunID)
	if _, err := f.store.writeReviewConclusion(pending, raw); err != nil {
		t.Fatal(err)
	}
	done, err := f.store.ConcludeReviewRun(run.RunID, f.key.Name, true, time.Now())
	if err != nil || done.State != ReviewConclusionRecorded || done.Attestation != taken.Attestation {
		t.Fatalf("an interrupted conclusion was not completed: %+v %v", done, err)
	}
}

// Review round 1, B3 and B4: a follow-up becomes an accepted risk only with
// the action the reviewer gave; a criterion's mappings cross as stated.
func TestReviewDraftConclusionKeepsWhatTheReviewerStated(t *testing.T) {
	draft := ReviewDraft{Schema: ReviewDraftSchema, Decision: "approved",
		Criteria: []ReviewDraftCriterion{{ID: "design-causality", Disposition: "passed", Evidence: "spec:x", Mappings: []ReviewCriterionMapping{{Delta: "SD-1", Basis: "R1", Disposition: "mapped", Rationale: "r"}}}},
		Findings: []ReviewDraftFinding{
			{ID: "F1", Severity: "low", Disposition: "follow-up", Summary: "s", Owner: "@o", ReviewBy: "2026-12-01"},
			{ID: "F2", Severity: "low", Disposition: "follow-up", Summary: "s", Action: "a", Owner: "@o", ReviewBy: "2026-12-01"},
		}}
	att := reviewAttestationFromDraft(ReviewRun{RunID: "rrn-0123456789abcdef"}, ReviewBundle{}, draft, "agent:r")
	if att.Findings[0].Disposition != "open" || att.Findings[1].Disposition != "accepted-risk" {
		t.Fatalf("findings: %+v", att.Findings)
	}
	if len(att.Criteria[0].Mappings) != 1 || att.Criteria[0].Mappings[0].Delta != "SD-1" {
		t.Fatalf("mappings: %+v", att.Criteria)
	}
	raw, _ := json.Marshal(draft)
	if parsed, err := ParseReviewDraft("decision: approved\n```pose-draft\n" + string(raw) + "\n```\n"); err != nil || len(parsed.Criteria[0].Mappings) != 1 {
		t.Fatalf("a draft with mappings: %v", err)
	}
}

// Review round 1, B5: a run whose record cannot be read keeps a refusal
// under its id; a preview says the preflight has not run (F1).
func TestConcludeReviewRunRecordsAnUnreadableRun(t *testing.T) {
	f, draftPath := conclusionFixture(t)
	if err := os.WriteFile(draftPath, []byte(draftFrom(t, approvedBundleAttestation(f.bundle, "agent:reviewer"), nil)), 0o600); err != nil {
		t.Fatal(err)
	}
	run := dispatchConclusion(t, f, "to corrupt")
	if preview, err := f.store.ConcludeReviewRun(run.RunID, f.key.Name, false, time.Now()); err != nil || !strings.Contains(strings.Join(preview.Reasons, ";"), "preflight") {
		t.Fatalf("a preview: %+v %v", preview, err)
	}
	if err := os.WriteFile(filepath.Join(f.store.Root, filepath.FromSlash(reviewRunsDir), run.RunID+".json"), []byte("{"), 0o644); err != nil {
		t.Fatal(err)
	}
	refused, err := f.store.ConcludeReviewRun(run.RunID, f.key.Name, true, time.Now())
	kept, _ := f.store.ReadReviewConclusion(run.RunID)
	if err != nil || refused.State != ReviewConclusionRefused || kept == nil || !strings.Contains(strings.Join(kept.Reasons, ";"), "cannot be read") {
		t.Fatalf("an unreadable run: %+v %v %+v", refused, err, kept)
	}
}
