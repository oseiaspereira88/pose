package cli

import (
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
)

// Per-spec gate work is independent, so it runs on several cores — and reports in
// the order it always did.
//
// `checkReviewCloseout` and `checkDeliveryContracts` each walk the repository's
// completed specs, and one spec's verdict never depends on another's. What made
// them slow was not that the reads were sequential but that each spec needs real
// work: a bundle preparation with Git reads. Caching removed the repetitions;
// nothing removes the remaining 116 distinct subjects, so the only lever left is
// doing them at the same time.
//
// Findings are buffered per item and replayed in item order. A gate whose output
// order depends on scheduling cannot be diffed between runs, and this repository
// relies on exactly that diff to prove a performance change did not alter a
// verdict — so parallelism is allowed to change when work happens and never what
// the gate prints, nor the order it prints it in.
//
// Safe because the per-item function touches no shared mutable state: Store carries
// only a root, the lookup tables in internal/pose are read-only, the two parse
// memos are mutex-guarded, and focusSurfaceGraph no longer writes through the
// graph it is handed — which is what lets one prebuilt graph serve every worker.
func (checker *nativeChecker) failOrWarnPerItem(count int, work func(i int) []string) {
	if count <= 0 {
		return
	}
	results := make([][]string, count)
	workers := checkWorkerCount(count)
	var next int64
	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				i := int(atomic.AddInt64(&next, 1)) - 1
				if i >= count {
					return
				}
				results[i] = work(i)
			}
		}()
	}
	wg.Wait()
	for _, messages := range results {
		for _, message := range messages {
			checker.failOrWarn(message)
		}
	}
}

// checkWorkerCount resolves the pool size for one gate run.
//
// The default stays the core count, and `POSE_CHECK_WORKERS` overrides it.
//
// The default was measured rather than assumed, and it survived: five runs each on a
// sixteen-core machine gave medians of 18.3s at four workers, 16.3s at eight and
// 16.8s at sixteen. Eight looked better than sixteen over two runs and does not over
// five — the ranges overlap, 0.49s apart — so the core count is left alone. Baking in
// a three-percent difference that noise explains would be worse than the knob.
//
// Past the core count it degrades, which is why there is no reason to raise it: the
// wall is flat at 18.7s for thirty-two and sixty-four workers while the summed item
// time nearly triples and the slowest item grows. The work is process spawning and
// CPU — 7,248 Git subprocesses in one run — not idle waiting, so extra workers
// contend instead of overlapping.
//
// The override exists because the right number is a property of the machine: a
// container pinned to fewer cores than the host reports, or an operator who wants the
// gate to leave room, had no way to say so.
//
// It is not a performance knob to tune per run. An unparseable or non-positive value
// is ignored rather than rejected, because a gate is the wrong place to fail over an
// environment variable.
func checkWorkerCount(count int) int {
	workers := runtime.NumCPU()
	if raw := strings.TrimSpace(os.Getenv("POSE_CHECK_WORKERS")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			workers = parsed
		}
	}
	if workers > count {
		workers = count
	}
	if workers < 1 {
		workers = 1
	}
	return workers
}
