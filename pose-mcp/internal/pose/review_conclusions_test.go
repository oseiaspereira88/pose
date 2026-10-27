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
