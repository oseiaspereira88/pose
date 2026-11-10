package cli

import (
	"runtime"
	"sync/atomic"
	"time"
)

// Spec pose-worktree-runtime-qos-and-measurement.

// runtimeAdmission is the runtime's policy for hosted commands: two lanes, each with its own
// slots and queue, so that a short query is never queued behind a heavy check, and a bound on
// what waits. The defaults are the ones the measurements of the spec were made with.
type runtimeAdmission struct {
	HeavySlots, ShortSlots int
	HeavyQueue, ShortQueue int
	MemoMiB                int
}

func defaultRuntimeAdmission() runtimeAdmission {
	heavy := 1
	short := 2
	if runtime.GOMAXPROCS(0) < 2 {
		short = 1
	}
	return runtimeAdmission{HeavySlots: heavy, ShortSlots: short, HeavyQueue: 4, ShortQueue: 16, MemoMiB: 256}
}

// shortCommands are the hosted commands that answer in well under a second on a repository of
// this size (measured in the spec: surface-check of one spec takes about 10 ms; roadmap-check
// takes minutes here); every other hosted command is heavy.
var shortCommands = map[string]bool{"surface-check": true}

// hostedLane is one lane of the admission policy and its counters.
type hostedLane struct {
	name     string
	slots    chan struct{}
	maxQueue int64
	queued   atomic.Int64
	running  atomic.Int64
	admitted atomic.Int64
	refused  atomic.Int64
	waitNS   atomic.Int64
	maxWait  atomic.Int64
}

func newHostedLane(name string, slots, queue int) *hostedLane {
	if slots < 1 {
		slots = 1
	}
	return &hostedLane{name: name, slots: make(chan struct{}, slots), maxQueue: int64(queue)}
}

// enter reserves a place in the queue; false means every slot is busy and the queue is full,
// and the command is refused before it waits. Two arrivals that find the same last free slot
// may both wait: the queue limit can be exceeded by the arrivals of one instant.
func (l *hostedLane) enter() bool {
	if l.queued.Add(1) > l.maxQueue && len(l.slots) == cap(l.slots) {
		l.queued.Add(-1)
		l.refused.Add(1)
		return false
	}
	return true
}

func (l *hostedLane) admit(wait time.Duration) {
	l.queued.Add(-1)
	l.running.Add(1)
	l.admitted.Add(1)
	l.waitNS.Add(int64(wait))
	for {
		cur := l.maxWait.Load()
		if int64(wait) <= cur || l.maxWait.CompareAndSwap(cur, int64(wait)) {
			return
		}
	}
}

func (l *hostedLane) withdrawn() { l.queued.Add(-1); l.refused.Add(1) }

func (l *hostedLane) metrics(out map[string]int64) {
	out[l.name+"_slots"] = int64(cap(l.slots))
	out[l.name+"_queue_limit"] = l.maxQueue
	out[l.name+"_queued"] = l.queued.Load()
	out[l.name+"_running"] = l.running.Load()
	out[l.name+"_admitted"] = l.admitted.Load()
	out[l.name+"_refused"] = l.refused.Load()
	out[l.name+"_wait_ms_total"] = l.waitNS.Load() / int64(time.Millisecond)
	out[l.name+"_wait_ms_max"] = l.maxWait.Load() / int64(time.Millisecond)
}

type hostedLanes struct{ heavy, short *hostedLane }

func newHostedLanes(p runtimeAdmission) *hostedLanes {
	return &hostedLanes{heavy: newHostedLane("heavy", p.HeavySlots, p.HeavyQueue), short: newHostedLane("short", p.ShortSlots, p.ShortQueue)}
}

func (h *hostedLanes) of(command string) *hostedLane {
	if shortCommands[command] {
		return h.short
	}
	return h.heavy
}

func (h *hostedLanes) metrics() map[string]int64 {
	out := map[string]int64{}
	h.heavy.metrics(out)
	h.short.metrics(out)
	return out
}
