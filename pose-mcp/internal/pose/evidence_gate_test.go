package pose

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// closedDerivedSeal is a done spec sealed over design evidence alone, with a profile
// whose criterion takes it, as engines before this gate sealed and approved it.
func closedDerivedSeal(t *testing.T) (string, Store, ReviewBundle) {
	t.Helper()
	root, s := reviewBundleFixture(t)
	policy := filepath.Join(root, ".pose/policy/review.json")
	raw, err := os.ReadFile(policy)
	if err != nil {
		t.Fatal(err)
	}
	text := strings.Replace(string(raw), `"overlay_profiles": ["frontend-review@1", "backend-review@1"]`, `"overlay_profiles": []`, 1)
	text = strings.Replace(text, `"schema_version": 2,`, `"schema_version": 2, "review_bundles":true, "review_bundles_adopted_at":"2026-08-13",`, 1)
	writeReviewFixture(t, root, ".pose/policy/review.json", text)
	writeReviewFixture(t, root, ".pose/review-profiles/spec-closeout.json", `{"schema_version":2,"id":"spec-closeout","version":2,"scope":"spec","criteria":[{"id":"correctness","description":"Structural correctness","kind":"judgment","evidence_classes":["structure"]}]}`)
	spec := filepath.Join(root, ".pose/specs/backend/spec.md")
	raw, err = os.ReadFile(spec)
	if err != nil {
		t.Fatal(err)
	}
	writeReviewFixture(t, root, ".pose/specs/backend/spec.md", strings.Replace(string(raw), "status: in-progress", "status: done\ncompleted_at: 2026-09-26", 1))
	dropValidationResults(t, root)
	writeReviewFixture(t, root, ".pose/indexes/validation-matrix.json", `{"deliveryProfiles":{"api-contract":{"kind":"contract","requiredEvidenceClasses":["structure"]}},"stacks":{}}`)
	commitEvidenceFixture(t, root)
	bundle, err := s.PrepareReviewBundle("spec:backend")
	if err != nil || len(bundle.Blockers) != 0 {
		t.Fatal(err, bundle.Blockers)
	}
	if len(bundle.Payload.Evidence) != 1 || holdsValidationResult(bundle.Payload.Evidence) {
		t.Fatalf("the bundle is not derived alone: %+v", bundle.Payload.Evidence)
	}
	// Written by hand: this engine refuses to seal it, which is the point.
	bundle.State, bundle.SealedAt = "sealed", "2026-09-26T18:00:00Z"
	bundle.Path = filepath.ToSlash(filepath.Join(".pose", "review-bundles", bundle.BundleID+".json"))
	sealed, err := json.MarshalIndent(bundle, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	writeReviewFixture(t, root, bundle.Path, string(sealed))
	if _, err := s.LoadReviewBundle(bundle.BundleID); err != nil {
		t.Fatal(err)
	}
	return root, s, bundle
}

func TestNoNewSealOverDerivedEvidence(t *testing.T) {
	root, s, bundle := closedDerivedSeal(t)
	if again, err := s.SealReviewBundle("spec:backend", time.Now()); err != nil || again.BundleID != bundle.BundleID {
		t.Fatalf("sealing the same content did not hand back the governing bundle: %+v %v", again.BundleID, err)
	}
	if err := os.Remove(filepath.Join(root, bundle.Path)); err != nil {
		t.Fatal(err)
	}
	if _, err := s.SealReviewBundle("spec:backend", time.Now()); err == nil || !strings.Contains(err.Error(), noValidationEvidenceBlocker) {
		t.Fatalf("a done spec sealed anew over derived evidence: %v", err)
	}
}

func dropValidationResults(t *testing.T, root string) {
	t.Helper()
	path := filepath.Join(root, ".pose/indexes/delivery-integrity.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var graph DeliveryIntegrityGraph
	if err := json.Unmarshal(raw, &graph); err != nil {
		t.Fatal(err)
	}
	graph.ValidationResults = nil
	raw, _ = json.Marshal(graph)
	writeReviewFixture(t, root, ".pose/indexes/delivery-integrity.json", string(raw))
}

func commitEvidenceFixture(t *testing.T, root string) {
	t.Helper()
	federatedTestGit(t, root, "init", "-q")
	federatedTestGit(t, root, "config", "user.name", "fixture")
	federatedTestGit(t, root, "config", "user.email", "fixture@example.invalid")
	federatedTestGit(t, root, "add", "--all")
	federatedTestGit(t, root, "commit", "-q", "-m", "fixture")
}

func TestAutoAttestRefusesDerivedEvidence(t *testing.T) {
	_, s, bundle := closedDerivedSeal(t)
	if _, err := s.PrepareReviewAttestation(bundle.BundleID, "agent:auto", time.Date(2026, 9, 26, 19, 0, 0, 0, time.UTC)); err == nil || !strings.Contains(err.Error(), "no passed structured validation evidence") {
		t.Fatalf("auto-attest took derived evidence for a validation result: %v", err)
	}
}

func TestClosedScopeKeepsItsDerivedSeal(t *testing.T) {
	root, s, bundle := closedDerivedSeal(t)
	at := time.Date(2026, 9, 26, 19, 0, 0, 0, time.UTC)
	if _, err := s.RecordReviewAttestation(approvedBundleAttestation(bundle, "agent:reviewer"), at); err != nil {
		t.Fatal(err)
	}
	stands := func(when string) {
		t.Helper()
		verification, err := s.VerifyReviewBundle("spec:backend")
		if err != nil || !verification.Approved || !verification.Fresh {
			t.Fatalf("%s: verify lost the approval: %+v %v", when, verification, err)
		}
		eval, err := s.ReviewCheck("spec:backend")
		if err != nil || !eval.Approved {
			t.Fatalf("%s: review-check lost the approval: %+v %v", when, eval, err)
		}
		plan, err := s.PlanCloseout("spec:backend")
		if err != nil || !plan.Terminal {
			t.Fatalf("%s: closeout lost the approval: %+v %v", when, plan, err)
		}
		_, _, _, _, _, _, blockers := federatedSourceProof(NativeFiles(), s, ArtifactRef{Kind: "spec", Slug: "backend", Project: "source"}, gitHeadAtRoot(root))
		if len(blockers) != 0 {
			t.Fatalf("%s: a consumer lost the proof: %v", when, blockers)
		}
	}
	stands("as sealed")
	writeReviewFixture(t, root, "unrelated.txt", "later\n")
	federatedTestGit(t, root, "add", "--all")
	federatedTestGit(t, root, "commit", "-q", "-m", "later work")
	stands("after later work")
}

func TestAResultAtADerivedPathIsValidation(t *testing.T) {
	root, s := reviewBundleFixture(t)
	path := filepath.Join(root, ".pose/indexes/delivery-integrity.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var graph DeliveryIntegrityGraph
	if err := json.Unmarshal(raw, &graph); err != nil {
		t.Fatal(err)
	}
	if len(graph.ValidationResults) == 0 {
		t.Fatal("the fixture has no result")
	}
	for i := range graph.ValidationResults {
		graph.ValidationResults[i].Report = "derived:results.json"
	}
	raw, _ = json.Marshal(graph)
	writeReviewFixture(t, root, ".pose/indexes/delivery-integrity.json", string(raw))
	commitEvidenceFixture(t, root)
	bundle, err := s.PrepareReviewBundle("spec:backend")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(strings.Join(bundle.Blockers, " "), "no passed structured validation evidence") {
		t.Fatalf("a result was read as derived for the path of its report: %v", bundle.Blockers)
	}
	if !holdsValidationResult([]ReviewBundleEvidence{{ID: "real-run", Check: "assess-design", EvidenceClass: "structure", Report: "derived:design-delta/run.json", ProvenanceDigest: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}}) {
		t.Fatal("a result with every field the operator gives it was read as derived")
	}
}

func TestDesignEvidenceIsRecognizedByItsOwnDigest(t *testing.T) {
	_, _, bundle := closedDerivedSeal(t)
	if !derivedDesignEvidence(bundle.Payload.Evidence[0]) {
		t.Fatalf("the entry the engine appends was not recognized: %+v", bundle.Payload.Evidence[0])
	}
}
