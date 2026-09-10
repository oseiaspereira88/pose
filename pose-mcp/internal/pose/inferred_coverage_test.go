// Coverage that rests on an inference is reported
// (spec pose-report-coverage-that-rests-on-inference).
//
// A result from a module containing the target answers for it, because a
// module-wide run usually does exercise its subtree. Usually is the word: a run
// configured to skip a directory inside it does not, and nothing in the result
// says which. That is the one part of the containment rule POSE cannot check.
//
// Rather than tighten it — which would break the layout where a module runs its
// checks once — the inference is made visible, so the cases someone should look
// at can be looked at.

package pose

import (
	"strings"
	"testing"
)

func inferredCoverageGraph(t *testing.T, targetModule, resultModule string) DeliveryIntegrityGraph {
	t.Helper()
	base := BuildDeliveryIntegrity(
		[]Spec{{Slug: "alpha", Status: "done"}},
		[]ArtifactClaim{{Spec: "alpha", Action: "modified", Path: "web/api/handler.go"}},
		[]ChangeSet{{ID: "cs-1", Spec: "alpha", Paths: []ObservedPath{{Action: "modified", Path: "web/api/handler.go"}}}},
		[]string{"web/api/handler.go"}, ArtifactPolicy{},
	)
	target := DeliveryTarget{Spec: "alpha", Ref: "capability:api", Kind: "capability", ID: "api", Module: targetModule, Profile: "composed-capability", Entrypoint: "cmd/app/main.go"}
	profiles := map[string]DeliveryProfile{"composed-capability": {Kind: "capability", RequiredEvidenceClasses: []string{"integration"}}}
	results := []DeliveryValidationResult{{
		ID: "r1", Module: resultModule, Check: "c", EvidenceClass: "integration",
		Severity: "required", Outcome: "pass", ProvenanceDigest: base.ProvenanceDigest,
	}}
	return BuildDeliverySurface(base, []Spec{{Slug: "alpha"}}, []DeliveryTarget{target}, results, nil, profiles, DeliveryPolicy{})
}

func inferredFinding(graph DeliveryIntegrityGraph) (DeliveryIntegrityFinding, bool) {
	for _, finding := range graph.Findings {
		if finding.Code == "inferred-coverage" {
			return finding, true
		}
	}
	return DeliveryIntegrityFinding{}, false
}

func TestCoverageFromAContainingModuleIsReported(t *testing.T) {
	graph := inferredCoverageGraph(t, "web/api", "web")
	finding, ok := inferredFinding(graph)
	if !ok {
		t.Fatal("a target covered only by its containing module's run reported nothing")
	}
	if !strings.Contains(finding.Message, "integration") || !strings.Contains(finding.Message, "web/api") {
		t.Errorf("the finding does not name the class and the target: %q", finding.Message)
	}
	if finding.Remediation == "" {
		t.Error("the finding names a problem with no remedy")
	}
	// Reporting is the point: the target is still covered. Turning this into a
	// blocker is the option that was rejected, because it breaks running a
	// module's checks once.
	for _, other := range graph.Findings {
		if other.Code == "unconnected-artifact" || other.Code == "surface-without-provenance" {
			t.Errorf("the target lost its coverage as well as being reported: %+v", other)
		}
	}
}

// A result naming the target's own module is not an inference, and neither is
// the repository root answering for everything.
func TestExactAndRootCoverageAreNotReported(t *testing.T) {
	if _, ok := inferredFinding(inferredCoverageGraph(t, "web/api", "web/api")); ok {
		t.Error("a result naming the target's own module was reported as inferred")
	}
	if _, ok := inferredFinding(inferredCoverageGraph(t, "web/api", ".")); ok {
		t.Error("a root result was reported as inferred")
	}
}

// One exact result is enough: a target that also has a containing module's run
// is not resting on the inference, so reporting it would be noise.
func TestAnExactResultSilencesTheReport(t *testing.T) {
	base := BuildDeliveryIntegrity(
		[]Spec{{Slug: "alpha", Status: "done"}},
		[]ArtifactClaim{{Spec: "alpha", Action: "modified", Path: "web/api/handler.go"}},
		[]ChangeSet{{ID: "cs-1", Spec: "alpha", Paths: []ObservedPath{{Action: "modified", Path: "web/api/handler.go"}}}},
		[]string{"web/api/handler.go"}, ArtifactPolicy{},
	)
	target := DeliveryTarget{Spec: "alpha", Ref: "capability:api", Kind: "capability", ID: "api", Module: "web/api", Profile: "composed-capability", Entrypoint: "cmd/app/main.go"}
	profiles := map[string]DeliveryProfile{"composed-capability": {Kind: "capability", RequiredEvidenceClasses: []string{"integration"}}}
	results := []DeliveryValidationResult{
		{ID: "r-parent", Module: "web", Check: "c", EvidenceClass: "integration", Severity: "required", Outcome: "pass", ProvenanceDigest: base.ProvenanceDigest},
		{ID: "r-exact", Module: "web/api", Check: "c", EvidenceClass: "integration", Severity: "required", Outcome: "pass", ProvenanceDigest: base.ProvenanceDigest},
	}
	graph := BuildDeliverySurface(base, []Spec{{Slug: "alpha"}}, []DeliveryTarget{target}, results, nil, profiles, DeliveryPolicy{})
	if finding, ok := inferredFinding(graph); ok {
		t.Errorf("a target with a result of its own was reported as inferred: %q", finding.Message)
	}
}

// The classifier itself, on the cases the delivery loop cannot easily reach.
func TestModuleCoverageIsInferredOnlyForContainment(t *testing.T) {
	for _, tc := range []struct {
		module, target string
		want           bool
	}{
		{"web", "web/api", true},
		{"web", "web/api/handlers", true},
		{"web/api", "web/api", false},
		{".", "web/api", false},
		{"web/api", ".", false},
		{"root", "web/api", false},
		{"web", "webhooks/api", false},
		{"web/", "web/api", true},
	} {
		if got := moduleCoverageIsInferred(tc.module, tc.target); got != tc.want {
			t.Errorf("moduleCoverageIsInferred(%q, %q) = %v, want %v", tc.module, tc.target, got, tc.want)
		}
	}
}
