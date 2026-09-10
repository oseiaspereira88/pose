// A demanded class nothing produces
// (spec pose-report-a-demanded-class-nothing-produces).
//
// `review.evidence-vocabulary` asks whether a class *could* be emitted by some
// check somewhere — it holds the profile to the closed vocabulary. This asks
// whether one actually is, in this repository.
//
// The two came apart when `go vet` moved from `build` to `lint`. `build` stayed
// a perfectly valid class, the selected profiles kept demanding it, and no
// registered check produced it any more. A gate depending on evidence the
// repository does not generate, with nothing saying so — found only because the
// same change added a `go build ./...` check by hand.

package cli

import (
	"path/filepath"
	"strings"
	"testing"
)

// classProducerFixture is an instance whose selected profile demands `unit` and
// `e2e`, with a matrix that produces whatever the caller says.
func classProducerFixture(t *testing.T, produced string) string {
	t.Helper()
	root := doctorTrailerFixture(t)
	mustWrite(t, filepath.Join(root, ".pose", "policy", "review.json"),
		`{"schema_version":2,"enabled":true,"profiles":{"spec":"acme@1"}}`)
	mustWrite(t, filepath.Join(root, ".pose", "review-profiles", "acme.json"),
		`{"schema_version":2,"id":"acme","version":1,"scope":"spec","criteria":[`+
			`{"id":"c1","description":"d","evidence_classes":["unit"]},`+
			`{"id":"c2","description":"d","evidence_classes":["e2e"]}]}`)
	mustWrite(t, filepath.Join(root, ".pose", "indexes", "validation-matrix.json"),
		`{"defaults":{"mode":"strict"},"deliveryProfiles":{},"stacks":{"go":{"checks":[`+
			`{"name":"test","program":"go","args":["test"],"severity":"required","evidenceClass":"`+produced+`"}]}}}`)
	return root
}

func TestDoctorReportsADemandedClassNoCheckProduces(t *testing.T) {
	root := classProducerFixture(t, "unit")

	f, ok := findDoctorFinding(runDoctorJSON(t, root), "validate.class-producers")
	if !ok {
		t.Fatal("no validate.class-producers finding")
	}
	if f.Level != "warn" {
		t.Fatalf("level = %q, want warn: %s", f.Level, f.Message)
	}
	if !strings.Contains(f.Message, "e2e") {
		t.Errorf("the finding does not name the class nothing produces: %q", f.Message)
	}
	// Naming the produced one would be the finding inverted.
	if strings.Contains(f.Message, "unit") {
		t.Errorf("the finding names a class a check does produce: %q", f.Message)
	}
	if f.Hint == "" {
		t.Error("the finding names a problem with no remedy")
	}
}

// This is the case that motivated it: the class stays valid and demanded, and
// the only check that produced it starts producing something else.
func TestMovingTheOnlyProducerIsReported(t *testing.T) {
	root := classProducerFixture(t, "unit")
	if f, _ := findDoctorFinding(runDoctorJSON(t, root), "validate.class-producers"); !strings.Contains(f.Message, "e2e") {
		t.Fatalf("fixture does not start from the expected state: %q", f.Message)
	}

	// The check now emits a different, still-valid class. Nothing produces
	// `unit` any more, and the profile still demands it.
	mustWrite(t, filepath.Join(root, ".pose", "indexes", "validation-matrix.json"),
		`{"defaults":{"mode":"strict"},"deliveryProfiles":{},"stacks":{"go":{"checks":[`+
			`{"name":"vet","program":"go","args":["vet"],"severity":"optional","evidenceClass":"lint"}]}}}`)
	f, ok := findDoctorFinding(runDoctorJSON(t, root), "validate.class-producers")
	if !ok || f.Level != "warn" {
		t.Fatalf("moving the only producer was not reported: %+v", f)
	}
	if !strings.Contains(f.Message, "unit") {
		t.Errorf("the finding does not name the class that lost its producer: %q", f.Message)
	}
}

// A module override producing the class counts: the question is whether the
// repository generates the evidence, not which stack does.
func TestAModuleOverrideCountsAsAProducer(t *testing.T) {
	root := classProducerFixture(t, "unit")
	mustWrite(t, filepath.Join(root, ".pose", "indexes", "validation-matrix.json"),
		`{"defaults":{"mode":"strict"},"deliveryProfiles":{},"stacks":{"go":{"checks":[`+
			`{"name":"test","program":"go","args":["test"],"severity":"required","evidenceClass":"unit"}]}},`+
			`"moduleOverrides":{"web":{"checks":[{"name":"e2e","program":"npm","args":["run","e2e"],"severity":"required","evidenceClass":"e2e"}]}}}`)

	f, ok := findDoctorFinding(runDoctorJSON(t, root), "validate.class-producers")
	if !ok || f.Level != "ok" {
		t.Errorf("a class produced by a module override was reported as unproduced: %+v", f)
	}
}

// An instance whose profiles demand nothing produces no finding, rather than an
// ok about an empty set.
func TestNoFindingWhenNothingIsDemanded(t *testing.T) {
	root := doctorTrailerFixture(t)
	mustWrite(t, filepath.Join(root, ".pose", "policy", "review.json"),
		`{"schema_version":2,"enabled":true,"profiles":{"spec":"acme@1"}}`)
	mustWrite(t, filepath.Join(root, ".pose", "review-profiles", "acme.json"),
		`{"schema_version":2,"id":"acme","version":1,"scope":"spec","criteria":[{"id":"c1","description":"d"}]}`)

	if f, ok := findDoctorFinding(runDoctorJSON(t, root), "validate.class-producers"); ok {
		t.Errorf("reported on an instance whose profiles demand no class: %+v", f)
	}
}
