package pose

import (
	"strings"
	"testing"
)

// Spec pose-review-subject-head-is-the-newest.

// R1: of the heads of the change sets of a scope, the subject's head is the newest commit,
// whatever the order of the sets (the sort is by id) and whichever of them is last.
func TestTheSubjectHeadIsTheNewestHeadOfItsChangeSets(t *testing.T) {
	root := snapshotRepo(t, map[string]string{"a.md": "one\n"})
	commit := func(name string) string {
		snapshotTestGit(t, root, "commit", "-q", "--allow-empty", "-m", name)
		return strings.TrimSpace(snapshotTestGit(t, root, "rev-parse", "HEAD"))
	}
	first, second, third := commit("first"), commit("second"), commit("third")
	s := Store{Root: root}
	for _, order := range [][]string{{first, second, third}, {third, first, second}, {second, third, first}} {
		var sets []ChangeSet
		for _, head := range order {
			sets = append(sets, ChangeSet{ID: "cs-" + head[:6], ResolvedHead: head})
		}
		if got := s.newestResolvedHead(sets); got != third {
			t.Fatalf("order %v: newest = %s, want %s", order, got, third)
		}
	}
	if got := s.newestResolvedHead([]ChangeSet{{ID: "one", ResolvedHead: first}, {ID: "same", ResolvedHead: first}}); got != "" {
		t.Fatalf("a single distinct head chose %q: there is nothing to choose", got)
	}
	if got := s.newestResolvedHead([]ChangeSet{{ID: "a", ResolvedHead: first}, {ID: "b", ResolvedHead: "0000000000000000000000000000000000000000"}}); got != "" {
		t.Fatalf("an unknown head was ordered: %q", got)
	}
}
