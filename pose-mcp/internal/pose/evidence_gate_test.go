package pose

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// sealWithDerivedEvidence writes the bundle an engine before this gate sealed: the
// same bundle, with nothing but what is derived from the subject.
func sealWithDerivedEvidence(t *testing.T, s Store, bundle ReviewBundle) ReviewBundle {
	t.Helper()
	bundle.Payload.Evidence = []ReviewBundleEvidence{{ID: "design-0", Check: "assess-design", EvidenceClass: "structure", Outcome: "pass", Report: "derived:design-delta/0"}}
	digest, err := reviewBundlePayloadDigest(bundle.Payload)
	if err != nil {
		t.Fatal(err)
	}
	old := bundle.BundleID
	bundle.BundleDigest, bundle.BundleID = digest, "rvb-"+strings.TrimPrefix(digest, "sha256:")[:16]
	bundle.Path = strings.ReplaceAll(bundle.Path, old, bundle.BundleID)
	raw, err := json.MarshalIndent(bundle, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(s.reviewBundlesDir(), bundle.BundleID+".json"), raw, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(s.reviewBundlesDir(), old+".json")); err != nil {
		t.Fatal(err)
	}
	return bundle
}

func TestAutoAttestRefusesDerivedEvidence(t *testing.T) {
	_, s := reviewBundleFixture(t)
	bundle, err := s.SealReviewBundle("spec:backend", time.Date(2026, 9, 26, 18, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err, bundle.Blockers)
	}
	derived := sealWithDerivedEvidence(t, s, bundle)
	if _, err := s.LoadReviewBundle(derived.BundleID); err != nil {
		t.Fatalf("the fixture does not read as a sealed bundle: %v", err)
	}
	if _, err := s.PrepareReviewAttestation(derived.BundleID, "agent:auto", time.Date(2026, 9, 26, 19, 0, 0, 0, time.UTC)); err == nil || !strings.Contains(err.Error(), "no passed structured validation evidence") {
		t.Fatalf("auto-attest took derived evidence for a validation result: %v", err)
	}
}

func TestClosedScopeKeepsItsDerivedSeal(t *testing.T) {
	root, s := closedMilestoneFixture(t)
	path := filepath.Join(root, ".pose/indexes/delivery-integrity.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var graph map[string]any
	if err := json.Unmarshal(raw, &graph); err != nil {
		t.Fatal(err)
	}
	before, err := s.VerifyReviewBundle("spec:backend")
	if err != nil || !before.Approved {
		t.Fatalf("the fixture is not approved: %+v %v", before, err)
	}
	delete(graph, "validation_results")
	raw, _ = json.Marshal(graph)
	federatedTestWrite(t, root, ".pose/indexes/delivery-integrity.json", string(raw))
	federatedTestGit(t, root, "add", "--all")
	federatedTestGit(t, root, "commit", "-q", "-m", "no result in the index")
	after, err := s.VerifyReviewBundle("spec:backend")
	if err != nil || !after.Approved {
		t.Fatalf("a closed scope lost its approval when its evidence left the index: %+v %v", after, err)
	}
}
