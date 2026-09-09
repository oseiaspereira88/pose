// Unread-key findings for the rest of the policies
// (spec pose-doctor-reports-unread-policy-keys, follow-up).
//
// The review policy got this first. Every other policy the engine models has
// the identical exposure — the decoder ignores what it does not know, so a
// misspelled key takes the default and the setting silently does nothing — and
// an operator who learns the finding exists for one file has no way to know it
// does not exist for the next.

package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDoctorReportsUnreadKeysInEveryModelledPolicy(t *testing.T) {
	// One misspelling per policy, each a near-miss of a real key: that is the
	// case that costs an operator most, because the file reads as configured
	// and is not.
	cases := []struct {
		file, check, body, typo string
	}{
		{"delivery.json", "delivery.policy-keys",
			`{"schema_version":1,"enabled":false,"results_path":"x.json","result_path":"x.json"}`, "result_path"},
		{"artifacts.json", "artifact.policy-keys",
			`{"schema_version":1,"enabled":false,"governed_roots":[],"governed_root":[]}`, "governed_root"},
		{"capabilities.json", "capability.policy-keys",
			`{"stale_after_days":30,"stale_after_commit":200}`, "stale_after_commit"},
		{"docs.json", "docs.policy-keys",
			`{"min_hits":1,"review_day":14}`, "review_day"},
		{"release.json", "release.policy-keys",
			`{"schema_version":1,"provider":"github","repositories":"a/b"}`, "repositories"},
		{"state.json", "state.policy-keys",
			`{"max_age_days":30,"max_commit":200}`, "max_commit"},
	}
	for _, tc := range cases {
		t.Run(tc.file, func(t *testing.T) {
			root := doctorTrailerFixture(t)
			mustWrite(t, filepath.Join(root, ".pose", "policy", tc.file), tc.body)

			f, ok := findDoctorFinding(runDoctorJSON(t, root), tc.check)
			if !ok {
				t.Fatalf("no %s finding: the policy is not held to its keys at all", tc.check)
			}
			if f.Level != "warn" {
				t.Fatalf("level=%q, want warn: %s", f.Level, f.Message)
			}
			if !strings.Contains(f.Message, tc.typo) {
				t.Errorf("the finding does not name %q: %q", tc.typo, f.Message)
			}
			if f.Hint == "" {
				t.Error("the finding does not say which keys are read, so it names a problem with no remedy")
			}
		})
	}
}

// `_comment` is how the shipped policies say what the file is for and why it
// ships disabled. Reporting it would make the finding fire on every fresh
// install, which is how an operator learns to ignore a check.
func TestDoctorDoesNotReportTheAnnotationKey(t *testing.T) {
	root := doctorTrailerFixture(t)
	mustWrite(t, filepath.Join(root, ".pose", "policy", "delivery.json"),
		`{"_comment":"why this ships disabled","schema_version":1,"enabled":false,"results_path":"x.json"}`)

	f, ok := findDoctorFinding(runDoctorJSON(t, root), "delivery.policy-keys")
	if !ok {
		t.Fatal("no delivery.policy-keys finding")
	}
	if f.Level != "ok" {
		t.Errorf("the annotation key was reported as unread: %s", f.Message)
	}
}

// capabilities.json is decoded twice, by two structs in two commands. Holding
// it to one of them inverts the finding: it tells the operator to remove
// settings the engine does read.
func TestDoctorAcceptsCapabilityKeysReadByEitherConsumer(t *testing.T) {
	root := doctorTrailerFixture(t)
	mustWrite(t, filepath.Join(root, ".pose", "policy", "capabilities.json"),
		`{"stale_after_days":30,"stale_after_commits":200,"min_hits":2,"level":"any","default_owner":"@team","review_days":21}`)

	f, ok := findDoctorFinding(runDoctorJSON(t, root), "capability.policy-keys")
	if !ok {
		t.Fatal("no capability.policy-keys finding")
	}
	if f.Level != "ok" {
		t.Errorf("a key one of the two readers models was reported as unread: %s", f.Message)
	}
}

// LoadReleasePolicy still falls back to the pre-.pose/policy location, so an
// instance that has not migrated reads its release policy from there. Checking
// only the current path would say nothing about that instance, and the silence
// would read as approval.
func TestDoctorReadsTheReleasePolicyWhereTheEngineDoes(t *testing.T) {
	root := doctorTrailerFixture(t)
	mustWrite(t, filepath.Join(root, ".pose", "release-policy.json"),
		`{"schema_version":1,"provider":"github","repositories":"a/b"}`)

	f, ok := findDoctorFinding(runDoctorJSON(t, root), "release.policy-keys")
	if !ok {
		t.Fatal("no release.policy-keys finding for a policy at the legacy location")
	}
	if f.Level != "warn" || !strings.Contains(f.Message, "repositories") {
		t.Errorf("the legacy location was not examined: %+v", f)
	}
}

// A policy the instance does not have is not a finding: absent means the
// defaults apply, and every one of these files is optional.
func TestDoctorSaysNothingAboutAnAbsentPolicy(t *testing.T) {
	root := doctorTrailerFixture(t)
	findings := runDoctorJSON(t, root)
	for _, check := range policyKeyChecks() {
		if _, err := os.Stat(filepath.Join(root, ".pose", "policy", check.file)); err == nil {
			continue
		}
		if f, ok := findDoctorFinding(findings, check.check); ok {
			t.Errorf("%s reported on an absent policy: %+v", check.check, f)
		}
	}
}

// The keys have to come from the structs. A restated list drifts the first time
// a field is added, and reports a real key as unknown — a failure that looks
// exactly like the misspelling the check exists to catch.
func TestEveryWiredPolicyKeyListMatchesItsStruct(t *testing.T) {
	root := doctorTrailerFixture(t)
	for _, check := range policyKeyChecks() {
		// Writing every known key back and requiring silence is what proves the
		// list describes the file rather than some other struct's fields.
		document := map[string]any{}
		for _, key := range check.known {
			document[key] = nil
		}
		encoded, err := json.Marshal(document)
		if err != nil {
			t.Fatal(err)
		}
		mustWrite(t, filepath.Join(root, ".pose", "policy", check.file), string(encoded))
	}
	for _, f := range runDoctorJSON(t, root) {
		if strings.HasSuffix(f.Check, ".policy-keys") && f.Level != "ok" {
			t.Errorf("%s reported its own known keys as unread: %s", f.Check, f.Message)
		}
	}
}
