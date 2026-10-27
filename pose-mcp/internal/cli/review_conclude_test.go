package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	posemodel "github.com/harne8/pose-mcp/internal/pose"
)

// R1, R3, R4 of pose-delegated-review-conclusions over the CLI: conclude
// needs an issuer; a run that captured no provenance records no attestation
// and keeps why; concluding it again returns its record; with the capability
// off, a delegated review records no conclusion on its own.
func TestReviewConcludeRefusesAndReplays(t *testing.T) {
	root := dispatchInstance(t)
	writeDelegation(t, root, delegationPolicy(approvingAgent))
	code, out, errOut := runPoseSplit(t, root, "delegate", "review", "spec:alpha", "--agent", "ok", "--json")
	var doc map[string]any
	if code != 0 || json.Unmarshal([]byte(out), &doc) != nil {
		t.Fatalf("review: %d %s %s", code, out, errOut)
	}
	run, _ := doc["run"].(string)
	if entries, _ := os.ReadDir(filepath.Join(root, ".pose", "review-conclusions")); len(entries) != 0 {
		t.Fatal("a conclusion was recorded with the capability off")
	}
	if code, _, errOut := runPoseSplit(t, root, "review", "conclude", run); code != 2 || !strings.Contains(errOut, "issuer") {
		t.Fatalf("conclude without an issuer: %d %s", code, errOut)
	}
	if code, out, _ := runPoseSplit(t, root, "review", "conclude", run, "--issuer", "fixture"); code != 1 || !strings.Contains(out, "review_conclusion.state=refused") || !strings.Contains(out, "review_conclusion.apply=false") {
		t.Fatalf("a preview: %d %s", code, out)
	}
	code, out, _ = runPoseSplit(t, root, "review", "conclude", run, "--issuer", "fixture", "--apply", "--json")
	var conclusion posemodel.ReviewConclusion
	if code != 1 || json.Unmarshal([]byte(out), &conclusion) != nil || conclusion.State != posemodel.ReviewConclusionRefused || !strings.Contains(strings.Join(conclusion.Reasons, ";"), "provenance") {
		t.Fatalf("a run without provenance: %d %s", code, out)
	}
	code, out, _ = runPoseSplit(t, root, "review", "conclude", run, "--issuer", "fixture", "--apply", "--json")
	if json.Unmarshal([]byte(out), &conclusion) != nil || !conclusion.Replayed {
		t.Fatalf("a replay: %d %s", code, out)
	}
}
