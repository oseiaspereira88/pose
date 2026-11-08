package pose

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// R3 and R6 (spec pose-bounded-isolated-graph-caches): what a cache key reflects,
// and which caches may outlive an operation.

func TestTheGraphCacheKeyIsTheContentNotTheSizeOrTheOwner(t *testing.T) {
	deliveryGraphCache.Reset()
	base := DeliveryGraphCacheStats()
	a := []byte(`{"schema_version":1,"input_digest":"aaaa","nodes":[],"edges":[],"claims":[],"change_sets":[],"reverse":{},"findings":[]}`)
	b := []byte(`{"schema_version":1,"input_digest":"bbbb","nodes":[],"edges":[],"claims":[],"change_sets":[],"reverse":{},"findings":[]}`)
	if len(a) != len(b) {
		t.Fatal("the fixtures must have the same size")
	}
	ga, _ := parseDeliveryIntegrityGraph("/work/one", a)
	gb, _ := parseDeliveryIntegrityGraph("/work/two", b)
	if ga.InputDigest != "aaaa" || gb.InputDigest != "bbbb" {
		t.Fatalf("a graph was served for other bytes: %q %q", ga.InputDigest, gb.InputDigest)
	}
	// The same size and the same owner as a, but different bytes: its own graph, not a's.
	gb2, _ := parseDeliveryIntegrityGraph("/work/one", b)
	if gb2.InputDigest != "bbbb" {
		t.Fatalf("same size and owner served the previous graph: %q", gb2.InputDigest)
	}
	// Two worktrees holding the same bytes share one entry: the graph is a pure function of them.
	if g, _ := parseDeliveryIntegrityGraph("/work/three", a); g.InputDigest != "aaaa" {
		t.Fatalf("identical bytes served another graph: %q", g.InputDigest)
	}
	s := DeliveryGraphCacheStats()
	if s.Misses-base.Misses != 2 || s.Hits-base.Hits != 2 || s.Entries != 2 {
		t.Fatalf("two contents must be two productions and identical bytes one entry: %+v", s)
	}
}

func gitFixture(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	run := func(args ...string) {
		cmd := exec.Command("git", append([]string{"-C", root, "-c", "user.name=t", "-c", "user.email=t@t"}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	run("init", "-q")
	for name, body := range files {
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	run("add", "-A")
	run("commit", "-q", "-m", "fixture")
	return root
}

func TestTheTestCatalogIsKeyedByTheMatrixAndMeteredPerRoot(t *testing.T) {
	root := gitFixture(t, map[string]string{
		"go.mod":                               "module x\n",
		"a_test.go":                            "package x\nfunc TestAlpha(t *testing.T) {}\n",
		".pose/indexes/validation-matrix.json": `{"moduleOverrides":{}}`,
	})
	testCatalogCache.Reset()
	base := testCatalogCache.Stats()
	if LoadTestCatalog(root) == nil || LoadTestCatalog(root) == nil {
		t.Fatal("no catalog")
	}
	s := testCatalogCache.Stats()
	if s.Misses-base.Misses != 1 || s.Hits-base.Hits != 1 {
		t.Fatalf("the second load must be a hit: %+v", s)
	}
	// A changed matrix is another key: the catalog is rebuilt, not served stale.
	if err := os.WriteFile(filepath.Join(root, ".pose", "indexes", "validation-matrix.json"), []byte(`{"moduleOverrides":{"x":{}}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	LoadTestCatalog(root)
	if s2 := testCatalogCache.Stats(); s2.Misses-base.Misses != 2 {
		t.Fatalf("a changed matrix was served from the old entry: %+v", s2)
	}
	// A tree Git cannot list is not an empty catalog and is not remembered.
	if LoadTestCatalog(t.TempDir()) != nil {
		t.Fatal("a directory Git cannot list produced a catalog")
	}
}

func TestADesignDeltaReportIsIsolatedAndBounded(t *testing.T) {
	designDeltaCache.Reset()
	report := DesignDeltaReport{Deltas: []StructuralDelta{{ID: "d1", SharedWith: []string{"other"}}}, Warnings: []string{"w"}}
	designDeltaCache.Store("/r", "k", report, 100)
	got, ok := designDeltaCache.Lookup("k")
	if !ok {
		t.Fatal("miss")
	}
	got.Deltas[0].ID = "mutated"
	got.Deltas[0].SharedWith[0] = "mutated"
	got.Warnings[0] = "mutated"
	again, _ := designDeltaCache.Lookup("k")
	if again.Deltas[0].ID != "d1" || again.Deltas[0].SharedWith[0] != "other" || again.Warnings[0] != "w" {
		t.Fatalf("a caller's write reached the cached report: %+v", again)
	}
	for i := 0; i < 2000; i++ {
		designDeltaCache.Store("/r", "bulk-"+strings.Repeat("x", 100)+string(rune(i)), DesignDeltaReport{}, 1<<20)
	}
	if s := designDeltaCache.Stats(); s.Retained > 32<<20 {
		t.Fatalf("the design-delta cache is not bounded in bytes: %+v", s)
	}
}

// The deep copy of the design-delta report covers every field by construction: a
// reflection test fills every slice, map and pointer and finds what the copy shares.
func TestADesignDeltaReportSharesNoReferenceWithItsCopies(t *testing.T) {
	var report DesignDeltaReport
	populate(reflect.ValueOf(&report).Elem(), 0)
	copied := copyDesignDeltaReport(report)
	if !reflect.DeepEqual(report, copied) {
		t.Fatal("the copy does not equal the original")
	}
	if shared := sharedReferences(report, copied); len(shared) != 0 {
		t.Fatalf("the copy shares references with the original: %v", shared)
	}
}
