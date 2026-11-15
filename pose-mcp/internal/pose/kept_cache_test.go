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
