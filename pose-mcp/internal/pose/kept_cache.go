package pose

import "sync"

// keptCache holds values computed from content-addressed inputs for the
// process (spec pose-federated-proof-reads-each-input-once). A key is
// computed once even when several callers miss it together: the first
// computes, the others wait for it. A result the computation declines to
// keep (a failed read) is not shared: each waiter then computes its own.
// Past its bound the cache is dropped, for a long-running process.
type keptCache[V any] struct {
	mu      sync.Mutex
	entries map[string]*keptEntry[V]
	bound   int
}

type keptEntry[V any] struct {
	ready chan struct{}
	value V
	kept  bool
}

func newKeptCache[V any](bound int) *keptCache[V] {
	return &keptCache[V]{entries: map[string]*keptEntry[V]{}, bound: bound}
}

// get returns the kept value of key, or computes it; compute reports whether
// its result may be kept.
func (c *keptCache[V]) get(key string, compute func() (V, bool)) V {
	c.mu.Lock()
	if entry, ok := c.entries[key]; ok {
		c.mu.Unlock()
		<-entry.ready
		if entry.kept {
			return entry.value
		}
		value, _ := compute()
		return value
	}
	if len(c.entries) >= c.bound {
		c.entries = map[string]*keptEntry[V]{}
	}
	entry := &keptEntry[V]{ready: make(chan struct{})}
	c.entries[key] = entry
	c.mu.Unlock()
	value, keep := compute()
	c.mu.Lock()
	entry.value, entry.kept = value, keep
	if !keep && c.entries[key] == entry {
		delete(c.entries, key)
	}
	c.mu.Unlock()
	close(entry.ready)
	return value
}

func (c *keptCache[V]) clear() {
	c.mu.Lock()
	c.entries = map[string]*keptEntry[V]{}
	c.mu.Unlock()
}

func (c *keptCache[V]) len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.entries)
}
