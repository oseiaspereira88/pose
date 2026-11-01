package pose

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
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
	breakFederationPolicy(t, root)
	_, verifyErr := s.VerifyReviewBundle("milestone:program/core")
	if verifyErr == nil {
		t.Fatal("the fixture did not fail the verification")
	}
	if _, err := s.ReviewCheck("milestone:program/core"); err == nil || err.Error() != verifyErr.Error() {
		t.Fatalf("closeout did not return the failure of the verification: %v", err)
	}
}

func breakFederationPolicy(t *testing.T, root string) {
	t.Helper()
	federatedTestWrite(t, root, ".pose/policy/federation.json", "{")
	federatedTestGit(t, root, "add", "--all")
	federatedTestGit(t, root, "commit", "-q", "-m", "break the federation policy")
}

// The exemption is for scopes with no bundle: a legacy attempt does not stand in for a
// sealed bundle that could not be verified.
func TestCloseoutRefusesALegacyAttemptOverAFailedVerification(t *testing.T) {
	root, s := closedMilestoneFixture(t)
	editRoadmapFixture(t, root, "slug: program", "slug: program\ncreated_at: 2026-08-13")
	path := filepath.Join(root, ".pose/policy/review.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var policy map[string]any
	if err := json.Unmarshal(raw, &policy); err != nil {
		t.Fatal(err)
	}
	policy["component_aware"] = false
	raw, err = json.Marshal(policy)
	if err != nil {
		t.Fatal(err)
	}
	federatedTestWrite(t, root, ".pose/policy/review.json", string(raw))
	digest, err := s.ScopeDigest("milestone:program/core")
	if err != nil {
		t.Fatal(err)
	}
	federatedTestWrite(t, root, ".pose/reviews/rvw-legacy.md", fmt.Sprintf("---\nschema_version: 1\nreview_id: rvw-legacy\nscope: milestone:program/core\nscope_digest: %s\nprofile: milestone-integration@1\nreviewer: agent:test-review\ndecision: approved\nreviewed_at: 2026-08-13T12:00:00Z\nevidence_refs: [integration:legacy]\n---\n\n## Criteria\n- integration [passed] evidence:integration:legacy\n\n## Findings\n", digest))
	breakFederationPolicy(t, root)
	if eval, err := s.ReviewCheck("milestone:program/core"); err == nil {
		t.Fatalf("a legacy attempt stood in for a bundle no verification could judge: %+v", eval)
	}
}

// A seal with no manifest has nothing to compare, whenever it was made: absence proves
// no age, since a store with no resolver seals none today.
func TestSealWithoutAManifestStales(t *testing.T) {
	root, s, _, _ := federatedTestMilestoneConsumer(t)
	editRoadmapFixture(t, root, "consumes: xref:source/roadmap:component", "created_at: 2026-08-01")
	local := s
	local.FederatedResolver = nil
	at := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	for i, ref := range []string{"spec:backend", "milestone:program/core"} {
		sealed := at.Add(time.Duration(i) * 2 * time.Minute)
		bundle, err := local.SealReviewBundle(ref, sealed)
		if err != nil {
			t.Fatal(err, bundle.Blockers)
		}
		if bundle.Payload.FederatedManifest != nil {
			t.Fatal("the fixture sealed a manifest")
		}
		attestChildReview(t, local, bundle, sealed.Add(time.Minute), "approved")
	}
	if eval := verifyAndCloseoutAgree(t, s, "milestone:program/core"); eval.Approved {
		t.Fatalf("a seal with no manifest stood against one: %+v", eval)
	}
}
