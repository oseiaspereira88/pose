package pose

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// verdictDraft is the pose-draft block a reviewer returns with a typed
// verdict.
func verdictDraft(t *testing.T, draft ReviewDraft) string {
	t.Helper()
	draft.Schema = ReviewDraftSchema
	if draft.Findings == nil {
		draft.Findings = []ReviewDraftFinding{}
	}
	raw, err := json.Marshal(draft)
	if err != nil {
		t.Fatal(err)
	}
	return "decision: " + draft.Decision + "\n```pose-draft\n" + string(raw) + "\n```\n"
}

func dispatchKind(t *testing.T, f runProvenanceFixture, kind, reason string) ReviewRun {
	t.Helper()
	plan, err := f.store.PlanReviewDispatch(f.bundle.BundleID, kind, "fixture labelled notes", "fake", reason)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Brief.TemplateVersion != reviewBriefTemplates[kind] {
		t.Fatalf("the %s brief is %s", kind, plan.Brief.TemplateVersion)
	}
	run, err := f.store.RunReviewDispatch(plan, time.Now)
	if err != nil || run.Status != ReviewRunCompleted {
		t.Fatalf("dispatch: %+v %v", run, err)
	}
	return run
}

// R1: an adjudication run's verdict is recorded as an adjudication record
// of the spec, read and verified; a question it says does not hold is what
// lint-spec fails on; a verdict that does not answer every question records
// nothing (R3).
func TestAdjudicationVerdictIsRecordedOnTheSpec(t *testing.T) {
	f, draftPath := conclusionFixture(t)
	questions, err := f.store.AdjudicationQuestions(f.bundle.Payload.Scope)
	if err != nil || len(questions) != 1 || questions[0].ID != "R1" {
		t.Fatalf("questions: %+v %v", questions, err)
	}
	// Nothing answered: refused, nothing recorded.
	if err := os.WriteFile(draftPath, []byte(verdictDraft(t, ReviewDraft{Decision: "approved"})), 0o600); err != nil {
		t.Fatal(err)
	}
	unanswered := dispatchKind(t, f, ReviewBriefKindAdjudication, "unanswered")
	refused, err := f.store.ConcludeReviewRun(unanswered.RunID, f.key.Name, true, time.Now())
	if err != nil || refused.State != ReviewConclusionRefused || !strings.Contains(strings.Join(refused.Reasons, ";"), "R1 is not answered") {
		t.Fatalf("an unanswered adjudication: %+v %v", refused, err)
	}
	if adjudications, _ := f.store.SpecAdjudications("backend"); len(adjudications) != 0 {
		t.Fatalf("a refused adjudication recorded: %+v", adjudications)
	}
	verdict := ReviewDraft{Decision: "changes-requested", Questions: []ReviewDraftQuestion{{ID: "R1", Verdict: AdjudicationDoesNotHold, Evidence: "file:api/server.go:3", Rationale: "the contract changed"}}}
	if err := os.WriteFile(draftPath, []byte(verdictDraft(t, verdict)), 0o600); err != nil {
		t.Fatal(err)
	}
	run := dispatchKind(t, f, ReviewBriefKindAdjudication, "answered")
	recorded, err := f.store.ConcludeReviewRun(run.RunID, f.key.Name, true, time.Now())
	if err != nil || recorded.State != ReviewConclusionRecorded || recorded.Destination != AdjudicationsDir+"/"+run.RunID+".json" || recorded.Attestation != "" {
		t.Fatalf("an adjudication: %+v %v", recorded, err)
	}
	adjudications, err := f.store.SpecAdjudications("backend")
	if err != nil || len(adjudications) != 1 || !adjudications[0].Verified || !adjudications[0].Current || adjudications[0].Record.Questions[0].Verdict != AdjudicationDoesNotHold {
		t.Fatalf("read back: %+v %v", adjudications, err)
	}
	if attestations, _ := f.store.ListReviewAttestations(f.bundle.BundleID); len(attestations) != 0 {
		t.Fatal("an adjudication recorded an attestation")
	}
	if again, err := f.store.ConcludeReviewRun(run.RunID, f.key.Name, true, time.Now()); err != nil || !again.Replayed {
		t.Fatalf("a replay: %+v %v", again, err)
	}
	// A record edited after the fact does not verify.
	path := filepath.Join(f.store.Root, filepath.FromSlash(recorded.Destination))
	raw, _ := os.ReadFile(path)
	if err := os.WriteFile(path, []byte(strings.Replace(string(raw), AdjudicationDoesNotHold, AdjudicationHolds, 1)), 0o644); err != nil {
		t.Fatal(err)
	}
	if adjudications, _ := f.store.SpecAdjudications("backend"); len(adjudications) != 1 || adjudications[0].Verified {
		t.Fatalf("an edited record verified: %+v", adjudications)
	}
}

// R2: a smoke run's observations are recorded as validation results of
// class smoke, one per delivery target, which only a verified document
// yields; an edited document yields nothing.
func TestSmokeVerdictIsRecordedAsSurfaceEvidence(t *testing.T) {
	f, draftPath := conclusionFixture(t)
	verdict := ReviewDraft{Decision: "approved", Surfaces: []ReviewDraftSurface{{Ref: "contract:backend-api", Outcome: SmokeObserved, Commands: []string{"go run ./api"}, Observed: "the API answered as R1 says"}}}
	if err := os.WriteFile(draftPath, []byte(verdictDraft(t, verdict)), 0o600); err != nil {
		t.Fatal(err)
	}
	run := dispatchKind(t, f, ReviewBriefKindSmoke, "smoke")
	recorded, err := f.store.ConcludeReviewRun(run.RunID, f.key.Name, true, time.Now())
	if err != nil || recorded.State != ReviewConclusionRecorded || recorded.Destination != SmokeResultsDir+"/"+run.RunID+".json" {
		t.Fatalf("a smoke run: %+v %v", recorded, err)
	}
	results, err := f.store.SmokeResults()
	if err != nil || len(results) != 1 || !results[0].Verified || len(results[0].Results) != 1 {
		t.Fatalf("read back: %+v %v", results, err)
	}
	r := results[0].Results[0]
	if r.EvidenceClass != SmokeEvidenceClass || r.Module != "api" || r.Outcome != "pass" || r.Severity != "required" || r.GitHead != run.SealedCommit {
		t.Fatalf("result: %+v", r)
	}
	path := filepath.Join(f.store.Root, filepath.FromSlash(recorded.Destination))
	raw, _ := os.ReadFile(path)
	if err := os.WriteFile(path, []byte(strings.Replace(string(raw), `"outcome": "pass"`, `"outcome": "fail"`, 1)), 0o644); err != nil {
		t.Fatal(err)
	}
	if results, _ := f.store.SmokeResults(); len(results) != 1 || results[0].Verified || len(results[0].Results) != 0 {
		t.Fatalf("an edited document verified: %+v", results)
	}
}

// R3: a smoke verdict naming a target the scope does not declare records
// nothing; a brief of an earlier template still verifies.
func TestSmokeVerdictRefusesAnUnaskedTarget(t *testing.T) {
	f, draftPath := conclusionFixture(t)
	verdict := ReviewDraft{Decision: "approved", Surfaces: []ReviewDraftSurface{
		{Ref: "contract:backend-api", Outcome: SmokeObserved, Commands: []string{"go run ./api"}, Observed: "ok"},
		{Ref: "surface:elsewhere", Outcome: SmokeObserved, Commands: []string{"true"}, Observed: "ok"},
	}}
	if err := os.WriteFile(draftPath, []byte(verdictDraft(t, verdict)), 0o600); err != nil {
		t.Fatal(err)
	}
	run := dispatchKind(t, f, ReviewBriefKindSmoke, "unasked")
	refused, err := f.store.ConcludeReviewRun(run.RunID, f.key.Name, true, time.Now())
	if err != nil || refused.State != ReviewConclusionRefused || !strings.Contains(strings.Join(refused.Reasons, ";"), "surface:elsewhere was not asked") {
		t.Fatalf("an unasked target: %+v %v", refused, err)
	}
	if results, _ := f.store.SmokeResults(); len(results) != 0 {
		t.Fatal("a refused smoke verdict recorded results")
	}
	earlier, err := f.store.renderReviewBriefAs(f.bundle.BundleID, ReviewBriefKindSmoke, "notes", smokeBriefTemplateFirst)
	if err != nil {
		t.Fatal(err)
	}
	matched, err := f.store.RenderReviewBriefMatching(f.bundle.BundleID, ReviewBriefKindSmoke, "notes", earlier.Digest)
	if err != nil || matched.TemplateVersion != smokeBriefTemplateFirst || strings.Contains(earlier.Text, "pose-draft") {
		t.Fatalf("an earlier smoke brief: %s %v", matched.TemplateVersion, err)
	}
}
