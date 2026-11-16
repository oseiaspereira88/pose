package pose

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

type ledgerMemoEntry[V any] struct {
	value V
	err   error
}

// ledgerMemoComputed sees each computation; tests count them.
var ledgerMemoComputed = func(key string) {}

// memoizedLedgerRead returns compute's result, kept per key in the
// read-view operation of files; without an operation it computes every
// time. The bool reports whether the value came from the memo.
func memoizedLedgerRead[V any](files Files, key string, compute func() (V, error)) (V, bool, error) {
	view, _ := governanceReadViewFromFiles(files)
	if view == nil || view.operation == nil {
		ledgerMemoComputed(key)
		value, err := compute()
		return value, false, err
	}
	op := view.operation
	if kept, ok := op.ledgerMemo.Load(key); ok {
		entry := kept.(ledgerMemoEntry[V])
		return entry.value, true, entry.err
	}
	ledgerMemoComputed(key)
	value, err := compute()
	kept, _ := op.ledgerMemo.LoadOrStore(key, ledgerMemoEntry[V]{value: value, err: err})
	entry := kept.(ledgerMemoEntry[V])
	return entry.value, false, entry.err
}
