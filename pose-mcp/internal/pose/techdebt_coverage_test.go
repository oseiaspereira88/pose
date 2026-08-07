package pose

import "testing"

func TestDocumentCoversDebtRequiresMarkerEvidence(t *testing.T) {
	item := TechDebtItem{
		ID:        "DEBT-001",
		File:      "pose-mcp/internal/scaffold/scaffold.go",
		Line:      23,
		Component: "pose-mcp",
	}

	covering := map[string]string{
		"file path":      "the panic in pose-mcp/internal/scaffold/scaffold.go must go",
		"stable debt id": "tracked as debt-001 in this cycle",
	}
	for name, document := range covering {
		if !documentCoversDebt(document, item) {
			t.Errorf("%s should count as coverage: %q", name, document)
		}
	}

	// A component name locates the debt; it does not commit anyone to it.
	// Treating it as coverage let any active spec touching the module silently
	// clear every marker inside it.
	notCovering := map[string]string{
		"delivery target line":   "contract:mcp-active-context module:pose-mcp profile:api-contract",
		"backticked component":   "this spec touches `pose-mcp` and its catalog",
		"components frontmatter": "components: pose-mcp",
		"unrelated file":         "fixes pose-mcp/internal/cli/doctor.go",
	}
	for name, document := range notCovering {
		if documentCoversDebt(document, item) {
			t.Errorf("%s must not count as coverage: %q", name, document)
		}
	}
}
