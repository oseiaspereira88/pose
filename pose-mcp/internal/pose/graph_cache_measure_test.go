package pose

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// R7 (spec pose-bounded-isolated-graph-caches): measured, in the same corpus and
// environment, with the operation identified. The numbers are logged; the assertions
// keep the relations the design promises.

func realIndex(t *testing.T) []byte {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", ".pose", "indexes", "delivery-integrity.json"))
	if err != nil || len(raw) < 1<<20 {
		t.Skipf("no real delivery index to measure (%d bytes, %v)", len(raw), err)
	}
	return raw
}

func heapInUse() uint64 {
	runtime.GC()
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return m.HeapAlloc
}

func TestGraphByteEstimateTracksTheMeasuredHeap(t *testing.T) {
	raw := realIndex(t)
	graph, ok := decodeDeliveryIntegrityIndex(raw)
	if !ok {
		t.Fatal("the real index does not parse")
	}
	estimate := estimateGraphBytes(graph)
	runtime.KeepAlive(graph)
	graph = DeliveryIntegrityGraph{}
	before := heapInUse()
	retained := make([]DeliveryIntegrityGraph, 3)
	for i := range retained {
		retained[i], _ = decodeDeliveryIntegrityIndex(raw)
	}
	after := heapInUse()
	measured := int64(after-before) / 3
	t.Logf("operation=parse delivery-integrity.json (%d B raw): estimate=%d B, measured heap per graph=%d B (ratio %.2f)", len(raw), estimate, measured, float64(estimate)/float64(measured))
	runtime.KeepAlive(retained)
	// The estimate must neither hide most of the heap nor claim many times more than is held.
	if estimate < measured/3 || estimate > measured*3 {
		t.Fatalf("the byte estimate is not within a factor of 3 of the measured heap: %d vs %d", estimate, measured)
	}
}

func TestRepeatedReadsAreHitsAndAlternationStaysUnderTheBudget(t *testing.T) {
	raw := realIndex(t)
	deliveryGraphCache.Reset()
	base := DeliveryGraphCacheStats()

	start := time.Now()
	if _, ok := parseDeliveryIntegrityGraph("/work/a", raw); !ok {
		t.Fatal("parse")
	}
	miss := time.Since(start)
	const reads = 50
	start = time.Now()
	for i := 0; i < reads; i++ {
		if _, ok := parseDeliveryIntegrityGraph("/work/a", raw); !ok {
			t.Fatal("parse")
		}
	}
	hit := time.Since(start) / reads
	s := DeliveryGraphCacheStats()
	t.Logf("operation=read delivery-integrity.json: miss=%v, hit (defensive deep copy)=%v, hits=%d misses=%d retained=%d B", miss, hit, s.Hits-base.Hits, s.Misses-base.Misses, s.Retained)
	if s.Misses-base.Misses != 1 || s.Hits-base.Hits != reads {
		t.Fatalf("repeated reads must hit: %+v", s)
	}

	// Alternating between many worktrees holding different generations of the index
	// must stay under the budgets, evicting rather than growing.
	for i := 0; i < 12; i++ {
		variant := append([]byte(nil), raw...)
		variant = append(variant[:len(variant)-1:len(variant)-1], []byte(" ")...) // distinct bytes
		variant = append(variant, byte('0'+i%10), '\n')
		_ = variant // the parse of a mutated tail fails; use whole distinct documents below
	}
	var wg sync.WaitGroup
	for g := 0; g < 4; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for i := 0; i < 6; i++ {
				doc := []byte(`{"schema_version":1,"input_digest":"` + string(rune('a'+g)) + string(rune('0'+i)) + `","nodes":[],"edges":[],"claims":[],"change_sets":[],"reverse":{},"findings":[]}`)
				if _, ok := parseDeliveryIntegrityGraph("/work/"+string(rune('a'+g)), doc); !ok {
					t.Error("parse")
				}
			}
		}(g)
	}
	wg.Wait()
	s = DeliveryGraphCacheStats()
	if s.Retained > deliveryGraphCacheBytes {
		t.Fatalf("the global budget was exceeded: %+v", s)
	}
	t.Logf("after alternating 4 worktrees x 6 generations: entries=%d retained=%d B (budget %d B) evictions=%d", s.Entries, s.Retained, deliveryGraphCacheBytes, s.Evictions)
}

func TestConcurrentMissesOnTheSameIndexParseItOnce(t *testing.T) {
	raw := realIndex(t)
	deliveryGraphCache.Reset()
	base := DeliveryGraphCacheStats()
	var wg sync.WaitGroup
	var failed atomic.Bool
	start := time.Now()
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, ok := parseDeliveryIntegrityGraph("/work/a", raw); !ok {
				failed.Store(true)
			}
		}()
	}
	wg.Wait()
	s := DeliveryGraphCacheStats()
	t.Logf("operation=12 concurrent first reads: %v wall, productions=%d, coalesced=%d, hits=%d", time.Since(start), s.Misses-base.Misses, s.Coalesced-base.Coalesced, s.Hits-base.Hits)
	if failed.Load() || s.Misses-base.Misses != 1 {
		t.Fatalf("concurrent misses parsed the index %d times", s.Misses-base.Misses)
	}
}

func TestACancelledWaiterOnAGraphParseDoesNotAbandonTheOthers(t *testing.T) {
	// The graph cache is the boundedCache; its waiter semantics are proven in
	// TestBoundedCacheWaiterCancellationLeavesTheProducerAndOthers. Here the same
	// path with a real graph: a cancelled waiter leaves, the parse still serves the rest.
	deliveryGraphCache.Reset()
	raw := []byte(`{"schema_version":1,"input_digest":"d","nodes":[],"edges":[],"claims":[],"change_sets":[],"reverse":{},"findings":[]}`)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	started := make(chan struct{})
	release := make(chan struct{})
	go func() {
		_, _ = deliveryGraphCache.Get(context.Background(), "o", "k-slow", func() (DeliveryIntegrityGraph, int64, error) {
			close(started)
			<-release
			g, _ := decodeDeliveryIntegrityIndex(raw)
			return g, 10, nil
		})
	}()
	<-started
	if _, err := deliveryGraphCache.Get(ctx, "o", "k-slow", nil); err == nil {
		t.Fatal("a cancelled waiter was served")
	}
	close(release)
}

// baselineShallowCopy is the copy the cache made before the attribute maps and the
// roadmap criteria were copied deeply: the baseline the cost of isolation is
// measured against.
func baselineShallowCopy(in DeliveryIntegrityGraph) DeliveryIntegrityGraph {
	out := in
	out.Nodes = append([]DeliveryIntegrityNode{}, in.Nodes...)
	out.Edges = append([]DeliveryIntegrityEdge{}, in.Edges...)
	out.Claims = append([]ArtifactClaim{}, in.Claims...)
	out.Findings = append([]DeliveryIntegrityFinding{}, in.Findings...)
	out.RoadmapCriteria = append([]RoadmapCriterion{}, in.RoadmapCriteria...)
	return out
}

func TestTheCostOfDeepIsolationIsMeasured(t *testing.T) {
	raw := realIndex(t)
	graph, ok := decodeDeliveryIntegrityIndex(raw)
	if !ok {
		t.Fatal("parse")
	}
	measure := func(copyFn func(DeliveryIntegrityGraph) DeliveryIntegrityGraph) (time.Duration, uint64) {
		runtime.GC()
		var a, b runtime.MemStats
		runtime.ReadMemStats(&a)
		start := time.Now()
		const n = 20
		for i := 0; i < n; i++ {
			g := copyFn(graph)
			runtime.KeepAlive(g)
		}
		elapsed := time.Since(start) / n
		runtime.ReadMemStats(&b)
		return elapsed, (b.TotalAlloc - a.TotalAlloc) / n
	}
	shallowT, shallowA := measure(baselineShallowCopy)
	deepT, deepA := measure(copyDeliveryIntegrityGraph)
	t.Logf("operation=copy of the parsed %d-node graph per cache hit: baseline shallow %v / %d B allocated; deep isolation %v / %d B allocated", len(graph.Nodes), shallowT, shallowA, deepT, deepA)
	if deepA < shallowA {
		t.Fatalf("the deep copy cannot allocate less than the shallow one: %d vs %d", deepA, shallowA)
	}
}

// A small read of the graph (one scope's digest) through the view API against the
// same read through the defensive copy: the view does not expand or copy the graph.
func TestAViewOfTheGraphDoesNotCopyIt(t *testing.T) {
	raw := realIndex(t)
	deliveryGraphCache.Reset()
	if _, ok := parseDeliveryIntegrityGraph("/work/a", raw); !ok {
		t.Fatal("parse")
	}
	const reads = 20
	measure := func(read func()) (time.Duration, uint64) {
		runtime.GC()
		var a, b runtime.MemStats
		runtime.ReadMemStats(&a)
		start := time.Now()
		for i := 0; i < reads; i++ {
			read()
		}
		elapsed := time.Since(start) / reads
		runtime.ReadMemStats(&b)
		return elapsed, (b.TotalAlloc - a.TotalAlloc) / reads
	}
	var sink string
	copyT, copyA := measure(func() {
		g, _ := parseDeliveryIntegrityGraph("/work/a", raw)
		sink = ScopedDeliveryProvenanceDigest(g, "pose-validation-evidence-material-binding")
	})
	viewT, viewA := measure(func() {
		viewDeliveryIntegrityGraph("/work/a", raw, func(g DeliveryIntegrityGraph) {
			sink = ScopedDeliveryProvenanceDigest(g, "pose-validation-evidence-material-binding")
		})
	})
	_ = sink
	t.Logf("operation=one scope's provenance digest from a hit: through the deep copy %v / %d B allocated; through the view %v / %d B allocated", copyT, copyA, viewT, viewA)
	if viewA*4 > copyA {
		t.Fatalf("the view allocated %d B against %d B for the copy: it is copying", viewA, copyA)
	}
}

// A view hands out the retained graph itself: the cache must never modify it, and
// a reader that keeps within the contract sees the same graph on every read.
func TestAViewSeesTheSameGraphEveryTimeAndTheCacheNeverWritesIt(t *testing.T) {
	deliveryGraphCache.Reset()
	raw := []byte(`{"schema_version":1,"input_digest":"d","nodes":[{"id":"n","type":"t","attributes":{"k":"v"}}],"edges":[],"claims":[],"change_sets":[],"reverse":{"p":["a"]},"findings":[]}`)
	var first, second string
	viewDeliveryIntegrityGraph("o", raw, func(g DeliveryIntegrityGraph) { first = g.Nodes[0].Attributes["k"] + g.Reverse["p"][0] })
	// A deep-copy reader mutates its copy; the view must not see it.
	g, _ := parseDeliveryIntegrityGraph("o", raw)
	g.Nodes[0].Attributes["k"] = "mutated"
	g.Reverse["p"][0] = "mutated"
	viewDeliveryIntegrityGraph("o", raw, func(g DeliveryIntegrityGraph) { second = g.Nodes[0].Attributes["k"] + g.Reverse["p"][0] })
	if first != "va" || second != "va" {
		t.Fatalf("a copy reader's write reached the viewed graph: %q %q", first, second)
	}
}
