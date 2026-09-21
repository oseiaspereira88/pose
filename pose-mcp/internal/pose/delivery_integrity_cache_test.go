package pose

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// A field added to DeliveryIntegrityGraph without being handled in
// copyDeliveryIntegrityGraph would be shared with every caller of the cache. This
// test fails on that omission by construction: it populates each field through
// reflection, copies, mutates the copy and asserts the original did not move.
func TestDeliveryGraphCopyCoversEveryField(t *testing.T) {
	var graph DeliveryIntegrityGraph
	value := reflect.ValueOf(&graph).Elem()
	typ := value.Type()
	for i := 0; i < typ.NumField(); i++ {
		field := value.Field(i)
		switch field.Kind() {
		case reflect.Int:
			field.SetInt(1)
		case reflect.String:
			field.SetString("x")
		case reflect.Slice:
			field.Set(reflect.MakeSlice(field.Type(), 1, 1))
		case reflect.Map:
			m := reflect.MakeMap(field.Type())
			m.SetMapIndex(reflect.ValueOf("k"), reflect.ValueOf([]string{"v"}))
			field.Set(m)
		default:
			t.Fatalf("field %s has kind %s, which this guard does not populate — extend it before adding the field", typ.Field(i).Name, field.Kind())
		}
	}
	graph.SchemaVersion = DeliveryIntegritySchemaVersion
	graph.ChangeSets[0] = ChangeSet{ID: "cs", Commits: []string{"a"}, Paths: []ObservedPath{{Action: "modified", Path: "p"}}}

	before, err := json.Marshal(graph)
	if err != nil {
		t.Fatal(err)
	}
	copied := copyDeliveryIntegrityGraph(graph)

	// Mutate every mutable field of the copy.
	cv := reflect.ValueOf(&copied).Elem()
	for i := 0; i < typ.NumField(); i++ {
		field := cv.Field(i)
		switch field.Kind() {
		case reflect.Slice:
			if field.Len() > 0 {
				field.Set(field.Slice(0, 0))
			}
		case reflect.Map:
			field.SetMapIndex(reflect.ValueOf("k"), reflect.ValueOf([]string{"mutated"}))
		}
	}
	copied.ChangeSets = copyDeliveryIntegrityGraph(graph).ChangeSets
	if len(copied.ChangeSets) > 0 {
		copied.ChangeSets[0].Commits[0] = "mutated"
		copied.ChangeSets[0].Paths[0].Path = "mutated"
	}

	after, err := json.Marshal(graph)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Errorf("mutating the copy changed the original — a field is shared\nbefore: %s\nafter:  %s", before, after)
	}
}

// Identical bytes must not be parsed twice, and a changed file must not serve the
// previous graph.
func TestDeliveryGraphCacheKeyedOnContent(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".pose", "indexes"), 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, ".pose", "indexes", "delivery-integrity.json")
	write := func(g DeliveryIntegrityGraph) {
		raw, err := json.Marshal(g)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, raw, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	store := Store{Root: root}

	write(DeliveryIntegrityGraph{SchemaVersion: DeliveryIntegritySchemaVersion,
		Findings: []DeliveryIntegrityFinding{{ID: "f1", Code: "first", Path: "a.go"}}})
	first, err := store.GetDeliveryIntegrity("")
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Findings) != 1 || first.Findings[0].Code != "first" {
		t.Fatalf("first read = %+v", first.Findings)
	}
	// A second read of the same bytes is served from the cache and must be an
	// independent value: truncating this one must not shorten the next.
	second, err := store.GetDeliveryIntegrity("")
	if err != nil {
		t.Fatal(err)
	}
	second.Findings = second.Findings[:0]
	third, err := store.GetDeliveryIntegrity("")
	if err != nil {
		t.Fatal(err)
	}
	if len(third.Findings) != 1 {
		t.Fatalf("a cached graph was consumed by an earlier caller: %+v", third.Findings)
	}

	// Rewriting the file with different content must not serve the old graph, even
	// within the same second.
	write(DeliveryIntegrityGraph{SchemaVersion: DeliveryIntegritySchemaVersion,
		Findings: []DeliveryIntegrityFinding{{ID: "f2", Code: "second", Path: "b.go"}}})
	updated, err := store.GetDeliveryIntegrity("")
	if err != nil {
		t.Fatal(err)
	}
	if len(updated.Findings) != 1 || updated.Findings[0].Code != "second" {
		t.Fatalf("a changed index served the cached graph: %+v", updated.Findings)
	}

	// The path filter must not narrow the graph for the next caller either.
	if _, err := store.GetDeliveryIntegrity("b.go"); err != nil {
		t.Fatal(err)
	}
	full, err := store.GetDeliveryIntegrity("")
	if err != nil {
		t.Fatal(err)
	}
	if len(full.Findings) != 1 || full.Findings[0].Code != "second" {
		t.Fatalf("the path filter narrowed the cached graph: %+v", full.Findings)
	}
}

// The same omission guard as the graph copy, for the design-delta report: a slice
// added to it without being copied would be shared through the memo.
func TestDesignDeltaCopyCoversEveryField(t *testing.T) {
	report := DesignDeltaReport{
		SchemaVersion: DesignDeltaSchemaVersion, ParserVersion: "p", Scope: "spec:x",
		Status: "observed", InputDigest: "sha256:a", CacheKey: "sha256:a",
		Subject:  DesignDeltaSubjectSummary{Entries: 1},
		Coverage: DesignDeltaCoverage{Detectors: []DesignDeltaDetector{{ID: "d", State: "observed"}}},
		Deltas:   []StructuralDelta{{ID: "sd", Kind: "dependency"}},
		Warnings: []string{"w"},
	}
	value := reflect.ValueOf(report)
	for i := 0; i < value.NumField(); i++ {
		field := value.Field(i)
		switch field.Kind() {
		case reflect.Int, reflect.String, reflect.Struct, reflect.Bool:
		case reflect.Slice:
			if field.Len() == 0 {
				t.Fatalf("field %s is a slice this guard left empty — populate it so the copy is measured", value.Type().Field(i).Name)
			}
		default:
			t.Fatalf("field %s has kind %s, which this guard does not cover", value.Type().Field(i).Name, field.Kind())
		}
	}
	before, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	copied := copyDesignDeltaReport(report)
	copied.Deltas[0].Kind = "mutated"
	copied.Warnings[0] = "mutated"
	copied.Coverage.Detectors[0].State = "mutated"
	after, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Errorf("mutating the copied report changed the original\nbefore: %s\nafter:  %s", before, after)
	}
}

// A memo hit must be an independent value, and a different digest must not be
// served from a previous one.
func TestDesignDeltaMemoIsKeyedAndCopied(t *testing.T) {
	storeDesignDelta("key-a", DesignDeltaReport{Status: "observed", Warnings: []string{"a"},
		Deltas: []StructuralDelta{{ID: "sd-a"}}})
	first, ok := cachedDesignDelta("key-a")
	if !ok || first.Deltas[0].ID != "sd-a" {
		t.Fatalf("memo did not return the stored report: %+v", first)
	}
	first.Deltas[0].ID = "mutated"
	first.Warnings[0] = "mutated"
	second, ok := cachedDesignDelta("key-a")
	if !ok || second.Deltas[0].ID != "sd-a" || second.Warnings[0] != "a" {
		t.Fatalf("a caller's edit reached the memo: %+v", second)
	}
	if _, ok := cachedDesignDelta("key-b"); ok {
		t.Fatal("an unknown digest was served from the memo")
	}
}

// A published root manifest must classify as governance, or a spec that changes it
// cannot seal a review bundle: the classifier refuses an unclassified subject path
// rather than guessing, which is correct and is why the list has to be complete.
func TestPublishedRootManifestsClassifyAsGovernance(t *testing.T) {
	scope, err := ParseScopeRef("spec:any")
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{
		"compatibility.json",
		"composition-contract.json",
		"pose-mcp/server.json",
		".pose/project.json",
		".pose/release-policy.json",
	} {
		class, include := reviewBundlePathClass(path, scope, nil)
		if class != "governance" || !include {
			t.Errorf("%s classified as %q include=%v, want governance and included", path, class, include)
		}
	}
	// A root file nobody declared stays unclassified, so the refusal still exists.
	if class, _ := reviewBundlePathClass("some-new-root-thing.json", scope, nil); class != "" {
		t.Errorf("an undeclared root file classified as %q; the refusal is the guard", class)
	}
}
