package pose

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// A milestone approved through the federated path, its roadmap still open.
func closedMilestoneFixture(t *testing.T) (string, Store) {
	t.Helper()
	root, s, _, _ := federatedTestMilestoneConsumer(t)
	now := time.Date(2026, 9, 26, 18, 0, 0, 0, time.UTC)
	child, err := s.SealReviewBundle("spec:backend", now)
	if err != nil {
		t.Fatal(err, child.Blockers)
	}
	attestChildReview(t, s, child, now.Add(time.Minute), "approved")
	milestone, err := s.SealReviewBundle("milestone:program/core", now.Add(2*time.Minute))
	if err != nil {
		t.Fatal(err, milestone.Blockers)
	}
	attestChildReview(t, s, milestone, now.Add(3*time.Minute), "approved")
	return root, s
}

func editRoadmapFixture(t *testing.T, root, from, to string) {
	t.Helper()
	path := filepath.Join(root, ".pose/roadmaps/program.md")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), from) {
		t.Fatalf("the roadmap fixture has no %q", from)
	}
	if err := os.WriteFile(path, []byte(strings.Replace(string(raw), from, to, 1)), 0o644); err != nil {
		t.Fatal(err)
	}
	federatedTestGit(t, root, "add", "--all")
	federatedTestGit(t, root, "commit", "-q", "-m", "roadmap edit")
}

func TestClosingARoadmapKeepsItsMilestoneApproved(t *testing.T) {
	root, s := closedMilestoneFixture(t)
	editRoadmapFixture(t, root, "status: active", "status: done")
	verification, err := s.VerifyReviewBundle("milestone:program/core")
	if err != nil {
		t.Fatal(err)
	}
	if verification.State != "closed" || !verification.Fresh || !verification.Approved {
		t.Fatalf("closing the roadmap staled its milestone: %+v", verification)
	}
}

func TestEditingAClosedRoadmapStalesItsMilestone(t *testing.T) {
	root, s := closedMilestoneFixture(t)
	editRoadmapFixture(t, root, "status: active", "status: done")
	editRoadmapFixture(t, root, "- after: core", "- after: core\n- exit: a gate added after the review")
	verification, err := s.VerifyReviewBundle("milestone:program/core")
	if err != nil {
		t.Fatal(err)
	}
	if verification.Approved || verification.State != "superseded" {
		t.Fatalf("an edit past the lifecycle kept the approval: %+v", verification)
	}
}

// Two revisions of the roadmap, and the manifests that record them.
func lifecycleManifests(t *testing.T) (Store, string, FederatedRoadmapManifest, FederatedRoadmapManifest) {
	t.Helper()
	root, s, _, _ := federatedTestMilestoneConsumer(t)
	before := gitHeadAtRoot(root)
	editRoadmapFixture(t, root, "status: active", "status: done")
	after := gitHeadAtRoot(root)
	coordinator := ArtifactRef{Project: "coordinator", Kind: "milestone", Slug: "program", Milestone: "core"}
	dependency := func(revision, digest, status string) FederatedDependency {
		return FederatedDependency{Ref: "xref:coordinator/roadmap:program", Identity: ArtifactRef{Project: "coordinator", Kind: "roadmap", Slug: "program"}, Status: status, SourceRevision: revision, SourceDigest: digest}
	}
	sealed := FederatedRoadmapManifest{SchemaVersion: 1, Coordinator: coordinator, CoordinatorRevision: before, TrustPolicyDigest: "trust",
		Dependencies: []FederatedDependency{dependency(before, "old", "done")}, Ready: true, Blockers: []string{}, Digest: "sealed"}
	current := sealed
	current.CoordinatorRevision, current.Digest = after, "current"
	current.Dependencies = []FederatedDependency{dependency(after, "new", "done")}
	return s, before, sealed, current
}

func TestManifestStandsOverALifecycleChange(t *testing.T) {
	s, _, sealed, current := lifecycleManifests(t)
	if !s.federatedManifestStands(&sealed, &current) {
		t.Fatal("a lifecycle change of the roadmap staled the manifest")
	}
	if !s.federatedManifestStands(&sealed, &sealed) || !s.federatedManifestStands(nil, nil) {
		t.Fatal("identical manifests must stand")
	}
}

func TestManifestStandsOverACompletedDependency(t *testing.T) {
	s, _, sealed, current := lifecycleManifests(t)
	sealed.Dependencies[0].Status = "active"
	sealed.Blockers, sealed.Ready = []string{"xref:coordinator/roadmap:program:not-done"}, false
	if !s.federatedManifestStands(&sealed, &current) {
		t.Fatal("a dependency reaching done staled the manifest")
	}
	kept := sealed
	kept.Blockers = []string{"xref:coordinator/roadmap:program:not-done", "xref:source/roadmap:component:consumer-trust-stale"}
	if s.federatedManifestStands(&kept, &current) {
		t.Fatal("a blocker the dependency did not clear was excused")
	}
}

func TestManifestDifferencesThatStale(t *testing.T) {
	s, before, sealed, base := lifecycleManifests(t)
	copied := func() FederatedRoadmapManifest {
		current := base
		current.Dependencies = append([]FederatedDependency{}, base.Dependencies...)
		return current
	}
	for name, change := range map[string]func(*FederatedRoadmapManifest){
		"trust": func(m *FederatedRoadmapManifest) { m.TrustPolicyDigest = "other" },
		"new blocker": func(m *FederatedRoadmapManifest) {
			m.Blockers = []string{"xref:source/roadmap:component:consumer-trust-stale"}
		},
		"coordinator":        func(m *FederatedRoadmapManifest) { m.Coordinator.Milestone = "later" },
		"unknown revision":   func(m *FederatedRoadmapManifest) { m.CoordinatorRevision = strings.Repeat("0", len(before)) },
		"status left done":   func(m *FederatedRoadmapManifest) { m.Dependencies[0].Status = "active" },
		"another project":    func(m *FederatedRoadmapManifest) { m.Dependencies[0].Identity.Project = "source" },
		"a spec":             func(m *FederatedRoadmapManifest) { m.Dependencies[0].Identity.Kind = "spec" },
		"another evidence":   func(m *FederatedRoadmapManifest) { m.Dependencies[0].ReviewBundleID = "other" },
		"fewer dependencies": func(m *FederatedRoadmapManifest) { m.Dependencies = nil },
	} {
		current := copied()
		change(&current)
		if s.federatedManifestStands(&sealed, &current) {
			t.Errorf("%s: the manifest stood over the difference", name)
		}
	}
}

func TestStatusIsOnlyTheOneInFrontMatter(t *testing.T) {
	raw := "---\nslug: a\nstatus: done\ncompleted_at: 2026-11-01\n---\n\nstatus: body\n"
	if got := withoutStatus(raw); got != "---\nslug: a\ncompleted_at: 2026-11-01\n---\n\nstatus: body\n" {
		t.Fatalf("%q", got)
	}
	if got := withoutStatus("status: done\n"); got != "status: done\n" {
		t.Fatalf("no front matter: %q", got)
	}
	padded := "---\nslug: a\n ---\nstatus: reviewed\n---\n"
	if got := withoutStatus(padded); got != padded {
		t.Fatalf("the parser's delimiter was not honoured: %q", got)
	}
}

func TestRoadmapStatusOtherThanDoneStales(t *testing.T) {
	root, s := closedMilestoneFixture(t)
	editRoadmapFixture(t, root, "status: active", "status: blocked")
	verification, err := s.VerifyReviewBundle("milestone:program/core")
	if err != nil {
		t.Fatal(err)
	}
	if verification.Approved {
		t.Fatalf("active to blocked kept the approval: %+v", verification)
	}
	root, s = closedMilestoneFixture(t)
	editRoadmapFixture(t, root, "status: active", "status: done")
	editRoadmapFixture(t, root, "status: done", "status: active")
	if verification, err = s.VerifyReviewBundle("milestone:program/core"); err != nil {
		t.Fatal(err)
	}
	if !verification.Approved {
		t.Fatalf("the sealed bytes again must stand: %+v", verification)
	}
	identity := ArtifactRef{Kind: "roadmap", Slug: "program"}
	before := gitHeadAtRoot(root)
	editRoadmapFixture(t, root, "status: active", "status: done")
	closed := gitHeadAtRoot(root)
	if s.sameRoadmapApartFromLifecycle(identity, closed, before) {
		t.Fatal("reopening a roadmap read as its transition to done")
	}
	editRoadmapFixture(t, root, "status: done", "status: active\ncompleted_at: 2026-11-01")
	if s.sameRoadmapApartFromLifecycle(identity, before, gitHeadAtRoot(root)) {
		t.Fatal("a field the transition does not write was excused")
	}
}

func TestBodyBehindAPaddedDelimiterStales(t *testing.T) {
	root, s := closedMilestoneFixture(t)
	editRoadmapFixture(t, root, "\n---\n", "\n ---\nstatus: the outcome is restricted\n---\n")
	before := gitHeadAtRoot(root)
	editRoadmapFixture(t, root, "status: active", "status: done")
	editRoadmapFixture(t, root, "the outcome is restricted", "the outcome is unrestricted")
	if s.sameRoadmapApartFromLifecycle(ArtifactRef{Kind: "roadmap", Slug: "program"}, before, gitHeadAtRoot(root)) {
		t.Fatal("a body edit behind a padded delimiter read as the transition")
	}
}

// A milestone waits on another roadmap, and its own roadmap's approval rests on
// it: both stand across the bytes the guarded transition writes to each roadmap.
func TestClosingARoadmapKeepsItsParentAndAWaitingMilestone(t *testing.T) {
	root, s, _, _ := federatedTestMilestoneConsumer(t)
	federatedTestWrite(t, root, ".pose/roadmaps/base.md", "---\nslug: base\nstatus: active\n---\n\n## Milestone: ground\n- exit: the ground holds\n")
	federatedTestWrite(t, root, ".pose/roadmaps/program.md", "---\nslug: program\nstatus: active\ndepends_on: base\nconsumes: xref:source/roadmap:component\n---\n\n## Milestone: core\n- specs: backend\n")
	federatedTestGit(t, root, "add", "--all")
	federatedTestGit(t, root, "commit", "-q", "-m", "two roadmaps")
	now := time.Date(2026, 9, 26, 18, 0, 0, 0, time.UTC)
	for i, scope := range []string{"spec:backend", "milestone:program/core"} {
		at := now.Add(time.Duration(i) * time.Minute)
		bundle, err := s.SealReviewBundle(scope, at)
		if err != nil {
			t.Fatal(scope, err, bundle.Blockers)
		}
		if scope != "spec:backend" && !federatedTestHasBlocker(bundle.Payload.FederatedManifest.Blockers, "roadmap:base:not-done") {
			t.Fatalf("the milestone does not wait on base: %+v", bundle.Payload.FederatedManifest.Blockers)
		}
		attestChildReview(t, s, bundle, at.Add(30*time.Second), "approved")
	}
	stands := func(scope string) {
		t.Helper()
		verification, err := s.VerifyReviewBundle(scope)
		if err != nil {
			t.Fatal(scope, err)
		}
		if verification.State != "closed" || !verification.Fresh || !verification.Approved {
			t.Fatalf("%s fell: %+v", scope, verification)
		}
	}
	closeRoadmap := func(name string) {
		t.Helper()
		path := filepath.Join(root, ".pose/roadmaps", name+".md")
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		// The bytes of applyLifecycleDone: a roadmap's close writes its status alone.
		if err := os.WriteFile(path, []byte(strings.Replace(string(raw), "status: active", "status: done", 1)), 0o644); err != nil {
			t.Fatal(err)
		}
		federatedTestGit(t, root, "add", "--all")
		federatedTestGit(t, root, "commit", "-q", "-m", "close "+name)
	}
	closeRoadmap("base")
	stands("milestone:program/core")
	at := now.Add(time.Hour)
	bundle, err := s.SealReviewBundle("roadmap:program", at)
	if err != nil {
		t.Fatal(err, bundle.Blockers)
	}
	attestChildReview(t, s, bundle, at.Add(time.Minute), "approved")
	closeRoadmap("program")
	stands("milestone:program/core")
	stands("roadmap:program")
}
