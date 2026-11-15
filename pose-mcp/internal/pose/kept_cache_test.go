package pose

import (
	"sync"
	"sync/atomic"
	"testing"
)

// Review round 1 (F4): callers that miss the same key together compute it
// once; the others wait for the first.
func TestAKeptValueIsComputedOnceForConcurrentMisses(t *testing.T) {
	cache := newKeptCache[int](10)
	var computed atomic.Int32
	release := make(chan struct{})
	var started sync.WaitGroup
	started.Add(1)
	var wg sync.WaitGroup
	results := make([]int, 8)
	for i := range results {
		wg.Add(1)
		go func() {
			defer wg.Done()
			results[i] = cache.get("k", func() (int, bool) {
				if computed.Add(1) == 1 {
					started.Done()
				}
				<-release
				return 42, true
			})
		}()
	}
	started.Wait()
	close(release)
	wg.Wait()
	if computed.Load() != 1 {
		t.Fatalf("eight concurrent misses computed %d times", computed.Load())
	}
	for i, v := range results {
		if v != 42 {
			t.Fatalf("caller %d got %d", i, v)
		}
	}
}

// F4: a result the computation declines to keep is not shared; the next
// caller computes again, and a later success is kept.
func TestADeclinedKeptValueIsComputedAgain(t *testing.T) {
	cache := newKeptCache[int](10)
	calls := 0
	compute := func() (int, bool) {
		calls++
		return calls, calls >= 2
	}
	if v := cache.get("k", compute); v != 1 {
		t.Fatalf("first %d", v)
	}
	if v := cache.get("k", compute); v != 2 {
		t.Fatalf("a declined result was reused: %d", v)
	}
	if v := cache.get("k", compute); v != 2 || calls != 2 {
		t.Fatalf("a kept result was computed again: %d after %d calls", v, calls)
	}
}

// Review round 1 (F1): an unreadable material is asked again, and a
// shallow checkout, which resolves no HEAD, is never kept.
func TestAnUnreadableMaterialIsAskedAgain(t *testing.T) {
	reads := 0
	answer := MaterialDigestUnreadable
	withMaterialHooks(t, func(string) string { return "0123456789012345678901234567890123456789" }, func(string, string, MaterialInputs) string {
		reads++
		return answer
	})
	sets := materialSets(1)
	if got := withCurrentMaterial("/r", sets)[0].MaterialDigest; got != MaterialDigestUnreadable {
		t.Fatalf("first %q", got)
	}
	answer = "sha256:recovered"
	if got := withCurrentMaterial("/r", sets)[0].MaterialDigest; got != "sha256:recovered" || reads != 2 {
		t.Fatalf("the unreadable answer was kept: %q after %d reads", got, reads)
	}
}

// Review round 2 (N1): callers that waited for a failed read retry through
// the cache: the recovered read is computed once and kept.
func TestARecoveredReadIsComputedOnceAndKept(t *testing.T) {
	cache := newKeptCache[int](10)
	var calls atomic.Int32
	release := make(chan struct{})
	first := make(chan struct{})
	compute := func() (int, bool) {
		n := calls.Add(1)
		if n == 1 {
			close(first)
			<-release
			return 0, false
		}
		return 7, true
	}
	var wg sync.WaitGroup
	results := make([]int, 6)
	wg.Add(1)
	go func() { defer wg.Done(); results[0] = cache.get("k", compute) }()
	<-first
	for i := 1; i < len(results); i++ {
		wg.Add(1)
		go func() { defer wg.Done(); results[i] = cache.get("k", compute) }()
	}
	close(release)
	wg.Wait()
	if calls.Load() != 2 {
		t.Fatalf("a failure and its recovery computed %d times, want 2", calls.Load())
	}
	if v := cache.get("k", compute); v != 7 || calls.Load() != 2 {
		t.Fatalf("the recovered read was not kept: %d after %d calls", v, calls.Load())
	}
}

// N2: crossing the bound while a key is computed keeps that key's entry, so
// a later caller of the same key waits for it instead of computing again.
func TestABoundCrossingKeepsAPendingKey(t *testing.T) {
	cache := newKeptCache[int](1)
	var calls atomic.Int32
	release := make(chan struct{})
	started := make(chan struct{})
	done := make(chan int)
	go func() {
		done <- cache.get("a", func() (int, bool) {
			calls.Add(1)
			close(started)
			<-release
			return 1, true
		})
	}()
	<-started
	cache.get("b", func() (int, bool) { return 2, true })
	second := make(chan int)
	go func() {
		second <- cache.get("a", func() (int, bool) { calls.Add(1); return 3, true })
	}()
	close(release)
	if v := <-done; v != 1 {
		t.Fatalf("first a %d", v)
	}
	if v := <-second; v != 1 || calls.Load() != 1 {
		t.Fatalf("the pending key was computed again across the bound: %d after %d calls", v, calls.Load())
	}
}

// Review round 3 (N3): a burst of distinct keys computed together keeps no
// more completed values than the bound once they all finish.
func TestAConcurrentBurstKeepsNoMoreThanTheBound(t *testing.T) {
	cache := newKeptCache[int](1)
	release := make(chan struct{})
	var started, wg sync.WaitGroup
	const burst = 5
	started.Add(burst)
	for i := 0; i < burst; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			cache.get(string(rune('a'+i)), func() (int, bool) {
				started.Done()
				<-release
				return i, true
			})
		}()
	}
	started.Wait()
	close(release)
	wg.Wait()
	cache.mu.Lock()
	retained, kept := len(cache.entries), cache.kept
	cache.mu.Unlock()
	if retained > 1 || kept > 1 {
		t.Fatalf("%d entries (%d counted) kept past a bound of 1", retained, kept)
	}
}
