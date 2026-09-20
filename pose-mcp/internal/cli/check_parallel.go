package cli

import (
	"runtime"
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
	workers := runtime.NumCPU()
	if workers > count {
		workers = count
	}
	if workers < 1 {
		workers = 1
	}
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
