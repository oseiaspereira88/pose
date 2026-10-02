package pose

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// compactFixture builds a real delivery graph: `targets` capabilities that share
// one module and one entrypoint, gated by `results` passing integration checks
// that were all produced by one validation run.
func compactFixture(t *testing.T, targets, results int) DeliveryIntegrityGraph {
	t.Helper()
	specs, claims, sets, tracked := []Spec{}, []ArtifactClaim{}, []ChangeSet{}, []string{}
	deliveries := []DeliveryTarget{}
	for i := 0; i < targets; i++ {
		slug := fmt.Sprintf("spec-%02d", i)
		path := fmt.Sprintf("web/file%02d.go", i)
		specs = append(specs, Spec{Slug: slug, Status: "done"})
		claims = append(claims, ArtifactClaim{Spec: slug, Action: "modified", Path: path})
		sets = append(sets, ChangeSet{ID: "cs-" + slug, Spec: slug, Paths: []ObservedPath{{Action: "modified", Path: path}}})
		tracked = append(tracked, path)
		deliveries = append(deliveries, DeliveryTarget{Spec: slug, Ref: "capability:cap-" + slug, Kind: "capability", ID: "cap-" + slug, Module: "web", Profile: "composed-capability", Entrypoint: "cmd/app/main.go"})
	}
	base := BuildDeliveryIntegrity(specs, claims, sets, tracked, ArtifactPolicy{})
	scope := map[string]string{}
	for i := 0; i < targets; i++ {
		slug := fmt.Sprintf("spec-%02d", i)
		scope[slug] = ScopedDeliveryProvenanceDigest(base, slug)
	}
	found := []DeliveryValidationResult{}
	for i := 0; i < results; i++ {
		found = append(found, DeliveryValidationResult{
			ID: fmt.Sprintf("web/go/check-%02d", i), Module: "web", Check: fmt.Sprintf("check-%02d", i), EvidenceClass: "integration", Severity: "required", Outcome: "pass",
			GitHead: "abc123", GeneratedAt: "2026-10-02T00:00:00Z", ProvenanceDigest: base.ProvenanceDigest, Report: ".pose/results/delivery-validation.json", ScopeProvenance: scope,
		})
	}
	profiles := map[string]DeliveryProfile{"composed-capability": {Kind: "capability", RequiredEvidenceClasses: []string{"integration"}}}
	return BuildDeliverySurface(base, specs, deliveries, found, nil, profiles, DeliveryPolicy{})
}

func countEdges(g DeliveryIntegrityGraph, kind string) int {
	n := 0
	for _, edge := range g.Edges {
		if edge.Type == kind {
			n++
		}
	}
	return n
}

func decodeForTest(t *testing.T, raw []byte) DeliveryIntegrityGraph {
	t.Helper()
	graph, ok := decodeDeliveryIntegrityIndex(raw)
	if !ok {
		t.Fatalf("index did not decode:\n%.400s", raw)
	}
	return graph
}

func asJSON(t *testing.T, value any) string {
	t.Helper()
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

// R5: the expanded form holds the same delivery-to-result pairs, findings and
// per-result provenance the writer started from.
func TestCompactIndexExpandsToTheGraphItWasWrittenFrom(t *testing.T) {
	graph := compactFixture(t, 6, 5)
	raw, err := graph.IndexJSON()
	if err != nil {
		t.Fatal(err)
	}
	expanded := decodeForTest(t, raw)
	if asJSON(t, expanded) != asJSON(t, graph) {
		t.Fatalf("expanding the index changed the graph:\nwritten  %.600s\nexpanded %.600s", asJSON(t, graph), asJSON(t, expanded))
	}
	if countEdges(graph, "validated-by") != (6+1)*5 {
		t.Fatalf("fixture does not exercise the product shape: %d validated-by edges", countEdges(graph, "validated-by"))
	}
	for _, result := range expanded.ValidationResults {
		if result.ScopeProvenance["spec-03"] == "" || result.GitHead != "abc123" || result.Report == "" {
			t.Fatalf("result lost its run data: %+v", result)
		}
	}
}

// R1: one run, one scope_provenance map in the file.
func TestCompactIndexStoresARunsScopeProvenanceOnce(t *testing.T) {
	graph := compactFixture(t, 4, 7)
	raw, err := graph.IndexJSON()
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Count(string(raw), `"spec-02": "sha256:`); got != 1 {
		t.Fatalf("scope_provenance entry is written %d times, want once", got)
	}
	var file compactDeliveryGraph
	if err := json.Unmarshal(raw, &file); err != nil {
		t.Fatal(err)
	}
	if len(file.ValidationRuns) != 1 || len(file.ValidationResults) != 7 {
		t.Fatalf("runs=%d results=%d", len(file.ValidationRuns), len(file.ValidationResults))
	}
	for _, result := range file.ValidationResults {
		if result.Run != file.ValidationRuns[0].ID {
			t.Fatalf("result %s does not reference the run", result.ID)
		}
	}
	// A caller that edits one result's map must not change another's.
	expanded := decodeForTest(t, raw)
	expanded.ValidationResults[0].ScopeProvenance["spec-00"] = "changed"
	if expanded.ValidationResults[1].ScopeProvenance["spec-00"] == "changed" {
		t.Fatal("results share one scope_provenance map")
	}
}

// R2, R3, R4: one set node per distinct result set, one validated-by edge per
// delivery, no repeated edge and no per-result path entries.
func TestCompactIndexSharesOneNodePerDistinctResultSet(t *testing.T) {
	graph := compactFixture(t, 6, 5)
	file := compactDeliveryIndex(graph)
	sets := 0
	for _, node := range file.Nodes {
		if node.Type == "validation-set" {
			sets++
		}
	}
	if sets != 1 {
		t.Fatalf("set nodes=%d, want 1 for one distinct result set", sets)
	}
	validated := 0
	for _, edge := range file.Edges {
		if edge.Type == "validated-by" {
			validated++
			if !strings.HasPrefix(edge.To, validationSetPrefix) {
				t.Fatalf("validated-by points at %s, not a set", edge.To)
			}
		}
	}
	// Six targets plus the one entrypoint they share.
	if validated != 7 || countEdges(DeliveryIntegrityGraph{Edges: file.Edges}, "contains") != 5 {
		t.Fatalf("validated-by=%d contains=%d", validated, countEdges(DeliveryIntegrityGraph{Edges: file.Edges}, "contains"))
	}
	seen := map[DeliveryIntegrityEdge]bool{}
	for _, edge := range file.Edges {
		if seen[edge] {
			t.Fatalf("edge written twice: %+v", edge)
		}
		seen[edge] = true
	}
	for key, path := range file.Paths {
		for _, element := range path {
			if strings.HasPrefix(element, validationResultPrefix) {
				t.Fatalf("path %s lists a result: %v", key, path)
			}
		}
	}
	// The graph itself no longer carries repeats either.
	seen = map[DeliveryIntegrityEdge]bool{}
	for _, edge := range graph.Edges {
		if seen[edge] {
			t.Fatalf("built graph repeats an edge: %+v", edge)
		}
		seen[edge] = true
	}
}

// R6: doubling the deliveries adds edges in proportion to the deliveries, not to
// deliveries times results.
func TestCompactIndexGrowsWithDeliveriesNotWithTheirProduct(t *testing.T) {
	small, large := compactDeliveryIndex(compactFixture(t, 10, 8)), compactDeliveryIndex(compactFixture(t, 20, 8))
	count := func(file compactDeliveryGraph, kind string) int {
		return countEdges(DeliveryIntegrityGraph{Edges: file.Edges}, kind)
	}
	if count(small, "contains") != count(large, "contains") {
		t.Fatalf("set membership grew with deliveries: %d -> %d", count(small, "contains"), count(large, "contains"))
	}
	if count(large, "validated-by")-count(small, "validated-by") != 10 {
		t.Fatalf("validated-by grew by %d for 10 more deliveries", count(large, "validated-by")-count(small, "validated-by"))
	}
	legacy := func(file DeliveryIntegrityGraph) int { return countEdges(file, "validated-by") }
	if legacy(compactFixture(t, 20, 8))-legacy(compactFixture(t, 10, 8)) != 10*8 {
		t.Fatal("fixture no longer reproduces the product shape the compact form removes")
	}
}

// The worst case: every delivery is gated by a different set. Size is then bounded
// by the sum of the set sizes, which is what the legacy form already cost.
func TestCompactIndexWorstCaseIsNoLargerThanTheExpandedPairs(t *testing.T) {
	graph := compactFixture(t, 4, 3)
	// Make every target's set distinct by dropping a different result from each.
	kept := graph.Edges[:0]
	drop := map[string]string{}
	i := 0
	for _, edge := range graph.Edges {
		if edge.Type == "validated-by" && strings.HasPrefix(edge.From, "delivery:") {
			if _, ok := drop[edge.From]; !ok {
				drop[edge.From] = fmt.Sprintf("%s%s", validationResultPrefix, fmt.Sprintf("web/go/check-%02d", i%3))
				i++
			}
			if drop[edge.From] == edge.To {
				continue
			}
		}
		kept = append(kept, edge)
	}
	graph.Edges = kept
	graph.Paths = map[string][]string{}
	file := compactDeliveryIndex(graph)
	// Membership is stored once per distinct set, so with every set distinct it is
	// exactly the number of expanded pairs: no cheaper, and no dearer.
	members := map[string]map[string]bool{}
	for _, edge := range graph.Edges {
		if edge.Type != "validated-by" {
			continue
		}
		if members[edge.From] == nil {
			members[edge.From] = map[string]bool{}
		}
		members[edge.From][edge.To] = true
	}
	distinct := map[string]int{}
	for _, set := range members {
		keys := []string{}
		for member := range set {
			keys = append(keys, member)
		}
		distinct[validationSetID(sortedCopy(keys))] = len(keys)
	}
	want := 0
	for _, size := range distinct {
		want += size
	}
	if got := countEdges(DeliveryIntegrityGraph{Edges: file.Edges}, "contains"); got != want || len(distinct) < 3 {
		t.Fatalf("contains=%d want %d over %d distinct sets", got, want, len(distinct))
	}
	expanded := decodeForTest(t, mustIndexJSON(t, graph))
	if asJSON(t, expanded.Edges) != asJSON(t, graph.Edges) {
		t.Fatal("distinct sets did not round-trip")
	}
}

func mustIndexJSON(t *testing.T, graph DeliveryIntegrityGraph) []byte {
	t.Helper()
	raw, err := graph.IndexJSON()
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

// R7: writing and reading the index leaves the digests that bundles depend on.
func TestCompactIndexLeavesTheProvenanceDigestsAlone(t *testing.T) {
	graph := compactFixture(t, 5, 4)
	expanded := decodeForTest(t, mustIndexJSON(t, graph))
	if expanded.ProvenanceDigest != graph.ProvenanceDigest || expanded.ProvenanceDigest == "" {
		t.Fatalf("provenance digest moved: %q -> %q", graph.ProvenanceDigest, expanded.ProvenanceDigest)
	}
	for i := 0; i < 5; i++ {
		slug := fmt.Sprintf("spec-%02d", i)
		if ScopedDeliveryProvenanceDigest(expanded, slug) != ScopedDeliveryProvenanceDigest(graph, slug) {
			t.Fatalf("scoped digest for %s moved", slug)
		}
	}
	// Evidence that was current before is still current after.
	for _, result := range expanded.ValidationResults {
		if !deliveryEvidenceCurrent(result, "spec-02", "done", expanded, nil) {
			t.Fatalf("result %s is no longer current evidence", result.ID)
		}
	}
}

// R8: schema 1 is read as written, schema 2 is expanded, nothing else is read.
func TestDeliveryIntegrityIndexReadsSchemaOneAndTwoAndNothingElse(t *testing.T) {
	graph := compactFixture(t, 3, 3)
	legacy, err := json.Marshal(graph)
	if err != nil {
		t.Fatal(err)
	}
	fromOne := decodeForTest(t, legacy)
	fromTwo := decodeForTest(t, mustIndexJSON(t, graph))
	if asJSON(t, fromOne) != asJSON(t, fromTwo) {
		t.Fatal("schema 1 and schema 2 read as different graphs")
	}
	if _, ok := decodeDeliveryIntegrityIndex([]byte(`{"schema_version":3}`)); ok {
		t.Fatal("an unknown schema was accepted")
	}
	if _, ok := decodeDeliveryIntegrityIndex([]byte(`{"reverse":{}}`)); ok {
		t.Fatal("an index with no schema was accepted")
	}
	for name, body := range map[string]string{
		"unknown run": `{"schema_version":2,"validation_results":[{"id":"r","module":"m","check":"c","evidence_class":"integration","outcome":"pass","run":"run-missing"}]}`,
		"unknown set": `{"schema_version":2,"edges":[{"from":"delivery:x","to":"validation-set:missing","type":"validated-by"}]}`,
	} {
		if _, ok := decodeDeliveryIntegrityIndex([]byte(body)); ok {
			t.Fatalf("%s was accepted", name)
		}
	}
}

// The index a command writes is the one the store reads back.
func TestStoreReadsTheIndexItWroteInSchemaTwo(t *testing.T) {
	root := t.TempDir()
	graph := compactFixture(t, 3, 3)
	path := filepath.Join(root, ".pose", "indexes", "delivery-integrity.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(mustIndexJSON(t, graph), '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := (Store{Root: root}).GetDeliveryIntegrity("")
	if err != nil {
		t.Fatal(err)
	}
	if asJSON(t, got) != asJSON(t, graph) {
		t.Fatal("the store did not read back the graph that was written")
	}
	var head struct {
		SchemaVersion int `json:"schema_version"`
	}
	raw, _ := os.ReadFile(path)
	_ = json.Unmarshal(raw, &head)
	if head.SchemaVersion != DeliveryIntegrityIndexSchemaVersion {
		t.Fatalf("file schema=%d", head.SchemaVersion)
	}
}

// R9: change-set-to-artifact edges follow from the change sets, so the index leaves
// them out and reading puts them back; when they do not follow, they stay.
func TestCompactIndexDerivesChangesEdgesFromTheChangeSets(t *testing.T) {
	graph := compactFixture(t, 5, 3)
	if countEdges(graph, "changes") == 0 {
		t.Fatal("fixture has no changes edges to derive")
	}
	file := compactDeliveryIndex(graph)
	if countEdges(DeliveryIntegrityGraph{Edges: file.Edges}, "changes") != 0 || len(file.ImpliedEdges) != 1 {
		t.Fatalf("changes edges were written: implied=%v", file.ImpliedEdges)
	}
	if asJSON(t, decodeForTest(t, mustIndexJSON(t, graph)).Edges) != asJSON(t, graph.Edges) {
		t.Fatal("the derived edges are not the edges that were left out")
	}

	// An edge the change sets do not imply is information the index cannot drop.
	extra := compactFixture(t, 5, 3)
	extra.Edges = append(extra.Edges, DeliveryIntegrityEdge{From: "change-set:cs-spec-00", To: "artifact:not/in/the/set.go", Type: "changes"})
	sort.Slice(extra.Edges, func(i, j int) bool { return edgeLess(extra.Edges[i], extra.Edges[j]) })
	kept := compactDeliveryIndex(extra)
	if len(kept.ImpliedEdges) != 0 || countEdges(DeliveryIntegrityGraph{Edges: kept.Edges}, "changes") != countEdges(extra, "changes") {
		t.Fatalf("an edge that is not implied was dropped: implied=%v", kept.ImpliedEdges)
	}
	if asJSON(t, decodeForTest(t, mustIndexJSON(t, extra)).Edges) != asJSON(t, extra.Edges) {
		t.Fatal("kept changes edges did not round-trip")
	}
	if _, ok := decodeDeliveryIntegrityIndex([]byte(`{"schema_version":2,"implied_edges":["reaches"]}`)); ok {
		t.Fatal("an unknown implied edge type was accepted")
	}
}
