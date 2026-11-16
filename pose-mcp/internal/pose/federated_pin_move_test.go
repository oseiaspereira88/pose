package pose

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

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

// pinMoveRepository is a dependency project whose spec alpha is committed
// three times: sealed, with only a follow-up disposed, and with a changed
// requirement. It returns the store that judges, and the three revisions.
func pinMoveRepository(t *testing.T) (Store, [3]string) {
	t.Helper()
	root := t.TempDir()
	git := func(args ...string) string {
		t.Helper()
		out, err := exec.Command("git", append([]string{"-C", root, "-c", "user.name=fixture", "-c", "user.email=fixture@example.invalid", "-c", "commit.gpgsign=false"}, args...)...).CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v %s", args, err, out)
		}
		return strings.TrimSpace(string(out))
	}
	spec := filepath.Join(root, ".pose", "specs", "2026-01-01-alpha.md")
	write := func(requirement, followup string) string {
		t.Helper()
		body := "---\nslug: alpha\nstatus: done\n---\n# Spec: alpha\n\n## 2. Requirements\n\n- R1: " + requirement + "\n\n## 7. Final Report\n\n### Follow-ups\n\n- " + followup + "\n"
		if err := os.MkdirAll(filepath.Dir(spec), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(spec, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		git("add", "-A")
		git("commit", "-qm", "alpha")
		return git("rev-parse", "HEAD")
	}
	git("init", "-q")
	revs := [3]string{
		write("The engine shall do one thing.", "[open] F1: later"),
		write("The engine shall do one thing.", "[covered: other] F1: done elsewhere"),
		write("The engine shall do two things.", "[covered: other] F1: done elsewhere"),
	}
	resolver := ArtifactResolver{Roots: NewRoots(RootsConfig{Explicit: map[string]string{"proj.engine": root}})}
	return Store{Root: t.TempDir(), FederatedResolver: &resolver}, revs
}

// Spec pose-closed-scope-stands-over-a-moved-federated-pin, R1: a closed
// scope's sealed manifest stands over a dependency of another project that
// moved with the trust pin and whose spec changed only in its Final Report.
func TestAClosedScopeStandsOverAMovedFederatedPin(t *testing.T) {
	store, revs := pinMoveRepository(t)
	sealed := pinMoveManifest("t1", pinMoveDependency(revs[0], "d1"))
	current := pinMoveManifest("t2", pinMoveDependency(revs[1], "d2"))
	if !store.manifestStandsOver(sealed, current) {
		t.Fatal("a dependency that moved only with the pin unapproved the closed scope")
	}
}

// Review round 1 (F1): a dependency whose requirements changed and that was
// approved again, with the same plan digest, still unapproves the closed
// scope.
func TestAMovedFederatedPinWithChangedRequirementsStales(t *testing.T) {
	store, revs := pinMoveRepository(t)
	sealed := pinMoveManifest("t1", pinMoveDependency(revs[0], "d1"))
	current := pinMoveManifest("t2", pinMoveDependency(revs[2], "d3"))
	if store.manifestStandsOver(sealed, current) {
		t.Fatal("a dependency with changed requirements kept the closed scope approved")
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
