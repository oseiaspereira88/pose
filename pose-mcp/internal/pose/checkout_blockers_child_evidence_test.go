package pose

import "testing"

// Spec pose-sealed-checkout-does-not-require-later-child-evidence.

// R1: a sealed checkout is not blocked by the missing review bundles of the children of its scope
// (evidence newer than the subject's head), and still is by every other blocker.
func TestTheSealedCheckoutIsNotBlockedByChildEvidenceItCannotHold(t *testing.T) {
	blockers := []string{
		"delivery integrity is unavailable: no generation",
		"child milestone:pose-shared-runtime/optional-worktree-runtime has no sealed review bundle: current semantic inputs are not represented by a sealed bundle",
		"child milestone:pose-shared-runtime/bounded-graph-caches has no sealed review bundle",
		"review policy is not adopted",
		"the roadmap names a milestone no spec belongs to",
		"children of the scope are not terminal",
	}
	got := checkoutBlockers(blockers)
	want := []string{"review policy is not adopted", "the roadmap names a milestone no spec belongs to", "children of the scope are not terminal"}
	if len(got) != len(want) {
		t.Fatalf("checkoutBlockers = %q, want %q", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("checkoutBlockers[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}
