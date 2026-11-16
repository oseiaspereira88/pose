package pose

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
)

// countLedgerComputations counts every computation of a memoized ledger
// value by the kind of value (the key's first field).
func countLedgerComputations(t *testing.T) func() map[string]int {
	t.Helper()
	var mu sync.Mutex
	counts := map[string]int{}
	keys := map[string]int{}
	original := ledgerMemoComputed
	ledgerMemoComputed = func(key string) {
		mu.Lock()
		counts[strings.SplitN(key, "\x00", 2)[0]]++
		keys[key]++
		mu.Unlock()
	}
	t.Cleanup(func() { ledgerMemoComputed = original })
	return func() map[string]int {
		mu.Lock()
		defer mu.Unlock()
		out := map[string]int{}
		for kind, n := range counts {
			out[kind] = n
		}
		for key, n := range keys {
			if n > out["max-per-key"] {
				out["max-per-key"] = n
			}
			_ = key
		}
		return out
	}
}

func ledgerOperation(t *testing.T, store Store) Store {
	t.Helper()
	base, view := store.beginGovernanceReadView()
	if view == nil || view.operation == nil {
		t.Fatal("no read-view operation")
	}
	return base.withGovernanceReadView(view, "review")
}

// Spec pose-review-ledger-is-read-once-per-operation, R1 and R2: every
// verification of an operation after the first is answered without reading
// or checking the ledger again.
func TestReviewLedgerIsReadOncePerOperation(t *testing.T) {
	f, run := approvedByARunThenDone(t)
	scope := f.bundle.Payload.Scope.Ref
	counts := countLedgerComputations(t)
	scoped := ledgerOperation(t, f.store)
	first, err := scoped.VerifyReviewBundle(scope)
	if err != nil {
		t.Fatal(err)
	}
	afterFirst := counts()
	for i := 0; i < 3; i++ {
		if _, err := scoped.VerifyReviewBundle(scope); err != nil {
			t.Fatal(err)
		}
		if _, _, err := scoped.ReadReviewRuns(run.BundleID); err != nil {
			t.Fatal(err)
		}
	}
	after := counts()
	if afterFirst["review-runs"] == 0 || afterFirst["transcript"] == 0 || afterFirst["bundle-header"] == 0 {
		t.Fatalf("the first verification read no ledger: %v", afterFirst)
	}
	if after["max-per-key"] != 1 || !reflect.DeepEqual(after, afterFirst) {
		t.Fatalf("later verifications read the ledger again: first %v, after %v", afterFirst, after)
	}
	if !first.Approved {
		t.Fatalf("the fixture's verification is not approved: %v", first.Blockers)
	}
}

// R3: a store with no read-view operation reads the ledger on every call.
func TestReviewLedgerWithoutAnOperationIsReadEveryTime(t *testing.T) {
	f, run := approvedByARunThenDone(t)
	counts := countLedgerComputations(t)
	for i := 0; i < 2; i++ {
		if _, _, err := f.store.ReadReviewRuns(run.BundleID); err != nil {
			t.Fatal(err)
		}
	}
	if got := counts()["review-runs"]; got != 2 {
		t.Fatalf("review runs read %d times without an operation, want 2", got)
	}
}

// R4: the answers are the same with and without the memo, a lost record and
// a caller that changes what it was given included.
func TestReviewLedgerMemoAnswersAsAFreshRead(t *testing.T) {
	f, run := approvedByARunThenDone(t)
	scope := f.bundle.Payload.Scope.Ref
	broken := filepath.Join(f.store.Root, filepath.FromSlash(reviewRunsDir), "rrn-0000000000000000.json")
	if err := os.WriteFile(broken, []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	freshRuns, freshLost, freshErr := f.store.ReadReviewRuns(run.BundleID)
	freshAtt, freshAttErr := f.store.ReadReviewAttestations(run.BundleID)
	fresh, freshVerifyErr := f.store.VerifyReviewBundle(scope)
	scoped := ledgerOperation(t, f.store)
	for i := 0; i < 2; i++ {
		runs, lost, err := scoped.ReadReviewRuns(run.BundleID)
		if !reflect.DeepEqual(runs, freshRuns) || !reflect.DeepEqual(lost, freshLost) || (err == nil) != (freshErr == nil) {
			t.Fatalf("read %d: runs or losses differ from a fresh read: lost %v, want %v", i, lost, freshLost)
		}
		if len(runs) > 0 {
			runs[0].RunID = "changed by the caller"
		}
		att, attErr := scoped.ReadReviewAttestations(run.BundleID)
		if !reflect.DeepEqual(att, freshAtt) || (attErr == nil) != (freshAttErr == nil) {
			t.Fatalf("read %d: attestations differ from a fresh read", i)
		}
		if len(att.Attestations) > 0 {
			att.Attestations[0].BundleID = "changed by the caller"
		}
		verification, err := scoped.VerifyReviewBundle(scope)
		if verification.Approved != fresh.Approved || verification.Fresh != fresh.Fresh || !reflect.DeepEqual(verification.Blockers, fresh.Blockers) || (err == nil) != (freshVerifyErr == nil) {
			t.Fatalf("read %d: verification differs from a fresh one: %v, want %v", i, verification.Blockers, fresh.Blockers)
		}
	}
	if len(freshLost) == 0 {
		t.Fatal("the unreadable run was not reported as lost")
	}
}
