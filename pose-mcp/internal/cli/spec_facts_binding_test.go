package cli

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/harne8/pose-mcp/internal/pose"
)

// Spec pose-spec-facts-evidence-binding: the human text and the JSON carry one classification.
func TestSpecsFactsLabelsCandidatesAndFreshnessInBothOutputs(t *testing.T) {
	root := t.TempDir()
	writeCloseoutCLIFile(t, root, ".pose/specs/2026-10-01-alpha.md", "---\nslug: alpha\nstatus: in-progress\n---\n\n# Spec: alpha\n")
	graphJSON := `{"schema_version":1,"change_sets":[{"id":"cs","spec":"alpha","head":"h1","resolved_head":"h1","commits":["c"],"paths":[{"action":"modified","path":"api/a.go"}]}],"validation_results":[{"id":"1","module":"api","check":"go-test","evidence_class":"unit","outcome":"pass","git_head":"h1","scope_provenance":{"alpha":"@scoped"}},{"id":"2","module":"web","check":"lint","evidence_class":"lint","outcome":"pass","git_head":"h1"}]}`
	var graph pose.DeliveryIntegrityGraph
	if err := json.Unmarshal([]byte(graphJSON), &graph); err != nil {
		t.Fatal(err)
	}
	writeCloseoutCLIFile(t, root, ".pose/indexes/delivery-integrity.json", strings.ReplaceAll(graphJSON, "@scoped", pose.ScopedDeliveryProvenanceDigest(graph, "alpha")))
	code, out := runPose(t, root, "specs", "facts", "alpha", "--json")
	if code != 0 {
		t.Fatalf("%d %s", code, out)
	}
	var doc struct {
		Checks     []map[string]any `json:"checks"`
		Candidates []map[string]any `json:"candidates"`
		Freshness  map[string]any   `json:"freshness"`
	}
	if err := json.Unmarshal([]byte(out[strings.Index(out, "{"):]), &doc); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	if len(doc.Checks) != 1 || doc.Checks[0]["check"] != "go-test" || len(doc.Candidates) != 1 || doc.Candidates[0]["check"] != "lint" || doc.Freshness["status"] == "" {
		t.Fatalf("doc = %+v", doc)
	}
	code, text := runPose(t, root, "specs", "facts", "alpha")
	if code != 0 {
		t.Fatalf("%d %s", code, text)
	}
	for _, want := range []string{"facts.freshness", "unknown", "associated by scope-provenance", "CANDIDATE, not evidence of this spec", "git_head=h1", "scoped=sha256:", "limits: "} {
		if !strings.Contains(text, want) {
			t.Errorf("the human output lacks %q:\n%s", want, text)
		}
	}
	if strings.Contains(text, "facts.check=web lint") || strings.Contains(text, "facts.check: web lint") {
		t.Errorf("a candidate was printed as evidence:\n%s", text)
	}
}

// F06 of the second review: the text names both ends of a rename, as the JSON does.
func TestSpecsFactsRendersBothEndsOfARename(t *testing.T) {
	root := t.TempDir()
	writeCloseoutCLIFile(t, root, ".pose/specs/2026-10-01-alpha.md", "---\nslug: alpha\nstatus: in-progress\n---\n\n# Spec: alpha\n")
	writeCloseoutCLIFile(t, root, ".pose/indexes/delivery-integrity.json", `{"schema_version":1,"change_sets":[{"id":"cs","spec":"alpha","head":"h1","resolved_head":"h1","commits":["c"],"paths":[{"action":"renamed","old_path":"api/a.go","new_path":"web/a.go"}]}]}`)
	code, text := runPose(t, root, "specs", "facts", "alpha")
	if code != 0 || !strings.Contains(text, "api/a.go → web/a.go") {
		t.Fatalf("the human output does not name both ends of the rename (%d):\n%s", code, text)
	}
}
