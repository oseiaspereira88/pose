// Evidence answers downward, not upward
// (spec pose-component-evidence-is-not-inherited-upward).
//
// `moduleMatchesTarget` accepted a prefix in either direction, so a result from
// `site/api` answered for a target at `site`. It covers one directory of the
// component and was accepted as covering all of it — the same shape as evidence
// from a sibling, which `pose-evidence-scoped-to-component` already refuses:
// real evidence, of the right class, silent about most of what it was asked
// about.
//
// The other direction stays. A module-wide run does exercise its subtree, and
// running checks once at the module root is how nearly every project is laid
// out.

package pose

import "testing"

func TestModuleMatchesTargetAnswersDownwardOnly(t *testing.T) {
	for _, tc := range []struct {
		name           string
		module, target string
		want           bool
	}{
		{"the same module", "site/api", "site/api", true},
		{"a module-wide run covers a target inside it", "site", "site/api", true},
		{"two levels down is still inside", "site", "site/api/handlers", true},
		{"a result from one directory does not answer for the whole component", "site/api", "site", false},
		{"nor from two levels down", "site/api/handlers", "site", false},
		{"siblings never answer for each other", "site/api", "site/web", false},
		{"a root result answers for everything", ".", "site/api", true},
		{"and a root target is answered by anything", "site/api", ".", true},
		{"`root` is spelled that way too", "root", "site/api", true},
		{"a name that merely shares a prefix is not inside", "site", "sitemap/api", false},
		{"trailing slashes do not change the answer", "site/", "site/api", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := moduleMatchesTarget(tc.module, tc.target); got != tc.want {
				t.Errorf("moduleMatchesTarget(%q, %q) = %v, want %v", tc.module, tc.target, got, tc.want)
			}
		})
	}
}

// The rule has to hold where it is consumed, not only in isolation: a target at
// `web` whose only evidence sits in `web/api` is a target with no evidence.
func TestDeliveryTargetIsNotSatisfiedByEvidenceFromOneDirectoryInsideIt(t *testing.T) {
	base := BuildDeliveryIntegrity(
		[]Spec{{Slug: "alpha", Status: "done"}},
		[]ArtifactClaim{{Spec: "alpha", Action: "modified", Path: "web/view.go"}},
		[]ChangeSet{{ID: "cs-1", Spec: "alpha", Paths: []ObservedPath{{Action: "modified", Path: "web/view.go"}}}},
		[]string{"web/view.go"}, ArtifactPolicy{},
	)
	target := DeliveryTarget{Spec: "alpha", Ref: "capability:runner", Kind: "capability", ID: "runner", Module: "web", Profile: "composed-capability", Entrypoint: "cmd/app/main.go"}
	profiles := map[string]DeliveryProfile{"composed-capability": {Kind: "capability", RequiredEvidenceClasses: []string{"integration"}}}

	fromInside := []DeliveryValidationResult{{ID: "api-integration", Module: "web/api", Check: "api", EvidenceClass: "integration", Severity: "required", Outcome: "pass", ProvenanceDigest: base.ProvenanceDigest}}
	graph := BuildDeliverySurface(base, []Spec{{Slug: "alpha"}}, []DeliveryTarget{target}, fromInside, nil, profiles, DeliveryPolicy{})
	if satisfied(graph, target.Ref) {
		t.Errorf("a target at %q was satisfied by evidence from %q, which covers one directory of it: %+v", target.Module, fromInside[0].Module, graph.Findings)
	}

	// And the direction that stays: the same result registered for the module
	// itself does answer for it, so the refusal above is about where the
	// evidence came from and not about the fixture being wrong.
	fromTheModule := []DeliveryValidationResult{{ID: "web-integration", Module: "web", Check: "web", EvidenceClass: "integration", Severity: "required", Outcome: "pass", ProvenanceDigest: base.ProvenanceDigest}}
	graph = BuildDeliverySurface(base, []Spec{{Slug: "alpha"}}, []DeliveryTarget{target}, fromTheModule, nil, profiles, DeliveryPolicy{})
	if !satisfied(graph, target.Ref) {
		t.Errorf("a module-wide result did not answer for its own target: %+v", graph.Findings)
	}
}

// A target inside a module still takes the module's run, which is the layout
// this must not break: checks run once at the module root.
func TestATargetInsideAModuleTakesTheModulesRun(t *testing.T) {
	base := BuildDeliveryIntegrity(
		[]Spec{{Slug: "alpha", Status: "done"}},
		[]ArtifactClaim{{Spec: "alpha", Action: "modified", Path: "web/api/handler.go"}},
		[]ChangeSet{{ID: "cs-1", Spec: "alpha", Paths: []ObservedPath{{Action: "modified", Path: "web/api/handler.go"}}}},
		[]string{"web/api/handler.go"}, ArtifactPolicy{},
	)
	target := DeliveryTarget{Spec: "alpha", Ref: "capability:api", Kind: "capability", ID: "api", Module: "web/api", Profile: "composed-capability", Entrypoint: "cmd/app/main.go"}
	profiles := map[string]DeliveryProfile{"composed-capability": {Kind: "capability", RequiredEvidenceClasses: []string{"integration"}}}
	results := []DeliveryValidationResult{{ID: "web-integration", Module: "web", Check: "web", EvidenceClass: "integration", Severity: "required", Outcome: "pass", ProvenanceDigest: base.ProvenanceDigest}}

	graph := BuildDeliverySurface(base, []Spec{{Slug: "alpha"}}, []DeliveryTarget{target}, results, nil, profiles, DeliveryPolicy{})
	if !satisfied(graph, target.Ref) {
		t.Errorf("a target at %q was not answered by its module's run at %q: %+v", target.Module, results[0].Module, graph.Findings)
	}
}

// satisfied reports whether the graph attached evidence to the target. It reads
// the edge rather than the absence of a finding: a fixture that never reaches
// the evidence loop also produces no finding, and would then read as satisfied.
func satisfied(graph DeliveryIntegrityGraph, ref string) bool {
	for _, edge := range graph.Edges {
		if edge.Type == "validated-by" && edge.From == "delivery:"+ref {
			return true
		}
	}
	return false
}
