package cli

// Diagnostics for the two failure modes where POSE silently reduces what it
// knows and reports a downstream symptom instead of the upstream loss (spec
// pose-diagnose-invisible-governance-failures). Each check is asserted in both
// directions: warn on a fixture that reproduces the real failure, ok on one
// that does not, so a check that observes nothing cannot pass by accident.

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestDoctorWarnsOnUnproducibleEvidenceClass(t *testing.T) {
	root := doctorTrailerFixture(t)
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
