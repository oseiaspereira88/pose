package pose

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Only the missing change set stands in for the evidence gap: a milestone of
// external members has no change set and no such blocker, and keeps its gate.
func TestOnlyTheMissingChangeSetSilencesTheEvidenceGap(t *testing.T) {
	if !reportsMissingChangeSet([]string{"other", missingChangeSetBlocker + "spec:x; the index holds no commit"}) {
		t.Fatal("the missing change set was not recognized")
	}
	for _, blockers := range [][]string{nil, {"subject path is missing", "unattributed " + missingChangeSetBlocker}} {
		if reportsMissingChangeSet(blockers) {
			t.Fatalf("another blocker silenced the evidence gap: %v", blockers)
		}
	}
}

func TestExternalMilestoneKeepsItsEvidenceGate(t *testing.T) {
	root, s, _, _ := federatedTestMilestoneConsumer(t)
	federatedTestWrite(t, root, ".pose/roadmaps/program.md", "---\nslug: program\nstatus: active\n---\n\n## Milestone: core\n- specs: xref:source/spec:backend\n")
	path := filepath.Join(root, ".pose/indexes/delivery-integrity.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var graph map[string]any
	if err := json.Unmarshal(raw, &graph); err != nil {
		t.Fatal(err)
	}
	if len(graph["deliveries"].([]any)) == 0 {
		t.Fatal("the fixture declares no delivery, so the scope requires no evidence")
	}
	delete(graph, "validation_results")
	raw, _ = json.Marshal(graph)
	federatedTestWrite(t, root, ".pose/indexes/delivery-integrity.json", string(raw))
	federatedTestGit(t, root, "add", "--all")
	federatedTestGit(t, root, "commit", "-q", "-m", "external member without evidence")
	bundle, err := s.PrepareReviewBundle("milestone:program/core")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.Join(bundle.Blockers, " "), "no passed structured validation evidence") {
		t.Fatalf("a milestone with no change set lost its evidence gate: %v", bundle.Blockers)
	}
	if _, err := s.SealReviewBundle("milestone:program/core", time.Now()); err == nil {
		t.Fatal("a milestone with no evidence sealed")
	}
}

func TestSpecWithoutAChangeSetStaysUnsealable(t *testing.T) {
	root, s := reviewBundleFixture(t)
	path := filepath.Join(root, ".pose/indexes/delivery-integrity.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var graph map[string]any
	if err := json.Unmarshal(raw, &graph); err != nil {
		t.Fatal(err)
	}
	graph["change_sets"] = []any{}
	raw, _ = json.Marshal(graph)
	writeReviewFixture(t, root, ".pose/indexes/delivery-integrity.json", string(raw))
	commitEvidenceFixture(t, root)
	_, err = s.SealReviewBundle("spec:backend", time.Now())
	if err == nil || !strings.Contains(err.Error(), missingChangeSetBlocker) {
		t.Fatalf("a spec with no change set sealed once its second blocker was gone: %v", err)
	}
}
