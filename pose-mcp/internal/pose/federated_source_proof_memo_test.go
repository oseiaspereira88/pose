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

// Review F2: two scopes that differ only in a field of their identity (two
// milestones of one roadmap) are proved apart, in either order.
func TestFederatedSourceProofKeepsEveryIdentityApart(t *testing.T) {
	f, _ := approvedByARunThenDone(t)
	core := ArtifactRef{Project: "proj.source", Kind: "milestone", Slug: "program", Milestone: "core"}
	later := ArtifactRef{Project: "proj.source", Kind: "milestone", Slug: "program", Milestone: "later"}
	for _, order := range [][]ArtifactRef{{core, later}, {later, core}} {
		scoped := ledgerOperation(t, f.store)
		answers := map[string][]string{}
		for _, identity := range order {
			_, _, _, _, _, _, blockers := federatedSourceProof(scoped.fsys(), scoped, identity, "")
			answers[identity.Milestone] = blockers
		}
		fresh := func(identity ArtifactRef) []string {
			_, _, _, _, _, _, blockers := federatedSourceProof(f.store.fsys(), f.store, identity, "")
			return blockers
		}
		for _, identity := range order {
			if want := fresh(identity); !reflect.DeepEqual(answers[identity.Milestone], want) {
				t.Fatalf("order %s first: milestone %s got %v, want %v", order[0].Milestone, identity.Milestone, answers[identity.Milestone], want)
			}
		}
		if reflect.DeepEqual(answers["core"], answers["later"]) {
			t.Fatalf("two milestones share an answer: %v", answers["core"])
		}
	}
}
