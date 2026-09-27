package pose

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func closeoutBlockers(t *testing.T, store Store, structure *ReviewPlanStructure, band string, required map[string]ReviewPlanCriterion, disposition string, mappings []ReviewCriterionMapping) []string {
	t.Helper()
	if disposition == "" {
		disposition = "passed"
	}
	return store.reviewCausalityCloseoutBlockers("spec:backend", structure, band, required, []ReviewCriterion{
		{ID: "design-causality", Disposition: disposition, Rationale: "examined both sides of the subject", Mappings: mappings},
		{ID: "correctness", Disposition: "passed", Evidence: "unit:u1"},
	})
}

func rewriteCausalitySpec(t *testing.T, store Store, from, to string) {
	t.Helper()
	path := filepath.Join(store.Root, ".pose", "specs", "backend", "spec.md")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), from) {
		t.Fatalf("fixture has no %q", from)
	}
	if err := os.WriteFile(path, []byte(strings.Replace(string(raw), from, to, 1)), 0o644); err != nil {
		t.Fatal(err)
	}
}

func goodCausalityMappings() []ReviewCriterionMapping {
	return []ReviewCriterionMapping{
		{Delta: "SD-aaaa1111", Basis: "D1", Rationale: "the library replaces the handwritten client D1 chose to drop"},
		{Delta: "SD-bbbb2222", Basis: "R1", Rationale: "the contract change is the justified surface R1 allows"},
	}
}

func TestABMCausalityCloseoutAcceptsAJustifiedLiveMapping(t *testing.T) {
	store := causalityFixture(t)
	structure, required := causalityInputs()
	if blockers := closeoutBlockers(t, store, structure, "elevated", required, "", goodCausalityMappings()); len(blockers) != 0 {
		t.Fatalf("a justified, live mapping was blocked: %+v", blockers)
	}
}

func TestABMCausalityCloseoutRefusesADeadBasis(t *testing.T) {
	store := causalityFixture(t)
	structure, required := causalityInputs()
	rewriteCausalitySpec(t, store, "- Status: verified", "- Status: invalidated")
	mappings := goodCausalityMappings()
	mappings[0].Basis = "A1"
	assertBlocker(t, closeoutBlockers(t, store, structure, "", required, "", mappings), "to assumption A1, which is invalidated")

	store = causalityFixture(t)
	rewriteCausalitySpec(t, store, "- Falsifier: the library stops being maintained.", "- Falsifier: the library stops being maintained.\n- Status: withdrawn")
	assertBlocker(t, closeoutBlockers(t, store, structure, "", required, "", goodCausalityMappings()), "to withdrawn decision D1")
}

func TestABMCausalityCloseoutSeparatesCompletenessFromAcceptance(t *testing.T) {
	store := causalityFixture(t)
	structure, required := causalityInputs()
	mappings := goodCausalityMappings()
	mappings[1].Rationale = ""
	assertBlocker(t, closeoutBlockers(t, store, structure, "", required, "", mappings), "maps SD-bbbb2222 to R1 without saying why")

	structure.Material = append(structure.Material, ReviewStructuralFact{ID: "SD-cccc3333", Kind: "dependency", Action: "added", Subject: "go:github.com/another/dep"})
	pasted := []ReviewCriterionMapping{
		{Delta: "SD-aaaa1111", Basis: "R1", Rationale: "Needed for R1."},
		{Delta: "SD-bbbb2222", Basis: "R1", Rationale: "needed for   R1."},
		{Delta: "SD-cccc3333", Basis: "R1", Rationale: "needed for R1."},
	}
	assertBlocker(t, closeoutBlockers(t, store, structure, "", required, "", pasted), "justifies 3 facts (SD-aaaa1111, SD-bbbb2222, SD-cccc3333) on R1 with one pasted reason")
	pasted[2].Rationale = "the second library covers the retry path R1 names"
	if blockers := closeoutBlockers(t, store, structure, "", required, "", pasted); len(blockers) != 0 {
		t.Fatalf("two identical reasons were treated as disproportionate: %+v", blockers)
	}
}

func TestABMCausalityCloseoutBoundsAcceptedRisk(t *testing.T) {
	store := causalityFixture(t)
	structure, required := causalityInputs()
	integrity := []ReviewCriterionMapping{
		goodCausalityMappings()[0],
		{Delta: "SD-bbbb2222", Disposition: ReviewMappingAcceptedRisk, Rationale: "ship now", Owner: "@api", ReviewBy: "2026-12-01"},
	}
	assertBlocker(t, closeoutBlockers(t, store, structure, "", required, "", integrity), "an integrity fact is mapped or raised as a finding, never waived")
	unowned := []ReviewCriterionMapping{
		{Delta: "SD-aaaa1111", Disposition: ReviewMappingAcceptedRisk, Rationale: "vendored later"},
		goodCausalityMappings()[1],
	}
	assertBlocker(t, closeoutBlockers(t, store, structure, "", required, "", unowned), "as a risk without an @owner and a review-by date")
	unowned[0].Owner, unowned[0].ReviewBy = "@platform", "2026-12-01"
	if blockers := closeoutBlockers(t, store, structure, "", required, "", unowned); len(blockers) != 0 {
		t.Fatalf("an owned, dated risk on a dependency was blocked: %+v", blockers)
	}
}

func TestABMCausalityCloseoutUnknownIsNotAbsence(t *testing.T) {
	store := causalityFixture(t)
	_, required := causalityInputs()
	unknown := &ReviewPlanStructure{Observed: true, Status: "partial", Unknown: []string{"detector:npm:unsupported"}}
	assertBlocker(t, closeoutBlockers(t, store, unknown, "", required, "not-applicable", nil), "not-applicable while structural coverage is unknown (detector:npm:unsupported)")
	clean := &ReviewPlanStructure{Observed: true, Status: "observed"}
	if blockers := closeoutBlockers(t, store, clean, "", required, "not-applicable", nil); len(blockers) != 0 {
		t.Fatalf("a fully observed empty structure could not be not-applicable: %+v", blockers)
	}
}

func TestABMCausalityCloseoutRaisesElevatedObligationsOnly(t *testing.T) {
	store := causalityFixture(t)
	structure, _ := causalityInputs()
	noOwner := map[string]ReviewPlanCriterion{"correctness": {ID: "correctness", Required: true, EvidenceClasses: []string{"unit"}}}
	for _, band := range []string{"elevated", "critical"} {
		assertBlocker(t, closeoutBlockers(t, store, structure, band, noOwner, "", nil), "the "+band+" band seals 2 material structural facts and no required criterion answers for observed structure")
	}
	if blockers := closeoutBlockers(t, store, structure, "baseline", noOwner, "", nil); len(blockers) != 0 {
		t.Fatalf("a baseline plan gained a causality gate: %+v", blockers)
	}
	trivial := &ReviewPlanStructure{Observed: true, Status: "observed"}
	if blockers := closeoutBlockers(t, store, trivial, "critical", noOwner, "", nil); len(blockers) != 0 {
		t.Fatalf("a trivial subject gained a causality gate: %+v", blockers)
	}
	for _, blocker := range closeoutBlockers(t, store, structure, "critical", noOwner, "", nil) {
		if strings.Contains(strings.ToLower(blocker), "human") {
			t.Fatalf("a critical band imposed a mandatory human: %s", blocker)
		}
	}
}

func TestABMCausalityCloseoutIsStampedOnlyOnAdoptionAndSealsTheBasis(t *testing.T) {
	root, store := reviewBundleFixture(t)
	legacy, err := store.PrepareReviewBundle("spec:backend")
	if err != nil {
		t.Fatal(err)
	}
	if governed, _ := BundleGovernedBy(legacy, CausalityCloseoutContract); governed || legacy.Payload.Basis != nil || legacy.Payload.Plan.Band != "" {
		t.Fatalf("an unadopted policy stamped the causality-closeout contract: %+v", legacy.Payload.GoverningContracts)
	}
	policyPath := filepath.Join(root, ".pose", "policy", "review.json")
	raw, err := os.ReadFile(policyPath)
	if err != nil {
		t.Fatal(err)
	}
	var policy map[string]any
	if err := json.Unmarshal(raw, &policy); err != nil {
		t.Fatal(err)
	}
	policy["causality_closeout_version"] = 1
	raw, _ = json.Marshal(policy)
	if err := os.WriteFile(policyPath, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	adopted, err := store.PrepareReviewBundle("spec:backend")
	if err != nil {
		t.Fatal(err)
	}
	if governed, _ := BundleGovernedBy(adopted, CausalityCloseoutContract); !governed || adopted.Payload.Basis["backend"] == "" {
		t.Fatalf("an adopted policy did not stamp the contract and seal the basis: %+v", adopted.Payload)
	}
	specPath := filepath.Join(root, ".pose", "specs", "backend", "spec.md")
	specRaw, _ := os.ReadFile(specPath)
	if err := os.WriteFile(specPath, []byte(strings.Replace(string(specRaw), "- R1: The backend shall remain compatible.", "- R1: The backend shall remain compatible with v1 clients.", 1)), 0o644); err != nil {
		t.Fatal(err)
	}
	changed, err := store.PrepareReviewBundle("spec:backend")
	if err != nil {
		t.Fatal(err)
	}
	if changed.Payload.Basis["backend"] == adopted.Payload.Basis["backend"] || changed.BundleDigest == adopted.BundleDigest {
		t.Fatal("a material requirement change did not change the sealed basis")
	}
	policy["causality_closeout_version"] = 2
	raw, _ = json.Marshal(policy)
	_ = os.WriteFile(policyPath, raw, 0o644)
	if _, err := store.GetReviewPolicy(); err == nil || !strings.Contains(err.Error(), "unsupported causality_closeout_version") {
		t.Fatalf("an unknown capability version was accepted: %v", err)
	}
}

// The rules run on the attestation path, and only for a stamped bundle.
func TestABMCausalityCloseoutRunsOnlyForStampedBundles(t *testing.T) {
	store := causalityFixture(t)
	structure, required := causalityInputs()
	criteria := []ReviewPlanCriterion{}
	for _, criterion := range required {
		criteria = append(criteria, criterion)
	}
	bundle := ReviewBundle{BundleID: "rvb-closeout", BundleDigest: "sha256:closeout", Payload: ReviewBundlePayload{
		Scope: ReviewBundleScope{Ref: "spec:backend", Kind: "spec", Slug: "backend"},
		Plan:  ReviewBundlePlan{Criteria: criteria, Structure: structure, Band: "elevated"},
	}}
	mappings := goodCausalityMappings()
	mappings[1].Rationale = ""
	att := ReviewAttestation{BundleID: "rvb-closeout", BundleDigest: "sha256:closeout", Decision: "approved", Criteria: []ReviewCriterion{
		{ID: "design-causality", Disposition: "passed", Rationale: "examined both sides", Mappings: mappings},
		{ID: "correctness", Disposition: "passed", Evidence: "unit:u1"},
	}}
	bundle.Payload.GoverningContracts = []string{"structural-causality"}
	for _, blocker := range store.validateBundleAttestation(bundle, att) {
		if strings.Contains(blocker, "without saying why") {
			t.Fatalf("an unstamped bundle was held to causality closeout: %s", blocker)
		}
	}
	bundle.Payload.GoverningContracts = []string{CausalityCloseoutContract, "structural-causality"}
	assertBlocker(t, store.validateBundleAttestation(bundle, att), "maps SD-bbbb2222 to R1 without saying why")
}
