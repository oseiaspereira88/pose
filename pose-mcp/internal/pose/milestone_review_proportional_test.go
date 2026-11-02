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
		if criterion.ID == "milestone-exit-gate" {
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
		if criterion.ID == "milestone-exit-gate" {
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
