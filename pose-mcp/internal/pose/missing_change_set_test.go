package pose

import "testing"

// Only the missing change set stands in for the evidence gap: a milestone of
// external members has no change set and no such blocker, and keeps its gate.
func TestOnlyTheMissingChangeSetSilencesTheEvidenceGap(t *testing.T) {
	if !reportsMissingChangeSet([]string{"other", missingChangeSetBlocker + "spec:x; the index holds no commit"}) {
		t.Fatal("the missing change set was not recognized")
	}
	for _, blockers := range [][]string{nil, {"subject path is missing", "unattributed " + missingChangeSetBlocker}} {
		if reportsMissingChangeSet(blockers) {
			t.Fatalf("another blocker silenced the evidence gap: %v", blockers)
		}
	}
}
