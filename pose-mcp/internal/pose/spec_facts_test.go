package pose

import "testing"

// Spec pose-progressive-spec-surface: facts come from the index, never invented.
func TestSpecFactsAreDerivedFromTheIndex(t *testing.T) {
	_, store := reviewBundleFixture(t)
	facts, err := store.DeriveSpecFacts("backend")
	if err != nil {
		t.Fatal(err)
	}
	if len(facts.Paths) != 1 || facts.Paths[0].Path != "api/server.go" || facts.Source == "" {
		t.Fatalf("paths: %+v", facts)
	}
}
