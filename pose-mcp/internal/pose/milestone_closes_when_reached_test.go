package pose

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// roadmapDependencyFixture is program, with milestones core and later, depending
// on prereq, a roadmap that is not done.
func roadmapDependencyFixture(t *testing.T) (string, Store) {
	t.Helper()
	root, s, _, _ := federatedTestMilestoneConsumer(t)
	federatedTestWrite(t, root, ".pose/roadmaps/prereq.md", "---\nslug: prereq\nstatus: active\n---\n\n# Prereq\n")
	editRoadmapFixture(t, root, "status: active", "status: active\ndepends_on: prereq")
	return root, s
}

func approveMember(t *testing.T, s Store, at time.Time) {
	t.Helper()
	child, err := s.SealReviewBundle("spec:backend", at)
	if err != nil {
		t.Fatal(err, child.Blockers)
	}
	attestChildReview(t, s, child, at.Add(time.Minute), "approved")
}

func approveCore(t *testing.T, s Store, at time.Time) {
	t.Helper()
	milestone, err := s.SealReviewBundle("milestone:program/core", at)
	if err != nil {
		t.Fatal(err, milestone.Blockers)
	}
	attestChildReview(t, s, milestone, at.Add(time.Minute), "approved")
}

func coreManifests(t *testing.T, s Store) (sealed, current *FederatedRoadmapManifest) {
	t.Helper()
	wider, err := s.milestoneAcceptance("coordinator", "program", "core", *s.FederatedResolver, true)
	if err != nil {
		t.Fatal(err)
	}
	narrow, err := s.FederatedMilestoneAcceptance("coordinator", "program", "core", *s.FederatedResolver)
	if err != nil {
		t.Fatal(err)
	}
	if len(wider.Manifest.Dependencies) <= len(narrow.Manifest.Dependencies) {
		t.Fatalf("the two manifests do not differ: %+v, %+v", wider.Manifest, narrow.Manifest)
	}
	return &wider.Manifest, &narrow.Manifest
}

func TestMilestoneClosesBeforeTheDependencyOfItsRoadmap(t *testing.T) {
	_, s := roadmapDependencyFixture(t)
	report, err := s.FederatedMilestoneAcceptance("coordinator", "program", "core", *s.FederatedResolver)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(strings.Join(report.Blockers, " "), "prereq") {
		t.Fatalf("the dependency of the roadmap gated a milestone: %v", report.Blockers)
	}
	for _, dep := range report.Manifest.Dependencies {
		if dep.Identity.Slug == "prereq" {
			t.Fatalf("the manifest of a milestone carries the dependency of its roadmap: %+v", dep)
		}
	}
	now := time.Date(2026, 9, 26, 18, 0, 0, 0, time.UTC)
	approveMember(t, s, now)
	approveCore(t, s, now.Add(2*time.Minute))
	if eval := verifyAndCloseoutAgree(t, s, "milestone:program/core"); !eval.Approved {
		t.Fatalf("the milestone did not close before the dependency of its roadmap: %+v", eval)
	}
}

func TestRoadmapStillWaitsForItsDependency(t *testing.T) {
	_, s := roadmapDependencyFixture(t)
	report, err := s.FederatedRoadmapAcceptance("coordinator", "program", *s.FederatedResolver)
	if err != nil {
		t.Fatal(err)
	}
	if report.Ready || !strings.Contains(strings.Join(report.Blockers, " "), "roadmap:prereq:not-done") {
		t.Fatalf("the roadmap no longer waits for its dependency: %+v", report)
	}
}

func TestAfterStillBindsAMilestoneToARoadmap(t *testing.T) {
	root, s := roadmapDependencyFixture(t)
	editRoadmapFixture(t, root, "## Milestone: core\n", "## Milestone: core\n- after: roadmap:prereq\n")
	report, err := s.FederatedMilestoneAcceptance("coordinator", "program", "core", *s.FederatedResolver)
	if err != nil {
		t.Fatal(err)
	}
	if report.Ready || !strings.Contains(strings.Join(report.Blockers, " "), "roadmap:prereq:not-done") {
		t.Fatalf("after: no longer binds a milestone to a roadmap: %+v", report)
	}
}

func TestSealedRoadmapEdgeKeepsTheApproval(t *testing.T) {
	_, s := roadmapDependencyFixture(t)
	sealed, current := coreManifests(t, s)
	if !s.federatedManifestStands(sealed, current) {
		t.Fatalf("a manifest sealed with the edge of its roadmap was staled by the rule that moved it:\n%+v\n%+v", sealed, current)
	}
}

func TestSealedRoadmapEdgeStandsWhenTheDependencyCloses(t *testing.T) {
	root, s := roadmapDependencyFixture(t)
	sealed, _ := coreManifests(t, s)
	if !federatedTestHasBlocker(sealed.Blockers, "roadmap:prereq:not-done") {
		t.Fatalf("the sealed manifest does not wait on the dependency: %v", sealed.Blockers)
	}
	federatedTestWrite(t, root, ".pose/roadmaps/prereq.md", "---\nslug: prereq\nstatus: done\n---\n\n# Prereq\n")
	federatedTestGit(t, root, "add", "--all")
	federatedTestGit(t, root, "commit", "-q", "-m", "close prereq")
	report, err := s.FederatedMilestoneAcceptance("coordinator", "program", "core", *s.FederatedResolver)
	if err != nil {
		t.Fatal(err)
	}
	if !s.federatedManifestStands(sealed, &report.Manifest) {
		t.Fatalf("closing the dependency staled the approval:\n%+v\n%+v", sealed, report.Manifest)
	}
}

func TestOtherMissingEdgeStales(t *testing.T) {
	t.Run("an edge the roadmap never declared", func(t *testing.T) {
		_, s := roadmapDependencyFixture(t)
		sealed, current := coreManifests(t, s)
		extra := sealed.Dependencies[0]
		extra.Ref, extra.Identity.Slug = "coordinator/roadmap:ghost", "ghost"
		sealed.Dependencies = append(sealed.Dependencies, extra)
		if s.federatedManifestStands(sealed, current) {
			t.Fatal("an edge nobody declares was excused")
		}
	})
	t.Run("an edge the roadmap dropped", func(t *testing.T) {
		root, s := roadmapDependencyFixture(t)
		sealed, _ := coreManifests(t, s)
		editRoadmapFixture(t, root, "depends_on: prereq\n", "")
		report, err := s.FederatedMilestoneAcceptance("coordinator", "program", "core", *s.FederatedResolver)
		if err != nil {
			t.Fatal(err)
		}
		if s.federatedManifestStands(sealed, &report.Manifest) {
			t.Fatal("an edit that dropped the dependency kept the approval")
		}
	})
	t.Run("an edge the roadmap replaced", func(t *testing.T) {
		root, s := roadmapDependencyFixture(t)
		federatedTestWrite(t, root, ".pose/roadmaps/other.md", "---\nslug: other\nstatus: active\n---\n\n# Other\n")
		sealed, _ := coreManifests(t, s)
		editRoadmapFixture(t, root, "depends_on: prereq", "depends_on: other")
		report, err := s.FederatedMilestoneAcceptance("coordinator", "program", "core", *s.FederatedResolver)
		if err != nil {
			t.Fatal(err)
		}
		if s.federatedManifestStands(sealed, &report.Manifest) {
			t.Fatal("an edit that replaced the dependency kept the approval")
		}
	})
	t.Run("a coordinator that is not a milestone", func(t *testing.T) {
		_, s := roadmapDependencyFixture(t)
		sealed, current := coreManifests(t, s)
		sealed.Coordinator.Kind, current.Coordinator.Kind = "roadmap", "roadmap"
		if s.federatedManifestStands(sealed, current) {
			t.Fatal("the excuse reached a coordinator other than a milestone")
		}
	})
	t.Run("a dependency that changed", func(t *testing.T) {
		root, s := roadmapDependencyFixture(t)
		sealed, _ := coreManifests(t, s)
		federatedTestWrite(t, root, ".pose/roadmaps/prereq.md", "---\nslug: prereq\nstatus: active\n---\n\n# Prereq, edited\n")
		federatedTestGit(t, root, "add", "--all")
		federatedTestGit(t, root, "commit", "-q", "-m", "prerequisite edit")
		report, err := s.FederatedMilestoneAcceptance("coordinator", "program", "core", *s.FederatedResolver)
		if err != nil {
			t.Fatal(err)
		}
		if s.federatedManifestStands(sealed, &report.Manifest) {
			t.Fatal("a sealed dependency that changed was excused")
		}
	})
}

func TestAttentionListsReachedMilestones(t *testing.T) {
	_, s := roadmapDependencyFixture(t)
	reachedOf := func(scope string) []string {
		t.Helper()
		report, err := s.ProjectObligations(ObligationQuery{Scope: scope})
		if err != nil {
			t.Fatal(err)
		}
		found := []string{}
		for _, o := range report.Obligations {
			if o.Rule == "milestone-reached" {
				found = append(found, o.Source.Detail)
				if o.Effects[0].Mode != EffectAdvisory {
					t.Fatalf("a reached milestone restricted a phase: %+v", o)
				}
			}
		}
		return found
	}
	const core = "milestone:program/core"
	for _, scope := range []string{"", "roadmap:program", core, "spec:backend"} {
		if got := reachedOf(scope); len(got) != 1 || got[0] != core {
			t.Fatalf("scope %q: a reached milestone is not listed: %v", scope, got)
		}
	}
	for _, scope := range []string{"milestone:program/later", "spec:pending", "roadmap:prereq"} {
		if got := reachedOf(scope); len(got) != 0 {
			t.Fatalf("scope %q listed a milestone outside it: %v", scope, got)
		}
	}
	now := time.Date(2026, 9, 26, 18, 0, 0, 0, time.UTC)
	approveMember(t, s, now)
	approveCore(t, s, now.Add(2*time.Minute))
	if got := reachedOf(""); len(got) != 0 {
		t.Fatalf("an approved milestone is still listed: %v", got)
	}
}

func TestReachedMilestoneOfADoneRoadmapIsNotListed(t *testing.T) {
	root, s := roadmapDependencyFixture(t)
	editRoadmapFixture(t, root, "status: active", "status: done")
	if got, err := s.ReachedMilestones(""); err != nil || len(got) != 0 {
		t.Fatalf("a done roadmap lists milestones: %v %v", got, err)
	}
}

func TestRoadmapIsReachedWhenEveryMilestoneIsApproved(t *testing.T) {
	root, s := roadmapDependencyFixture(t)
	if s.RoadmapReached("program") {
		t.Fatal("a roadmap with open milestones read reached")
	}
	editRoadmapFixture(t, root, "\n## Milestone: later\n- after: core\n- specs: pending\n", "")
	now := time.Date(2026, 9, 26, 18, 0, 0, 0, time.UTC)
	approveMember(t, s, now)
	approveCore(t, s, now.Add(2*time.Minute))
	if !s.RoadmapReached("program") {
		t.Fatal("a roadmap with every milestone approved did not read reached")
	}
}

func TestStartWarnsOfAnUnclosedAfter(t *testing.T) {
	_, s := roadmapDependencyFixture(t)
	before, err := s.PreviewStart("pending")
	if err != nil {
		t.Fatal(err)
	}
	if len(before.Warnings) != 1 || !strings.Contains(before.Warnings[0], "pose close milestone:program/core") {
		t.Fatalf("start did not report the unclosed milestone: %+v", before)
	}
	now := time.Date(2026, 9, 26, 18, 0, 0, 0, time.UTC)
	approveMember(t, s, now)
	approveCore(t, s, now.Add(2*time.Minute))
	after, err := s.PreviewStart("pending")
	if err != nil {
		t.Fatal(err)
	}
	if len(after.Warnings) != 0 {
		t.Fatalf("start reported a closed milestone: %v", after.Warnings)
	}
	if before.Digest != after.Digest || startPlanDigest(before) != before.Digest {
		t.Fatalf("a warning moved the digest of the reviewed plan: %s, %s", before.Digest, after.Digest)
	}
	if !before.Ready {
		t.Fatalf("a warning refused the start: %+v", before)
	}
}

func TestUncommittedSourceSaysSo(t *testing.T) {
	root, s := roadmapDependencyFixture(t)
	federatedTestWrite(t, root, ".pose/roadmaps/prereq.md", "---\nslug: prereq\nstatus: done\n---\n\n# Prereq\n")
	report, err := s.FederatedRoadmapAcceptance("coordinator", "program", *s.FederatedResolver)
	if err != nil {
		t.Fatal(err)
	}
	blockers := strings.Join(report.Blockers, "; ")
	if !strings.Contains(blockers, "roadmap:prereq:source-artifact-uncommitted: commit the artifact") {
		t.Fatalf("the blocker does not say the artifact has to be committed: %s", blockers)
	}
	for _, dep := range report.Manifest.Dependencies {
		if dep.Identity.Slug == "prereq" && dep.ResolutionState != federatedSourceUncommitted {
			t.Fatalf("the state is not the code: %+v", dep)
		}
	}
}

type deniedFiles struct {
	Files
	suffix string
}

func (f deniedFiles) ReadFile(name string) ([]byte, error) {
	if strings.HasSuffix(name, f.suffix) {
		return nil, fs.ErrPermission
	}
	return f.Files.ReadFile(name)
}

func (f deniedFiles) ReadDir(name string) ([]fs.DirEntry, error) {
	if strings.HasSuffix(name, f.suffix) {
		return nil, fs.ErrPermission
	}
	return f.Files.ReadDir(name)
}

type profileReads struct {
	Files
	count *atomic.Int64
}

func (f profileReads) ReadFile(name string) ([]byte, error) {
	if strings.Contains(name, "review-profiles") {
		f.count.Add(1)
	}
	return f.Files.ReadFile(name)
}

// A review of each candidate took `pose state --attention` from 4 to 212 seconds here.
func TestReachedListRunsNoReview(t *testing.T) {
	_, s := roadmapDependencyFixture(t)
	reads := &atomic.Int64{}
	s.FS = profileReads{Files: NativeFiles(), count: reads}
	cov := &Coverage{Producer: "closeout", State: CoverageStateCurrent}
	items, err := s.milestoneReachedObligations("coordinator", "", cov)
	if err != nil || len(items) != 1 {
		t.Fatalf("the milestone is not listed: %v %v", items, err)
	}
	if reads.Load() != 0 {
		t.Fatalf("listing reached milestones ran a review: %d profile reads", reads.Load())
	}
}

func TestAStaleApprovalIsLeftToTheClose(t *testing.T) {
	root, s := roadmapDependencyFixture(t)
	now := time.Date(2026, 9, 26, 18, 0, 0, 0, time.UTC)
	approveMember(t, s, now)
	approveCore(t, s, now.Add(2*time.Minute))
	editRoadmapFixture(t, root, "## Milestone: core", "Edited.\n\n## Milestone: core")
	cov := &Coverage{Producer: "closeout", State: CoverageStateCurrent}
	if items, err := s.milestoneReachedObligations("coordinator", "", cov); err != nil || len(items) != 0 {
		t.Fatalf("a recorded approval was listed: %v %v", items, err)
	}
	if reached, err := s.ReachedMilestones("backend"); err != nil || len(reached) != 1 {
		t.Fatalf("the close of a member does not see the stale approval: %v %v", reached, err)
	}
}

func TestAnUnreadableApprovalIsNotReached(t *testing.T) {
	_, s := roadmapDependencyFixture(t)
	approveMember(t, s, time.Date(2026, 9, 26, 18, 0, 0, 0, time.UTC))
	s.FS = deniedFiles{Files: NativeFiles(), suffix: "review-attestations"}
	cov := &Coverage{Producer: "closeout", State: CoverageStateCurrent}
	items, err := s.milestoneReachedObligations("coordinator", "", cov)
	if err != nil || len(items) != 0 || cov.State != CoverageStateIncomplete {
		t.Fatalf("a lost read reported a reached milestone: %v %v %+v", items, err, cov)
	}
	s.FS = deniedFiles{Files: NativeFiles(), suffix: "review.json"}
	if reached, err := s.ReachedMilestones(""); err == nil {
		t.Fatalf("a failed review read as no approval: %v", reached)
	}
}

func TestAStartWarningNeverRefuses(t *testing.T) {
	_, s := roadmapDependencyFixture(t)
	want, err := s.PreviewStart("pending")
	if err != nil {
		t.Fatal(err)
	}
	s.FS = deniedFiles{Files: NativeFiles(), suffix: "roadmaps"}
	plan, err := s.PreviewStart("pending")
	if err != nil {
		t.Fatalf("a warning that cannot be read refused the start: %v", err)
	}
	if plan.Digest != want.Digest || len(plan.Warnings) != 1 || !strings.Contains(plan.Warnings[0], "could not be read") {
		t.Fatalf("the plan does not report the lost read: %+v", plan)
	}
	s.FS = deniedFiles{Files: NativeFiles(), suffix: "review.json"}
	plan, err = s.PreviewStart("pending")
	if err != nil {
		t.Fatalf("a lost approval refused the start: %v", err)
	}
	if !strings.Contains(strings.Join(plan.Warnings, " "), "approval could not be read") {
		t.Fatalf("a lost approval read as an unclosed predecessor: %+v", plan.Warnings)
	}
}

func attentionListsCore(t *testing.T, s Store) bool {
	t.Helper()
	cov := &Coverage{Producer: "closeout", State: CoverageStateCurrent}
	items, err := s.milestoneReachedObligations("coordinator", "milestone:program/core", cov)
	if err != nil || cov.State != CoverageStateCurrent {
		t.Fatalf("the read was not complete: %v %+v", err, cov)
	}
	return len(items) == 1
}

func TestALaterRejectionIsNotAnApproval(t *testing.T) {
	_, s := roadmapDependencyFixture(t)
	now := time.Date(2026, 9, 26, 18, 0, 0, 0, time.UTC)
	approveMember(t, s, now)
	bundle, err := s.SealReviewBundle("milestone:program/core", now.Add(2*time.Minute))
	if err != nil {
		t.Fatal(err, bundle.Blockers)
	}
	attestChildReview(t, s, bundle, now.Add(3*time.Minute), "approved")
	if attentionListsCore(t, s) {
		t.Fatal("an approved milestone is listed")
	}
	attestChildReview(t, s, bundle, now.Add(4*time.Minute), "rejected")
	if !attentionListsCore(t, s) {
		t.Fatal("an approval followed by a rejection hid the milestone")
	}
}

func TestAReservedApprovalIsAnApproval(t *testing.T) {
	root, s := roadmapDependencyFixture(t)
	path := filepath.Join(root, ".pose/policy/review.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	policy := strings.Replace(string(raw), `{`, `{"allow_approved_with_reservations":true,`, 1)
	federatedTestWrite(t, root, ".pose/policy/review.json", policy)
	federatedTestGit(t, root, "add", "--all")
	federatedTestGit(t, root, "commit", "-q", "-m", "allow reservations")
	now := time.Date(2026, 9, 26, 18, 0, 0, 0, time.UTC)
	approveMember(t, s, now)
	bundle, err := s.SealReviewBundle("milestone:program/core", now.Add(2*time.Minute))
	if err != nil {
		t.Fatal(err, bundle.Blockers)
	}
	attestChildReview(t, s, bundle, now.Add(3*time.Minute), "approved-with-reservations")
	if eval, err := s.ReviewCheck("milestone:program/core"); err != nil || !eval.Approved {
		t.Fatalf("the fixture is not approved: %+v %v", eval, err)
	}
	if attentionListsCore(t, s) {
		t.Fatal("an approval with reservations read as none")
	}
}

func TestAnAmbiguousRecordIsNotAnApproval(t *testing.T) {
	_, s := roadmapDependencyFixture(t)
	now := time.Date(2026, 9, 26, 18, 0, 0, 0, time.UTC)
	approveMember(t, s, now)
	approveCore(t, s, now.Add(2*time.Minute))
	path := s.reviewAttestationsDir()
	entries, err := os.ReadDir(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		raw, err := os.ReadFile(filepath.Join(path, entry.Name()))
		if err != nil {
			t.Fatal(err)
		}
		edited := strings.Replace(string(raw), `"decision": "approved"`, `"decision": "rejected", "decision": "approved"`, 1)
		if err := os.WriteFile(filepath.Join(path, entry.Name()), []byte(edited), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	cov := &Coverage{Producer: "closeout", State: CoverageStateCurrent}
	items, err := s.milestoneReachedObligations("coordinator", "", cov)
	if err != nil || len(items) != 0 || cov.State != CoverageStateIncomplete {
		t.Fatalf("an ambiguous record read as a clean answer: %v %v %+v", items, err, cov)
	}
}
