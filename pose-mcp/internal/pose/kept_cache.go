package pose

import "sync"

// keptCache holds values computed from content-addressed inputs for the
// process (spec pose-federated-proof-reads-each-input-once). A key is
// computed once even when several callers miss it together: the first
// computes, the others wait for it. A result the computation declines to
// keep (a failed read) is not shared: the waiters acquire the key again.
// Past its bound the kept values are dropped, for a long-running process;
// a key being computed keeps its entry.
type keptCache[V any] struct {
	mu      sync.Mutex
	entries map[string]*keptEntry[V]
	bound   int
	// kept counts the completed, kept entries, which the bound limits.
	kept int
}

type keptEntry[V any] struct {
	ready chan struct{}
	value V
	kept  bool
	done  bool
}

func newKeptCache[V any](bound int) *keptCache[V] {
	return &keptCache[V]{entries: map[string]*keptEntry[V]{}, bound: bound}
}

// get returns the kept value of key, or computes it; compute reports whether
// its result may be kept. A caller that waited for a result that was not
// kept acquires the key again, so a recovered read is computed once and
// shared (review round 2, N1).
func (c *keptCache[V]) get(key string, compute func() (V, bool)) V {
	for {
		c.mu.Lock()
		if entry, ok := c.entries[key]; ok {
			c.mu.Unlock()
			<-entry.ready
			if entry.kept {
				return entry.value
			}
			continue
		}
		entry := &keptEntry[V]{ready: make(chan struct{})}
		c.entries[key] = entry
		c.mu.Unlock()
		value, keep := compute()
		c.mu.Lock()
		entry.value, entry.kept, entry.done = value, keep, true
		if !keep && c.entries[key] == entry {
			delete(c.entries, key)
		}
		if keep && c.entries[key] == entry {
			c.kept++
			c.evictLocked(key)
		}
		c.mu.Unlock()
		close(entry.ready)
		return value
	}
}

// evictLocked enforces the bound when a value is kept: past it, every other
// completed value is dropped. A key still being computed keeps its entry,
// so its waiters and a later caller never compute it twice (review rounds 2
// and 3, N2 and N3).
func (c *keptCache[V]) evictLocked(except string) {
	if c.kept <= c.bound {
		return
	}
	for key, entry := range c.entries {
		if entry.done && key != except {
			delete(c.entries, key)
			c.kept--
		}
	}
}

func (c *keptCache[V]) clear() {
	c.mu.Lock()
	c.entries = map[string]*keptEntry[V]{}
	c.kept = 0
	c.mu.Unlock()
}

func (c *keptCache[V]) len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.entries)
}
