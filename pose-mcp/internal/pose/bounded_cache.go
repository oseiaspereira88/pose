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
//     lack of capacity receives ErrCacheBusy, which it retries or reports.
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
	maxInflightKeys       int
	entryOverhead         int64
	maxOwnerInflightKeys  int
	copySlots             chan struct{}

	entries  map[string]*cacheEntry[V]
	lru      *list.List
	used     int64
	owners   map[string]int64
	inflight map[string]*cacheFlight[V]

	producers     int
	inflightBytes int64
	ownerFlights  map[string]int
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
	// pins counts readers (views, clones in progress) holding the value; an evicted
	// entry stays charged until the last pin goes, so what the cache keeps alive is
	// what it accounts for.
	pins    int
	evicted bool
}

type cacheFlight[V any] struct {
	done    chan struct{}
	value   V
	ok      bool // value is set; false with err == nil means "ask again"
	err     error
	waiters int
	owner   string
	// finished, holders and held keep a shared result charged while waiters still
	// copy it: the reservation is released by the last waiter, not by the producer.
	finished bool
	holders  int
	held     int64
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

// ErrCacheBusy is returned to a caller refused because the cache has no room for
// another production, queued key or waiter. It is an overload the caller sees: it
// retries or reports it, and never produces outside the cache's budgets.
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
		maxInflightBytes: maxBytes, maxOwnerInflightBytes: maxOwnerBytes, maxProducers: 8, maxWaiters: 128, entryOverhead: defaultEntryOverhead, maxInflightKeys: 64, maxOwnerInflightKeys: 16,
		copySlots: make(chan struct{}, slots),
		entries:   map[string]*cacheEntry[V]{}, lru: list.New(), owners: map[string]int64{},
		inflight: map[string]*cacheFlight[V]{}, ownerFlights: map[string]int{}, ownerInflight: map[string]int64{}, admissionWake: make(chan struct{}),
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
	return c.get(ctx, owner, key, reserve, produce, nil)
}

// View is Get for a reader that promises neither to modify nor to keep the value:
// for a retained entry it receives the value itself, without the clone, pinned for
// as long as fn runs (an entry evicted meanwhile stays charged until the pin goes).
// A value produced or shared during the call is cloned like Get's: the in-flight
// result is not an entry to pin. This is what makes a small read of a large graph
// cheap (spec pose-bounded-isolated-graph-caches R1).
func (c *boundedCache[V]) View(ctx context.Context, owner, key string, reserve int64, produce func() (V, int64, error), fn func(V)) error {
	_, err := c.get(ctx, owner, key, reserve, produce, fn)
	return err
}

// get serves a retained entry (passing it to view, pinned, when view is set, else as
// a clone), shares an in-flight production, or becomes the producer.
func (c *boundedCache[V]) get(ctx context.Context, owner, key string, reserve int64, produce func() (V, int64, error), view func(V)) (V, error) {
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
				entry.pins++
				v := entry.value
				c.mu.Unlock()
				if view != nil {
					view(v)
					c.unpin(entry)
					return v, nil
				}
				out := c.doClone(v)
				c.unpin(entry)
				return out, nil
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
				// The waiter leaves: its place goes back to the key, or its hold on
				// the shared result goes back to the flight if it had finished.
				c.mu.Lock()
				if flight.finished {
					c.dropHolderLocked(flight)
				} else {
					flight.waiters--
				}
				c.mu.Unlock()
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
			out := c.doClone(v)
			c.mu.Lock()
			c.dropHolderLocked(flight)
			c.mu.Unlock()
			if view != nil {
				view(out)
			}
			return out, nil
		}
		// A new production: bounded in keys, globally and per owner, before it exists.
		if len(c.inflight) >= c.maxInflightKeys || (c.maxOwnerInflightKeys > 0 && c.ownerFlights[owner] >= c.maxOwnerInflightKeys) {
			c.stats.Busy++
			c.mu.Unlock()
			return zero, ErrCacheBusy
		}
		flight := &cacheFlight[V]{done: make(chan struct{}), owner: owner}
		c.inflight[key] = flight
		c.ownerFlights[owner]++
		c.mu.Unlock()

		// Admission: the production waits until its reservation fits.
		if err := c.admit(ctx, owner, reserve); err != nil {
			c.finishFlight(key, flight, err)
			return zero, err
		}
		out, err := c.produce(owner, key, reserve, produce, flight)
		if err == nil && view != nil {
			view(out)
		}
		return out, err
	}
}

// finishFlight ends a flight that produced nothing: it frees the key and its slot
// and releases the waiters with err.
func (c *boundedCache[V]) finishFlight(key string, flight *cacheFlight[V], err error) {
	c.mu.Lock()
	c.removeFlightLocked(key, flight)
	flight.err = err
	c.mu.Unlock()
	close(flight.done)
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

// produce runs the production the caller was admitted for. The reservation is held
// until the result is published (cloned, stored and handed to the waiters): the
// values being copied are charged to the cache for as long as they are alive in it.
func (c *boundedCache[V]) produce(owner, key string, reserve int64, produce func() (V, int64, error), flight *cacheFlight[V]) (V, error) {
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
				c.stats.Failures++
				c.mu.Unlock()
				c.finishFlight(key, flight, errCacheProducerPanicked)
				panic(r)
			}
		}()
		value, size, err = produce()
	}()
	if err != nil {
		c.release(owner, reserve)
		c.mu.Lock()
		c.stats.Failures++
		c.mu.Unlock()
		c.finishFlight(key, flight, err)
		return zero, err
	}
	c.mu.Lock()
	retain := size >= 0 && size <= c.maxBytes && (c.maxOwnerBytes <= 0 || size <= c.maxOwnerBytes)
	if !retain {
		c.stats.Oversize++
	}
	c.mu.Unlock()
	// Publication is one critical section: the shared result, the waiters it has,
	// the removal of the flight (so none joins later) and the release of the
	// reservation are decided together. A value that is not retained is cloned only
	// when someone waits for it; a waiter that joins while the copy is being made
	// (outside the lock, so that independent keys are never delayed by it) is seen
	// when the lock is taken again, and the copy is made for it too.
	var shared V
	have := false
	for {
		if !have && retain {
			shared, have = c.doClone(value), true
		}
		c.mu.Lock()
		if !have && flight.waiters > 0 {
			c.mu.Unlock()
			shared, have = c.doClone(value), true
			continue
		}
		break
	}
	if retain {
		c.storeLocked(owner, key, shared, size)
	}
	if flight.waiters > 0 {
		flight.value, flight.ok = shared, true
		flight.held = reserve // charged until the last waiter has copied the result
	}
	flight.finished = true
	flight.holders = flight.waiters
	c.removeFlightLocked(key, flight)
	if flight.holders == 0 {
		c.releaseLocked(owner, reserve)
	}
	c.mu.Unlock()
	close(flight.done)
	return value, nil
}

// releaseLocked gives a production's slot and reservation back. The caller holds mu.
func (c *boundedCache[V]) releaseLocked(owner string, reserve int64) {
	if reserve < 0 {
		reserve = 0
	}
	c.producers--
	c.inflightBytes -= reserve
	c.ownerInflight[owner] -= reserve
	if c.ownerInflight[owner] <= 0 {
		delete(c.ownerInflight, owner)
	}
	close(c.admissionWake)
	c.admissionWake = make(chan struct{})
}

// removeFlightLocked takes a flight out of the key table. The caller holds mu.
func (c *boundedCache[V]) removeFlightLocked(key string, flight *cacheFlight[V]) {
	if c.inflight[key] == flight {
		delete(c.inflight, key)
		c.ownerFlights[flight.owner]--
		if c.ownerFlights[flight.owner] <= 0 {
			delete(c.ownerFlights, flight.owner)
		}
	}
}

// dropHolderLocked is a waiter letting go of a finished flight's shared result; the
// last one releases the reservation the producer left charged. The caller holds mu.
func (c *boundedCache[V]) dropHolderLocked(flight *cacheFlight[V]) {
	if flight.holders <= 0 {
		return
	}
	flight.holders--
	if flight.holders == 0 {
		c.releaseLocked(flight.owner, flight.held)
	}
}

// unpin drops one pin; the last pin of an evicted entry frees what it was charged.
func (c *boundedCache[V]) unpin(entry *cacheEntry[V]) {
	c.mu.Lock()
	entry.pins--
	if entry.pins == 0 && entry.evicted {
		c.used -= entry.size
		c.owners[entry.owner] -= entry.size
		if c.owners[entry.owner] <= 0 {
			delete(c.owners, entry.owner)
		}
	}
	c.mu.Unlock()
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
	entry.pins++
	v := entry.value
	c.mu.Unlock()
	out := c.doClone(v)
	c.unpin(entry)
	return out, true
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

// entryOverhead is what an entry costs beyond its payload: the entry, its list node,
// the lookup slots and the owner accounting, measured on 64-bit platforms at about
// 136 bytes for the first two and rounded up for the rest; the key is charged by length.
const defaultEntryOverhead = 256

func (c *boundedCache[V]) storeLocked(owner, key string, value V, size int64) {
	if size < 0 {
		size = 0
	}
	size += c.entryOverhead + int64(len(key)) + int64(len(owner))
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
	// What readers still pin counts against the budgets: when evicting everything
	// else is not enough, the value is not retained instead of exceeding them.
	if c.used+size > c.maxBytes || (c.maxOwnerBytes > 0 && c.owners[owner]+size > c.maxOwnerBytes) {
		c.stats.Oversize++
		return
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

// removeLocked takes an entry out of the cache. A pinned entry stays charged until
// its last reader lets go.
func (c *boundedCache[V]) removeLocked(entry *cacheEntry[V]) {
	c.lru.Remove(entry.el)
	delete(c.entries, entry.key)
	if entry.pins > 0 {
		entry.evicted = true
		return
	}
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
