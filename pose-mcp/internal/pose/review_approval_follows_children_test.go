package pose

import (
	"encoding/json"
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

// bundleOf is a bundle of the shared content, with a digest, so that its
// replacement holds.
func bundleOf(t *testing.T, id, replaces string) ReviewBundle {
	t.Helper()
	bundle := ReviewBundle{BundleID: id}
	if replaces != "" {
		bundle.Payload.Implementation = &ReviewBundleImplementation{Replaces: replaces}
	}
	digest, err := reviewBundlePayloadDigest(bundle.Payload)
	if err != nil {
		t.Fatal(err)
	}
	bundle.BundleDigest = digest
	return bundle
}

func TestReviewBundleLineage(t *testing.T) {
	for _, c := range []struct {
		name    string
		bundles []ReviewBundle
		want    string
	}{
		{"seal order without replacements", []ReviewBundle{bundleOf(t, "a", ""), bundleOf(t, "b", ""), bundleOf(t, "c", "")}, "c b a"},
		{"a replacement in order", []ReviewBundle{bundleOf(t, "a", ""), bundleOf(t, "b", "a")}, "b a"},
		{"a backdated replacement", []ReviewBundle{bundleOf(t, "b", "a"), bundleOf(t, "a", "")}, "b a"},
		{"takes the newest place of its chain", []ReviewBundle{bundleOf(t, "y", "x"), bundleOf(t, "z", ""), bundleOf(t, "x", "")}, "y x z"},
		{"a chain", []ReviewBundle{bundleOf(t, "c", "b"), bundleOf(t, "b", "a"), bundleOf(t, "a", "")}, "c b a"},
		{"two replacing one", []ReviewBundle{bundleOf(t, "a", ""), bundleOf(t, "b", "a"), bundleOf(t, "c", "a")}, "c b a"},
		{"a replacement of itself", []ReviewBundle{bundleOf(t, "a", "a"), bundleOf(t, "b", "")}, "b a"},
		{"a cycle", []ReviewBundle{bundleOf(t, "a", "b"), bundleOf(t, "b", "a")}, "b a"},
		{"replacing what is not listed", []ReviewBundle{bundleOf(t, "a", "gone"), bundleOf(t, "b", "")}, "b a"},
	} {
		got := []string{}
		for _, bundle := range reviewBundleLineage(c.bundles) {
			got = append(got, bundle.BundleID)
		}
		if strings.Join(got, " ") != c.want {
			t.Errorf("%s: %v, want %s", c.name, got, c.want)
		}
	}
	other := bundleOf(t, "other", "")
	other.Payload.Scope.Sections = []ReviewBundleInput{{Kind: "semantic-section", Path: "another", Digest: digestText("another")}}
	forged := []ReviewBundle{bundleOf(t, "a", "other"), other}
	if got := reviewBundleLineage(forged); got[0].BundleID != "other" {
		t.Errorf("a replacement of other content counted: %s first", got[0].BundleID)
	}
	if replaced := replacedReviewBundles(forged); replaced["other"] {
		t.Error("a replacement of other content retired its target")
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
	if bundle, _ := s.latestNegativeReview("spec:backend"); bundle != nil {
		t.Fatalf("the replaced rejection reads as the latest review: %s", bundle.BundleID)
	}
	commitReviewFixture(t, root, "!false")
	retained, why := s.retainedChildReviewBundle("spec:backend")
	if retained == nil || retained.BundleID != second.BundleID {
		t.Fatalf("retained %+v: %s", retained, why)
	}
}

// persistSealedBundle writes bundle as a sealed record of its digest, as a
// historical import would: what a sealed record holds is data.
func persistSealedBundle(t *testing.T, s Store, bundle ReviewBundle, at time.Time) ReviewBundle {
	t.Helper()
	digest, err := reviewBundlePayloadDigest(bundle.Payload)
	if err != nil {
		t.Fatal(err)
	}
	bundle.BundleDigest, bundle.BundleID = digest, "rvb-"+strings.TrimPrefix(digest, "sha256:")[:16]
	bundle.State, bundle.SealedAt = "sealed", at.Format(time.RFC3339)
	dir, err := ensureReviewArtifactDir(s.Root, ".pose/review-bundles", true)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.MarshalIndent(bundle, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, bundle.BundleID+".json"), raw, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := s.LoadReviewBundle(bundle.BundleID); err != nil {
		t.Fatal(err)
	}
	return bundle
}

func persistAttestation(t *testing.T, s Store, bundle ReviewBundle, at time.Time) {
	t.Helper()
	attestation := approvedBundleAttestation(bundle, "agent:independent-probe")
	attestation.AttestedAt = at.Format(time.RFC3339)
	attestation, err := completeReviewAttestation(bundle, attestation, at)
	if err != nil {
		t.Fatal(err)
	}
	dir, err := ensureReviewArtifactDir(s.Root, ".pose/review-attestations", true)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(attestation)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, attestation.AttestationID+".json"), raw, 0o644); err != nil {
		t.Fatal(err)
	}
}

// Sealed children are data: a record naming its own scope, or a cycle through
// another, ends in a refusal instead of recursing (AC-R1-01).
func TestChildCycleEndsInARefusal(t *testing.T) {
	for name, multi := range map[string]bool{"self": false, "through another scope": true} {
		t.Run(name, func(t *testing.T) {
			root, s, child, milestone, now := approvedParentFixture(t, true)
			target := child
			if multi {
				target = milestone
			}
			child.Payload.Children = []ReviewBundleChild{{Scope: target.Payload.Scope.Ref, BundleID: target.BundleID, BundleDigest: target.BundleDigest}}
			child = persistSealedBundle(t, s, child, now.Add(time.Minute))
			persistAttestation(t, s, child, now.Add(2*time.Minute))
			commitReviewFixture(t, root, "!false")
			done := make(chan ReviewBundleVerification, 1)
			go func() {
				verification, _ := s.VerifyReviewBundle("spec:backend")
				done <- verification
			}()
			select {
			case verification := <-done:
				if verification.Approved {
					t.Fatalf("a cycle verified approved: %+v", verification)
				}
				if !strings.Contains(strings.Join(verification.Blockers, "\n"), "form a cycle") {
					t.Fatalf("the cycle is not named: %v", verification.Blockers)
				}
			case <-time.After(30 * time.Second):
				t.Fatal("verification did not end")
			}
		})
	}
}

// A replacement of a legacy roadmap-coordinated milestone bundle governs in
// both directions (AC-R1-02).
func TestLegacyCoordinatedMatchingFollowsLineage(t *testing.T) {
	for _, c := range []struct {
		name, first, second string
		approved            bool
	}{
		{"an approved replacement of a rejection", "rejected", "approved", true},
		{"a rejected replacement of an approval", "approved", "rejected", false},
	} {
		t.Run(c.name, func(t *testing.T) {
			_, s, _, _ := federatedTestMilestoneConsumer(t)
			now := time.Date(2026, 9, 26, 18, 0, 0, 0, time.UTC)
			child, err := s.SealReviewBundle("spec:backend", now)
			if err != nil {
				t.Fatal(err)
			}
			attestChildReview(t, s, child, now.Add(time.Minute), "approved")
			first, err := s.prepareReviewBundle("milestone:program/core", true)
			if err != nil || len(first.Blockers) > 0 {
				t.Fatal(err, first.Blockers)
			}
			first = persistSealedBundle(t, s, first, now)
			attestChildReview(t, s, first, now.Add(time.Minute), c.first)
			second := first
			identity, _, err := s.SealingIdentity("openai/gpt-6.1-sol")
			if err != nil {
				t.Fatal(err)
			}
			identity.Replaces = first.BundleID
			second.Payload.Implementation = &identity
			second = persistSealedBundle(t, s, second, now.Add(time.Minute))
			attestChildReview(t, s, second, now.Add(2*time.Minute), c.second)
			second = persistSealedBundle(t, s, second, now.Add(-time.Minute))
			current, err := s.CurrentReviewBundle("milestone:program/core")
			if err != nil || current == nil || current.BundleID != second.BundleID {
				t.Fatalf("current %+v %v", current, err)
			}
			verification, err := s.VerifyReviewBundle("milestone:program/core")
			if err != nil || verification.Approved != c.approved {
				t.Fatalf("%+v %v", verification, err)
			}
		})
	}
}

// Two sealers that both read A: each successor comes before A (AC-R1-03).
func TestSealedBranchLineage(t *testing.T) {
	root, s, _, _ := federatedTestMilestoneConsumer(t)
	s.FederatedResolver = nil
	now := time.Date(2026, 9, 26, 18, 0, 0, 0, time.UTC)
	a, err := s.SealReviewBundle("spec:backend", now)
	if err != nil {
		t.Fatal(err)
	}
	attestChildReview(t, s, a, now.Add(time.Minute), "rejected")
	b, err := s.SealReviewBundleLead("spec:backend", "openai/gpt-6.1-sol", now.Add(2*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	attestChildReview(t, s, b, now.Add(3*time.Minute), "approved")
	path := filepath.Join(root, ".pose/review-bundles", b.BundleID+".json")
	hidden := filepath.Join(t.TempDir(), "hidden.json")
	if err := os.Rename(path, hidden); err != nil {
		t.Fatal(err)
	}
	c, err := s.SealReviewBundleLead("spec:backend", "anthropic/claude-opus-4", now.Add(4*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(hidden, path); err != nil {
		t.Fatal(err)
	}
	if b.Payload.Implementation.Replaces != a.BundleID || c.Payload.Implementation.Replaces != a.BundleID {
		t.Fatal("not a branch")
	}
	commitReviewFixture(t, root, "!false")
	retained, why := s.retainedChildReviewBundle("spec:backend")
	if retained == nil || retained.BundleID != b.BundleID {
		t.Fatalf("retained %+v: %s", retained, why)
	}
}

// A replaced approval hides no negative review that retention reaches (AC-R1-04).
func TestLatestNegativeSkipsAReplacedApproval(t *testing.T) {
	root, s, _, _ := federatedTestMilestoneConsumer(t)
	s.FederatedResolver = nil
	now := time.Date(2026, 9, 26, 18, 0, 0, 0, time.UTC)
	negative, err := s.SealReviewBundle("spec:backend", now)
	if err != nil {
		t.Fatal(err)
	}
	attestChildReview(t, s, negative, now.Add(time.Minute), "rejected")
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
	if err := os.WriteFile(path, append(raw, []byte("\n## 5. Decisions\n- D1: another decision.\n")...), 0o644); err != nil {
		t.Fatal(err)
	}
	federatedTestGit(t, root, "add", "--all")
	federatedTestGit(t, root, "commit", "-qm", "new semantic scope")
	approved, err := s.SealReviewBundle("spec:backend", now.Add(2*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	attestChildReview(t, s, approved, now.Add(3*time.Minute), "approved")
	replacement, err := s.SealReviewBundleLead("spec:backend", "openai/gpt-6.1-sol", now.Add(4*time.Minute))
	if err != nil || replacement.Payload.Implementation.Replaces != approved.BundleID {
		t.Fatalf("%+v %v", replacement, err)
	}
	commitReviewFixture(t, root, "!false")
	if bundle, _ := s.latestNegativeReview("spec:backend"); bundle == nil || bundle.BundleID != negative.BundleID {
		t.Fatalf("latest negative: %+v", bundle)
	}
}

// A sealed `replaces` of other content retires nothing (AC-R1-05).
func TestForgedReplacementRetiresNothing(t *testing.T) {
	root, s, second, now := replacementFixture(t, "approved")
	attestChildReview(t, s, second, now.Add(2*time.Minute), "approved")
	forged := second
	forged.Payload.Scope.Sections = append(append([]ReviewBundleInput{}, forged.Payload.Scope.Sections...), ReviewBundleInput{Kind: "semantic-section", Path: "forged", Digest: digestText("another subject")})
	identity := *second.Payload.Implementation
	identity.Replaces = second.BundleID
	forged.Payload.Implementation = &identity
	persistSealedBundle(t, s, forged, now.Add(4*time.Minute))
	if verification, err := s.VerifyReviewBundle("spec:backend"); err != nil || !verification.Approved {
		t.Fatalf("control: %+v %v", verification, err)
	}
	commitReviewFixture(t, root, "!false")
	if verification, err := s.VerifyReviewBundle("spec:backend"); err != nil || !verification.Approved {
		t.Fatalf("a forged edge retired a standing approval: %+v %v", verification, err)
	}
}

// Effective governance reads the bundle verification reads (AC-R1-06).
func TestEffectiveGovernanceFollowsLineage(t *testing.T) {
	_, s, second, now := replacementFixture(t, "approved")
	attestChildReview(t, s, second, now.Add(2*time.Minute), "approved")
	verification, err := s.VerifyReviewBundle("spec:backend")
	if err != nil || verification.Bundle == nil || verification.Bundle.BundleID != second.BundleID {
		t.Fatalf("%+v %v", verification, err)
	}
	governance, err := s.EffectiveGovernance("spec:backend")
	if err != nil {
		t.Fatal(err)
	}
	if governance.ScopeBundle == nil || governance.ScopeBundle.BundleID != second.BundleID {
		t.Fatalf("governance reads %+v", governance.ScopeBundle)
	}
}

// One public call verifies a scope once (AC-R1-08).
func TestChildVerificationIsOncePerOperation(t *testing.T) {
	root, s, child, _, now := approvedParentFixture(t, false)
	attestChildReview(t, s, child, now.Add(5*time.Minute), "rejected")
	for name, call := range map[string]func(Store) error{
		"verify":       func(s Store) error { _, err := s.VerifyReviewBundle("milestone:program/core"); return err },
		"review-check": func(s Store) error { _, err := s.ReviewCheck("milestone:program/core"); return err },
		"closeout":     func(s Store) error { _, err := s.GetCloseoutState("milestone:program/core"); return err },
	} {
		counted := map[string]int{}
		observed := s
		observed.onReviewVerify = func(scope string) { counted[scope]++ }
		if err := call(observed); err != nil {
			t.Fatal(name, err)
		}
		if counted["spec:backend"] != 1 {
			t.Errorf("%s verified the child %d times", name, counted["spec:backend"])
		}
	}
	operation := s.reviewOperation()
	first, err := operation.verifyReviewBundle("spec:backend")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(filepath.Join(root, ".pose", "review-bundles")); err != nil {
		t.Fatal(err)
	}
	if second, err := operation.verifyReviewBundle("spec:backend"); err != nil || second.State != first.State {
		t.Fatalf("verified twice: %+v %v", second, err)
	}
	if fresh, err := s.VerifyReviewBundle("spec:backend"); err != nil || fresh.State == first.State {
		t.Fatalf("a new operation reused the memo: %+v %v", fresh, err)
	}
}
