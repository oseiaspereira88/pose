package pose

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// Lifetime and overload of the caches (spec pose-bounded-isolated-graph-caches R4,
// R5): everything the cache keeps alive is charged for as long as it is alive, the
// queue of keys is bounded, a cancelled waiter gives its place back, and overload is
// the same refusal on every consumer.

func TestAQueueOfDistinctKeysIsBounded(t *testing.T) {
	c := intCache(1<<30, 0, 0)
	c.maxProducers, c.maxInflightKeys, c.maxOwnerInflightKeys = 1, 4, 3
	hold := make(chan struct{})
	started := make(chan struct{})
	var once sync.Once
	var busy, ran atomic.Int64
	var wg sync.WaitGroup
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, err := c.Get(context.Background(), "o", string(rune('a'+i)), 0, func() ([]int, int64, error) {
				once.Do(func() { close(started) })
				ran.Add(1)
				<-hold
				return []int{i}, 1, nil
			})
			if errors.Is(err, ErrCacheBusy) {
				busy.Add(1)
			}
		}(i)
	}
	<-started
	time.Sleep(80 * time.Millisecond)
	c.mu.Lock()
	queued := len(c.inflight)
	c.mu.Unlock()
	close(hold)
	wg.Wait()
	if queued > 3 {
		t.Fatalf("%d keys were queued for one owner against a limit of 3", queued)
	}
	if busy.Load() != 9 {
		t.Fatalf("%d of 12 callers were refused, want 9", busy.Load())
	}
}

func TestACancelledWaiterGivesItsPlaceBack(t *testing.T) {
	c := intCache(1<<30, 0, 0)
	c.maxWaiters = 2
	release := make(chan struct{})
	started := make(chan struct{})
	go func() {
		_, _ = c.Get(context.Background(), "o", "k", 0, func() ([]int, int64, error) {
			close(started)
			<-release
			return []int{1}, 1, nil
		})
	}()
	<-started
	for i := 0; i < 20; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
		_, err := c.Get(ctx, "o", "k", 0, produceInts(1, 2))
		cancel()
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("waiter %d: %v (cancelled callers must not exhaust the waiter capacity)", i, err)
		}
	}
	close(release)
}

func TestTheReservationIsHeldUntilTheResultIsPublished(t *testing.T) {
	gate := make(chan struct{})
	c := newBoundedCache(1<<30, 0, 0, func(v []int) []int {
		<-gate // every copy waits: a finished production is still being published
		return append([]int(nil), v...)
	})
	c.maxInflightBytes, c.maxOwnerInflightBytes, c.maxProducers, c.maxInflightKeys, c.maxOwnerInflightKeys = 100, 100, 8, 1000, 1000
	var wg sync.WaitGroup
	var started atomic.Int64
	for i := 0; i < 6; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, _ = c.Get(context.Background(), "o", string(rune('a'+i)), 40, func() ([]int, int64, error) {
				started.Add(1)
				return []int{i}, 1, nil
			})
		}(i)
	}
	time.Sleep(120 * time.Millisecond)
	if n := started.Load(); n > 2 {
		close(gate)
		wg.Wait()
		t.Fatalf("%d productions ran while the earlier results were still being copied: the reservation was released before publication", n)
	}
	s := c.Stats()
	close(gate)
	wg.Wait()
	if s.PeakInflightBytes > 100 {
		t.Fatalf("reserved bytes peaked at %d against 100", s.PeakInflightBytes)
	}
}

func TestAViewPinsTheEntryAndAnEvictedEntryStaysChargedUntilItLetsGo(t *testing.T) {
	c := intCache(100, 0, 0)
	if _, err := c.Get(context.Background(), "o", "a", 0, produceInts(60, 1)); err != nil {
		t.Fatal(err)
	}
	inView := make(chan struct{})
	leave := make(chan struct{})
	done := make(chan struct{})
	go func() {
		_ = c.View(context.Background(), "o", "a", 0, produceInts(60, 1), func([]int) {
			close(inView)
			<-leave
		})
		close(done)
	}()
	<-inView
	// "b" needs the room "a" holds, and "a" is pinned: b cannot be retained without
	// exceeding the budget while the reader still holds a.
	if _, err := c.Get(context.Background(), "o", "b", 0, produceInts(60, 2)); err != nil {
		t.Fatal(err)
	}
	s := c.Stats()
	if s.Retained > 100 {
		t.Fatalf("the budget was exceeded while a reader held an entry: %+v", s)
	}
	close(leave)
	<-done
	// Once the reader lets go the room is free again.
	if _, err := c.Get(context.Background(), "o", "c", 0, produceInts(60, 3)); err != nil {
		t.Fatal(err)
	}
	if s := c.Stats(); s.Retained > 100 {
		t.Fatalf("after the reader left: %+v", s)
	}
}

func TestOverloadIsTheSameRefusalOnEveryConsumer(t *testing.T) {
	// The graph cache: every waiter slot and production slot taken.
	deliveryGraphCache.Reset()
	deliveryGraphCache.mu.Lock()
	oldK, oldO, oldW := deliveryGraphCache.maxInflightKeys, deliveryGraphCache.maxOwnerInflightKeys, deliveryGraphCache.maxWaiters
	deliveryGraphCache.maxInflightKeys, deliveryGraphCache.maxOwnerInflightKeys, deliveryGraphCache.maxWaiters = 1, 1, 0
	deliveryGraphCache.mu.Unlock()
	defer func() {
		deliveryGraphCache.mu.Lock()
		deliveryGraphCache.maxInflightKeys, deliveryGraphCache.maxOwnerInflightKeys, deliveryGraphCache.maxWaiters = oldK, oldO, oldW
		deliveryGraphCache.mu.Unlock()
	}()
	hold := make(chan struct{})
	started := make(chan struct{})
	go func() {
		_, _ = deliveryGraphCache.Get(context.Background(), "/w", "blocker", 0, func() (DeliveryIntegrityGraph, int64, error) {
			close(started)
			<-hold
			return DeliveryIntegrityGraph{}, 1, nil
		})
	}()
	<-started
	raw := []byte(`{"schema_version":1,"input_digest":"d","nodes":[],"edges":[],"claims":[],"change_sets":[],"reverse":{},"findings":[]}`)
	if _, err := parseDeliveryIntegrityGraphErr("/w", raw); !errors.Is(err, ErrCacheBusy) {
		t.Fatalf("the parse did not refuse under overload: %v", err)
	}
	if err := viewDeliveryIntegrityGraphErr("/w", raw, func(DeliveryIntegrityGraph) {}); !errors.Is(err, ErrCacheBusy) {
		t.Fatalf("the view did not refuse under overload: %v", err)
	}
	if err := deliveryIndexError(ErrCacheBusy); !errors.Is(err, ErrCacheBusy) || !strings.Contains(err.Error(), "busy") {
		t.Fatalf("the graph consumers must report an overload as one: %v", err)
	}
	close(hold)
}

func TestTheTestCatalogDoesNotKeepTheFilesItReadItsNamesFrom(t *testing.T) {
	big := "package x\n\nfunc TestOnlyName(t *testing.T) {}\n\n// " + strings.Repeat("filler ", 299_000) + "\n"
	root := gitFixture(t, map[string]string{
		"go.mod":                               "module x\n",
		"a_test.go":                            big,
		".pose/indexes/validation-matrix.json": `{"moduleOverrides":{}}`,
	})
	if len(big) < 2<<20-20_000 || len(big) > 2<<20 {
		t.Skipf("the fixture is %d bytes", len(big))
	}
	before := heapInUse()
	var catalogs []*TestCatalog
	for i := 0; i < 5; i++ {
		catalogs = append(catalogs, buildTestCatalog(root))
	}
	after := heapInUse()
	runtime.KeepAlive(catalogs)
	retained := int64(after) - int64(before)
	t.Logf("5 catalogs built from a %d B test file hold %d B", len(big), retained)
	if retained > 1<<20 {
		t.Fatalf("5 catalogs hold %d B: a test name keeps its whole file alive", retained)
	}
	if !catalogs[0].names["TestOnlyName"] {
		t.Fatal("the name was lost")
	}
}

func TestASharedResultStaysChargedUntilItsLastWaiterHasCopiedIt(t *testing.T) {
	gate := make(chan struct{})
	var blockCopies atomic.Bool
	c := newBoundedCache(1<<30, 0, 0, func(v []int) []int {
		if blockCopies.Load() {
			<-gate
		}
		return append([]int(nil), v...)
	})
	release := make(chan struct{})
	started := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		_, _ = c.Get(context.Background(), "o", "k", 50, func() ([]int, int64, error) {
			close(started)
			<-release
			return []int{1}, Unretainable, nil // shared with the waiters, not retained
		})
	}()
	<-started
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = c.Get(context.Background(), "o", "k", 50, produceInts(1, 9))
		}()
	}
	time.Sleep(50 * time.Millisecond)
	blockCopies.Store(true) // the waiters' copies of the shared result will block
	close(release)
	time.Sleep(80 * time.Millisecond)
	c.mu.Lock()
	charged := c.inflightBytes
	c.mu.Unlock()
	if charged != 50 {
		t.Fatalf("the shared result is charged %d while its waiters still copy it, want 50", charged)
	}
	close(gate)
	wg.Wait()
	c.mu.Lock()
	charged, producers := c.inflightBytes, c.producers
	c.mu.Unlock()
	if charged != 0 || producers != 0 {
		t.Fatalf("after the last waiter: charged %d, producers %d", charged, producers)
	}
}

func TestAnOverloadedCatalogCacheIsReportedNotTakenForNoCatalog(t *testing.T) {
	testCatalogCache.Reset()
	testCatalogCache.mu.Lock()
	old := testCatalogCache.maxInflightKeys
	testCatalogCache.maxInflightKeys = 0
	testCatalogCache.mu.Unlock()
	oldWait := testCatalogBusyWait
	testCatalogBusyWait = 120 * time.Millisecond
	defer func() {
		testCatalogCache.mu.Lock()
		testCatalogCache.maxInflightKeys = old
		testCatalogCache.mu.Unlock()
		testCatalogBusyWait = oldWait
	}()
	catalog, err := LoadTestCatalogChecked(t.TempDir())
	if catalog != nil || !errors.Is(err, ErrCacheBusy) {
		t.Fatalf("an overloaded cache must be an error the gate can report: %v %v", catalog, err)
	}
}

func TestAPublicationRacingWithCancelledAndLateWaitersKeepsTheCountersExact(t *testing.T) {
	for round := 0; round < 200; round++ {
		c := intCache(1<<30, 0, 0)
		c.maxWaiters = 64
		release := make(chan struct{})
		started := make(chan struct{})
		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = c.Get(context.Background(), "o", "k", 10, func() ([]int, int64, error) {
				close(started)
				<-release
				return []int{1}, Unretainable, nil
			})
		}()
		<-started
		for i := 0; i < 8; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				ctx, cancel := context.WithCancel(context.Background())
				if i%2 == 0 {
					go func() { time.Sleep(time.Duration(i) * 100 * time.Microsecond); cancel() }()
				}
				_, _ = c.Get(ctx, "o", "k", 10, produceInts(1, 2))
				cancel()
			}(i)
		}
		time.Sleep(time.Millisecond)
		close(release)
		wg.Wait()
		c.mu.Lock()
		p, b := c.producers, c.inflightBytes
		c.mu.Unlock()
		if p != 0 || b != 0 {
			t.Fatalf("round %d: producers=%d inflightBytes=%d after everything finished", round, p, b)
		}
	}
}

func TestACommitWithMorePathsThanTheReaderAcceptsIsUnknownNotClean(t *testing.T) {
	var raw bytes.Buffer
	for i := 0; i <= commitPathsMaxPaths; i++ {
		fmt.Fprintf(&raw, "p%d\x00", i)
	}
	if _, err := parseCommitPaths(raw.Bytes()); !errors.Is(err, errCommitPathsTooLarge) {
		t.Fatalf("a dense commit must be refused at the path ceiling: %v", err)
	}
	paths, err := parseCommitPaths([]byte("a\x00b/c\x00\x00d"))
	if err != nil || len(paths) != 4-1 || !paths["b/c"] || !paths["d"] {
		t.Fatalf("paths=%v err=%v", paths, err)
	}
	root := gitFixture(t, map[string]string{"a.txt": "a"})
	if _, ok := commitChangedPathsGovernedIn(NativeFiles(), root, "test", "0000000000000000000000000000000000000000"); ok {
		t.Fatal("an unknown commit must read as unknown")
	}
	head, _ := exec.Command("git", "-C", root, "rev-parse", "HEAD").Output()
	got, ok := commitChangedPathsGovernedIn(NativeFiles(), root, "test", strings.TrimSpace(string(head)))
	if !ok || !got["a.txt"] {
		t.Fatalf("the governed reader lost the commit's paths: %v %v", got, ok)
	}
}

func TestManySmallEntriesAreChargedTheirMetadataToo(t *testing.T) {
	c := newBoundedCache(100_000, 0, 0, func(v []int) []int { return append([]int(nil), v...) })
	for i := 0; i < 5000; i++ {
		c.Store("o", "key-"+strings.Repeat("k", 40)+string(rune(i+0x100)), []int{i}, 1)
	}
	s := c.Stats()
	if s.Retained > 100_000 {
		t.Fatalf("retained %d against a budget of 100000", s.Retained)
	}
	if s.Entries > 100_000/defaultEntryOverhead {
		t.Fatalf("%d entries of 1 payload byte fit in 100000: their metadata was not charged", s.Entries)
	}
}

func TestAPaddedTitleDoesNotKeepItsPaddingAlive(t *testing.T) {
	catalog := &TestCatalog{names: map[string]bool{}, slugs: map[string]bool{}, files: map[string]bool{}}
	runtime.GC()
	before := heapInUse()
	for i := 0; i < 8; i++ {
		padded := strings.Repeat(" ", 1<<20) + "title-" + strings.Repeat("x", i+1) + strings.Repeat(" ", 1<<20)
		catalog.add(padded)
	}
	runtime.GC()
	after := heapInUse()
	if held := int64(after) - int64(before); held > 1<<20 {
		t.Fatalf("8 padded titles hold %d B: the trimmed name keeps its padding", held)
	}
	runtime.KeepAlive(catalog)
}

func TestTheOwnersBytesAreChargedToEachEntry(t *testing.T) {
	c := newBoundedCache(100_000, 0, 0, func(v []int) []int { return append([]int(nil), v...) })
	owner := strings.Repeat("r", 4000)
	for i := 0; i < 100; i++ {
		c.Store(owner, "k"+strings.Repeat("k", 10)+string(rune(i+0x100)), []int{i}, 1)
	}
	if s := c.Stats(); s.Retained > 100_000 || s.Entries > 100_000/4000 {
		t.Fatalf("entries under a 4000-byte owner were not charged for it: %+v", s)
	}
}
