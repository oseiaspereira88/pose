package cli

// Diagnostics for the two failure modes where POSE silently reduces what it
// knows and reports a downstream symptom instead of the upstream loss (spec
// pose-diagnose-invisible-governance-failures). Each check is asserted in both
// directions: warn on a fixture that reproduces the real failure, ok on one
// that does not, so a check that observes nothing cannot pass by accident.

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	posemodel "github.com/harne8/pose-mcp/internal/pose"
)

func TestDoctorWarnsOnUnproducibleEvidenceClass(t *testing.T) {
	root := doctorTrailerFixture(t)
	mustWrite(t, filepath.Join(root, ".pose", "policy", "review.json"),
		`{"schema_version":2,"enabled":true,"profiles":{"spec":"spec-closeout@1"}}`)
	// `validation` is the class the shipped spec-closeout profile demanded for
	// years while pose validate refused to register it on any check, leaving
	// auto-attest as the only path that completed a review.
	mustWrite(t, filepath.Join(root, ".pose", "review-profiles", "spec-closeout.json"),
		`{"schema_version":2,"id":"spec-closeout","version":1,"scope":"spec",`+
			`"criteria":[{"id":"correctness","description":"d","evidence_classes":["test","validation"]}],`+
			`"tools":[{"id":"validate","requiredness":"required","evidence_classes":["validation"]}]}`)

	f, ok := findDoctorFinding(runDoctorJSON(t, root), "review.evidence-vocabulary")
	if !ok {
		t.Fatal("expected a review.evidence-vocabulary finding")
	}
	if f.Level != "warn" {
		t.Errorf("level=%q, want warn", f.Level)
	}
	for _, want := range []string{"spec-closeout.json", "test", "validation"} {
		if !strings.Contains(f.Message, want) {
			t.Errorf("message does not name %q: %q", want, f.Message)
		}
	}
	if strings.Contains(f.Message, "integration") {
		t.Errorf("a producible class was reported as unreachable: %q", f.Message)
	}
}

func TestDoctorSilentWhenEveryEvidenceClassIsProducible(t *testing.T) {
	root := doctorTrailerFixture(t)
	mustWrite(t, filepath.Join(root, ".pose", "policy", "review.json"),
		`{"schema_version":2,"enabled":true,"profiles":{"spec":"spec-closeout@1"}}`)
	mustWrite(t, filepath.Join(root, ".pose", "review-profiles", "spec-closeout.json"),
		`{"schema_version":2,"id":"spec-closeout","version":1,"scope":"spec",`+
			`"criteria":[{"id":"correctness","description":"d","evidence_classes":["unit","integration"]}],`+
			`"tools":[{"id":"validate","requiredness":"required","evidence_classes":["build","reachability"]}]}`)

	f, ok := findDoctorFinding(runDoctorJSON(t, root), "review.evidence-vocabulary")
	if !ok {
		t.Fatal("expected a review.evidence-vocabulary finding")
	}
	if f.Level != "ok" {
		t.Errorf("level=%q, want ok: %s", f.Level, f.Message)
	}
}

func TestDoctorIgnoresAProfileThePolicyDoesNotSelect(t *testing.T) {
	root := doctorTrailerFixture(t)
	// The shape a project lands in the moment it owns its profiles and leaves
	// the shipped ones on disk: the unselected file still names unproducible
	// classes, but it builds no plan, so reporting it is noise to dismiss.
	mustWrite(t, filepath.Join(root, ".pose", "policy", "review.json"),
		`{"schema_version":2,"enabled":true,"profiles":{"spec":"acme-spec-closeout@1"}}`)
	mustWrite(t, filepath.Join(root, ".pose", "review-profiles", "acme-spec-closeout.json"),
		`{"schema_version":2,"id":"acme-spec-closeout","version":1,"scope":"spec",`+
			`"criteria":[{"id":"correctness","description":"d","evidence_classes":["unit"]}],`+
			`"tools":[{"id":"validate","requiredness":"required","evidence_classes":["build"]}]}`)
	mustWrite(t, filepath.Join(root, ".pose", "review-profiles", "spec-closeout.json"),
		`{"schema_version":2,"id":"spec-closeout","version":1,"scope":"spec",`+
			`"criteria":[{"id":"correctness","description":"d","evidence_classes":["validation"]}],`+
			`"tools":[{"id":"validate","requiredness":"required","evidence_classes":["validation"]}]}`)

	f, ok := findDoctorFinding(runDoctorJSON(t, root), "review.evidence-vocabulary")
	if !ok {
		t.Fatal("expected a review.evidence-vocabulary finding")
	}
	if f.Level != "ok" {
		t.Errorf("level=%q, want ok — the unselected profile builds no plan: %s", f.Level, f.Message)
	}
	if strings.Contains(f.Message, "spec-closeout.json") {
		t.Errorf("reported a profile the policy does not select: %q", f.Message)
	}
}

func TestDoctorWarnsOnCheckWithoutEvidenceClass(t *testing.T) {
	root := doctorTrailerFixture(t)
	// A check with no class still runs and still passes, but review evidence
	// collection discards its result: the module looks covered and contributes
	// nothing.
	mustWrite(t, filepath.Join(root, ".pose", "indexes", "validation-matrix.json"),
		`{"defaults":{"mode":"strict"},"stacks":{},"moduleOverrides":{"api":{"stack":"go","checks":[`+
			`{"name":"test","program":"go","args":["test","./..."],"severity":"required","evidenceClass":"unit"},`+
			`{"name":"vet","program":"go","args":["vet","./..."],"severity":"optional"}]}}}`)

	f, ok := findDoctorFinding(runDoctorJSON(t, root), "validate.evidence-class-coverage")
	if !ok {
		t.Fatal("expected a validate.evidence-class-coverage finding")
	}
	if f.Level != "warn" {
		t.Errorf("level=%q, want warn", f.Level)
	}
	if !strings.Contains(f.Message, "api/vet") {
		t.Errorf("message does not name the unclassed check: %q", f.Message)
	}
	if strings.Contains(f.Message, "api/test") {
		t.Errorf("a classified check was reported as unclassed: %q", f.Message)
	}
}

func TestDoctorSilentWhenEveryCheckDeclaresAnEvidenceClass(t *testing.T) {
	root := doctorTrailerFixture(t)
	mustWrite(t, filepath.Join(root, ".pose", "indexes", "validation-matrix.json"),
		`{"defaults":{"mode":"strict"},"stacks":{},"moduleOverrides":{"api":{"stack":"go","checks":[`+
			`{"name":"test","program":"go","args":["test","./..."],"severity":"required","evidenceClass":"unit"}]}}}`)

	f, ok := findDoctorFinding(runDoctorJSON(t, root), "validate.evidence-class-coverage")
	if !ok {
		t.Fatal("expected a validate.evidence-class-coverage finding")
	}
	if f.Level != "ok" {
		t.Errorf("level=%q, want ok: %s", f.Level, f.Message)
	}
}

func TestDoctorWarnsWhenRecordedReviewsPredateTheContract(t *testing.T) {
	// `.pose/policy/` is not machinery, so an update delivers a stricter engine
	// and no statement of when the instance received it. Without the date, work
	// reviewed under the previous contract fails, and the operator sees a wall
	// of failures about closeouts nobody touched.
	root := doctorTrailerFixture(t)
	mustWrite(t, filepath.Join(root, ".pose", "policy", "review.json"),
		`{"schema_version":2,"enabled":true,"profiles":{"spec":"spec-closeout@1"}}`)
	mustWrite(t, filepath.Join(root, ".pose", "review-attestations", "rva-old.json"),
		`{"schema_version":1,"attestation_id":"rva-old","bundle_id":"rvb-old","decision":"approved"}`)

	f, ok := findDoctorFinding(runDoctorJSON(t, root), "review.contract-adoption")
	if !ok {
		t.Fatal("expected a review.contract-adoption finding")
	}
	if f.Level != "warn" {
		t.Errorf("level=%q, want warn: %s", f.Level, f.Message)
	}
	// Every registered contract is named, so adding one to the registry is
	// enough to have it reported here.
	for _, contract := range posemodel.ReviewContracts() {
		if !strings.Contains(f.Message, contract.ID) {
			t.Errorf("the finding does not name contract %q: %q", contract.ID, f.Message)
		}
	}

	// A date recorded for every contract leaves nothing to report. The dates are
	// built from the registry rather than listed here: a fixture that names the
	// contracts it expects would pass whatever the registry holds, which is the
	// property under test.
	entries := []string{}
	for _, contract := range posemodel.ReviewContracts() {
		entries = append(entries, fmt.Sprintf("%q:%q", contract.ID, "2026-08-13"))
	}
	mustWrite(t, filepath.Join(root, ".pose", "policy", "review.json"),
		`{"schema_version":2,"enabled":true,"profiles":{"spec":"spec-closeout@1"},`+
			`"contract_adoptions":{`+strings.Join(entries, ",")+`}}`)
	f, ok = findDoctorFinding(runDoctorJSON(t, root), "review.contract-adoption")
	if !ok {
		t.Fatal("expected a review.contract-adoption finding")
	}
	if f.Level != "ok" {
		t.Errorf("level=%q, want ok once every contract has a date: %s", f.Level, f.Message)
	}

	// And an instance with no recorded review has nothing to grandfather, so it
	// is not nagged into setting dates it does not need.
	if err := os.Remove(filepath.Join(root, ".pose", "review-attestations", "rva-old.json")); err != nil {
		t.Fatal(err)
	}
	mustWrite(t, filepath.Join(root, ".pose", "policy", "review.json"),
		`{"schema_version":2,"enabled":true,"profiles":{"spec":"spec-closeout@1"}}`)
	if f, ok := findDoctorFinding(runDoctorJSON(t, root), "review.contract-adoption"); !ok || f.Level != "ok" {
		t.Errorf("a fresh instance was asked for an adoption date: %+v", f)
	}
}
