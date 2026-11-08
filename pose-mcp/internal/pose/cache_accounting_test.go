package pose

import (
	"context"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// Accounting of what the caches hold and what they let run (spec
// pose-bounded-isolated-graph-caches R4, R5): variable-sized fields are charged what
// they hold, productions and waiters have budgets, copies do not block other keys,
// and what is not retained is not copied for retention.

func TestTheEstimateChargesEveryVariableSizedField(t *testing.T) {
	big := strings.Repeat("x", 4<<20)
	cases := map[string]DeliveryIntegrityGraph{
		"a roadmap criterion with a large Raw and Refs": {RoadmapCriteria: []RoadmapCriterion{{Refs: []string{big}, Reasons: []string{big}}}},
		"a removal with large maps":                     {Removals: []ArtifactRemoval{{Supersedes: map[string]string{"k": big}, Refusals: map[string]string{"k": big}}}},
		"a change set with material inputs":             {ChangeSets: []ChangeSet{{MaterialInputs: &MaterialInputs{Names: []string{big}, Moved: []string{big}}}}},
		"a node attribute":                              {Nodes: []DeliveryIntegrityNode{{Attributes: map[string]string{"k": big}}}},
	}
	for name, g := range cases {
		if got := estimateGraphBytes(g); got < 4<<20 {
			t.Errorf("%s is charged %d B for at least 4 MiB held", name, got)
		}
	}
	report := DesignDeltaReport{Deltas: []StructuralDelta{{SharedWith: []string{big}}}}
	if got := estimateRetainedBytes(report); got < 4<<20 {
		t.Errorf("a design delta's SharedWith is charged %d B for 4 MiB held", got)
	}
}

func TestProductionsAreAdmittedWithinTheInflightBudget(t *testing.T) {
	c := intCache(1<<30, 0, 0)
	c.maxInflightBytes, c.maxOwnerInflightBytes, c.maxProducers, c.maxInflightKeys, c.maxOwnerInflightKeys = 100, 100, 8, 1000, 1000
	var running, peak atomic.Int64
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			key := string(rune('a' + i))
			_, err := c.Get(context.Background(), "o", key, 40, func() ([]int, int64, error) {
				n := running.Add(1)
				for {
					p := peak.Load()
					if n <= p || peak.CompareAndSwap(p, n) {
						break
					}
				}
				time.Sleep(5 * time.Millisecond)
				running.Add(-1)
				return []int{i}, 1, nil
			})
			if err != nil {
				t.Error(err)
			}
		}(i)
	}
	wg.Wait()
	if peak.Load() > 2 {
		t.Fatalf("%d productions ran at once against an in-flight budget that holds two reservations of 40 in 100", peak.Load())
	}
	if s := c.Stats(); s.PeakInflightBytes > 100 {
		t.Fatalf("reserved bytes peaked at %d against a budget of 100", s.PeakInflightBytes)
	}
}

func TestAReservationLargerThanTheBudgetRunsAloneInsteadOfStarving(t *testing.T) {
	c := intCache(1<<30, 0, 0)
	c.maxInflightBytes = 10
	v, err := c.Get(context.Background(), "o", "huge", 1000, produceInts(1, 7))
	if err != nil || v[0] != 7 {
		t.Fatalf("an oversize reservation starved: %v %v", v, err)
	}
}

func TestTheNumberOfProductionsIsCapped(t *testing.T) {
	c := intCache(1<<30, 0, 0)
	c.maxProducers, c.maxInflightKeys, c.maxOwnerInflightKeys = 3, 1000, 1000
	var running, peak atomic.Int64
	var wg sync.WaitGroup
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, _ = c.Get(context.Background(), "o", string(rune('a'+i)), 0, func() ([]int, int64, error) {
				n := running.Add(1)
				for {
					p := peak.Load()
					if n <= p || peak.CompareAndSwap(p, n) {
						break
					}
				}
				time.Sleep(5 * time.Millisecond)
				running.Add(-1)
				return []int{i}, 1, nil
			})
		}(i)
	}
	wg.Wait()
	if peak.Load() > 3 {
		t.Fatalf("%d productions ran at once against a cap of 3", peak.Load())
	}
}

func TestWaitersPerKeyAreCappedAndTheRefusedCallerIsToldToProduceForItself(t *testing.T) {
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
	var wg sync.WaitGroup
	var busy atomic.Int64
	for i := 0; i < 6; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := c.Get(context.Background(), "o", "k", 0, produceInts(1, 2))
			if err == ErrCacheBusy {
				busy.Add(1)
			}
		}()
	}
	time.Sleep(60 * time.Millisecond)
	close(release)
	wg.Wait()
	if busy.Load() != 4 {
		t.Fatalf("with room for 2 waiters, %d of 6 were refused, want 4", busy.Load())
	}
	if c.Stats().Busy != 4 {
		t.Fatalf("stats = %+v", c.Stats())
	}
}

func TestACopyDoesNotDelayAnotherKey(t *testing.T) {
	var slow atomic.Bool
	gate := make(chan struct{})
	started := make(chan struct{})
	var once sync.Once
	c := newBoundedCache(1<<30, 0, 0, func(v []int) []int {
		if slow.Load() && len(v) == 1 && v[0] == 99 {
			once.Do(func() { close(started) })
			<-gate
		}
		return append([]int(nil), v...)
	})
	if _, err := c.Get(context.Background(), "o", "big", 0, produceInts(1, 99)); err != nil {
		t.Fatal(err)
	}
	slow.Store(true)
	go func() { _, _ = c.Get(context.Background(), "o", "big", 0, produceInts(1, 99)) }() // a hit whose copy blocks
	<-started
	done := make(chan struct{})
	go func() {
		_, _ = c.Get(context.Background(), "o", "other", 0, produceInts(1, 5))
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("a large copy of one key delayed another key")
	}
	close(gate)
}

func TestAValueNotRetainedIsNotCopiedForRetention(t *testing.T) {
	var copies atomic.Int64
	c := newBoundedCache(100, 0, 0, func(v []int) []int { copies.Add(1); return append([]int(nil), v...) })
	if _, err := c.Get(context.Background(), "o", "big", 0, produceInts(500, 1)); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Get(context.Background(), "o", "partial", 0, produceInts(Unretainable, 2)); err != nil {
		t.Fatal(err)
	}
	if copies.Load() != 0 {
		t.Fatalf("%d copies were made of values that are not retained and have no waiter", copies.Load())
	}
	if s := c.Stats(); s.Entries != 0 || s.Oversize != 2 {
		t.Fatalf("stats = %+v", s)
	}
}

func TestADesignDeltaIsAssessedOncePerKeyAndOnlyACompleteOneIsRetained(t *testing.T) {
	designDeltaCache.Reset()
	var runs atomic.Int64
	gate := make(chan struct{})
	assess := func(status string) func() (DesignDeltaReport, int64, error) {
		return func() (DesignDeltaReport, int64, error) {
			runs.Add(1)
			<-gate
			r := DesignDeltaReport{Status: status}
			size := Unretainable
			if status == "observed" {
				size = estimateRetainedBytes(r)
			}
			return r, size, nil
		}
	}
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = designDeltaCache.Get(context.Background(), "/r", "/r\x00complete", 1, assess("observed"))
		}()
	}
	time.Sleep(40 * time.Millisecond)
	close(gate)
	wg.Wait()
	if runs.Load() != 1 {
		t.Fatalf("simultaneous misses assessed %d times", runs.Load())
	}
	// A partial report is served and not kept: the next call recomputes.
	gate2 := make(chan struct{})
	close(gate2)
	gate = gate2
	before := runs.Load()
	for i := 0; i < 2; i++ {
		if _, err := designDeltaCache.Get(context.Background(), "/r", "/r\x00partial", 1, assess("partial")); err != nil {
			t.Fatal(err)
		}
	}
	if runs.Load()-before != 2 {
		t.Fatalf("a partial report was remembered: %d runs for 2 calls", runs.Load()-before)
	}
	// Another root with the same subject digest is another key and another owner.
	if _, ok := designDeltaCache.Lookup("/other\x00complete"); ok {
		t.Fatal("a report was served across roots")
	}
}

func TestTheDesignDeltaBudgetIsPerWorktree(t *testing.T) {
	designDeltaCache.Reset()
	for i := 0; i < 40; i++ {
		designDeltaCache.Store("/a", "a"+string(rune('0'+i%10))+string(rune('a'+i)), DesignDeltaReport{}, 1<<20)
	}
	designDeltaCache.Store("/b", "b-keep", DesignDeltaReport{}, 1<<20)
	for i := 0; i < 40; i++ {
		designDeltaCache.Store("/a", "a2"+string(rune('0'+i%10))+string(rune('a'+i)), DesignDeltaReport{}, 1<<20)
	}
	if _, ok := designDeltaCache.Lookup("b-keep"); !ok {
		t.Fatal("one worktree evicted another's report")
	}
	designDeltaCache.mu.Lock()
	a := designDeltaCache.owners["/a"]
	designDeltaCache.mu.Unlock()
	if a > 16<<20 {
		t.Fatalf("a worktree retains %d bytes against 16 MiB", a)
	}
}
