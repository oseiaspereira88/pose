package pose

import "testing"

func pinMoveDependency(revision, digest string) FederatedDependency {
	return FederatedDependency{
		Ref: "xref:proj.engine/spec:alpha", Relationship: "prerequisite", Relationships: []string{"prerequisite"},
		Identity:        ArtifactRef{Project: "proj.engine", Kind: "spec", Slug: "alpha"},
		ResolutionState: "resolved", Status: "done", SourceRevision: revision, SourceDigest: digest,
		ReviewBundleID: "rvb-" + digest, ReviewBundleDigest: "sha256:" + digest,
		PlanDigest: "sha256:plan", GoverningContracts: []string{"explicit-judgment"},
	}
}

func pinMoveManifest(trust string, deps ...FederatedDependency) *FederatedRoadmapManifest {
	return &FederatedRoadmapManifest{SchemaVersion: 1, Coordinator: ArtifactRef{Project: "proj.local", Kind: "spec", Slug: "consumer"},
		CoordinatorRevision: "c1", TrustPolicyDigest: trust, Dependencies: deps, Ready: true, Blockers: []string{}, Digest: "sha256:" + trust}
}

// Spec pose-closed-scope-stands-over-a-moved-federated-pin, R1: a closed
// scope's sealed manifest stands over a dependency of another project that
// moved with the trust pin and is still the same commitment.
func TestAClosedScopeStandsOverAMovedFederatedPin(t *testing.T) {
	sealed := pinMoveManifest("t1", pinMoveDependency("r1", "d1"))
	current := pinMoveManifest("t2", pinMoveDependency("r2", "d2"))
	if !(Store{}).manifestStandsOver(sealed, current) {
		t.Fatal("a dependency that moved only with the pin unapproved the closed scope")
	}
}

// R2: anything that changes the commitment, or a current blocker naming the
// dependency, still unapproves it.
func TestAMovedFederatedPinThatChangesTheCommitmentStales(t *testing.T) {
	for name, change := range map[string]func(*FederatedDependency, *FederatedRoadmapManifest){
		"plan":         func(d *FederatedDependency, _ *FederatedRoadmapManifest) { d.PlanDigest = "sha256:other-plan" },
		"contracts":    func(d *FederatedDependency, _ *FederatedRoadmapManifest) { d.GoverningContracts = []string{"other"} },
		"status":       func(d *FederatedDependency, _ *FederatedRoadmapManifest) { d.Status = "in-progress" },
		"resolution":   func(d *FederatedDependency, _ *FederatedRoadmapManifest) { d.ResolutionState = "unknown-project" },
		"identity":     func(d *FederatedDependency, _ *FederatedRoadmapManifest) { d.Identity.Slug = "beta" },
		"relationship": func(d *FederatedDependency, _ *FederatedRoadmapManifest) { d.Relationship = "after" },
		"no review":    func(d *FederatedDependency, _ *FederatedRoadmapManifest) { d.ReviewBundleID = "" },
		"blocker": func(d *FederatedDependency, m *FederatedRoadmapManifest) {
			m.Blockers = []string{d.Ref + ":review-not-approved"}
			m.Ready = false
		},
	} {
		t.Run(name, func(t *testing.T) {
			sealed := pinMoveManifest("t1", pinMoveDependency("r1", "d1"))
			dep := pinMoveDependency("r2", "d2")
			current := pinMoveManifest("t2")
			change(&dep, current)
			current.Dependencies = []FederatedDependency{dep}
			if (Store{}).manifestStandsOver(sealed, current) {
				t.Fatalf("a moved pin that changed the %s kept the closed scope approved", name)
			}
		})
	}
}

// R3: the trust policy digest follows a moved pin only when the dependencies
// were judged; a manifest whose only change is the trust policy stales.
func TestATrustPolicyChangeAloneStales(t *testing.T) {
	sealed := pinMoveManifest("t1", pinMoveDependency("r1", "d1"))
	current := pinMoveManifest("t2", pinMoveDependency("r1", "d1"))
	if (Store{}).manifestStandsOver(sealed, current) {
		t.Fatal("a changed trust policy with no moved dependency kept the closed scope approved")
	}
}
