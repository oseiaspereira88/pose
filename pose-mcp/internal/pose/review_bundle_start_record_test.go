package pose

import "testing"

// A start record is the baseline atomic start reconciles against, so it is an
// authority record like an action journal, not derived output. Unclassified,
// it made every spec started with `pose start --apply` impossible to seal.
func TestStartRecordIsAGovernanceSubjectPath(t *testing.T) {
	scope := ScopeRef{Kind: "spec", Slug: "demo"}
	class, include := reviewBundlePathClass(".pose/starts/demo.json", scope, nil)
	if class != "governance" || !include {
		t.Fatalf("start record class = %q include = %v, want governance in the subject", class, include)
	}
}
