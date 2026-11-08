package pose

import (
	"container/list"
	"context"
	"sync"
	"time"
)

// boundedCache is the one cache the rebuildable derived data of this package goes
// through (spec pose-bounded-isolated-graph-caches). It is bounded in bytes, not in
// entries: each value reports its retained size, the cache evicts least recently
// used entries to stay under a global budget and under a budget per owner (a
// worktree or root), and a value larger than its budget is served but not retained.
// A miss is produced once per key however many callers ask at the same time; the
// production runs outside the lock, so parsing one key never delays another.
// Failures are not remembered: the next call produces again. A waiter whose context
// ends leaves without disturbing the producer or the other waiters.
//
// The cache holds rebuildable data only. It never remembers a verdict, a
// satisfaction or an authority: those are recomputed from current inputs.
type boundedCache[V any] struct {
	mu            sync.Mutex
	maxBytes      int64
	maxOwnerBytes int64
	ttl           time.Duration
	now           func() time.Time
	clone         func(V) V

	entries  map[string]*cacheEntry[V]
	lru      *list.List
	used     int64
	owners   map[string]int64
	inflight map[string]*cacheFlight[V]
	stats    CacheStats
}

type cacheEntry[V any] struct {
	key, owner string
	value      V
	size       int64
	at         time.Time
	el         *list.Element
}

type cacheFlight[V any] struct {
	done  chan struct{}
	value V
	err   error
}

// CacheStats counts what a cache did, for measurement (spec R7).
type CacheStats struct {
	Hits      int64 // served from a retained entry
	Coalesced int64 // served by a production another caller was already running
	Misses    int64 // productions run
	Evictions int64
	Oversize  int64 // produced values too large to retain
	Expired   int64
	Failures  int64 // productions that failed (never retained)
	Entries   int
	Retained  int64 // bytes retained now
}

func newBoundedCache[V any](maxBytes, maxOwnerBytes int64, ttl time.Duration, clone func(V) V) *boundedCache[V] {
	return &boundedCache[V]{
		maxBytes: maxBytes, maxOwnerBytes: maxOwnerBytes, ttl: ttl, now: time.Now, clone: clone,
		entries: map[string]*cacheEntry[V]{}, lru: list.New(), owners: map[string]int64{}, inflight: map[string]*cacheFlight[V]{},
	}
}

// Get returns the value for key, producing it with produce when none is retained.
// produce returns the value and its retained size in bytes. Every value handed out
// is a clone: the retained one is never shared with a caller.
func (c *boundedCache[V]) Get(ctx context.Context, owner, key string, produce func() (V, int64, error)) (V, error) {
	return c.get(ctx, owner, key, produce, true)
}

// View is Get for a reader that promises neither to modify nor to keep the value:
// it receives the retained value itself, without the clone, which is what makes a
// small read of a large graph cheap (spec pose-bounded-isolated-graph-caches R1).
// The cache never modifies a retained value, so a view stays valid after its entry
// is evicted or replaced.
func (c *boundedCache[V]) View(ctx context.Context, owner, key string, produce func() (V, int64, error), fn func(V)) error {
	v, err := c.get(ctx, owner, key, produce, false)
	if err != nil {
		return err
	}
	fn(v)
	return nil
}

func (c *boundedCache[V]) get(ctx context.Context, owner, key string, produce func() (V, int64, error), clone bool) (V, error) {
	var zero V
	out := func(v V) V {
		if clone {
			return c.clone(v)
		}
		return v
	}
	c.mu.Lock()
	if entry, ok := c.entries[key]; ok {
		if c.ttl > 0 && c.now().Sub(entry.at) > c.ttl {
			c.removeLocked(entry)
			c.stats.Expired++
		} else {
			c.lru.MoveToFront(entry.el)
			c.stats.Hits++
			v := out(entry.value)
			c.mu.Unlock()
			return v, nil
		}
	}
	if flight, ok := c.inflight[key]; ok {
		c.mu.Unlock()
		select {
		case <-flight.done:
		case <-ctx.Done():
			return zero, ctx.Err()
		}
		if flight.err != nil {
			return zero, flight.err
		}
		c.mu.Lock()
		c.stats.Coalesced++
		v := out(flight.value)
		c.mu.Unlock()
		return v, nil
	}
	flight := &cacheFlight[V]{done: make(chan struct{})}
	c.inflight[key] = flight
	c.stats.Misses++
	c.mu.Unlock()

	var (
		value V
		size  int64
		err   error
	)
	func() {
		// A panicking producer must still release its waiters and the key.
		defer func() {
			if r := recover(); r != nil {
				c.mu.Lock()
				delete(c.inflight, key)
				c.stats.Failures++
				flight.err = errCacheProducerPanicked
				c.mu.Unlock()
				close(flight.done)
				panic(r)
			}
		}()
		value, size, err = produce()
	}()

	c.mu.Lock()
	delete(c.inflight, key)
	if err != nil {
		c.stats.Failures++
		flight.err = err
		c.mu.Unlock()
		close(flight.done)
		return zero, err
	}
	flight.value = c.clone(value)
	c.storeLocked(owner, key, flight.value, size)
	c.mu.Unlock()
	close(flight.done)
	return value, nil
}

// Lookup returns a clone of the retained value for key, if any.
func (c *boundedCache[V]) Lookup(key string) (V, bool) {
	var zero V
	c.mu.Lock()
	defer c.mu.Unlock()
	entry, ok := c.entries[key]
	if !ok {
		c.stats.Misses++
		return zero, false
	}
	if c.ttl > 0 && c.now().Sub(entry.at) > c.ttl {
		c.removeLocked(entry)
		c.stats.Expired++
		c.stats.Misses++
		return zero, false
	}
	c.lru.MoveToFront(entry.el)
	c.stats.Hits++
	return c.clone(entry.value), true
}

// Store retains a clone of value for key under the same budgets as Get.
func (c *boundedCache[V]) Store(owner, key string, value V, size int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.storeLocked(owner, key, c.clone(value), size)
}

func (c *boundedCache[V]) storeLocked(owner, key string, value V, size int64) {
	if size < 0 {
		size = 0
	}
	if size > c.maxBytes || (c.maxOwnerBytes > 0 && size > c.maxOwnerBytes) {
		c.stats.Oversize++
		return
	}
	if old, ok := c.entries[key]; ok {
		c.removeLocked(old)
	}
	for c.maxOwnerBytes > 0 && c.owners[owner]+size > c.maxOwnerBytes {
		if !c.evictLocked(owner) {
			break
		}
	}
	for c.used+size > c.maxBytes {
		if !c.evictLocked("") {
			break
		}
	}
	entry := &cacheEntry[V]{key: key, owner: owner, value: value, size: size, at: c.now()}
	entry.el = c.lru.PushFront(entry)
	c.entries[key] = entry
	c.used += size
	c.owners[owner] += size
}

// evictLocked removes the least recently used entry, of one owner when owner is
// set. It reports whether it removed one.
func (c *boundedCache[V]) evictLocked(owner string) bool {
	for el := c.lru.Back(); el != nil; el = el.Prev() {
		entry := el.Value.(*cacheEntry[V])
		if owner == "" || entry.owner == owner {
			c.removeLocked(entry)
			c.stats.Evictions++
			return true
		}
	}
	return false
}

func (c *boundedCache[V]) removeLocked(entry *cacheEntry[V]) {
	c.lru.Remove(entry.el)
	delete(c.entries, entry.key)
	c.used -= entry.size
	c.owners[entry.owner] -= entry.size
	if c.owners[entry.owner] <= 0 {
		delete(c.owners, entry.owner)
	}
}

// Stats returns the counters and what is retained now.
func (c *boundedCache[V]) Stats() CacheStats {
	c.mu.Lock()
	defer c.mu.Unlock()
	s := c.stats
	s.Entries, s.Retained = len(c.entries), c.used
	return s
}

// Reset drops every retained entry (tests, and callers that know the world changed).
func (c *boundedCache[V]) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries, c.lru, c.used, c.owners = map[string]*cacheEntry[V]{}, list.New(), 0, map[string]int64{}
}

var errCacheProducerPanicked = &cacheError{"the cache producer panicked"}

type cacheError struct{ msg string }

func (e *cacheError) Error() string { return e.msg }
