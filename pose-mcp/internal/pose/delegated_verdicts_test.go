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

// Review round 1, B01: a run that does not meet the independence its plan
// requires records no verdict, and a record of one does not verify.
func TestVerdictsMeetThePlansIndependence(t *testing.T) {
	draftPath := filepath.Join(t.TempDir(), "draft.txt")
	f := provenanceRunFixtureAs(t, nil, "openai", "implementation-model", func(a *ReviewerAdapter) {
		a.Command = []string{"sh", "-c", "cat >/dev/null; cat '" + draftPath + "'"}
	})
	verdict := ReviewDraft{Decision: "approved", Questions: []ReviewDraftQuestion{{ID: "R1", Verdict: AdjudicationHolds, Rationale: "it does"}}}
	if err := os.WriteFile(draftPath, []byte(verdictDraft(t, verdict)), 0o600); err != nil {
		t.Fatal(err)
	}
	run := dispatchKind(t, f, ReviewBriefKindAdjudication, "same model")
	refused, err := f.store.ConcludeReviewRun(run.RunID, f.key.Name, true, time.Now())
	if err != nil || refused.State != ReviewConclusionRefused || !strings.Contains(strings.Join(refused.Reasons, ";"), "match the implementation") {
		t.Fatalf("a same-model adjudication: %+v %v", refused, err)
	}
	proof, err := f.store.SignReviewRunProof(run.RunID, f.key, time.Hour, time.Now(), true)
	if err != nil {
		t.Fatal(err)
	}
	writeAdjudication(t, f.store, adjudicationRecordFor(run, f.bundle, verdict, proof.ProofID, f.key.Name, time.Now()))
	if a, _ := f.store.SpecAdjudications("backend"); len(a) != 1 || a[0].Verified {
		t.Fatalf("a same-model record verified: %+v", a)
	}
}

func writeAdjudication(t *testing.T, s Store, rec AdjudicationRecord) {
	t.Helper()
	raw, _ := json.Marshal(rec)
	dir := filepath.Join(s.Root, filepath.FromSlash(AdjudicationsDir))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, rec.RunID+".json"), raw, 0o644); err != nil {
		t.Fatal(err)
	}
}

// Review round 1, B04 and F02: a record written without the conclusion —
// for a verdict that does not answer what was asked, or naming another
// issuer than the proof's — does not verify.
func TestVerdictRecordsVerifyLikeTheConclusion(t *testing.T) {
	f, draftPath := conclusionFixture(t)
	verdict := ReviewDraft{Decision: "approved", Questions: []ReviewDraftQuestion{{ID: "R999", Verdict: AdjudicationHolds, Rationale: "unasked"}}}
	if err := os.WriteFile(draftPath, []byte(verdictDraft(t, verdict)), 0o600); err != nil {
		t.Fatal(err)
	}
	run := dispatchKind(t, f, ReviewBriefKindAdjudication, "unasked")
	proof, err := f.store.SignReviewRunProof(run.RunID, f.key, time.Hour, time.Now(), true)
	if err != nil {
		t.Fatal(err)
	}
	writeAdjudication(t, f.store, adjudicationRecordFor(run, f.bundle, verdict, proof.ProofID, f.key.Name, time.Now()))
	if a, _ := f.store.SpecAdjudications("backend"); len(a) != 1 || a[0].Verified || !strings.Contains(a[0].Why, "not answered") {
		t.Fatalf("an unanswered record verified: %+v", a)
	}
	answered := ReviewDraft{Decision: "approved", Questions: []ReviewDraftQuestion{{ID: "R1", Verdict: AdjudicationHolds, Rationale: "it does"}}}
	if err := os.WriteFile(draftPath, []byte(verdictDraft(t, answered)), 0o600); err != nil {
		t.Fatal(err)
	}
	other := dispatchKind(t, f, ReviewBriefKindAdjudication, "other issuer")
	proof, err = f.store.SignReviewRunProof(other.RunID, f.key, time.Hour, time.Now(), true)
	if err != nil {
		t.Fatal(err)
	}
	writeAdjudication(t, f.store, adjudicationRecordFor(other, f.bundle, answered, proof.ProofID, "someone-else", time.Now()))
	for _, a := range func() []VerifiedAdjudication { a, _ := f.store.SpecAdjudications("backend"); return a }() {
		if a.Record.RunID == other.RunID && (a.Verified || !strings.Contains(a.Why, "issuer")) {
			t.Fatalf("a record naming another issuer verified: %+v", a)
		}
	}
}

// Review round 1, B02 and B03: a smoke result takes nothing from the
// document's bytes but what the run's draft concludes — head, time and
// provenance come from the run and the index — and it covers the target it
// observed, of its spec, and no other.
func TestSmokeResultsAreDerivedAndTargeted(t *testing.T) {
	f, draftPath := conclusionFixture(t)
	verdict := ReviewDraft{Decision: "approved", Surfaces: []ReviewDraftSurface{{Ref: "contract:backend-api", Outcome: SmokeObserved, Commands: []string{"go run ./api"}, Observed: "ok"}}}
	if err := os.WriteFile(draftPath, []byte(verdictDraft(t, verdict)), 0o600); err != nil {
		t.Fatal(err)
	}
	run := dispatchKind(t, f, ReviewBriefKindSmoke, "derived")
	recorded, err := f.store.ConcludeReviewRun(run.RunID, f.key.Name, true, time.Now())
	if err != nil || recorded.State != ReviewConclusionRecorded {
		t.Fatalf("conclude: %+v %v", recorded, err)
	}
	path := filepath.Join(f.store.Root, filepath.FromSlash(recorded.Destination))
	raw, _ := os.ReadFile(path)
	var doc map[string]any
	_ = json.Unmarshal(raw, &doc)
	doc["scope_provenance"] = map[string]any{"foreign": "sha256:" + strings.Repeat("f", 64)}
	doc["provenance_digest"] = "sha256:" + strings.Repeat("e", 64)
	doc["generated_at"] = "2099-01-01T00:00:00Z"
	raw, _ = json.Marshal(doc)
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	results, _ := f.store.SmokeResults()
	if len(results) != 1 || !results[0].Verified || len(results[0].Results) != 1 {
		t.Fatalf("results: %+v", results)
	}
	r := results[0].Results[0]
	if r.GeneratedAt != run.EndedAt || r.ProvenanceDigest != "" || r.ScopeProvenance["foreign"] != "" || r.Target != "contract:backend-api" || r.TargetSpec != "backend" {
		t.Fatalf("a result took the document's bytes: %+v", r)
	}
	if !resultCoversTarget(r, DeliveryTarget{Spec: "backend", Ref: "contract:backend-api"}) || resultCoversTarget(r, DeliveryTarget{Spec: "backend", Ref: "contract:never-exercised"}) || resultCoversTarget(r, DeliveryTarget{Spec: "foreign", Ref: "contract:backend-api"}) {
		t.Fatal("a smoke result covers a target it did not observe")
	}
	if resultCoversTarget(DeliveryValidationResult{EvidenceClass: SmokeEvidenceClass}, DeliveryTarget{Spec: "backend", Ref: "contract:backend-api"}) {
		t.Fatal("a smoke result naming no target covers one")
	}
}

// Review round 1, B05: a question the current bundle's adjudication says
// does not hold blocks the closeout.
func TestAdjudicationBlocksTheCloseout(t *testing.T) {
	f, draftPath := conclusionFixture(t)
	verdict := ReviewDraft{Decision: "changes-requested", Questions: []ReviewDraftQuestion{{ID: "R1", Verdict: AdjudicationDoesNotHold, Rationale: "it does not"}}}
	if err := os.WriteFile(draftPath, []byte(verdictDraft(t, verdict)), 0o600); err != nil {
		t.Fatal(err)
	}
	run := dispatchKind(t, f, ReviewBriefKindAdjudication, "blocks")
	if c, err := f.store.ConcludeReviewRun(run.RunID, f.key.Name, true, time.Now()); err != nil || c.State != ReviewConclusionRecorded {
		t.Fatalf("conclude: %+v %v", c, err)
	}
	state, err := f.store.getCloseoutState("spec:backend", false)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, d := range state.Diagnostics {
		found = found || d.Code == AdjudicationDoesNotHoldCode && strings.Contains(d.Message, "R1 does not hold")
	}
	if !found {
		t.Fatalf("the closeout state does not block: %+v", state.Diagnostics)
	}
}
