package pose

import (
	"strings"
	"testing"
)

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

// A marker's id must come from the marker itself, so inserting another marker
// earlier in the same file cannot make an existing citation point somewhere
// else (spec pose-governance-gate-activation, R1).
func TestTechDebtIDIsDerivedFromMarkerIdentityNotScanPosition(t *testing.T) {
	first := techDebtID("internal/pose/state.go", "TODO", "// TODO: reconcile the ledger")
	same := techDebtID("internal/pose/state.go", "TODO", "// TODO: reconcile the ledger")
	if first != same {
		t.Errorf("the same marker must keep the same id: %s != %s", first, same)
	}

	// Adding an unrelated marker anywhere in the scan leaves this one alone.
	other := techDebtID("internal/pose/state.go", "FIXME", "// FIXME: drop the cache")
	if other == first {
		t.Error("different markers must not collide")
	}

	// The same text in another file is a different debt.
	elsewhere := techDebtID("internal/cli/report.go", "TODO", "// TODO: reconcile the ledger")
	if elsewhere == first {
		t.Error("the same marker text in another file must be a different debt")
	}

	// Moving down the file is not a new debt: the line number is excluded on
	// purpose, or an id would change whenever unrelated code is inserted above.
	if !strings.HasPrefix(first, "DEBT-") || len(first) != len("DEBT-")+8 {
		t.Errorf("unexpected id shape: %s", first)
	}
}
