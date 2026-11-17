package pose

import (
	"reflect"
	"testing"
)

// Spec pose-federated-source-proof-is-computed-once-per-operation, R1: a
// dependency proved several times in one operation is computed once, and
// every answer is the same as the first; R2: without an operation it is
// computed every time; R3: a caller that changes its answer does not change
// a later one.
func TestFederatedSourceProofIsComputedOncePerOperation(t *testing.T) {
	f, _ := approvedByARunThenDone(t)
	identity := ArtifactRef{Project: "proj.source", Kind: "spec", Slug: "not-a-spec-of-this-fixture"}
	counts := countLedgerComputations(t)
	scoped := ledgerOperation(t, f.store)
	var first []string
	for i := 0; i < 3; i++ {
		_, _, _, _, _, _, blockers := federatedSourceProof(scoped.fsys(), scoped, identity, "")
		if len(blockers) == 0 {
			t.Fatal("a missing dependency was proved")
		}
		if i == 0 {
			first = append([]string(nil), blockers...)
		} else if !reflect.DeepEqual(blockers, first) {
			t.Fatalf("answer %d differs: %v, want %v", i, blockers, first)
		}
		blockers[0] = "changed by the caller"
	}
	if got := counts()["federated-source-proof"]; got != 1 {
		t.Fatalf("three proofs in one operation computed %d times, want 1", got)
	}
	for i := 0; i < 2; i++ {
		federatedSourceProof(f.store.fsys(), f.store, identity, "")
	}
	if got := counts()["federated-source-proof"]; got != 3 {
		t.Fatalf("proofs without an operation: %d computations in total, want 3", got)
	}
}
