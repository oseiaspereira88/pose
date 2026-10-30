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
