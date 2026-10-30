package pose

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// withRemoval rewrites the fixture's indexed graph with removal and returns
// the fixture's subject path removed from disk.
func withRemoval(t *testing.T, root string, removal *ArtifactRemoval) {
	t.Helper()
	path := filepath.Join(root, ".pose", "indexes", "delivery-integrity.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var graph DeliveryIntegrityGraph
	if err := json.Unmarshal(raw, &graph); err != nil {
		t.Fatal(err)
	}
	graph.Removals = nil
	if removal != nil {
		graph.Removals = []ArtifactRemoval{*removal}
	}
	raw, err = json.MarshalIndent(graph, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	writeReviewFixture(t, root, ".pose/indexes/delivery-integrity.json", string(raw))
	if err := os.Remove(filepath.Join(root, "api", "server.go")); err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
}

func TestReviewSubjectReadsALaterRemovalFromItsWitness(t *testing.T) {
	unreadable := func(bundle ReviewBundle) bool {
		for _, blocker := range bundle.Blockers {
			if strings.Contains(blocker, "api/server.go cannot be read") {
				return true
			}
		}
		return false
	}
	// R1: a removal the graph accepted for the spec is the path's identity.
	root, store := reviewBundleFixture(t)
	withRemoval(t, root, &ArtifactRemoval{Spec: "later", Path: "api/server.go", Commit: "c1", Supersedes: map[string]string{"backend": "c1"}})
	bundle, err := store.PrepareReviewBundle("spec:backend")
	if err != nil {
		t.Fatal(err)
	}
	if unreadable(bundle) {
		t.Fatalf("an accepted later removal still refused the subject: %v", bundle.Blockers)
	}
	found := false
	for _, entry := range bundle.Payload.Subject.Entries {
		found = found || entry.Path == "api/server.go" && entry.Digest == digestBytes([]byte("removed-by later c1"))
	}
	if !found {
		t.Fatalf("the subject entry is not the removal witness: %+v", bundle.Payload.Subject.Entries)
	}
	// R2: no witness, a witness for another spec or another path, or a
	// refused removal, keep the unreadable path.
	for name, removal := range map[string]*ArtifactRemoval{
		"no witness":    nil,
		"another spec":  {Spec: "later", Path: "api/server.go", Commit: "c1", Supersedes: map[string]string{"other": "c1"}},
		"another path":  {Spec: "later", Path: "api/other.go", Commit: "c1", Supersedes: map[string]string{"backend": "c1"}},
		"refused":       {Spec: "later", Path: "api/server.go", Commit: "c1", Refusals: map[string]string{"backend": "this spec's change to it does not precede the removal"}},
		"never deleted": {Spec: "later", Path: "api/server.go", Refusals: map[string]string{"backend": "its change set deletes nothing there"}},
		// SR-R1-01: another owner's acceptance never stands for the subject's
		// refusal, and a spec's own removal satisfies nothing.
		"peer accepted, subject refused": {Spec: "later", Path: "api/server.go", Commit: "c1", Supersedes: map[string]string{"peer": "c1"}, Refusals: map[string]string{"backend": "this spec's change to it does not precede the removal"}},
		"own removal":                    {Spec: "backend", Path: "api/server.go", Commit: "c1", Supersedes: map[string]string{"backend": "c1"}},
	} {
		root, store := reviewBundleFixture(t)
		withRemoval(t, root, removal)
		bundle, err := store.PrepareReviewBundle("spec:backend")
		if err != nil {
			t.Fatal(err)
		}
		if !unreadable(bundle) {
			t.Fatalf("%s: an absent path without an accepted removal was not refused: %v", name, bundle.Blockers)
		}
	}
}

func TestMilestoneSealRetainsADoneChildsApprovedReview(t *testing.T) {
	childBlocker := func(bundle ReviewBundle) bool {
		for _, blocker := range bundle.Blockers {
			if strings.Contains(blocker, "child spec:backend has no sealed review bundle") {
				return true
			}
		}
		return false
	}
	setup := func(t *testing.T, decision string) (string, Store, ReviewBundle) {
		t.Helper()
		root, store, _, _ := federatedTestMilestoneConsumer(t)
		now := time.Date(2026, 9, 26, 18, 0, 0, 0, time.UTC)
		specBundle, err := store.SealReviewBundle("spec:backend", now)
		if err != nil {
			t.Fatalf("seal spec: %v; blockers=%v", err, specBundle.Blockers)
		}
		attestation := approvedBundleAttestation(specBundle, "agent:spec-review")
		attestation.AttestedAt = now.Add(time.Minute).Format(time.RFC3339)
		if _, err := store.RecordReviewAttestation(attestation, now.Add(time.Minute)); err != nil {
			t.Fatal(err)
		}
		if decision != "approved" {
			later := attestation
			later.Decision = decision
			later.AttestedAt = now.Add(2 * time.Minute).Format(time.RFC3339)
			if _, err := store.RecordReviewAttestation(later, now.Add(2*time.Minute)); err != nil {
				t.Fatal(err)
			}
		}
		// A commit after the spec closed changes what is prepared for it.
		writeReviewFixture(t, root, "api/server.go", "package api\n\nfunc Ready() bool { return !false }\n")
		federatedTestGit(t, root, "add", "--all")
		federatedTestGit(t, root, "commit", "-q", "-m", "after closeout")
		if current, err := store.CurrentReviewBundle("spec:backend"); err != nil || current != nil {
			t.Fatalf("the fixture did not change the child's preparation: %+v %v", current, err)
		}
		return root, store, specBundle
	}
	// R3: a done child's retained approved review is the child.
	_, store, specBundle := setup(t, "approved")
	milestone, err := store.PrepareReviewBundle("milestone:program/core")
	if err != nil {
		t.Fatal(err)
	}
	if childBlocker(milestone) {
		t.Fatalf("a done child's approved review was not retained: %v", milestone.Blockers)
	}
	retained := false
	for _, child := range milestone.Payload.Children {
		retained = retained || child.Scope == "spec:backend" && child.BundleID == specBundle.BundleID
	}
	if !retained {
		t.Fatalf("the milestone did not record the retained child bundle: %+v", milestone.Payload.Children)
	}
	// A later negative review, or a child that is not done, still blocks.
	_, store, _ = setup(t, "changes-requested")
	if milestone, err := store.PrepareReviewBundle("milestone:program/core"); err != nil || !childBlocker(milestone) {
		t.Fatalf("a child whose latest review requested changes was retained: %v %v", milestone.Blockers, err)
	}
	root, store, _ := setup(t, "approved")
	specPath := filepath.Join(root, ".pose", "specs", "backend", "spec.md")
	raw, err := os.ReadFile(specPath)
	if err != nil {
		t.Fatal(err)
	}
	writeReviewFixture(t, root, ".pose/specs/backend/spec.md", strings.Replace(string(raw), "status: done", "status: in-progress", 1))
	federatedTestGit(t, root, "add", "--all")
	federatedTestGit(t, root, "commit", "-q", "-m", "reopened")
	if milestone, err := store.PrepareReviewBundle("milestone:program/core"); err != nil || !childBlocker(milestone) {
		t.Fatalf("a child that is not done was retained: %v %v", milestone.Blockers, err)
	}
}

// SR-R1-02: an entry several selected specs observe is a removal only when
// the removal holds for each of them, whichever comes first.
func TestReviewSubjectRemovalHoldsForEveryObserver(t *testing.T) {
	graph := DeliveryIntegrityGraph{Removals: []ArtifactRemoval{{Spec: "later", Path: "x.go", Commit: "c1", Supersedes: map[string]string{"peer": "c1"}}}}
	for _, owners := range [][]string{{"backend", "peer"}, {"peer", "backend"}, {""}, nil} {
		if _, ok := reviewBundleRemovalWitness(graph, "x.go", owners); ok {
			t.Fatalf("a removal accepted for peer only stood for %v", owners)
		}
	}
	graph.Removals[0].Supersedes["backend"] = "c2"
	witness, ok := reviewBundleRemovalWitness(graph, "x.go", []string{"peer", "backend", "peer"})
	if !ok || witness != "removed-by later c1\nremoved-by later c2" {
		t.Fatalf("a removal accepted for every observer: %v %q", ok, witness)
	}
}

// SR-R1-03: a retained child is what its own verification retains, so a
// revoked federated authority refuses it.
func TestRetainedChildFollowsItsVerification(t *testing.T) {
	root, store, trust, _ := federatedTestMilestoneConsumer(t)
	now := time.Date(2026, 9, 26, 18, 0, 0, 0, time.UTC)
	attest := func(bundle ReviewBundle) {
		t.Helper()
		attestation := approvedBundleAttestation(bundle, "agent:review")
		attestation.AttestedAt = now.Add(time.Minute).Format(time.RFC3339)
		if _, err := store.RecordReviewAttestation(attestation, now.Add(time.Minute)); err != nil {
			t.Fatal(err)
		}
	}
	specBundle, err := store.SealReviewBundle("spec:backend", now)
	if err != nil {
		t.Fatalf("seal spec: %v; blockers=%v", err, specBundle.Blockers)
	}
	attest(specBundle)
	milestone, err := store.SealReviewBundle("milestone:program/core", now)
	if err != nil {
		t.Fatalf("seal milestone: %v; blockers=%v", err, milestone.Blockers)
	}
	attest(milestone)
	writeReviewFixture(t, root, "api/server.go", "package api\n\nfunc Ready() bool { return !false }\n")
	federatedTestGit(t, root, "add", "--all")
	federatedTestGit(t, root, "commit", "-q", "-m", "after closeout")
	if retained, why := store.retainedChildReviewBundle("milestone:program/core"); retained == nil || retained.BundleID != milestone.BundleID {
		t.Fatalf("a done milestone's approved review was not retained: %v %q", retained, why)
	}
	federatedTestTrust(t, root, map[string]FederatedProjectTrust{})
	if retained, _ := store.retainedChildReviewBundle("milestone:program/core"); retained != nil {
		t.Fatal("a child whose federated authority was revoked was retained")
	}
	federatedTestTrust(t, root, map[string]FederatedProjectTrust{"source": trust})
	if retained, _ := store.retainedChildReviewBundle("milestone:program/core"); retained == nil {
		t.Fatal("restored trust did not retain the child again")
	}
}

// SR-R1-02 through a milestone: two members observe the removed path, the
// one whose change set sorts first with the accepted removal.
func TestMilestoneSubjectRemovalHoldsForEveryMember(t *testing.T) {
	// SR-R2-01: also when the member's commits are contained in the peer's,
	// so that reduction keeps one set for both.
	for _, c := range []struct {
		both     bool
		id       string
		subsumed bool
	}{{false, "cs-a-peer", false}, {false, "cs-z-peer", false}, {true, "cs-a-peer", false}, {true, "cs-z-peer", false},
		{false, "cs-a-peer", true}, {false, "cs-z-peer", true}, {true, "cs-a-peer", true}, {true, "cs-z-peer", true}} {
		both := c.both
		root, store, _, _ := federatedTestMilestoneConsumer(t)
		writeReviewFixture(t, root, ".pose/specs/peer/spec.md", "---\nslug: peer\nstatus: done\ncreated_at: 2026-08-13\ncomponents: api\n---\n\n# Spec: peer\n\n## 3. Technical Plan\n\n### Artifacts\n- modified: api/server.go\n")
		writeReviewFixture(t, root, ".pose/roadmaps/program.md", "---\nslug: program\nstatus: active\nconsumes: xref:source/roadmap:component\n---\n\n## Milestone: core\n- specs: backend, peer\n\n## Milestone: later\n- after: core\n- specs: pending\n")
		path := filepath.Join(root, ".pose", "indexes", "delivery-integrity.json")
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		var graph DeliveryIntegrityGraph
		if err := json.Unmarshal(raw, &graph); err != nil {
			t.Fatal(err)
		}
		peerSet := graph.ChangeSets[0]
		peerSet.ID, peerSet.Spec = c.id, "peer"
		if c.subsumed {
			graph.ChangeSets[0].Commits = []string{"shared"}
			peerSet.Commits = []string{"shared", "extra"}
		}
		graph.ChangeSets = append([]ChangeSet{peerSet}, graph.ChangeSets...)
		removal := ArtifactRemoval{Spec: "later", Path: "api/server.go", Commit: "c1", Supersedes: map[string]string{"peer": "c1"}}
		if both {
			removal.Supersedes["backend"] = "c1"
		}
		graph.Removals = []ArtifactRemoval{removal}
		raw, err = json.MarshalIndent(graph, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		writeReviewFixture(t, root, ".pose/indexes/delivery-integrity.json", string(raw))
		if err := os.Remove(filepath.Join(root, "api", "server.go")); err != nil {
			t.Fatal(err)
		}
		federatedTestGit(t, root, "add", "--all")
		federatedTestGit(t, root, "commit", "-q", "-m", "a later spec removes the file")
		bundle, err := store.PrepareReviewBundle("milestone:program/core")
		if err != nil {
			t.Fatal(err)
		}
		unreadable := false
		for _, blocker := range bundle.Blockers {
			// The milestone's own subject, not a member's verification.
			unreadable = unreadable || strings.HasPrefix(blocker, "review subject path api/server.go cannot be read")
		}
		if unreadable == both {
			t.Fatalf("removal for every member %v, peer set %s, subsumed %v: unreadable %v; blockers %v; change sets %v", both, c.id, c.subsumed, unreadable, bundle.Blockers, bundle.Payload.Subject.ChangeSets)
		}
	}
}
