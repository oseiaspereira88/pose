// Unread-key coverage for the shipped policies
// (spec pose-doctor-reports-unread-policy-keys, follow-up).
//
// The review policy got the unread-key finding first, because that is where the
// strict decoder had to be dropped. Extending it to three more files by hand is
// the shape this repository keeps being bitten by — a list that is right the day
// it is written and silently wrong afterwards, like the shellcheck file list and
// the workflow checkout depths.
//
// So the pairing is checked: every policy the engine ships is either held to its
// keys or exempted here, in writing, with the reason.

package cli

import (
	"io/fs"
	"sort"
	"strings"
	"testing"

	"github.com/harne8/pose-mcp/internal/scaffold"
)

// policyFilesWithoutAModelledStruct are shipped policies the doctor does not
// hold to a key list, and why. Each entry is a decision, not a backlog item:
// removing one means wiring the file into policyKeyChecks.
var policyFilesWithoutAModelledStruct = map[string]string{
	// Both are read through anonymous structs declared inside the commands that
	// use them, so there is no single declaration to derive a key list from —
	// and writing one out here would be the restated list this whole mechanism
	// exists to avoid. Worse, both would report true findings the engine itself
	// causes: changelog.json ships `categories`, which its only reader does not
	// model, and dor.json ships neither of the keys readiness.go looks for.
	// Giving them named structs is its own change.
	"changelog.json": "read through an anonymous struct in check.go; no named policy type to derive keys from",
	"dor.json":       "read through anonymous structs in readiness.go and check.go; no named policy type to derive keys from",
}

func TestEveryShippedPolicyIsHeldToItsKeysOrExempted(t *testing.T) {
	entries, err := fs.ReadDir(scaffold.Dist(), ".pose/policy")
	if err != nil {
		t.Fatalf("reading the shipped policies: %v", err)
	}
	wired := map[string]bool{}
	for _, check := range policyKeyChecks() {
		wired[check.file] = true
	}
	shipped := 0
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		shipped++
		name := entry.Name()
		_, exempt := policyFilesWithoutAModelledStruct[name]
		if wired[name] && exempt {
			t.Errorf("%s is both wired into policyKeyChecks and exempted; the exemption is stale", name)
			continue
		}
		if !wired[name] && !exempt {
			t.Errorf("%s ships but its keys are unchecked: wire it into policyKeyChecks, or record in policyFilesWithoutAModelledStruct why it has no key list — an operator who learns the finding exists for one policy cannot know it does not exist for this one", name)
		}
	}
	if shipped == 0 {
		t.Fatal("no shipped policy was examined — the scaffold lookup is broken, not the coverage")
	}
	for name := range policyFilesWithoutAModelledStruct {
		found := false
		for _, entry := range entries {
			if entry.Name() == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("%s is exempted but no longer shipped; drop the exemption", name)
		}
	}
}

// The keys have to come from the structs, or the finding drifts from what is
// actually read — which is the whole reason the review list was derived rather
// than written down.
func TestPolicyKeyChecksDeriveTheirKeysFromTheStructs(t *testing.T) {
	for _, check := range policyKeyChecks() {
		if len(check.known) == 0 {
			t.Errorf("%s has no known keys, so every key in the file would be reported unread", check.file)
			continue
		}
		if !sort.StringsAreSorted(check.known) {
			t.Errorf("%s: known keys are not sorted, so the finding's remedy line is unstable: %v", check.file, check.known)
		}
	}
}

// capabilities.json is read by two structs, and the finding is only correct
// when it knows about both. With one of them the other's keys read as unread,
// which is the finding inverted: it would tell an operator to delete settings
// the engine does use.
func TestCapabilityPolicyKeysCoverBothReaders(t *testing.T) {
	var known []string
	for _, check := range policyKeyChecks() {
		if check.file == "capabilities.json" {
			known = check.known
		}
	}
	if known == nil {
		t.Fatal("capabilities.json is not wired into policyKeyChecks")
	}
	for _, key := range []string{"stale_after_days", "stale_after_commits", "min_hits", "level", "default_owner", "review_days"} {
		found := false
		for _, k := range known {
			if k == key {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("%q is read from capabilities.json but is not in the known keys, so the doctor would report a working setting as unread", key)
		}
	}
}
