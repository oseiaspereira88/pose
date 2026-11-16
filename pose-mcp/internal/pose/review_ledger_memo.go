package pose

import "sync"

// The review ledger of a project is read by every bundle verification an
// operation makes: a closeout that verifies 64 federated dependencies read
// pose-dist's ledger 280 times and checked transcript digests 29,680 times.
// Within a read-view operation the view already returns the same bytes for
// the same read, so a value derived only from those reads is the same on
// every request: it is computed once and kept for the life of the operation
// (spec pose-review-ledger-is-read-once-per-operation, ADR
// governance-reads-use-bounded-operation-scoped-views D14). The first
// computation reads through the view, so the operation's closure and final
// audit cover every input as before.

type reviewRunsRead struct {
	runs []ReviewRun
	lost []string
}

// ledgerMemoCell computes its value once: concurrent requests for the same
// key wait for the one computation.
type ledgerMemoCell[V any] struct {
	once  sync.Once
	value V
	err   error
}

// ledgerMemoComputed sees each computation; tests count them.
var ledgerMemoComputed = func(key string) {}

// memoizedLedgerRead returns compute's result, kept per key in the
// read-view operation of files; without an operation it computes every
// time. The bool reports whether the value came from the memo.
func memoizedLedgerRead[V any](files Files, key string, compute func() (V, error)) (V, bool, error) {
	view, viewFiles := governanceReadViewFromFiles(files)
	if view == nil || view.operation == nil || viewFiles == nil {
		// Without an operation, or without the producer the reads are
		// attributed to, every request computes and reads for itself.
		ledgerMemoComputed(key)
		value, err := compute()
		return value, false, err
	}
	// A kept value is served only while the view is open, as a read is: a
	// finalized view rejects it (review F4).
	if err := view.begin(); err != nil {
		var zero V
		return zero, false, err
	}
	defer view.end()
	// A kept value is the producer's own: each producer reads once through
	// the view, so the final audit attributes a changed input to every
	// producer that consumed it (review N3).
	key += "\x00producer=" + viewFiles.producer + "\x00project=" + viewFiles.project
	kept, _ := view.operation.ledgerMemo.LoadOrStore(key, &ledgerMemoCell[V]{})
	cell := kept.(*ledgerMemoCell[V])
	computed := false
	cell.once.Do(func() {
		computed = true
		ledgerMemoComputed(key)
		cell.value, cell.err = compute()
	})
	return cell.value, !computed, cell.err
}

// copyReviewRuns gives a caller runs that share nothing with the kept ones:
// deepCopyValue copies every exported field, and the unexported ones are
// copied here (a reflection test holds this list to ReviewRun's fields).
func copyReviewRuns(runs []ReviewRun) []ReviewRun {
	if runs == nil {
		return nil
	}
	out := make([]ReviewRun, len(runs))
	for i, run := range runs {
		out[i] = deepCopyValue(run)
		out[i].sessionForRedaction = run.sessionForRedaction
		out[i].eventsDir = run.eventsDir
		if run.eventsBytes != nil {
			out[i].eventsBytes = append(make([]byte, 0, len(run.eventsBytes)), run.eventsBytes...)
		}
	}
	return out
}
