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
	if !strings.Contains(f.Message, "c2") || !strings.Contains(f.Message, "e2e") {
		t.Errorf("the finding does not name the criterion nothing can satisfy: %q", f.Message)
	}
	// Naming the satisfiable one would be the finding inverted.
	if strings.Contains(f.Message, "c1") {
		t.Errorf("the finding names a criterion a produced class satisfies: %q", f.Message)
	}
	if f.Hint == "" {
		t.Error("the finding names a problem with no remedy")
	}
}

// This is the case that motivated it: the class stays valid and demanded, and
// the only check that produced it starts producing something else.
func TestMovingTheOnlyProducerIsReported(t *testing.T) {
	root := classProducerFixture(t, "unit")
	if f, _ := findDoctorFinding(runDoctorJSON(t, root), "validate.class-producers"); !strings.Contains(f.Message, "c2") {
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
	if !strings.Contains(f.Message, "c1") {
		t.Errorf("the finding does not name the criterion that lost its only producer: %q", f.Message)
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

// `evidence_classes` is a disjunction: the attestation cites one of them and the
// first match satisfies the criterion. Reporting classes one by one said `e2e`
// was unproduced while every criterion listing it also accepted `unit`, which is
// produced — a true statement naming nothing to fix, which is how a check earns
// being ignored.
func TestACriterionAcceptingAProducedClassIsNotReported(t *testing.T) {
	root := doctorTrailerFixture(t)
	mustWrite(t, filepath.Join(root, ".pose", "policy", "review.json"),
		`{"schema_version":2,"enabled":true,"profiles":{"spec":"acme@1"}}`)
	mustWrite(t, filepath.Join(root, ".pose", "review-profiles", "acme.json"),
		`{"schema_version":2,"id":"acme","version":1,"scope":"spec","criteria":[`+
			`{"id":"c1","description":"d","evidence_classes":["e2e","unit"]}]}`)
	mustWrite(t, filepath.Join(root, ".pose", "indexes", "validation-matrix.json"),
		`{"defaults":{"mode":"strict"},"deliveryProfiles":{},"stacks":{"go":{"checks":[`+
			`{"name":"test","program":"go","args":["test"],"severity":"required","evidenceClass":"unit"}]}}}`)

	f, ok := findDoctorFinding(runDoctorJSON(t, root), "validate.class-producers")
	if !ok || f.Level != "ok" {
		t.Errorf("a criterion accepting a produced class was reported: %+v", f)
	}
}

// A profile selected by a language the repository does not have can never apply
// to any component, so its criteria are not ones anyone here will be held to.
func TestAProfileForAnAbsentLanguageIsNotReported(t *testing.T) {
	root := doctorTrailerFixture(t)
	mustWrite(t, filepath.Join(root, ".pose", "policy", "review.json"),
		`{"schema_version":2,"enabled":true,"profiles":{"spec":"acme@1"},"overlay_profiles":["web@1"]}`)
	mustWrite(t, filepath.Join(root, ".pose", "review-profiles", "acme.json"),
		`{"schema_version":2,"id":"acme","version":1,"scope":"spec","criteria":[`+
			`{"id":"c1","description":"d","evidence_classes":["unit"]}]}`)
	mustWrite(t, filepath.Join(root, ".pose", "review-profiles", "web.json"),
		`{"schema_version":2,"id":"web","version":1,"scope":"spec","selectors":{"languages":["typescript"]},`+
			`"criteria":[{"id":"a11y","description":"d","evidence_classes":["a11y"]}]}`)
	mustWrite(t, filepath.Join(root, ".pose", "indexes", "validation-matrix.json"),
		`{"defaults":{"mode":"strict"},"deliveryProfiles":{},"stacks":{"go":{"checks":[`+
			`{"name":"test","program":"go","args":["test"],"severity":"required","evidenceClass":"unit"}]}}}`)
	mustWrite(t, filepath.Join(root, ".pose", "indexes", "module-metadata.json"),
		`{"schemaVersion":1,"defaults":{},"modules":{"api":{"domain":"go","criticality":"medium"}}}`)

	if f, ok := findDoctorFinding(runDoctorJSON(t, root), "validate.class-producers"); !ok || f.Level != "ok" {
		t.Errorf("a profile for a language this repository does not have was reported: %+v", f)
	}

	// And it is reported once such a component exists — the skip is about what
	// the repository contains, not about the profile being ignorable.
	mustWrite(t, filepath.Join(root, ".pose", "indexes", "module-metadata.json"),
		`{"schemaVersion":1,"defaults":{},"modules":{"api":{"domain":"go","criticality":"medium"},"web":{"domain":"typescript","criticality":"medium"}}}`)
	f, ok := findDoctorFinding(runDoctorJSON(t, root), "validate.class-producers")
	if !ok || f.Level != "warn" || !strings.Contains(f.Message, "a11y") {
		t.Errorf("adding a component in that language did not bring the criterion back: %+v", f)
	}
}
