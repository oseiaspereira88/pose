package pose

import (
	"strings"
	"testing"
)

// Spec pose-done-spec-keeps-its-sealed-plan.

const backendSpecPath = ".pose/specs/backend/spec.md"

// backend is the spec of the fixture that has an attributed change set; here it is the
// only member of the milestone, which declares a gate, and it starts in progress.
func soleMemberWithAGate(t *testing.T) (string, Store) {
	t.Helper()
	root, s, _, _ := federatedTestMilestoneConsumer(t)
	editRoadmapFixture(t, root, "- specs: backend, xref:source/spec:backend\n", "- specs: backend\n\n**Exit gate:** the export\nholds on restart.\n")
	federatedTestWrite(t, root, backendSpecPath, strings.Replace(mustReadFile(t, root, backendSpecPath), "status: done", "status: in-progress", 1))
	federatedTestGit(t, root, "add", "--all")
	federatedTestGit(t, root, "commit", "-q", "-m", "reopened")
	return root, s
}

func markDone(t *testing.T, root string) {
	t.Helper()
	federatedTestWrite(t, root, backendSpecPath, strings.Replace(mustReadFile(t, root, backendSpecPath), "status: in-progress", "status: done", 1))
	federatedTestGit(t, root, "add", "--all")
	federatedTestGit(t, root, "commit", "-q", "-m", "done")
}

func gateCriteria(criteria []ReviewPlanCriterion) []string {
	ids := []string{}
	for _, criterion := range criteria {
		if strings.HasPrefix(criterion.ID, "milestone-exit-gate-") {
			ids = append(ids, criterion.ID)
		}
	}
	return ids
}

// R1, R6: the spec was sealed with the gate and approved; once done, its plan still
// carries it, so the sealed bundle still represents the spec and the approval is the
// fresh one, not an approval retained over drift.
func TestADoneSpecSealedWithTheGateKeepsItsPlanAndItsApproval(t *testing.T) {
	root, s := soleMemberWithAGate(t)
	bundle, err := s.SealReviewBundle("spec:backend", reviewFixtureNow)
	if err != nil {
		t.Fatal(err, bundle.Blockers)
	}
	if len(gateCriteria(bundle.Payload.Plan.Criteria)) != 1 {
		t.Fatalf("the open spec was not sealed with the gate: %+v", bundle.Payload.Plan.Criteria)
	}
	attestChildReview(t, s, bundle, reviewFixtureNow.Add(60e9), "approved")
	markDone(t, root)
	plan, err := s.ReviewPlan("spec:backend")
	if err != nil {
		t.Fatal(err)
	}
	if len(gateCriteria(plan.Criteria)) != 1 {
		t.Fatalf("a done spec sealed with the gate lost it from its plan: %+v", plan.Criteria)
	}
	verification, err := s.VerifyReviewBundle("spec:backend")
	if err != nil {
		t.Fatal(err)
	}
	if !verification.Approved || !verification.Fresh || verification.State != "closed" {
		t.Fatalf("the approval did not survive the transition to done: %+v", verification)
	}
	if strings.Contains(strings.Join(verification.Warnings, " "), retainedCompletedReviewWarning) || verification.NextAction != "scope is closed with a fresh bundle attestation" {
		t.Fatalf("the approval stands only as one retained over drift, not over the bundle the spec was sealed with: %v / %s", verification.Warnings, verification.NextAction)
	}
}

// R2: a done spec that no sealed bundle carries the gate for is prepared as before.
func TestADoneSpecSealedWithoutTheGateGainsNone(t *testing.T) {
	root, s := soleMemberWithAGate(t)
	// Sealed before the roadmap declared a gate: edit the roadmap after sealing.
	editRoadmapFixture(t, root, "\n**Exit gate:** the export\nholds on restart.\n", "\n")
	bundle, err := s.SealReviewBundle("spec:backend", reviewFixtureNow)
	if err != nil {
		t.Fatal(err, bundle.Blockers)
	}
	if len(gateCriteria(bundle.Payload.Plan.Criteria)) != 0 {
		t.Fatalf("the fixture sealed with a gate: %+v", bundle.Payload.Plan.Criteria)
	}
	editRoadmapFixture(t, root, "- specs: backend\n", "- specs: backend\n\n**Exit gate:** declared later.\n")
	markDone(t, root)
	plan, err := s.ReviewPlan("spec:backend")
	if err != nil {
		t.Fatal(err)
	}
	if got := gateCriteria(plan.Criteria); len(got) != 0 {
		t.Fatalf("a done spec sealed without a gate gained %v", got)
	}
}

// R3: an open spec is prepared as before; R2: a spec of several members, as before.
func TestAnOpenSpecStillOwesTheGate(t *testing.T) {
	_, s := soleMemberWithAGate(t)
	plan, err := s.ReviewPlan("spec:backend")
	if err != nil {
		t.Fatal(err)
	}
	if len(gateCriteria(plan.Criteria)) != 1 {
		t.Fatalf("the open spec does not carry the gate: %+v", plan.Criteria)
	}
}
