package cli

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	posemodel "github.com/harne8/pose-mcp/internal/pose"
)

// Spec pose-delegated-verdict-destinations R1 over the CLI: an adjudication
// run concluded with review conclude records its verdict on the spec, and
// lint-spec fails on a requirement the current bundle's adjudication says
// does not hold.
func TestAdjudicationVerdictFailsLintSpec(t *testing.T) {
	root, draftPath, _ := cliProvenanceInstance(t)
	if _, out := runPose(t, root, "lint-spec", "alpha"); strings.Contains(out, "adjudicat") {
		t.Fatalf("lint-spec before: %s", out)
	}
	draft := posemodel.ReviewDraft{Schema: posemodel.ReviewDraftSchema, Decision: "changes-requested", Findings: []posemodel.ReviewDraftFinding{},
		Questions: []posemodel.ReviewDraftQuestion{{ID: "R1", Verdict: posemodel.AdjudicationDoesNotHold, Evidence: "integration:val-alpha", Rationale: "the fixture does not work"}}}
	raw, _ := json.Marshal(draft)
	if err := os.WriteFile(draftPath, []byte("decision: changes-requested\n```pose-draft\n"+string(raw)+"\n```\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if code, out := runPose(t, root, "delegate", "adjudicate", "spec:alpha", "--agent", "fake"); code != 0 {
		t.Fatal(code, out)
	}
	runs, _, _ := posemodel.Store{Root: root}.ReadReviewRuns("")
	if len(runs) != 1 {
		t.Fatalf("runs: %d", len(runs))
	}
	code, out := runPose(t, root, "review", "conclude", runs[0].RunID, "--issuer", "fixture-executor", "--apply", "--json")
	if code != 0 || !strings.Contains(out, posemodel.AdjudicationsDir) {
		t.Fatalf("conclude: %d %s", code, out)
	}
	if code, out := runPose(t, root, "lint-spec", "alpha"); code == 0 || !strings.Contains(out, "R1 does not hold") || !strings.Contains(out, "spec.adjudications.failures=1") {
		t.Fatalf("lint-spec after: %d %s", code, out)
	}
	// The closeout refuses too (review round 1, B05).
	if code, out := runPose(t, root, "close", "spec:alpha"); code == 0 || !strings.Contains(out, "close.refused.adjudication") {
		t.Fatalf("close: %d %s", code, out)
	}
}

// closableWithAdjudication is an instance whose spec:alpha lints cleanly and
// whose current bundle is approved by a concluded review, with a completed
// adjudication run whose verdict — R1 does not hold — is not yet concluded.
func closableWithAdjudication(t *testing.T) (string, posemodel.Store, string) {
	t.Helper()
	root, draftPath, bundleID := cliProvenanceInstance(t)
	s := posemodel.Store{Root: root}
	sp, err := s.GetSpec("alpha")
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(sp.Path)
	if err := os.WriteFile(sp.Path, []byte(strings.ReplaceAll(string(raw), "test:TestFixture", "check:unit")), 0o644); err != nil {
		t.Fatal(err)
	}
	draft := posemodel.ReviewDraft{Schema: posemodel.ReviewDraftSchema, Decision: "changes-requested", Findings: []posemodel.ReviewDraftFinding{},
		Questions: []posemodel.ReviewDraftQuestion{{ID: "R1", Verdict: posemodel.AdjudicationDoesNotHold, Rationale: "a counterexample"}}}
	raw, _ = json.Marshal(draft)
	if err := os.WriteFile(draftPath, []byte("decision: changes-requested\n```pose-draft\n"+string(raw)+"\n```\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if code, out := runPose(t, root, "delegate", "adjudicate", "spec:alpha", "--agent", "fake"); code != 0 {
		t.Fatal(code, out)
	}
	runs, _, _ := s.ReadReviewRuns("")
	adjudication := runs[0].RunID
	writeRunConclusion(t, s, bundleID, draftPath)
	if code, out := runPose(t, root, "delegate", "review", "spec:alpha", "--agent", "fake", "--reason", "the review"); code != 0 {
		t.Fatal(code, out)
	}
	runs, _, _ = s.ReadReviewRuns("")
	for _, r := range runs {
		if r.Kind == posemodel.ReviewBriefKindReview {
			if code, out := runPose(t, root, "review", "conclude", r.RunID, "--issuer", "fixture-executor", "--apply"); code != 0 {
				t.Fatal(code, out)
			}
		}
	}
	if code, out := runPose(t, root, "review", "verify", "spec:alpha"); code != 0 {
		t.Fatal(code, out)
	}
	return root, s, adjudication
}

// Review round 2, R2-B01: an adjudication published while close runs either
// lands before close's locked re-check, which refuses, or waits for the
// lifecycle lock close holds and lands after the transition.
func TestCloseAndAdjudicationAreSerialized(t *testing.T) {
	t.Run("before the locked re-check", func(t *testing.T) {
		root, s, adjudication := closableWithAdjudication(t)
		closeGatesJudged = func() {
			if code, out := runPose(t, root, "review", "conclude", adjudication, "--issuer", "fixture-executor", "--apply"); code != 0 {
				t.Fatal(code, out)
			}
		}
		t.Cleanup(func() { closeGatesJudged = nil })
		if code, out := runPose(t, root, "close", "spec:alpha"); code == 0 {
			t.Fatalf("close published over a negative adjudication: %s", out)
		}
		if sp, _ := s.GetSpec("alpha"); sp.Status != "in-progress" {
			t.Fatalf("status %s", sp.Status)
		}
	})
	t.Run("while close holds the lock", func(t *testing.T) {
		root, s, adjudication := closableWithAdjudication(t)
		concluded := make(chan string, 1)
		guard := func() error {
			go func() {
				_, out := runPose(t, root, "review", "conclude", adjudication, "--issuer", "fixture-executor", "--apply")
				concluded <- out
			}()
			time.Sleep(500 * time.Millisecond)
			if a, _ := s.SpecAdjudications("alpha"); len(a) != 0 {
				t.Error("an adjudication was published while close held the lifecycle lock")
			}
			return nil
		}
		var stdout, stderr strings.Builder
		if code := cmdCloseLocal(root, "spec:alpha", guard, &stdout, &stderr); code != 0 {
			t.Fatalf("close: %d %s %s", code, stdout.String(), stderr.String())
		}
		<-concluded
		if a, _ := s.SpecAdjudications("alpha"); len(a) != 1 {
			t.Fatalf("the adjudication did not land after the transition: %+v", a)
		}
	})
}
