package pose

import (
	"strings"
	"testing"
)

func verifyAndCloseoutAgree(t *testing.T, s Store, ref string) ReviewEvaluation {
	t.Helper()
	verification, err := s.VerifyReviewBundle(ref)
	if err != nil {
		t.Fatal(err)
	}
	eval, err := s.ReviewCheck(ref)
	if err != nil {
		t.Fatal(err)
	}
	if verification.Approved != eval.Approved || verification.Fresh != eval.Fresh {
		t.Fatalf("verify and closeout disagree on %s: verify %+v, closeout %+v", ref, verification, eval)
	}
	return eval
}

// The roadmap stays active: the milestone is done only because its members are.
func TestCloseoutRefusesARoadmapEditVerifyRefused(t *testing.T) {
	root, s := closedMilestoneFixture(t)
	editRoadmapFixture(t, root, "## Milestone: core", "Edited.\n\n## Milestone: core")
	eval := verifyAndCloseoutAgree(t, s, "milestone:program/core")
	if eval.Approved || eval.Fresh || eval.BundleState != "superseded" {
		t.Fatalf("a closed milestone stood over an edited roadmap: %+v", eval)
	}
	if !strings.Contains(strings.Join(eval.Blockers, " "), "federated manifest of milestone:program/core changed") {
		t.Fatalf("the refusal does not name the manifest: %v", eval.Blockers)
	}
	state, err := s.GetCloseoutState("milestone:program/core")
	if err != nil {
		t.Fatal(err)
	}
	if state.Terminal {
		t.Fatalf("the closeout state read terminal over an edited roadmap: %+v", state)
	}
}

func TestCloseoutRefusesARoadmapEditWhenClosed(t *testing.T) {
	root, s := closedMilestoneFixture(t)
	editRoadmapFixture(t, root, "status: active", "status: done")
	editRoadmapFixture(t, root, "## Milestone: core", "Edited.\n\n## Milestone: core")
	if eval := verifyAndCloseoutAgree(t, s, "milestone:program/core"); eval.Approved {
		t.Fatalf("a closed milestone stood over an edit of its closed roadmap: %+v", eval)
	}
}

func TestCloseoutKeepsWhatVerifyRetains(t *testing.T) {
	root, s := closedMilestoneFixture(t)
	editRoadmapFixture(t, root, "status: active", "status: done")
	eval := verifyAndCloseoutAgree(t, s, "milestone:program/core")
	if !eval.Approved || eval.BundleState != "closed" {
		t.Fatalf("the close of its roadmap staled the milestone in closeout: %+v", eval)
	}
}

func TestCloseoutRefusesRevokedTrust(t *testing.T) {
	root, s := closedMilestoneFixture(t)
	if eval := verifyAndCloseoutAgree(t, s, "milestone:program/core"); !eval.Approved {
		t.Fatalf("the approval did not stand before the revocation: %+v", eval)
	}
	federatedTestTrust(t, root, map[string]FederatedProjectTrust{})
	federatedTestGit(t, root, "add", "--all")
	federatedTestGit(t, root, "commit", "-q", "-m", "revoke")
	if eval := verifyAndCloseoutAgree(t, s, "milestone:program/core"); eval.Approved {
		t.Fatalf("closeout kept an approval over revoked trust: %+v", eval)
	}
}

func TestRetainedApprovalIsNotJudgedWithoutAResolver(t *testing.T) {
	bundle := &ReviewBundle{}
	bundle.Payload.FederatedManifest = &FederatedRoadmapManifest{Digest: "sealed"}
	if !(Store{}).retainedApprovalStands(bundle, &FederatedRoadmapManifest{Digest: "other"}) {
		t.Fatal("a store with no resolver judged a manifest it cannot resolve")
	}
}

func TestCloseoutRefusesAFailedVerification(t *testing.T) {
	root, s := closedMilestoneFixture(t)
	federatedTestWrite(t, root, ".pose/policy/federation.json", "{")
	federatedTestGit(t, root, "add", "--all")
	federatedTestGit(t, root, "commit", "-q", "-m", "break the federation policy")
	if _, err := s.VerifyReviewBundle("milestone:program/core"); err == nil {
		t.Fatal("the fixture did not fail the verification")
	}
	if eval, err := s.ReviewCheck("milestone:program/core"); err == nil && eval.Approved {
		t.Fatalf("closeout kept an approval no verification could judge: %+v", eval)
	}
}

func TestManifestAbsentFromAnOldSeal(t *testing.T) {
	s := Store{}
	here, there := ArtifactRef{Project: "here"}, ArtifactRef{Project: "there"}
	local := &FederatedRoadmapManifest{Coordinator: here, Ready: true, Dependencies: []FederatedDependency{{Identity: here, Status: "done"}}}
	if !s.federatedManifestStands(nil, local) {
		t.Fatal("a seal older than manifests was staled by a manifest within its project")
	}
	for name, current := range map[string]*FederatedRoadmapManifest{
		"another project": {Coordinator: here, Ready: true, Dependencies: []FederatedDependency{{Identity: there, Status: "done"}}},
		"a blocker":       {Coordinator: here, Ready: true, Blockers: []string{"xref:here/spec:a:not-done"}},
		"not ready":       {Coordinator: here},
	} {
		if s.federatedManifestStands(nil, current) {
			t.Errorf("a seal older than manifests stood over %s", name)
		}
	}
	if s.federatedManifestStands(local, nil) {
		t.Error("a manifest gone from the current scope stood")
	}
}
