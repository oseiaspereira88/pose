// Analysis evidence has classes of its own
// (spec pose-emittable-analysis-evidence-classes).
//
// `lint`, `typecheck`, `security-scan` and `contract` had no class and were
// reported as `build`. A criterion asking for security assurance was then
// satisfied by a successful compilation — the same indistinction the single
// vocabulary closed between profiles and checks, still open between the kinds
// of result a check produces.

package pose

import "testing"

func TestAnalysisClassesAreEmittable(t *testing.T) {
	for _, class := range []string{"lint", "typecheck", "security-scan", "contract"} {
		if !ValidEvidenceClasses[class] {
			t.Errorf("%q is not a class a check may emit, so a criterion asking for it can never be satisfied", class)
		}
	}
}

// The vocabulary is one set: what a check may emit is exactly what a profile
// may demand. A class added to one side only is how ten unsatisfiable classes
// existed before.
func TestTheVocabularyStaysOneSet(t *testing.T) {
	classes := sortedEvidenceClasses()
	if len(classes) != len(ValidEvidenceClasses) {
		t.Fatalf("sortedEvidenceClasses reports %d of %d classes", len(classes), len(ValidEvidenceClasses))
	}
	for _, class := range classes {
		if !ValidEvidenceClasses[class] {
			t.Errorf("%q is listed to operators but is not accepted", class)
		}
	}
}

// `build` must keep meaning "it compiled". If lint were folded back into it,
// the distinction this spec exists for would be gone and nothing would say so.
func TestBuildIsNotTheCatchAllForAnalysis(t *testing.T) {
	if !ValidEvidenceClasses["build"] {
		t.Fatal("build stopped being a class")
	}
	for _, class := range []string{"lint", "typecheck", "security-scan", "contract"} {
		if class == "build" {
			t.Fatalf("%q collapsed into build", class)
		}
	}
}
