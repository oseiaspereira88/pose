// The release notes say when a release introduces a contract
// (spec pose-adoption-stamp-stays-readable, follow-up).
//
// v2.0.0 added `evidence_vocabulary_reconciled_at`, and v1.8.1 then refused the
// whole review policy: its decoder used DisallowUnknownFields, so one key it did
// not know invalidated the file. The remedy is to update every engine reading
// that repository, and nothing in the encoding avoids it — the key names a
// contract the older version does not have.
//
// The notes for that release said none of this. Saying it per release, by hand,
// is the shape that fails the first time someone forgets; the registry already
// knows which contracts exist, so it is the registry that says it.

package pose

import (
	"strings"
	"testing"
)

func TestReleaseNotesWarnWhenTheReleaseIntroducesAContract(t *testing.T) {
	notes := RenderReleaseNotes("2.0.0", []ReleaseFragment{
		{Spec: "alpha", Category: "added", Body: "Something."},
	})
	if !strings.Contains(notes, "## Compatibility") {
		t.Fatalf("a release that introduces a contract says nothing about it:\n%s", notes)
	}
	if !strings.Contains(notes, "evidence-vocabulary") {
		t.Errorf("the section does not name the contract:\n%s", notes)
	}
	// The reason has to be there, not just the fact: an operator who is not told
	// that every reader must move together will move one.
	for _, phrase := range []string{"older than 2.0.0", "move together"} {
		if !strings.Contains(notes, phrase) {
			t.Errorf("the section does not say %q:\n%s", phrase, notes)
		}
	}
	// The fragments still render; the section is added, not substituted.
	if !strings.Contains(notes, "Something.") {
		t.Error("the compatibility section replaced the release's own notes")
	}
}

// A release that introduces nothing must not carry the warning, or it stops
// meaning anything.
func TestReleaseNotesAreSilentWhenNoContractIsIntroduced(t *testing.T) {
	notes := RenderReleaseNotes("2.0.1", []ReleaseFragment{
		{Spec: "alpha", Category: "fixed", Body: "Something."},
	})
	if strings.Contains(notes, "## Compatibility") {
		t.Errorf("a release introducing no contract carried the warning:\n%s", notes)
	}
}

// The version is matched with or without the leading v, because both forms
// reach this: the manifest carries `v2.0.0` and the registry records `2.0.0`.
func TestContractsIntroducedInAcceptsBothVersionForms(t *testing.T) {
	if got := ContractsIntroducedIn("v2.0.0"); len(got) != 1 || got[0].ID != "evidence-vocabulary" {
		t.Errorf("v-prefixed = %+v", got)
	}
	if got := ContractsIntroducedIn("2.0.0"); len(got) != 1 {
		t.Errorf("bare = %+v", got)
	}
	if got := ContractsIntroducedIn("1.1.0"); len(got) != 2 {
		t.Errorf("a release introducing two contracts reported %d", len(got))
	}
}

// Every contract has to say which release shipped it, or the notes for the
// release that adds the next one will be silent — the exact failure this exists
// to close, one contract later.
func TestEveryContractRecordsTheReleaseThatIntroducedIt(t *testing.T) {
	for _, contract := range ReviewContracts() {
		if contract.IntroducedIn == "" {
			t.Errorf("contract %q does not say which release introduced it, so that release's notes will not warn about it", contract.ID)
		}
	}
}
