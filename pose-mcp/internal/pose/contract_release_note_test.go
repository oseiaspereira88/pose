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
	// The reason has to be there, not just the fact.
	for _, phrase := range []string{"older than 2.0.0", "never applies it"} {
		if !strings.Contains(notes, phrase) {
			t.Errorf("the section does not say %q:\n%s", phrase, notes)
		}
	}
	// evidence-vocabulary is carried by a top-level key, so for this one the
	// stronger claim is true and has to be made.
	for _, phrase := range []string{"evidence_vocabulary_reconciled_at", "before 2.0.2", "move together"} {
		if !strings.Contains(notes, phrase) {
			t.Errorf("the section does not say %q for a contract carried by a top-level key:\n%s", phrase, notes)
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

// The claim has to match how the adoption is actually written. A contract
// recorded as an id inside `contract_adoptions` does not make the policy
// unreadable: the map is a key those engines already model, and the strict
// decoder that refused unknown keys was dropped in 2.0.2 anyway. Telling every
// user of a future contract to upgrade in lockstep would be false, and false in
// the direction that costs them work.
func TestTheWarningMatchesHowTheAdoptionIsWritten(t *testing.T) {
	carried := ReviewContract{
		ID: "carried-in-the-map", Summary: "something", IntroducedIn: "9.9.9",
	}
	if carried.AdoptionAddsATopLevelKey() {
		t.Fatal("a contract with no legacy field must not claim a top-level key")
	}
	for _, contract := range ReviewContracts() {
		if !contract.AdoptionAddsATopLevelKey() {
			continue
		}
		notes := RenderReleaseNotes(contract.IntroducedIn, nil)
		if !strings.Contains(notes, contract.LegacyField) {
			t.Errorf("%s is carried by %q and the notes do not name it:\n%s", contract.ID, contract.LegacyField, notes)
		}
		if !strings.Contains(notes, "move together") {
			t.Errorf("%s adds a top-level key and the notes do not say readers must move together", contract.ID)
		}
	}
}

// Every recorded version has to round-trip through the lookup, or a registry
// entry written `v3.1.0` is a release that silently carries no section.
func TestEveryContractRoundTripsThroughTheLookup(t *testing.T) {
	for _, contract := range ReviewContracts() {
		found := false
		for _, got := range ContractsIntroducedIn(contract.IntroducedIn) {
			if got.ID == contract.ID {
				found = true
			}
		}
		if !found {
			t.Errorf("contract %q records IntroducedIn=%q but ContractsIntroducedIn(%q) does not return it", contract.ID, contract.IntroducedIn, contract.IntroducedIn)
		}
		// And with the other spelling, since both reach this: the manifest
		// carries `v2.0.0` and the registry records `2.0.0`.
		for _, spelling := range []string{"v" + strings.TrimPrefix(contract.IntroducedIn, "v"), strings.TrimPrefix(contract.IntroducedIn, "v")} {
			if len(ContractsIntroducedIn(spelling)) == 0 {
				t.Errorf("contract %q is not found under the spelling %q", contract.ID, spelling)
			}
		}
	}
}
