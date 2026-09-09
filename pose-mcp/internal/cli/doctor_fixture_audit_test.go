package cli

import (
	"path/filepath"
	"strings"
	"testing"

	posemodel "github.com/harne8/pose-mcp/internal/pose"
)

// Audit of the doctor fixtures (spec pose-doctor-fixtures-exercise-production-path).
//
// Coverage over the diagnostics showed four branches no test reached. These are
// the ones that describe behaviour rather than a defensive guard, and each was
// found by measuring which code the suite executes — not by reading it, which
// is how the same class of gap survived twice already.

// `review.evidence-vocabulary` inspects a profile's criteria and its tools. Only
// the criteria half was ever executed, so a regression in the tool half would
// have been invisible — the check has two halves and the suite proved one.
func TestDoctorReportsAnUnproducibleClassDemandedByATool(t *testing.T) {
	root := doctorTrailerFixture(t)
	mustWrite(t, filepath.Join(root, ".pose", "policy", "review.json"),
		`{"schema_version":2,"enabled":true,"profiles":{"spec":"acme-spec-closeout@1"}}`)
	// Criteria are clean; only the tool demands a class no check may emit.
	mustWrite(t, filepath.Join(root, ".pose", "review-profiles", "acme-spec-closeout.json"),
		`{"schema_version":2,"id":"acme-spec-closeout","version":1,"scope":"spec",`+
			`"criteria":[{"id":"correctness","description":"d","evidence_classes":["unit"]}],`+
			`"tools":[{"id":"validate","requiredness":"required","evidence_classes":["telemetry"]}]}`)

	f, ok := findDoctorFinding(runDoctorJSON(t, root), "review.evidence-vocabulary")
	if !ok {
		t.Fatal("expected a review.evidence-vocabulary finding")
	}
	if f.Level != "warn" {
		t.Errorf("level=%q, want warn: %s", f.Level, f.Message)
	}
	if !strings.Contains(f.Message, "telemetry") {
		t.Errorf("the finding does not name the class the tool demands: %q", f.Message)
	}
}

// The ok branch of `policy.artifact-roots` was never executed, so "this check
// says ok when the roots resolve" was an assumption. A check that only ever
// proves its warn is a check that could be reporting on nothing.
func TestDoctorAcceptsArtifactRootsThatResolve(t *testing.T) {
	root := doctorTrailerFixture(t)
	mustWrite(t, filepath.Join(root, "src", "keep.txt"), "x\n")
	mustWrite(t, filepath.Join(root, ".pose", "policy", "artifacts.json"),
		`{"schema_version":1,"enabled":true,"governed_roots":["src"]}`)

	f, ok := findDoctorFinding(runDoctorJSON(t, root), "policy.artifact-roots")
	if !ok {
		t.Fatal("expected a policy.artifact-roots finding")
	}
	if f.Level != "ok" {
		t.Errorf("level=%q, want ok when the root exists: %s", f.Level, f.Message)
	}

	// And the warn still fires, so the ok is not the only reachable answer.
	mustWrite(t, filepath.Join(root, ".pose", "policy", "artifacts.json"),
		`{"schema_version":1,"enabled":true,"governed_roots":["nowhere"]}`)
	if f, ok := findDoctorFinding(runDoctorJSON(t, root), "policy.artifact-roots"); !ok || f.Level != "warn" {
		t.Errorf("a governed root that does not exist was accepted: %+v", f)
	}
}

// `mcp.config` has three answers and the suite executed one. The branch that
// went untested is the one that matters most: a configuration that exists and
// points somewhere other than the native binary.
func TestDoctorReportsAnMCPConfigThatIsNotTheNativeBinary(t *testing.T) {
	root := doctorTrailerFixture(t)
	mustWrite(t, filepath.Join(root, ".mcp.json"),
		`{"mcpServers":{"pose":{"command":"npx","args":["-y","pose-mcp"]}}}`)

	f, ok := findDoctorFinding(runDoctorJSON(t, root), "mcp.config")
	if !ok {
		t.Fatal("expected an mcp.config finding")
	}
	if f.Level != "warn" {
		t.Errorf("level=%q, want warn for a non-native command: %s", f.Level, f.Message)
	}

	mustWrite(t, filepath.Join(root, ".mcp.json"),
		`{"mcpServers":{"pose":{"command": "pose","args":["mcp"]}}}`)
	if f, ok := findDoctorFinding(runDoctorJSON(t, root), "mcp.config"); !ok || f.Level != "ok" {
		t.Errorf("a native configuration was not accepted: %+v", f)
	}
}

// Dropping DisallowUnknownFields kept a newer field from breaking an older
// binary, and gave up telling an operator that a key they wrote is not read.
// A misspelling now takes the default and the setting silently does nothing;
// this is the finding that recovers it.
func TestDoctorReportsReviewPolicyKeysTheEngineDoesNotRead(t *testing.T) {
	root := doctorTrailerFixture(t)
	mustWrite(t, filepath.Join(root, ".pose", "policy", "review.json"),
		`{"schema_version":2,"enabled":true,"profiles":{"spec":"spec-closeout@1"},`+
			`"contract_adoption":{"evidence-vocabulary":"2026-09-09"},"allow_criteria_reuse":true}`)

	f, ok := findDoctorFinding(runDoctorJSON(t, root), "review.policy-keys")
	if !ok {
		t.Fatal("expected a review.policy-keys finding")
	}
	if f.Level != "warn" {
		t.Errorf("level=%q, want warn: %s", f.Level, f.Message)
	}
	// Both misspellings are near-misses of real keys, which is the case that
	// costs an operator most: the file reads as configured and is not.
	for _, want := range []string{"contract_adoption", "allow_criteria_reuse"} {
		if !strings.Contains(f.Message, want) {
			t.Errorf("the finding does not name %q: %q", want, f.Message)
		}
	}
	if !strings.Contains(f.Hint, "contract_adoptions") {
		t.Errorf("the hint does not list the keys that are read: %q", f.Hint)
	}

	mustWrite(t, filepath.Join(root, ".pose", "policy", "review.json"),
		`{"schema_version":2,"enabled":true,"profiles":{"spec":"spec-closeout@1"},"contract_adoptions":{}}`)
	if f, ok := findDoctorFinding(runDoctorJSON(t, root), "review.policy-keys"); !ok || f.Level != "ok" {
		t.Errorf("a policy with only known keys was reported: %+v", f)
	}
}

// The known-key list is derived from the struct, not restated beside it. A
// hand-maintained list would drift the first time a field is added and report
// a real key as unknown — the failure would look exactly like the misspelling
// it is meant to catch.
func TestKnownPolicyKeysComeFromTheStruct(t *testing.T) {
	keys := posemodel.ReviewPolicyKnownKeys()
	if len(keys) < 10 {
		t.Fatalf("only %d keys derived, which suggests the reflection found nothing: %v", len(keys), keys)
	}
	for _, want := range []string{"schema_version", "profiles", "contract_adoptions", "review_bundles_adopted_at"} {
		found := false
		for _, key := range keys {
			if key == want {
				found = true
			}
		}
		if !found {
			t.Errorf("%q is a field of ReviewPolicy but is not derived: %v", want, keys)
		}
	}
	for _, absent := range []string{"id", "disposition", "rationale"} {
		for _, key := range keys {
			if key == absent {
				t.Errorf("%q belongs to a nested struct and must not be listed as a policy key", absent)
			}
		}
	}
}
