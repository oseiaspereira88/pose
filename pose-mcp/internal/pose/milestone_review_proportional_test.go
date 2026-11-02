package pose

import (
	"strings"
	"testing"
	"time"
)

var reviewFixtureNow = time.Date(2026, 9, 26, 18, 0, 0, 0, time.UTC)

// A milestone with no reader of its own: program/core has the one member backend.
func inheritedFixture(t *testing.T) (string, Store) {
	t.Helper()
	root, s, _, _ := federatedTestMilestoneConsumer(t)
	editRoadmapFixture(t, root, "- specs: backend, xref:source/spec:backend\n", "- specs: backend\n")
	approveMember(t, s, reviewFixtureNow)
	return root, s
}

func TestOneMemberMilestoneInheritsItsSpec(t *testing.T) {
	_, s := inheritedFixture(t)
	verification, err := s.VerifyReviewBundle("milestone:program/core")
	if err != nil {
		t.Fatal(err)
	}
	if !verification.Approved || !verification.Fresh || verification.State != "closed" {
		t.Fatalf("the milestone did not inherit: %+v", verification)
	}
	if !strings.Contains(strings.Join(verification.Warnings, " "), "inherited from spec:backend") {
		t.Fatalf("the verification does not say it is inherited: %v", verification.Warnings)
	}
	if eval := verifyAndCloseoutAgree(t, s, "milestone:program/core"); !eval.Approved {
		t.Fatalf("closeout disagrees: %+v", eval)
	}
	state, err := s.GetCloseoutState("milestone:program/core")
	if err != nil {
		t.Fatal(err)
	}
	if !state.Terminal {
		t.Fatalf("the milestone is not terminal: %+v", state.Blockers)
	}
}

func TestAMilestoneDoesNotInheritAnOpenOrUnapprovedSpec(t *testing.T) {
	root, s, _, _ := federatedTestMilestoneConsumer(t)
	editRoadmapFixture(t, root, "- specs: backend, xref:source/spec:backend\n", "- specs: backend\n")
	if verification, err := s.VerifyReviewBundle("milestone:program/core"); err == nil && verification.Approved {
		t.Fatalf("a milestone inherited a review that does not exist: %+v", verification)
	}
	if verification, err := s.VerifyReviewBundle("milestone:program/later"); err == nil && verification.Approved {
		t.Fatalf("a milestone inherited from a draft: %+v", verification)
	}
}

func TestANegativeReviewOfAMilestoneStands(t *testing.T) {
	_, s := inheritedFixture(t)
	bundle, err := s.SealReviewBundle("milestone:program/core", reviewFixtureNow.Add(2*time.Minute))
	if err != nil {
		t.Fatal(err, bundle.Blockers)
	}
	attestChildReview(t, s, bundle, reviewFixtureNow.Add(3*time.Minute), "changes-requested")
	verification, err := s.VerifyReviewBundle("milestone:program/core")
	if err != nil {
		t.Fatal(err)
	}
	if verification.Approved {
		t.Fatalf("a recorded refusal was inherited away: %+v", verification)
	}
}

func TestARecordedMilestoneApprovalStays(t *testing.T) {
	_, s := inheritedFixture(t)
	approveCore(t, s, reviewFixtureNow.Add(2*time.Minute))
	verification, err := s.VerifyReviewBundle("milestone:program/core")
	if err != nil {
		t.Fatal(err)
	}
	if !verification.Approved || verification.inherited || verification.Bundle == nil || verification.Bundle.Payload.Scope.Ref != "milestone:program/core" {
		t.Fatalf("a recorded approval was replaced: %+v", verification)
	}
}

func TestAMilestoneInheritsNoAcceptance(t *testing.T) {
	root, s := inheritedFixture(t)
	federatedTestTrust(t, root, map[string]FederatedProjectTrust{})
	federatedTestGit(t, root, "add", "--all")
	federatedTestGit(t, root, "commit", "-q", "-m", "revoke")
	if verification, err := s.VerifyReviewBundle("milestone:program/core"); err == nil && verification.Approved {
		t.Fatalf("a milestone inherited over revoked trust: %+v", verification)
	}
	root2, s2 := inheritedFixture(t)
	federatedTestWrite(t, root2, ".pose/roadmaps/open.md", "---\nslug: open\nstatus: active\n---\n\n# Open\n")
	editRoadmapFixture(t, root2, "## Milestone: core\n", "## Milestone: core\n- after: roadmap:open\n")
	if verification, err := s2.VerifyReviewBundle("milestone:program/core"); err == nil && verification.Approved {
		t.Fatalf("a milestone inherited over an open predecessor: %+v", verification)
	}
}

func TestOtherMilestonesOweTheirReview(t *testing.T) {
	for name, specs := range map[string]string{"several": "backend, pending", "external": "xref:source/spec:backend"} {
		t.Run(name, func(t *testing.T) {
			root, s := inheritedFixture(t)
			editRoadmapFixture(t, root, "- specs: backend\n", "- specs: "+specs+"\n")
			if _, ok := s.soleMemberOf(ScopeRef{Kind: "milestone", Roadmap: "program", Milestone: "core"}); ok {
				t.Fatal("the milestone reads as having one member")
			}
			if verification, err := s.VerifyReviewBundle("milestone:program/core"); err == nil && verification.Approved {
				t.Fatalf("the milestone did not owe its review: %+v", verification)
			}
		})
	}
}

func TestTheSpecAnswersForTheExitGate(t *testing.T) {
	root, s, _, _ := federatedTestMilestoneConsumer(t)
	editRoadmapFixture(t, root, "- specs: pending\n", "- specs: pending\n\n**Exit gate:** the export\nholds on restart.\n")
	federatedTestWrite(t, root, ".pose/specs/pending.md", strings.Replace(mustReadFile(t, root, ".pose/specs/pending.md"), "status: draft", "status: in-progress", 1))
	federatedTestGit(t, root, "add", "--all")
	federatedTestGit(t, root, "commit", "-q", "-m", "started")
	plan, err := s.ReviewPlan("spec:pending")
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, criterion := range plan.Criteria {
		if strings.HasPrefix(criterion.ID, "milestone-exit-gate-") {
			found = criterion.Required && ReviewCriterionKind(criterion) == ReviewCriterionKindJudgment && strings.Contains(criterion.Description, "the export holds on restart.")
		}
	}
	if !found {
		t.Fatalf("the plan does not carry the gate: %+v", plan.Criteria)
	}
	done, err := s.ReviewPlan("spec:backend")
	if err != nil {
		t.Fatal(err)
	}
	for _, criterion := range done.Criteria {
		if strings.HasPrefix(criterion.ID, "milestone-exit-gate-") {
			t.Fatalf("a done spec gained a criterion: %+v", criterion)
		}
	}
}

func TestARoadmapDeclaresUnreviewedMilestones(t *testing.T) {
	root, s := inheritedFixture(t)
	federatedTestWrite(t, root, ".pose/specs/extra.md", "---\nslug: extra\nstatus: done\ncreated_at: 2026-08-13\ncompleted_at: 2026-08-14\n---\n\n# Spec: extra\n")
	editRoadmapFixture(t, root, "- specs: backend\n", "- specs: backend, extra\n")
	const core = "milestone:program/core"
	if verification, err := s.VerifyReviewBundle(core); err == nil && verification.Approved {
		t.Fatalf("the fixture stands with no declaration: %+v", verification)
	}
	editRoadmapFixture(t, root, "status: active\n", "status: active\nunreviewed_milestones: closed before milestones were reviewed\n")
	verification, err := s.VerifyReviewBundle(core)
	if err != nil {
		t.Fatal(err)
	}
	if !verification.Approved || !strings.Contains(strings.Join(verification.Warnings, " "), "closed before milestones were reviewed") {
		t.Fatalf("the declaration did not stand with its reason: %+v", verification)
	}
	if s.declaredUnreviewed(ScopeRef{Kind: "milestone", Roadmap: "program", Milestone: "later"}) != "" {
		t.Fatal("a milestone with an open member stood on the declaration")
	}
	if eval, err := s.ReviewCheck("roadmap:program"); err != nil || eval.Approved {
		t.Fatalf("the declaration approved the roadmap: %+v %v", eval, err)
	}
}

func TestRoadmapSealsOverInheritedMilestones(t *testing.T) {
	root, s := inheritedFixture(t)
	editRoadmapFixture(t, root, "\n## Milestone: later\n- after: core\n- specs: pending\n", "")
	bundle, err := s.SealReviewBundle("roadmap:program", reviewFixtureNow.Add(2*time.Minute))
	if err != nil {
		t.Fatal(err, bundle.Blockers)
	}
	if len(bundle.Payload.Children) != 1 || bundle.Payload.Children[0].Scope != "milestone:program/core" {
		t.Fatalf("the roadmap does not seal over its milestone: %+v", bundle.Payload.Children)
	}
	cov := &Coverage{Producer: "closeout", State: CoverageStateCurrent}
	if items, err := s.milestoneReachedObligations("coordinator", "", cov); err != nil || len(items) != 0 {
		t.Fatalf("an inherited milestone was listed as reached: %v %v", items, err)
	}
	reached, err := s.ReachedMilestones("backend")
	if err != nil || len(reached) != 0 {
		t.Fatalf("closing the member named a milestone with nothing to do: %v %v", reached, err)
	}
}

func TestAnApprovedOpenSpecIsNotInherited(t *testing.T) {
	root, s, _, _ := federatedTestMilestoneConsumer(t)
	editRoadmapFixture(t, root, "- specs: backend, xref:source/spec:backend\n", "- specs: backend\n")
	path := ".pose/specs/backend/spec.md"
	federatedTestWrite(t, root, path, strings.Replace(mustReadFile(t, root, path), "status: done", "status: in-progress", 1))
	federatedTestGit(t, root, "add", "--all")
	federatedTestGit(t, root, "commit", "-q", "-m", "reopened")
	approveMember(t, s, reviewFixtureNow)
	member, err := s.VerifyReviewBundle("spec:backend")
	if err != nil || !member.Approved || !member.Fresh || member.State == "closed" {
		t.Fatalf("the fixture is not an approved open spec: %+v %v", member, err)
	}
	if verification, err := s.VerifyReviewBundle("milestone:program/core"); err != nil || verification.Approved {
		t.Fatalf("a milestone inherited from a spec not closed: %+v %v", verification, err)
	}
}

func TestTheReasonIsSealedByTheRoadmap(t *testing.T) {
	root, s := inheritedFixture(t)
	editRoadmapFixture(t, root, "\n## Milestone: later\n- after: core\n- specs: pending\n", "")
	sealedWith := func(reason string) (ReviewBundle, string) {
		t.Helper()
		if reason != "" {
			editRoadmapFixture(t, root, "status: active\n", "status: active\nunreviewed_milestones: "+reason+"\n")
		}
		bundle, err := s.PrepareReviewBundle("roadmap:program")
		if err != nil {
			t.Fatal(err)
		}
		digest, err := s.ScopeDigest("roadmap:program")
		if err != nil {
			t.Fatal(err)
		}
		return bundle, digest
	}
	none, noneDigest := sealedWith("")
	first, firstDigest := sealedWith("reason one")
	editRoadmapFixture(t, root, "unreviewed_milestones: reason one\n", "unreviewed_milestones: reason two\n")
	second, secondDigest := sealedWith("")
	if none.BundleDigest == first.BundleDigest || first.BundleDigest == second.BundleDigest {
		t.Fatal("the bundle does not bind the reason")
	}
	if noneDigest == firstDigest || firstDigest == secondDigest {
		t.Fatal("the scope digest does not bind the reason")
	}
	found := false
	for _, section := range first.Payload.Scope.Sections {
		found = found || section.Path == "reason one"
	}
	if !found {
		t.Fatalf("a reviewer cannot read the reason: %+v", first.Payload.Scope.Sections)
	}
}

func TestAttentionKeepsWhatDoesNotStand(t *testing.T) {
	list := func(s Store) int {
		t.Helper()
		cov := &Coverage{Producer: "closeout", State: CoverageStateCurrent}
		items, err := s.milestoneReachedObligations("coordinator", "", cov)
		if err != nil {
			t.Fatal(err)
		}
		return len(items)
	}
	root, unapproved, _, _ := federatedTestMilestoneConsumer(t)
	editRoadmapFixture(t, root, "- specs: backend, xref:source/spec:backend\n", "- specs: backend\n")
	if list(unapproved) != 1 {
		t.Fatal("a milestone whose only member has no approval was hidden")
	}
	_, refused := inheritedFixture(t)
	bundle, err := refused.SealReviewBundle("milestone:program/core", reviewFixtureNow.Add(2*time.Minute))
	if err != nil {
		t.Fatal(err, bundle.Blockers)
	}
	attestChildReview(t, refused, bundle, reviewFixtureNow.Add(3*time.Minute), "changes-requested")
	if list(refused) != 1 {
		t.Fatal("a recorded refusal was hidden")
	}
	_, inherited := inheritedFixture(t)
	if list(inherited) != 0 {
		t.Fatal("an inherited milestone is listed")
	}
}

func TestADeclarationRescuesATerminalSingleton(t *testing.T) {
	root, s, _, _ := federatedTestMilestoneConsumer(t)
	editRoadmapFixture(t, root, "- specs: backend, xref:source/spec:backend\n", "- specs: backend\n")
	if verification, err := s.VerifyReviewBundle("milestone:program/core"); err == nil && verification.Approved {
		t.Fatalf("the fixture stands: %+v", verification)
	}
	editRoadmapFixture(t, root, "status: active\n", "status: active\nunreviewed_milestones: closed before reviews\n")
	verification, err := s.VerifyReviewBundle("milestone:program/core")
	if err != nil || !verification.Approved || verification.ApprovalOrigin != "declared:roadmap:program" {
		t.Fatalf("the declaration did not rescue it: %+v %v", verification, err)
	}
	state, err := s.GetCloseoutState("milestone:program/core")
	if err != nil || !state.Terminal {
		t.Fatalf("closeout disagrees: %+v %v", state.Blockers, err)
	}
}

func TestAnExternalConsumerHoldsTheRoadmapOfADeclaredMilestone(t *testing.T) {
	root, s := inheritedFixture(t)
	federatedTestWrite(t, root, ".pose/specs/extra.md", "---\nslug: extra\nstatus: done\ncreated_at: 2026-08-13\ncompleted_at: 2026-08-14\n---\n\n# Spec: extra\n")
	editRoadmapFixture(t, root, "- specs: backend\n", "- specs: backend, extra\n")
	editRoadmapFixture(t, root, "status: active\n", "status: active\nunreviewed_milestones: closed before reviews\n")
	editRoadmapFixture(t, root, "\n## Milestone: later\n- after: core\n- specs: pending\n", "")
	identity := ArtifactRef{Kind: "milestone", Slug: "program", Milestone: "core", Project: "coordinator"}
	_, _, _, _, _, _, blockers := federatedSourceProof(NativeFiles(), s, identity, gitHeadAtRoot(root))
	if !strings.Contains(strings.Join(blockers, " "), "source-review-not-approved") {
		t.Fatalf("a declaration stood as proof with no reviewed roadmap: %v", blockers)
	}
	roadmap, err := s.SealReviewBundle("roadmap:program", reviewFixtureNow.Add(2*time.Minute))
	if err != nil {
		t.Fatal(err, roadmap.Blockers)
	}
	attestChildReview(t, s, roadmap, reviewFixtureNow.Add(3*time.Minute), "approved")
	id, _, _, _, _, _, blockers := federatedSourceProof(NativeFiles(), s, identity, gitHeadAtRoot(root))
	if id != roadmap.BundleID || len(blockers) != 0 {
		t.Fatalf("the review of the roadmap is not the proof: %q %v", id, blockers)
	}
}

func TestAPolicyWithoutBundlesKeepsItsOwnAnswer(t *testing.T) {
	root, s := inheritedFixture(t)
	path := ".pose/policy/review.json"
	federatedTestWrite(t, root, path, strings.Replace(mustReadFile(t, root, path), `"review_bundles":true`, `"review_bundles":false`, 1))
	federatedTestGit(t, root, "add", "--all")
	federatedTestGit(t, root, "commit", "-q", "-m", "no bundles")
	if verification, err := s.VerifyReviewBundle("milestone:program/core"); err == nil && verification.ApprovalOrigin != "" {
		t.Fatalf("inheritance ran under a policy without bundles: %+v", verification)
	}
}

func TestEachExitGateHasItsOwnCriterion(t *testing.T) {
	root, s, _, _ := federatedTestMilestoneConsumer(t)
	editRoadmapFixture(t, root, "- specs: pending\n", "- specs: pending\n\n**Exit gate:** one.\n- target_due: 2026-12-01\n")
	federatedTestWrite(t, root, ".pose/roadmaps/old.md", "---\nslug: old\nstatus: abandoned\n---\n\n## Milestone: earlier\n- specs: pending\n\n**Exit gate:** two.\n")
	federatedTestWrite(t, root, ".pose/roadmaps/other.md", "---\nslug: other\nstatus: active\n---\n\n## Milestone: also\n- specs: pending\n\n**Exit gate:** three.\n")
	federatedTestWrite(t, root, ".pose/specs/pending.md", strings.Replace(mustReadFile(t, root, ".pose/specs/pending.md"), "status: draft", "status: in-progress", 1))
	federatedTestGit(t, root, "add", "--all")
	federatedTestGit(t, root, "commit", "-q", "-m", "gates")
	plan, err := s.ReviewPlan("spec:pending")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(strings.Join(plan.Blockers, " "), "conflicting") {
		t.Fatalf("two gates collided: %v", plan.Blockers)
	}
	gates := []string{}
	for _, criterion := range plan.Criteria {
		if strings.HasPrefix(criterion.ID, "milestone-exit-gate-") {
			gates = append(gates, criterion.Description)
		}
	}
	joined := strings.Join(gates, " | ")
	if len(gates) != 2 || strings.Contains(joined, "two.") || strings.Contains(joined, "target_due") || !strings.Contains(joined, "holds: one.") || !strings.Contains(joined, "three.") {
		t.Fatalf("the plan does not hold the gates of the live roadmaps: %v", gates)
	}
}

func TestARecordedLegacyRefusalStands(t *testing.T) {
	_, s := inheritedFixture(t)
	writeReviewFixture(t, s.Root, ".pose/reviews/rvw-20260926T180000Z-aaaa1111.md", "---\nschema_version: 1\nreview_id: rvw-20260926T180000Z-aaaa1111\nscope: milestone:program/core\ndecision: rejected\nreviewed_at: 2026-09-26T18:00:00Z\n---\n\n## Findings\n\nNone.\n")
	attempts, err := s.ReadReviewAttempts("milestone:program/core")
	if err != nil || len(attempts.Attempts) != 1 {
		t.Fatalf("the fixture attempt was not read: %+v %v", attempts, err)
	}
	if verification, err := s.VerifyReviewBundle("milestone:program/core"); err != nil || verification.Approved {
		t.Fatalf("a legacy refusal was inherited away: %+v %v", verification, err)
	}
}

func TestRoadmapSealsOverADeclaredMilestone(t *testing.T) {
	root, s := inheritedFixture(t)
	federatedTestWrite(t, root, ".pose/specs/extra.md", "---\nslug: extra\nstatus: done\ncreated_at: 2026-08-13\ncompleted_at: 2026-08-14\n---\n\n# Spec: extra\n")
	editRoadmapFixture(t, root, "- specs: backend\n", "- specs: backend, extra\n")
	editRoadmapFixture(t, root, "\n## Milestone: later\n- after: core\n- specs: pending\n", "")
	editRoadmapFixture(t, root, "status: active\n", "status: active\nunreviewed_milestones: closed before reviews\n")
	bundle, err := s.SealReviewBundle("roadmap:program", reviewFixtureNow.Add(2*time.Minute))
	if err != nil {
		t.Fatal(err, bundle.Blockers)
	}
	if len(bundle.Payload.Children) != 0 {
		t.Fatalf("a declared milestone has no bundle to seal over: %+v", bundle.Payload.Children)
	}
}
