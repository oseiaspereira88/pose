package pose

import (
	"container/list"
	"context"
	"errors"
	"runtime"
	"sync"
	"time"
)

// boundedCache is the one cache the rebuildable derived data of this package goes
// through (spec pose-bounded-isolated-graph-caches).
//
// What it bounds, and what it does not:
//
//   - Retained bytes: each value reports its size; the cache evicts least recently
//     used entries under a global budget and a budget per owner (a worktree root). A
//     value over its budget is served and neither retained nor cloned for retention.
//   - In-flight work: a production reserves bytes before it runs, globally and per
//     owner, and waits (or leaves with its context) until they fit; the number of
//     productions and of waiters per key is capped too. A caller that is refused for
//     lack of waiter capacity receives ErrCacheBusy and produces for itself.
//   - Copies in flight: cloning runs outside the lock and takes one of a few copy
//     slots, so independent keys are never delayed by a copy and the copies being
//     made at one moment are few.
//   - Not bounded: the clone a caller keeps after it returns. That memory belongs to
//     the caller; a reader that only looks up fields uses View and keeps nothing.
//
// A missed key is produced once however many callers ask at the same time. Failures
// are not remembered. A waiter whose context ends leaves without disturbing the
// producer or the other waiters. The cache holds rebuildable data only: never a
// verdict, a satisfaction or an authority.
type boundedCache[V any] struct {
	mu            sync.Mutex
	maxBytes      int64
	maxOwnerBytes int64
	ttl           time.Duration
	now           func() time.Time
	clone         func(V) V

	maxInflightBytes      int64
	maxOwnerInflightBytes int64
	maxProducers          int
	maxWaiters            int
	copySlots             chan struct{}

	entries  map[string]*cacheEntry[V]
	lru      *list.List
	used     int64
	owners   map[string]int64
	inflight map[string]*cacheFlight[V]

	producers     int
	inflightBytes int64
	ownerInflight map[string]int64
	admissionWake chan struct{}
	peakProducers int
	peakInflight  int64
	stats         CacheStats
}

type cacheEntry[V any] struct {
	key, owner string
	value      V
	size       int64
	at         time.Time
	el         *list.Element
}

type cacheFlight[V any] struct {
	done    chan struct{}
	value   V
	ok      bool // value is set; false with err == nil means "ask again"
	err     error
	waiters int
}

// CacheStats counts what a cache did, for measurement (spec R7).
type CacheStats struct {
	Hits      int64 // served from a retained entry
	Coalesced int64 // served by a production another caller was already running
	Misses    int64 // productions run
	Evictions int64
	Oversize  int64 // produced values not retained: over a budget, or declared unretainable
	Expired   int64
	Failures  int64 // productions that failed (never retained)
	Busy      int64 // callers refused for lack of waiter capacity
	Entries   int
	Retained  int64 // bytes retained now
	// PeakProducers and PeakInflightBytes are the most productions and reserved bytes
	// that were in flight at once.
	PeakProducers     int
	PeakInflightBytes int64
}

// ErrCacheBusy is returned to a caller refused because too many are already waiting
// on the same key. The caller produces for itself, uncached.
var ErrCacheBusy = errors.New("pose: the cache has no waiter capacity for this key")

// Unretainable is the size a producer returns for a value that must be served and
// not remembered (a partial or unknown result a later call should recompute).
const Unretainable = int64(-1)

func newBoundedCache[V any](maxBytes, maxOwnerBytes int64, ttl time.Duration, clone func(V) V) *boundedCache[V] {
	slots := runtime.GOMAXPROCS(0)
	if slots < 2 {
		slots = 2
	}
	return &boundedCache[V]{
		maxBytes: maxBytes, maxOwnerBytes: maxOwnerBytes, ttl: ttl, now: time.Now, clone: clone,
		maxInflightBytes: maxBytes, maxOwnerInflightBytes: maxOwnerBytes, maxProducers: 8, maxWaiters: 128,
		copySlots: make(chan struct{}, slots),
		entries:   map[string]*cacheEntry[V]{}, lru: list.New(), owners: map[string]int64{},
		inflight: map[string]*cacheFlight[V]{}, ownerInflight: map[string]int64{}, admissionWake: make(chan struct{}),
	}
}

func (c *boundedCache[V]) doClone(v V) V {
	c.copySlots <- struct{}{}
	defer func() { <-c.copySlots }()
	return c.clone(v)
}

// Get returns the value for key, producing it with produce when none is retained.
// reserve is the caller's estimate of the bytes the production will hold while it
// runs. produce returns the value and its retained size (Unretainable to serve it
// without keeping it). Every value handed out is a clone.
func (c *boundedCache[V]) Get(ctx context.Context, owner, key string, reserve int64, produce func() (V, int64, error)) (V, error) {
	return c.get(ctx, owner, key, reserve, produce, true)
}

// View is Get for a reader that promises neither to modify nor to keep the value:
// it receives the retained value itself, without the clone, which is what makes a
// small read of a large graph cheap (spec pose-bounded-isolated-graph-caches R1).
// The cache never modifies a retained value, so a view stays valid after its entry
// is evicted or replaced.
func (c *boundedCache[V]) View(ctx context.Context, owner, key string, reserve int64, produce func() (V, int64, error), fn func(V)) error {
	v, err := c.get(ctx, owner, key, reserve, produce, false)
	if err != nil {
		return err
	}
	fn(v)
	return nil
}

func (c *boundedCache[V]) get(ctx context.Context, owner, key string, reserve int64, produce func() (V, int64, error), clone bool) (V, error) {
	var zero V
	for {
		c.mu.Lock()
		if entry, ok := c.entries[key]; ok {
			if c.ttl > 0 && c.now().Sub(entry.at) > c.ttl {
				c.removeLocked(entry)
				c.stats.Expired++
			} else {
				c.lru.MoveToFront(entry.el)
				c.stats.Hits++
				v := entry.value
				c.mu.Unlock()
				if clone {
					return c.doClone(v), nil
				}
				return v, nil
			}
		}
		if flight, ok := c.inflight[key]; ok {
			if flight.waiters >= c.maxWaiters {
				c.stats.Busy++
				c.mu.Unlock()
				return zero, ErrCacheBusy
			}
			flight.waiters++
			c.mu.Unlock()
			select {
			case <-flight.done:
			case <-ctx.Done():
				return zero, ctx.Err()
			}
			if flight.err != nil {
				return zero, flight.err
			}
			if !flight.ok {
				continue // the producer kept nothing to share: ask again
			}
			c.mu.Lock()
			c.stats.Coalesced++
			v := flight.value
			c.mu.Unlock()
			if clone {
				return c.doClone(v), nil
			}
			return v, nil
		}
		flight := &cacheFlight[V]{done: make(chan struct{})}
		c.inflight[key] = flight
		c.mu.Unlock()

		// Admission: the production waits until its reservation fits.
		if err := c.admit(ctx, owner, reserve); err != nil {
			c.mu.Lock()
			delete(c.inflight, key)
			flight.err = err
			c.mu.Unlock()
			close(flight.done)
			return zero, err
		}
		return c.produce(ctx, owner, key, reserve, produce, flight, clone)
	}
}

// admit reserves reserve bytes for a production, waiting until they fit under the
// global and the per-owner in-flight budgets and a production slot is free. A
// reservation larger than a budget is admitted alone, so nothing starves.
func (c *boundedCache[V]) admit(ctx context.Context, owner string, reserve int64) error {
	if reserve < 0 {
		reserve = 0
	}
	for {
		c.mu.Lock()
		fits := c.producers < c.maxProducers &&
			(c.inflightBytes == 0 || c.inflightBytes+reserve <= c.maxInflightBytes) &&
			(c.ownerInflight[owner] == 0 || c.maxOwnerInflightBytes <= 0 || c.ownerInflight[owner]+reserve <= c.maxOwnerInflightBytes)
		if fits {
			c.producers++
			c.inflightBytes += reserve
			c.ownerInflight[owner] += reserve
			if c.producers > c.peakProducers {
				c.peakProducers = c.producers
			}
			if c.inflightBytes > c.peakInflight {
				c.peakInflight = c.inflightBytes
			}
			c.mu.Unlock()
			return nil
		}
		wake := c.admissionWake
		c.mu.Unlock()
		select {
		case <-wake:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (c *boundedCache[V]) release(owner string, reserve int64) {
	if reserve < 0 {
		reserve = 0
	}
	c.mu.Lock()
	c.producers--
	c.inflightBytes -= reserve
	c.ownerInflight[owner] -= reserve
	if c.ownerInflight[owner] <= 0 {
		delete(c.ownerInflight, owner)
	}
	close(c.admissionWake)
	c.admissionWake = make(chan struct{})
	c.mu.Unlock()
}

func (c *boundedCache[V]) produce(ctx context.Context, owner, key string, reserve int64, produce func() (V, int64, error), flight *cacheFlight[V], clone bool) (V, error) {
	var zero V
	c.mu.Lock()
	c.stats.Misses++
	c.mu.Unlock()
	var (
		value V
		size  int64
		err   error
	)
	func() {
		// A panicking producer must still release its waiters, its reservation and the key.
		defer func() {
			if r := recover(); r != nil {
				c.release(owner, reserve)
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
	c.release(owner, reserve)

	if err != nil {
		c.mu.Lock()
		delete(c.inflight, key)
		c.stats.Failures++
		flight.err = err
		c.mu.Unlock()
		close(flight.done)
		return zero, err
	}
	// Decide before cloning: a value that will not be retained is cloned only if
	// someone is waiting for it.
	c.mu.Lock()
	retain := size >= 0 && size <= c.maxBytes && (c.maxOwnerBytes <= 0 || size <= c.maxOwnerBytes)
	waiters := flight.waiters
	if !retain {
		c.stats.Oversize++
	}
	c.mu.Unlock()
	var shared V
	if retain || waiters > 0 {
		shared = c.doClone(value) // outside the lock: independent keys are never delayed by a copy
	}
	c.mu.Lock()
	if retain {
		c.storeLocked(owner, key, shared, size)
	}
	if retain || waiters > 0 {
		flight.value, flight.ok = shared, true
	}
	delete(c.inflight, key)
	c.mu.Unlock()
	close(flight.done)
	return value, nil
}

// Lookup returns a clone of the retained value for key, if any.
func (c *boundedCache[V]) Lookup(key string) (V, bool) {
	var zero V
	c.mu.Lock()
	entry, ok := c.entries[key]
	if !ok {
		c.stats.Misses++
		c.mu.Unlock()
		return zero, false
	}
	if c.ttl > 0 && c.now().Sub(entry.at) > c.ttl {
		c.removeLocked(entry)
		c.stats.Expired++
		c.stats.Misses++
		c.mu.Unlock()
		return zero, false
	}
	c.lru.MoveToFront(entry.el)
	c.stats.Hits++
	v := entry.value
	c.mu.Unlock()
	return c.doClone(v), true
}

// Store retains a clone of value for key under the same budgets as Get.
func (c *boundedCache[V]) Store(owner, key string, value V, size int64) {
	if size < 0 || size > c.maxBytes || (c.maxOwnerBytes > 0 && size > c.maxOwnerBytes) {
		c.mu.Lock()
		c.stats.Oversize++
		c.mu.Unlock()
		return
	}
	cl := c.doClone(value)
	c.mu.Lock()
	defer c.mu.Unlock()
	c.storeLocked(owner, key, cl, size)
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
	s.PeakProducers, s.PeakInflightBytes = c.peakProducers, c.peakInflight
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
