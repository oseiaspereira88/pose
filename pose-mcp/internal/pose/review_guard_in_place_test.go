package pose

import (
	"reflect"
	"sort"
	"testing"
)

// guardFromACopy is the guard's change-set part computed from a deep copy of
// the graph, as it was before the guard read the graph in place.
func guardFromACopy(t *testing.T, s Store, wanted map[string]bool) ([]string, string) {
	t.Helper()
	graph, err := s.GetDeliveryIntegrity("")
	if err != nil {
		t.Fatal(err)
	}
	sets := []ChangeSet{}
	for _, set := range graph.ChangeSets {
		if wanted[set.Spec] {
			sets = append(sets, set)
		}
	}
	sort.Slice(sets, func(i, j int) bool { return sets[i].ID < sets[j].ID })
	paths, revision := []string{}, ""
	for _, set := range sets {
		for _, observed := range set.Paths {
			for _, path := range []string{observed.Path, observed.OldPath, observed.NewPath} {
				if path != "" && reviewGovernanceContractPath(path) {
					paths = append(paths, path)
				}
			}
		}
		if revision == "" {
			revision = set.ResolvedBase
		}
	}
	return uniqueSorted(paths), revision
}

// Spec pose-review-guard-reads-the-integrity-graph-in-place, R1 and R2: the
// guard of a scope that changes the review contract is what a copy of the
// graph gives, twice, and reading in place leaves the retained graph intact.
func TestReviewGuardReadsTheIntegrityGraphInPlace(t *testing.T) {
	_, store := protectedBaselineFixture(t)
	retained := func() DeliveryIntegrityGraph {
		var copied DeliveryIntegrityGraph
		if err := store.viewDeliveryIntegrity(func(graph DeliveryIntegrityGraph) { copied = copyDeliveryIntegrityGraph(graph) }); err != nil {
			t.Fatal(err)
		}
		return copied
	}
	before := retained()
	for i := 0; i < 2; i++ {
		guard := store.resolveReviewGovernanceGuard(ScopeRef{Kind: "spec", Slug: "governance"})
		paths, revision := guardFromACopy(t, store, map[string]bool{"governance": true})
		if len(guard.Paths) == 0 || guard.Revision == "" {
			t.Fatalf("the fixture's guard has no contract change: %+v", guard)
		}
		if !reflect.DeepEqual(guard.Paths, paths) || guard.Revision != revision {
			t.Fatalf("read %d: guard %v@%s, a copy gives %v@%s", i, guard.Paths, guard.Revision, paths, revision)
		}
	}
	if after := retained(); !reflect.DeepEqual(before, after) {
		t.Fatal("reading the graph in place changed the retained graph")
	}
}
