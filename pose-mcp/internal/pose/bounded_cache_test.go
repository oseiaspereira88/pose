package pose

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func intCache(max, ownerMax int64, ttl time.Duration) *boundedCache[[]int] {
	return newBoundedCache(max, ownerMax, ttl, func(v []int) []int { return append([]int(nil), v...) })
}

func produceInts(size int64, values ...int) func() ([]int, int64, error) {
	return func() ([]int, int64, error) { return values, size, nil }
}

func TestBoundedCacheEvictsByBytesNotEntries(t *testing.T) {
	c := intCache(100, 0, 0)
	for i, key := range []string{"a", "b", "c", "d"} {
		if _, err := c.Get(context.Background(), "o", key, 0, produceInts(40, i)); err != nil {
			t.Fatal(err)
		}
	}
	s := c.Stats()
	if s.Retained > 100 || s.Entries != 2 || s.Evictions != 2 {
		t.Fatalf("100 bytes cannot hold four 40-byte entries: %+v", s)
	}
	// The least recently used went first: a and b are gone, c and d stay.
	if _, err := c.Get(context.Background(), "o", "d", 0, produceInts(40, 9)); err != nil {
		t.Fatal(err)
	}
	if c.Stats().Hits != 1 {
		t.Fatalf("the most recent entry was evicted: %+v", c.Stats())
	}
}

func TestBoundedCacheHoldsAnOwnerToItsBudget(t *testing.T) {
	c := intCache(1000, 100, 0)
	for _, owner := range []string{"a", "b"} {
		for i := 0; i < 5; i++ {
			if _, err := c.Get(context.Background(), owner, owner+string(rune('0'+i)), 0, produceInts(40, i)); err != nil {
				t.Fatal(err)
			}
		}
	}
	c.mu.Lock()
	a, b := c.owners["a"], c.owners["b"]
	c.mu.Unlock()
	if a > 100 || b > 100 || a == 0 || b == 0 {
		t.Fatalf("an owner exceeded its budget or starved the other: a=%d b=%d", a, b)
	}
}

func TestBoundedCacheDoesNotRetainWhatExceedsTheBudget(t *testing.T) {
	c := intCache(100, 0, 0)
	v, err := c.Get(context.Background(), "o", "big", 0, produceInts(500, 1, 2, 3))
	if err != nil || len(v) != 3 {
		t.Fatalf("an oversize value must still be served: %v %v", v, err)
	}
	if s := c.Stats(); s.Entries != 0 || s.Retained != 0 || s.Oversize != 1 {
		t.Fatalf("an oversize value was retained: %+v", s)
	}
}

func TestBoundedCacheProducesOncePerKeyUnderConcurrentMisses(t *testing.T) {
	c := intCache(1000, 0, 0)
	var produced atomic.Int64
	release := make(chan struct{})
	var wg sync.WaitGroup
	results := make([][]int, 16)
	for i := range results {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			v, err := c.Get(context.Background(), "o", "k", 0, func() ([]int, int64, error) {
				produced.Add(1)
				<-release
				return []int{1, 2, 3}, 10, nil
			})
			if err != nil {
				t.Error(err)
			}
			results[i] = v
		}(i)
	}
	time.Sleep(50 * time.Millisecond)
	close(release)
	wg.Wait()
	if produced.Load() != 1 {
		t.Fatalf("concurrent misses produced %d times", produced.Load())
	}
	// Every caller got its own copy.
	results[0][0] = 99
	for _, r := range results[1:] {
		if r[0] != 1 {
			t.Fatal("two callers share one slice")
		}
	}
	if s := c.Stats(); s.Misses != 1 {
		t.Fatalf("misses = %d, want 1: %+v", s.Misses, s)
	}
}

func TestBoundedCacheDoesNotSerializeIndependentKeys(t *testing.T) {
	c := intCache(1000, 0, 0)
	hold := make(chan struct{})
	started := make(chan struct{})
	go func() {
		_, _ = c.Get(context.Background(), "o", "slow", 0, func() ([]int, int64, error) {
			close(started)
			<-hold
			return []int{1}, 1, nil
		})
	}()
	<-started
	done := make(chan struct{})
	go func() {
		_, _ = c.Get(context.Background(), "o", "fast", 0, produceInts(1, 2))
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("a slow production of one key blocked another key")
	}
	close(hold)
}

func TestBoundedCacheWaiterCancellationLeavesTheProducerAndOthers(t *testing.T) {
	c := intCache(1000, 0, 0)
	release := make(chan struct{})
	started := make(chan struct{})
	var once sync.Once
	produce := func() ([]int, int64, error) {
		once.Do(func() { close(started) })
		<-release
		return []int{7}, 1, nil
	}
	producerDone := make(chan []int, 1)
	go func() {
		v, _ := c.Get(context.Background(), "o", "k", 0, produce)
		producerDone <- v
	}()
	<-started
	ctx, cancel := context.WithCancel(context.Background())
	cancelled := make(chan error, 1)
	go func() {
		_, err := c.Get(ctx, "o", "k", 0, produce)
		cancelled <- err
	}()
	patient := make(chan []int, 1)
	go func() {
		v, _ := c.Get(context.Background(), "o", "k", 0, produce)
		patient <- v
	}()
	time.Sleep(30 * time.Millisecond)
	cancel()
	if err := <-cancelled; !errors.Is(err, context.Canceled) {
		t.Fatalf("the cancelled waiter got %v", err)
	}
	close(release)
	if v := <-producerDone; len(v) != 1 || v[0] != 7 {
		t.Fatalf("the producer was disturbed: %v", v)
	}
	if v := <-patient; len(v) != 1 || v[0] != 7 {
		t.Fatalf("another waiter was abandoned: %v", v)
	}
}

func TestBoundedCacheDoesNotRememberAFailure(t *testing.T) {
	c := intCache(1000, 0, 0)
	boom := errors.New("boom")
	if _, err := c.Get(context.Background(), "o", "k", 0, func() ([]int, int64, error) { return nil, 0, boom }); !errors.Is(err, boom) {
		t.Fatalf("got %v", err)
	}
	v, err := c.Get(context.Background(), "o", "k", 0, produceInts(1, 5))
	if err != nil || v[0] != 5 {
		t.Fatalf("a failure was memorized as the answer: %v %v", v, err)
	}
	if s := c.Stats(); s.Failures != 1 || s.Misses != 2 {
		t.Fatalf("a failure must be produced again: %+v", s)
	}
}

func TestBoundedCacheExpiresByTTL(t *testing.T) {
	c := intCache(1000, 0, time.Minute)
	now := time.Now()
	c.now = func() time.Time { return now }
	_, _ = c.Get(context.Background(), "o", "k", 0, produceInts(1, 1))
	now = now.Add(2 * time.Minute)
	_, _ = c.Get(context.Background(), "o", "k", 0, produceInts(1, 2))
	if s := c.Stats(); s.Expired != 1 || s.Misses != 2 || s.Hits != 0 {
		t.Fatalf("an expired entry was served: %+v", s)
	}
}

func TestBoundedCacheSurvivesAlternatingOwnersUnderRace(t *testing.T) {
	c := intCache(2000, 600, 0)
	var wg sync.WaitGroup
	for g := 0; g < 8; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for i := 0; i < 300; i++ {
				owner := string(rune('a' + (i+g)%4))
				key := owner + string(rune('0'+i%7))
				v, err := c.Get(context.Background(), owner, key, 0, produceInts(100, i))
				if err != nil {
					t.Error(err)
					return
				}
				v[0] = -1 // a caller writing its copy must reach nobody
			}
		}(g)
	}
	wg.Wait()
	s := c.Stats()
	if s.Retained > 2000 {
		t.Fatalf("the global budget was exceeded: %+v", s)
	}
	c.mu.Lock()
	for owner, used := range c.owners {
		if used > 600 {
			t.Errorf("owner %s retains %d bytes against 600", owner, used)
		}
	}
	for _, e := range c.entries {
		if e.value[0] == -1 {
			t.Error("a caller's write reached a retained entry")
		}
	}
	c.mu.Unlock()
}
