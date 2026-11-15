package pose

import (
	"strings"
	"testing"
)

func withMaterialHooks(t *testing.T, head func(string) string, at func(string, string, MaterialInputs) string) {
	t.Helper()
	oldDigests, oldHead, oldAt := CurrentMaterialDigests, currentMaterialHead, currentMaterialAt
	CurrentMaterialDigests = nil
	SetCurrentMaterialAt(head, at)
	currentMaterialKept.clear()
	t.Cleanup(func() {
		CurrentMaterialDigests, currentMaterialHead, currentMaterialAt = oldDigests, oldHead, oldAt
		currentMaterialKept.clear()
	})
}

func materialSets(n int) []ChangeSet {
	sets := make([]ChangeSet, n)
	for i := range sets {
		in := MaterialInputs{Names: []string{strings.Repeat("a", i+1) + ".go"}}
		sets[i] = ChangeSet{Spec: "s", MaterialDigest: "sha256:indexed", MaterialInputs: &in}
	}
	return sets
}

// Spec pose-federated-proof-reads-each-input-once, R1: HEAD is resolved
// once per graph, and the material of the same inputs at the same commit is
// read from the tree once per process.
func TestCurrentMaterialIsReadOncePerCommit(t *testing.T) {
	heads, reads := 0, 0
	withMaterialHooks(t, func(string) string { heads++; return strings.Repeat("a", 40) }, func(_, head string, in MaterialInputs) string {
		reads++
		return "sha256:" + head[:4] + in.Names[0]
	})
	sets := materialSets(5)
	first := withCurrentMaterial("/r", sets)
	second := withCurrentMaterial("/r", sets)
	if heads != 2 || reads != 5 {
		t.Fatalf("heads=%d reads=%d, want 2 and 5", heads, reads)
	}
	for i := range first {
		if first[i].MaterialDigest != second[i].MaterialDigest || first[i].MaterialDigest == "sha256:indexed" {
			t.Fatalf("set %d: %q then %q", i, first[i].MaterialDigest, second[i].MaterialDigest)
		}
	}
}

// R2: a HEAD that moved is read at its new commit.
func TestCurrentMaterialFollowsAMovedHead(t *testing.T) {
	head := strings.Repeat("a", 40)
	withMaterialHooks(t, func(string) string { return head }, func(_, at string, in MaterialInputs) string { return "sha256:" + at[:1] })
	sets := materialSets(1)
	before := withCurrentMaterial("/r", sets)[0].MaterialDigest
	head = strings.Repeat("b", 40)
	after := withCurrentMaterial("/r", sets)[0].MaterialDigest
	if before != "sha256:a" || after != "sha256:b" {
		t.Fatalf("before %q after %q", before, after)
	}
}

// R3: an unresolvable HEAD reads the tree of HEAD for each set and keeps
// nothing.
func TestCurrentMaterialWithoutAHeadIsNotKept(t *testing.T) {
	reads := 0
	withMaterialHooks(t, func(string) string { return "" }, func(_, at string, in MaterialInputs) string {
		reads++
		if at != "HEAD" {
			t.Fatalf("read at %q, want HEAD", at)
		}
		return "sha256:head"
	})
	sets := materialSets(2)
	withCurrentMaterial("/r", sets)
	withCurrentMaterial("/r", sets)
	if reads != 4 {
		t.Fatalf("reads=%d, want 4", reads)
	}
}

// R4: the kept digests of a long-running process are bounded.
func TestCurrentMaterialKeptDigestsAreBounded(t *testing.T) {
	head := 0
	withMaterialHooks(t, func(string) string { head++; return strings.Repeat(string(rune('a'+head%6)), 40) }, func(string, string, MaterialInputs) string { return "sha256:x" })
	old := currentMaterialKept.bound
	currentMaterialKept.bound = 3
	t.Cleanup(func() { currentMaterialKept.bound = old })
	for i := 0; i < 10; i++ {
		withCurrentMaterial("/r", materialSets(1))
	}
	kept := currentMaterialKept.len()
	if kept > 3 {
		t.Fatalf("%d digests kept past a bound of 3", kept)
	}
}
