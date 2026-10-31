package pose

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func attestChildReview(t *testing.T, s Store, bundle ReviewBundle, at time.Time, decision string) {
	t.Helper()
	attestation := approvedBundleAttestation(bundle, "agent:independent-probe")
	attestation.Decision, attestation.AttestedAt = decision, at.Format(time.RFC3339)
	if _, err := s.RecordReviewAttestation(attestation, at); err != nil {
		t.Fatal(err)
	}
}

func commitReviewFixture(t *testing.T, root, body string) {
	t.Helper()
	writeReviewFixture(t, root, "api/server.go", "package api\n\nfunc Ready() bool { return "+body+" }\n")
	federatedTestGit(t, root, "add", "--all")
	federatedTestGit(t, root, "commit", "-qm", "drift "+body)
}

// approvedParentFixture is a done spec approved, then drifted, and its
// milestone approved over the retained child; with current set, the child's
// inputs are restored first so that it matches.
func approvedParentFixture(t *testing.T, current bool) (string, Store, ReviewBundle, ReviewBundle, time.Time) {
	t.Helper()
	root, s, _, _ := federatedTestMilestoneConsumer(t)
	s.FederatedResolver = nil
	now := time.Date(2026, 9, 26, 18, 0, 0, 0, time.UTC)
	child, err := s.SealReviewBundle("spec:backend", now)
	if err != nil {
		t.Fatal(err, child.Blockers)
	}
	attestChildReview(t, s, child, now.Add(time.Minute), "approved")
	if !current {
		commitReviewFixture(t, root, "!false")
	}
	parent, err := s.SealReviewBundle("milestone:program/core", now.Add(3*time.Minute))
	if err != nil {
		t.Fatal(err, parent.Blockers)
	}
	attestChildReview(t, s, parent, now.Add(4*time.Minute), "approved")
	verification, err := s.VerifyReviewBundle("milestone:program/core")
	if err != nil || !verification.Approved || verification.State != "closed" {
		t.Fatalf("the parent stands before the child falls: %+v %v", verification, err)
	}
	return root, s, child, parent, now
}

func assertParentFallen(t *testing.T, s Store, scope, child string) {
	t.Helper()
	verification, err := s.VerifyReviewBundle(scope)
	if err != nil {
		t.Fatal(err)
	}
	if verification.Approved {
		t.Fatalf("%s stays approved over %s: %+v", scope, child, verification)
	}
	if text := strings.Join(verification.Blockers, "\n"); !strings.Contains(text, reviewChildPrefix+child+" does not stand") {
		t.Fatalf("%s does not name %s: %s", scope, child, text)
	}
	if verification.NextAction != "restore the review of the children of "+scope {
		t.Fatalf("next action: %q", verification.NextAction)
	}
	eval, err := s.ReviewCheck(scope)
	if err != nil {
		t.Fatal(err)
	}
	if eval.Approved || !strings.Contains(strings.Join(eval.Blockers, "\n"), child) {
		t.Fatalf("review check of %s: %+v", scope, eval)
	}
	state, err := s.GetCloseoutState(scope)
	if err != nil {
		t.Fatal(err)
	}
	if state.Terminal {
		t.Fatalf("closeout of %s is terminal", scope)
	}
}

func TestParentApprovalFollowsAChildsRejection(t *testing.T) {
	for name, current := range map[string]bool{"retained child": false, "current child": true} {
		t.Run(name, func(t *testing.T) {
			_, s, child, parent, now := approvedParentFixture(t, current)
			before, err := s.PrepareReviewBundle("milestone:program/core")
			if err != nil {
				t.Fatal(err)
			}
			attestChildReview(t, s, child, now.Add(5*time.Minute), "rejected")
			assertParentFallen(t, s, "milestone:program/core", "spec:backend")
			prepared, err := s.PrepareReviewBundle("milestone:program/core")
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(strings.Join(prepared.Blockers, "\n"), "spec:backend") {
				t.Fatalf("preparation: %v", prepared.Blockers)
			}
			if current && prepared.BundleDigest != before.BundleDigest {
				t.Fatal("a child's later attestation changed the parent's identity")
			}
			_ = parent
		})
	}
}

// A parent recovers with its child, with no reseal.
func TestParentApprovalRecoversWithItsChild(t *testing.T) {
	_, s, child, _, now := approvedParentFixture(t, true)
	attestChildReview(t, s, child, now.Add(5*time.Minute), "rejected")
	assertParentFallen(t, s, "milestone:program/core", "spec:backend")
	attestChildReview(t, s, child, now.Add(6*time.Minute), "approved")
	verification, err := s.VerifyReviewBundle("milestone:program/core")
	if err != nil || !verification.Approved || verification.State != "closed" {
		t.Fatalf("%+v %v", verification, err)
	}
}

func TestParentApprovalFollowsLostChildHistory(t *testing.T) {
	root, s, child, _, _ := approvedParentFixture(t, false)
	lost := filepath.Join(root, ".pose", "review-attestations", "rva-ffffffffffffffff.json")
	if err := os.WriteFile(lost, []byte(`{"bundle_id":"`+child.BundleID+`"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if verification, err := s.VerifyReviewBundle("spec:backend"); err != nil || verification.Approved {
		t.Fatalf("the child stands over a lost record: %+v %v", verification, err)
	}
	assertParentFallen(t, s, "milestone:program/core", "spec:backend")
}

// The rule is transitive: a roadmap reads its milestone's verification, which
// reads its spec's.
func TestRoadmapApprovalFollowsItsDescendants(t *testing.T) {
	root, s, child, milestone, now := approvedParentFixture(t, false)
	children := []ReviewBundleChild{{Scope: "milestone:program/core", BundleID: milestone.BundleID, BundleDigest: milestone.BundleDigest}}
	if blockers := s.reviewChildBlockers(children); len(blockers) != 0 {
		t.Fatalf("control: %v", blockers)
	}
	commitReviewFixture(t, root, "1 == 1")
	if blockers := s.reviewChildBlockers(children); len(blockers) != 0 {
		t.Fatalf("drift alone: %v", blockers)
	}
	attestChildReview(t, s, child, now.Add(5*time.Minute), "rejected")
	blockers := s.reviewChildBlockers(children)
	if text := strings.Join(blockers, "\n"); len(blockers) != 1 || !strings.Contains(text, "milestone:program/core does not stand") || !strings.Contains(text, "spec:backend does not stand") {
		t.Fatalf("blockers: %v", blockers)
	}
	retained, blockersOfRoadmap, err := s.reviewBundleChildren(ScopeRef{Kind: "roadmap", Slug: "program"})
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range retained {
		if c.Scope == "milestone:program/core" {
			t.Fatalf("the milestone was retained: %+v", retained)
		}
	}
	if !strings.Contains(strings.Join(blockersOfRoadmap, "\n"), "milestone:program/core") {
		t.Fatalf("roadmap: %v", blockersOfRoadmap)
	}
}

// A reopened child stops standing under a done parent, though it stays approved.
func TestChildMustBeClosedUnderADoneParent(t *testing.T) {
	root, s, _, _, _ := approvedParentFixture(t, true)
	spec, err := s.GetSpec("backend")
	if err != nil {
		t.Fatal(err)
	}
	path := spec.Path
	if !filepath.IsAbs(path) {
		path = filepath.Join(root, path)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(strings.Replace(string(raw), "status: done", "status: in-progress", 1)), 0o644); err != nil {
		t.Fatal(err)
	}
	if why := s.childReviewStanding("spec:backend", false); why == "" {
		verification, _ := s.VerifyReviewBundle("spec:backend")
		if verification.State == "closed" {
			t.Fatalf("the reopened spec reads closed: %+v", verification)
		}
		if why := s.childReviewStanding("spec:backend", true); why == "" {
			t.Fatal("a reopened child stands under a done parent")
		}
	}
}

func bundleOf(id, replaces string) ReviewBundle {
	bundle := ReviewBundle{BundleID: id}
	if replaces != "" {
		bundle.Payload.Implementation = &ReviewBundleImplementation{Replaces: replaces}
	}
	return bundle
}

func TestReviewBundleLineage(t *testing.T) {
	for _, c := range []struct {
		name    string
		bundles []ReviewBundle
		want    string
	}{
		{"seal order without replacements", []ReviewBundle{bundleOf("a", ""), bundleOf("b", ""), bundleOf("c", "")}, "c b a"},
		{"a replacement in order", []ReviewBundle{bundleOf("a", ""), bundleOf("b", "a")}, "b a"},
		{"a backdated replacement", []ReviewBundle{bundleOf("b", "a"), bundleOf("a", "")}, "b a"},
		{"takes the newest place of its chain", []ReviewBundle{bundleOf("y", "x"), bundleOf("z", ""), bundleOf("x", "")}, "y x z"},
		{"a chain", []ReviewBundle{bundleOf("c", "b"), bundleOf("b", "a"), bundleOf("a", "")}, "c b a"},
		{"two replacing one", []ReviewBundle{bundleOf("a", ""), bundleOf("b", "a"), bundleOf("c", "a")}, "c a b"},
		{"a cycle", []ReviewBundle{bundleOf("a", "b"), bundleOf("b", "a")}, "b a"},
		{"replacing what is not listed", []ReviewBundle{bundleOf("a", "gone"), bundleOf("b", "")}, "b a"},
	} {
		got := []string{}
		for _, bundle := range reviewBundleLineage(c.bundles) {
			got = append(got, bundle.BundleID)
		}
		if strings.Join(got, " ") != c.want {
			t.Errorf("%s: %v, want %s", c.name, got, c.want)
		}
	}
}

// replacementFixture seals a child and records decision on it, then replaces
// it with a backdated bundle under a declared identity.
func replacementFixture(t *testing.T, decision string) (string, Store, ReviewBundle, time.Time) {
	t.Helper()
	root, s, _, _ := federatedTestMilestoneConsumer(t)
	s.FederatedResolver = nil
	now := time.Date(2026, 9, 26, 18, 0, 0, 0, time.UTC)
	first, err := s.SealReviewBundle("spec:backend", now)
	if err != nil {
		t.Fatal(err, first.Blockers)
	}
	attestChildReview(t, s, first, now.Add(time.Minute), decision)
	second, err := s.SealReviewBundleLead("spec:backend", "openai/gpt-6.1-sol", now.Add(-time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if second.Payload.Implementation.Replaces != first.BundleID {
		t.Fatal("the replacement was not declared")
	}
	return root, s, second, now
}

func TestRetentionHonoursReplacementLineage(t *testing.T) {
	root, s, second, now := replacementFixture(t, "approved")
	attestChildReview(t, s, second, now.Add(2*time.Minute), "approved")
	before, err := s.CurrentReviewBundle("spec:backend")
	if err != nil || before == nil || before.BundleID != second.BundleID {
		t.Fatalf("the replacement governs: %+v %v", before, err)
	}
	commitReviewFixture(t, root, "!false")
	retained, why := s.retainedChildReviewBundle("spec:backend")
	if retained == nil || retained.BundleID != second.BundleID {
		t.Fatalf("retained %+v: %s", retained, why)
	}
}

func TestReplacedApprovalIsNotRetained(t *testing.T) {
	root, s, _, _ := replacementFixture(t, "approved")
	commitReviewFixture(t, root, "!false")
	if retained, _ := s.retainedChildReviewBundle("spec:backend"); retained != nil {
		t.Fatalf("a replaced approval was retained: %s", retained.BundleID)
	}
}

func TestReplacementsRejectionIsNotOverriddenByWhatItReplaced(t *testing.T) {
	root, s, second, now := replacementFixture(t, "approved")
	attestChildReview(t, s, second, now.Add(2*time.Minute), "rejected")
	commitReviewFixture(t, root, "!false")
	if retained, _ := s.retainedChildReviewBundle("spec:backend"); retained != nil {
		t.Fatalf("an approval stood over a rejection: %s", retained.BundleID)
	}
	if bundle, att := s.latestNegativeReview("spec:backend"); bundle == nil || bundle.BundleID != second.BundleID || att.Decision != "rejected" {
		t.Fatalf("latest negative review: %+v", bundle)
	}
}

// A negative review of a replaced bundle counts when its replacement says nothing.
func TestReplacedRejectionStillCounts(t *testing.T) {
	root, s, _, _ := replacementFixture(t, "rejected")
	commitReviewFixture(t, root, "!false")
	if retained, _ := s.retainedChildReviewBundle("spec:backend"); retained != nil {
		t.Fatalf("retained over a rejection: %s", retained.BundleID)
	}
}

// An approved replacement stands though what it replaced was rejected and,
// by seal time, looks newer.
func TestReplacementStandsOverWhatItReplaced(t *testing.T) {
	root, s, second, now := replacementFixture(t, "rejected")
	attestChildReview(t, s, second, now.Add(2*time.Minute), "approved")
	commitReviewFixture(t, root, "!false")
	retained, why := s.retainedChildReviewBundle("spec:backend")
	if retained == nil || retained.BundleID != second.BundleID {
		t.Fatalf("retained %+v: %s", retained, why)
	}
}

// One operation verifies a child once.
func TestChildVerificationIsOncePerOperation(t *testing.T) {
	root, s, _, _, _ := approvedParentFixture(t, false)
	operation := s.reviewOperation()
	first, err := operation.childVerification("spec:backend")
	if err != nil || !first.Approved {
		t.Fatalf("%+v %v", first, err)
	}
	if err := os.RemoveAll(filepath.Join(root, ".pose", "review-attestations")); err != nil {
		t.Fatal(err)
	}
	second, err := operation.childVerification("spec:backend")
	if err != nil || !second.Approved {
		t.Fatalf("verified twice: %+v %v", second, err)
	}
	if fresh, err := s.reviewOperation().childVerification("spec:backend"); err != nil || fresh.Approved {
		t.Fatalf("a new operation reused the memo: %+v %v", fresh, err)
	}
}
