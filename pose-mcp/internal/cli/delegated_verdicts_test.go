package cli

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

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
	if code, out := runPose(t, root, "review", "dispatch", "spec:alpha", "--via", "fake", "--kind", "adjudication", "--apply"); code != 0 {
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
