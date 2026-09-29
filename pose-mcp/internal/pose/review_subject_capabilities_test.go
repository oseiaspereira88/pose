package pose

import (
	"encoding/json"
	"strings"
	"testing"
)

// The capability assessment is written by POSE itself (pose capability) and
// its bullets are the authority on each mechanism's state; the history beside
// it is the append-only record of snapshots taken from it. A change set that
// touched either could not be sealed: both fell through every classification
// rule and blocked as "unclassified review subject path".
func TestReviewSubjectClassifiesTheCapabilityAssessment(t *testing.T) {
	root, store := reviewBundleFixture(t)
	writeReviewFixture(t, root, ".pose/capabilities/assessment.md", "---\nschema_version: 1\n---\n")
	writeReviewFixture(t, root, ".pose/capabilities/history.jsonl", "{}\n")
	graph, err := store.GetDeliveryIntegrity("")
	if err != nil {
		t.Fatal(err)
	}
	graph.ChangeSets[0].Paths = append(graph.ChangeSets[0].Paths,
		ObservedPath{Action: "modified", Path: ".pose/capabilities/assessment.md"},
		ObservedPath{Action: "modified", Path: ".pose/capabilities/history.jsonl"},
	)
	raw, _ := json.Marshal(graph)
	writeReviewFixture(t, root, ".pose/indexes/delivery-integrity.json", string(raw))
	bundle, err := store.PrepareReviewBundle("spec:backend")
	if err != nil {
		t.Fatal(err)
	}
	if blockers := strings.Join(bundle.Blockers, " "); strings.Contains(blockers, ".pose/capabilities/") {
		t.Fatalf("capability paths blocked the bundle: %s", blockers)
	}
	classes := map[string]string{}
	for _, entry := range bundle.Payload.Subject.Entries {
		classes[entry.Path] = entry.Class
	}
	// The assessment is authored state a reviewer must see change, so it is
	// in the subject; the history is derived from it and is recorded as an
	// excluded input instead.
	if got := classes[".pose/capabilities/assessment.md"]; got != "governance" {
		t.Errorf("assessment class = %q, want governance", got)
	}
	if _, ok := classes[".pose/capabilities/history.jsonl"]; ok {
		t.Error("capability history is in the review subject")
	}
	excluded := false
	for _, input := range bundle.ExcludedInputs {
		if input.Path == ".pose/capabilities/history.jsonl" && input.Kind == "derived-evidence" {
			excluded = true
		}
	}
	if !excluded {
		t.Errorf("capability history is not an excluded derived-evidence input: %+v", bundle.ExcludedInputs)
	}
}
