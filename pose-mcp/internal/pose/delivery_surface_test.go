package pose

import (
	"strings"
	"testing"
)

func TestDeliveryTargetsRequireExactTypedFrontmatterAndBodyRefs(t *testing.T) {
	spec := Spec{Slug: "alpha", Delivers: []string{"surface:dashboard", "capability:runner"}, Body: "### Delivery targets\n- surface:dashboard module:web profile:web-ui entrypoint:web/main.go\n- capability:runner module:internal/runner profile:composed-capability entrypoint:cmd/app/main.go\n\n### Risks\nnone\n"}
	targets, found, err := ParseDeliveryTargets(spec)
	if err != nil || !found || len(targets) != 2 || targets[1].Ref != "surface:dashboard" {
		t.Fatalf("targets=%+v found=%t err=%v", targets, found, err)
	}
	spec.Delivers = []string{"surface:dashboard"}
	if _, _, err := ParseDeliveryTargets(spec); err == nil || !strings.Contains(err.Error(), "exact same refs") {
		t.Fatalf("frontmatter/body drift accepted: %v", err)
	}
}

func TestRoadmapCriteriaRejectRawCommandsAndRequireRegisteredRefs(t *testing.T) {
	rm := Roadmap{Slug: "alpha", Body: "## Cut criteria\n- C1: surface:dashboard check:web-reachability evidence:e2e\n- C2: `npm test`\n- C3: prose only\n"}
	criteria, errors := ParseRoadmapCriteria(rm)
	if len(criteria) != 1 || !criteria[0].Passed && len(criteria[0].Reasons) > 0 || len(errors) != 2 {
		t.Fatalf("criteria=%+v errors=%v", criteria, errors)
	}
}

func TestDeliverySurfaceFailsGreenArtifactWithUnreachableSurface(t *testing.T) {
	base := BuildDeliveryIntegrity(
		[]Spec{{Slug: "alpha", Status: "done"}},
		[]ArtifactClaim{{Spec: "alpha", Action: "modified", Path: "web/view.go"}},
		[]ChangeSet{{ID: "cs-1", Spec: "alpha", Paths: []ObservedPath{{Action: "modified", Path: "web/view.go"}}}},
		[]string{"web/view.go"}, ArtifactPolicy{GovernedRoots: []string{"web"}},
	)
	target := DeliveryTarget{Spec: "alpha", Ref: "surface:dashboard", Kind: "surface", ID: "dashboard", Module: "web", Profile: "web-ui", Entrypoint: "cmd/app/main.go"}
	profiles := map[string]DeliveryProfile{"web-ui": {Kind: "surface", RequiredEvidenceClasses: []string{"reachability"}, AnyEvidenceClasses: []string{"integration", "e2e"}}}
	graph := BuildDeliverySurface(base, []Spec{{Slug: "alpha"}}, []DeliveryTarget{target}, []DeliveryValidationResult{{ID: "unit", Module: "web", Check: "unit", EvidenceClass: "unit", Severity: "required", Outcome: "pass", ProvenanceDigest: base.ProvenanceDigest}}, nil, profiles, DeliveryPolicy{Severities: map[string]string{"surface-without-reachability": "error"}})
	found := false
	for _, finding := range graph.Findings {
		if finding.Code == "surface-without-reachability" && finding.Severity == "error" {
			found = true
		}
	}
	if !found {
		t.Fatalf("unreachable surface passed: %+v", graph.Findings)
	}
}

func TestDeliverySurfacePassesCurrentReachabilityAndIntegration(t *testing.T) {
	base := BuildDeliveryIntegrity([]Spec{{Slug: "alpha"}}, []ArtifactClaim{{Spec: "alpha", Action: "modified", Path: "web/view.go"}}, []ChangeSet{{ID: "cs-1", Spec: "alpha", Paths: []ObservedPath{{Action: "modified", Path: "web/view.go"}}}}, []string{"web/view.go"}, ArtifactPolicy{})
	target := DeliveryTarget{Spec: "alpha", Ref: "surface:dashboard", Kind: "surface", ID: "dashboard", Module: "web", Profile: "web-ui", Entrypoint: "cmd/app/main.go"}
	profiles := map[string]DeliveryProfile{"web-ui": {Kind: "surface", RequiredEvidenceClasses: []string{"reachability"}, AnyEvidenceClasses: []string{"integration", "e2e"}}}
	results := []DeliveryValidationResult{{ID: "reach", Module: "web", Check: "web-reach", EvidenceClass: "reachability", Severity: "required", Outcome: "pass", ProvenanceDigest: base.ProvenanceDigest}, {ID: "e2e", Module: "web", Check: "web-e2e", EvidenceClass: "e2e", Severity: "required", Outcome: "pass", ProvenanceDigest: base.ProvenanceDigest}}
	graph := BuildDeliverySurface(base, []Spec{{Slug: "alpha"}}, []DeliveryTarget{target}, results, nil, profiles, DeliveryPolicy{})
	for _, finding := range graph.Findings {
		if finding.Code == "surface-without-reachability" {
			t.Fatalf("valid surface failed: %+v", graph.Findings)
		}
	}
	if len(graph.Paths[target.Ref]) < 4 {
		t.Fatalf("missing explainable path: %v", graph.Paths)
	}
}

func TestDeferredIntegrationDoesNotAssertDeliveryOrSatisfyRoadmapCriterion(t *testing.T) {
	specs := []Spec{
		{Slug: "alpha", Status: "in-progress", Body: "## 2. Requirements\n- R1: defer the dashboard integration.\n\n## 6. Validation\n\n### Requirement trace\n- R1 [deferred-integration: spec:beta] surface:dashboard\n"},
		{Slug: "beta", Status: "planned"},
	}
	base := BuildDeliveryIntegrity(specs, nil, nil, nil, ArtifactPolicy{})
	target := DeliveryTarget{Spec: "alpha", Ref: "surface:dashboard", Kind: "surface", ID: "dashboard", Module: "web", Profile: "web-ui", Entrypoint: "cmd/app/main.go"}
	profiles := map[string]DeliveryProfile{"web-ui": {Kind: "surface", RequiredEvidenceClasses: []string{"reachability"}}}
	roadmaps := []Roadmap{{Slug: "release", Body: "## Cut criteria\n- C1: surface:dashboard\n"}}
	graph := BuildDeliverySurface(base, specs, []DeliveryTarget{target}, nil, roadmaps, profiles, DeliveryPolicy{})
	for _, edge := range graph.Edges {
		if edge.From == "spec:alpha" && edge.To == "delivery:surface:dashboard" && edge.Type == "delivers" {
			t.Fatal("deferred surface was asserted as delivered")
		}
	}
	if len(graph.RoadmapCriteria) != 1 || graph.RoadmapCriteria[0].Passed || !strings.Contains(strings.Join(graph.RoadmapCriteria[0].Reasons, " "), "deferred delivery ref") {
		t.Fatalf("deferred surface satisfied roadmap criterion: %+v", graph.RoadmapCriteria)
	}
	for _, finding := range graph.Findings {
		if finding.Code == "surface-without-reachability" || finding.Code == "surface-without-provenance" {
			t.Fatalf("valid deferral produced delivery finding: %+v", graph.Findings)
		}
	}
}

func TestReviewBundleDeliveryChangeDoesNotStaleUnrelatedClosedScopes(t *testing.T) {
	alphaSet := ChangeSet{ID: "cs-alpha", Spec: "alpha", Paths: []ObservedPath{{Action: "modified", Path: "svc/alpha.go"}}, DiffDigest: "sha256:alpha", ResolvedHead: "alpha-head"}
	alphaGraph := BuildDeliveryIntegrity([]Spec{{Slug: "alpha"}}, []ArtifactClaim{{Spec: "alpha", Action: "modified", Path: "svc/alpha.go"}}, []ChangeSet{alphaSet}, []string{"svc/alpha.go"}, ArtifactPolicy{})
	alphaScoped := ScopedDeliveryProvenanceDigest(alphaGraph, "alpha")
	betaSet := ChangeSet{ID: "cs-beta", Spec: "beta", Paths: []ObservedPath{{Action: "modified", Path: "svc/beta.go"}}, DiffDigest: "sha256:beta", ResolvedHead: "beta-head"}
	combined := BuildDeliveryIntegrity(
		[]Spec{{Slug: "alpha", Status: "done"}, {Slug: "beta", Status: "in-progress"}},
		[]ArtifactClaim{{Spec: "alpha", Action: "modified", Path: "svc/alpha.go"}, {Spec: "beta", Action: "modified", Path: "svc/beta.go"}},
		[]ChangeSet{alphaSet, betaSet}, []string{"svc/alpha.go", "svc/beta.go"}, ArtifactPolicy{},
	)
	targets := []DeliveryTarget{
		{Spec: "alpha", Ref: "contract:alpha", Kind: "contract", Module: "svc", Profile: "api-contract", Entrypoint: "svc/alpha.go"},
		{Spec: "beta", Ref: "contract:beta", Kind: "contract", Module: "svc", Profile: "api-contract", Entrypoint: "svc/beta.go"},
	}
	results := []DeliveryValidationResult{{ID: "alpha-integration", Module: "svc", Check: "integration", EvidenceClass: "integration", Severity: "required", Outcome: "pass", GitHead: "alpha-head", ScopeProvenance: map[string]string{"alpha": alphaScoped}}}
	profiles := map[string]DeliveryProfile{"api-contract": {Kind: "contract", RequiredEvidenceClasses: []string{"integration"}}}
	graph := BuildDeliverySurface(combined, []Spec{{Slug: "alpha", Status: "done"}, {Slug: "beta", Status: "in-progress"}}, targets, results, nil, profiles, DeliveryPolicy{})
	for _, finding := range graph.Findings {
		if finding.Spec == "alpha" && (finding.Code == "stale-evidence" || finding.Code == "unconsumed-capability") {
			t.Fatalf("unrelated beta delivery invalidated alpha evidence: %+v", finding)
		}
	}
}

func TestReviewBundlePreservesLegacyEvidenceOnlyForClosedScopes(t *testing.T) {
	set := ChangeSet{ID: "cs-alpha", Spec: "alpha", Paths: []ObservedPath{{Action: "modified", Path: "svc/alpha.go"}}}
	base := BuildDeliveryIntegrity([]Spec{{Slug: "alpha"}}, []ArtifactClaim{{Spec: "alpha", Action: "modified", Path: "svc/alpha.go"}}, []ChangeSet{set}, []string{"svc/alpha.go"}, ArtifactPolicy{})
	target := DeliveryTarget{Spec: "alpha", Ref: "contract:alpha", Kind: "contract", Module: "svc", Profile: "api-contract", Entrypoint: "svc/alpha.go"}
	result := DeliveryValidationResult{ID: "legacy", Module: "svc", Check: "integration", EvidenceClass: "integration", Severity: "required", Outcome: "pass", GeneratedAt: "2026-08-01T00:00:00Z", ProvenanceDigest: "sha256:legacy-module-wide"}
	profiles := map[string]DeliveryProfile{"api-contract": {Kind: "contract", RequiredEvidenceClasses: []string{"integration"}}}
	closed := BuildDeliverySurface(base, []Spec{{Slug: "alpha", Status: "done"}}, []DeliveryTarget{target}, []DeliveryValidationResult{result}, nil, profiles, DeliveryPolicy{})
	for _, finding := range closed.Findings {
		if finding.Spec == "alpha" && (finding.Code == "stale-evidence" || finding.Code == "unconsumed-capability") {
			t.Fatalf("legacy closed evidence was retroactively invalidated: %+v", finding)
		}
	}
	open := BuildDeliverySurface(base, []Spec{{Slug: "alpha", Status: "in-progress"}}, []DeliveryTarget{target}, []DeliveryValidationResult{result}, nil, profiles, DeliveryPolicy{})
	stale := false
	for _, finding := range open.Findings {
		if finding.Spec == "alpha" && finding.Code == "stale-evidence" {
			stale = true
		}
	}
	if !stale {
		t.Fatal("legacy module-wide evidence was accepted for an open scope")
	}
}
