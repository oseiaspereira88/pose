package pose

import (
	"encoding/json"
	"io/fs"
	"reflect"
	"strings"
	"testing"

	"github.com/harne8/pose-mcp/internal/scaffold"
)

// structuralFixture commits a base with one direct dependency, then leaves the
// working tree and the attributed change set describing the head, so the plan
// observes the subject the way a review would.
func structuralFixture(t *testing.T, baseManifest, headManifest string, extra ...string) (string, Store) {
	t.Helper()
	root, store := componentReviewFixture(t)
	designDeltaGit(t, root, "init", "-q")
	designDeltaGit(t, root, "config", "user.email", "pose@example.test")
	designDeltaGit(t, root, "config", "user.name", "POSE Test")
	writeReviewFixture(t, root, ".pose/indexes/repo-map.json", `{"services":[{"name":"api","path":"api","language":"go","criticality":"medium","metadataStatus":{"source":"declared"}}]}`)
	writeReviewFixture(t, root, ".pose/review-profiles/structural-materiality.json", mustReadDistProfile(t, "structural-materiality"))
	writeReviewFixture(t, root, "api/go.mod", baseManifest)
	writeReviewFixture(t, root, "api/README.md", "# api\n")
	writeReviewFixture(t, root, ".pose/specs/backend/spec.md", "---\nslug: backend\nstatus: in-progress\ncreated_at: 2026-09-20\ncomponents: api\n---\n"+
		"# Spec\n\n## 2. Requirements\n\n- R1: keep the api dependency surface justified.\n\n### Artifacts\n- modified: api/go.mod\n- modified: api/README.md\n")
	designDeltaGit(t, root, "add", "--", ".")
	designDeltaGit(t, root, "commit", "-q", "-m", "base")
	base := strings.TrimSpace(string(mustDesignDeltaGitOutput(t, root, "rev-parse", "HEAD")))
	writeReviewFixture(t, root, "api/go.mod", headManifest)
	writeReviewFixture(t, root, "api/README.md", "# api\n\nDocumented.\n")
	for i := 0; i+1 < len(extra); i += 2 {
		writeReviewFixture(t, root, extra[i], extra[i+1])
	}
	designDeltaGit(t, root, "add", "--", ".")
	designDeltaGit(t, root, "commit", "-q", "-m", "head")
	head := strings.TrimSpace(string(mustDesignDeltaGitOutput(t, root, "rev-parse", "HEAD")))
	paths := []ObservedPath{{Action: "modified", Path: "api/go.mod"}, {Action: "modified", Path: "api/README.md"}}
	for i := 0; i+1 < len(extra); i += 2 {
		paths = append(paths, ObservedPath{Action: "modified", Path: extra[i]})
	}
	graph := DeliveryIntegrityGraph{
		SchemaVersion: DeliveryIntegritySchemaVersion,
		ChangeSets: []ChangeSet{{
			ID: "cs-backend", Spec: "backend", ResolvedBase: base, ResolvedHead: head, Paths: paths,
		}},
		Reverse: map[string][]string{"api/go.mod": {"backend"}},
	}
	raw, err := json.Marshal(graph)
	if err != nil {
		t.Fatal(err)
	}
	writeReviewFixture(t, root, ".pose/indexes/delivery-integrity.json", string(raw))
	return root, store
}

func adoptStructuralMateriality(t *testing.T, store Store) {
	t.Helper()
	policy, _, err := store.loadReviewPolicy()
	if err != nil {
		t.Fatal(err)
	}
	policy.OverlayProfiles = append(policy.OverlayProfiles, "structural-materiality@1")
	raw, err := json.Marshal(policy)
	if err != nil {
		t.Fatal(err)
	}
	writeReviewFixture(t, store.Root, ".pose/policy/review.json", string(raw))
}

const structuralBaseManifest = "module example.test/api\n\ngo 1.24\n\nrequire (\n\tgithub.com/kept/dep v1.0.0\n)\n"

// R6: an undeclared scope with a real dependency change is material. The same
// scope with nothing structural observed gains nothing at all.
func TestABMStructuralSelectorsObserveWhatWasNotDeclared(t *testing.T) {
	head := "module example.test/api\n\ngo 1.24\n\nrequire (\n\tgithub.com/kept/dep v1.0.0\n\tgithub.com/new/dep v2.0.0\n)\n"
	_, store := structuralFixture(t, structuralBaseManifest, head)
	before := progressivePlan(t, store)
	if before.Structure != nil {
		t.Fatalf("structure resolved without an adopted structural selector: %+v", before.Structure)
	}
	adoptStructuralMateriality(t, store)
	plan := progressivePlan(t, store)
	if plan.Structure == nil || !plan.Structure.Observed || !containsFold(plan.Structure.Kinds, "dependency") {
		t.Fatalf("subject was not observed: %+v", plan.Structure)
	}
	if plan.Structure.SubjectDigest == "" || plan.Structure.InputDigest == "" {
		t.Fatalf("observation is not anchored to the subject: %+v", plan.Structure)
	}
	material := []string{}
	for _, fact := range plan.Structure.Material {
		material = append(material, fact.Kind+"/"+fact.Action+"/"+fact.Subject)
	}
	if !containsFold(material, "dependency/added/go:github.com/new/dep") {
		t.Fatalf("the added direct dependency is not material: %+v", material)
	}
	criterion := ReviewPlanCriterion{}
	for _, candidate := range plan.Criteria {
		if candidate.ID == "design-causality" {
			criterion = candidate
		}
	}
	if !criterion.RequiresStructuralMapping || ReviewCriterionKind(criterion) != ReviewCriterionKindJudgment {
		t.Fatalf("the mapping obligation did not compose: %+v", criterion)
	}
	band := planBandFor(t, plan, "structural-materiality@1")
	if band.Trigger != "structural_kind=dependency" || band.Basis != reviewBasisObserved ||
		!strings.Contains(band.Source, "subject-observation") {
		t.Fatalf("structural trigger is not explained as an observation: %+v", band)
	}
	if !reflect.DeepEqual(plan.Projection.ObservedStructure, []string{"dependency"}) || !plan.Projection.ScopeExpanded {
		t.Fatalf("observation was reported as a declared forecast: %+v", plan.Projection)
	}
	if !containsFold(plan.Projection.AddedCriteria, "design-causality") {
		t.Fatalf("expansion did not attribute the added obligation: %+v", plan.Projection)
	}
	if plan.PlanDigest == before.PlanDigest {
		t.Fatal("material observation left plan identity unchanged")
	}
}

// Transitive dependencies, lock files, renames and unreadable manifests are
// observed and reported, and none of them creates an obligation.
func TestABMStructuralMaterialityExcludesTransitiveAndUncertain(t *testing.T) {
	head := "module example.test/api\n\ngo 1.24\n\nrequire (\n\tgithub.com/kept/dep v1.0.0\n\tgithub.com/transitive/dep v1.0.0 // indirect\n)\n"
	_, store := structuralFixture(t, structuralBaseManifest, head,
		"api/go.sum", "github.com/transitive/dep v1.0.0 h1:abc=\n",
		"api/Cargo.toml", "[package]\nname = \"api\"\n")
	adoptStructuralMateriality(t, store)
	plan := progressivePlan(t, store)
	if plan.Structure == nil || !plan.Structure.Observed {
		t.Fatalf("subject was not observed: %+v", plan.Structure)
	}
	if len(plan.Structure.Material) != 0 || len(plan.Structure.Kinds) != 0 {
		t.Fatalf("transitive, lock or unsupported input became material: %+v", plan.Structure.Material)
	}
	if len(plan.Structure.Unknown) == 0 {
		t.Fatalf("unsupported manifest was not reported as uncertainty: %+v", plan.Structure)
	}
	for _, band := range plan.Bands {
		if band.Policy == "structural-materiality@1" {
			t.Fatalf("a non-material observation selected the profile: %+v", band)
		}
	}
	if planHasCriterion(plan, "design-causality") {
		t.Fatalf("a non-material observation added an obligation: %+v", plan.Criteria)
	}
}

func mustReadDistProfile(t *testing.T, id string) string {
	t.Helper()
	raw, err := fs.ReadFile(scaffold.Dist(), ".pose/review-profiles/"+id+".json")
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

// The R7 corpus works against the gate directly, so each rule is measured on its
// own predicate instead of through a whole sealed bundle.
func causalityFixture(t *testing.T) Store {
	t.Helper()
	root, store := componentReviewFixture(t)
	writeReviewFixture(t, root, ".pose/specs/backend/spec.md", `---
slug: backend
status: in-progress
created_at: 2026-09-20
components: api
---
# Spec

## 2. Requirements

- R1: keep the dependency surface justified.
- C1: no new runtime process.

## 5. Decisions

### Assumption A1
- Claim: the new library is maintained and vendorable.
- Status: verified
- Evidence: report:2026-09-18-pose-abm-corroboration
- Scope: api module, revision 2026-09-20
- Affects: R1

### Assumption A9
- Claim: unrelated to any requirement.
- Status: unknown

### Decision D1
- Basis: R1, A1
- Minimal option: keep the handwritten client.
- Selected option: adopt the library.
- Rationale: R1 is satisfied with less code.
- Consequences: one direct dependency.
- Falsifier: the library stops being maintained.

### Decision D9
- Basis: A9
- Selected option: something unjustified.
`)
	return store
}

func causalityInputs() (*ReviewPlanStructure, map[string]ReviewPlanCriterion) {
	structure := &ReviewPlanStructure{
		Observed: true, Status: "observed", Kinds: []string{"dependency", "public-contract"},
		Material: []ReviewStructuralFact{
			{ID: "SD-aaaa1111", Kind: "dependency", Action: "added", Subject: "go:github.com/new/dep"},
			{ID: "SD-bbbb2222", Kind: "public-contract", Action: "changed", Subject: "api/openapi.yaml"},
		},
	}
	required := map[string]ReviewPlanCriterion{
		"design-causality": {ID: "design-causality", Required: true, Kind: ReviewCriterionKindJudgment, RequiresStructuralMapping: true},
		"correctness":      {ID: "correctness", Required: true, EvidenceClasses: []string{"unit"}},
	}
	return structure, required
}

func causalityBlockers(t *testing.T, store Store, mappings []ReviewCriterionMapping, disposition string) []string {
	t.Helper()
	structure, required := causalityInputs()
	if disposition == "" {
		disposition = "passed"
	}
	return store.reviewStructuralCausalityBlockers("spec:backend", structure, required,
		[]ReviewCriterion{
			{ID: "design-causality", Disposition: disposition, Rationale: "examined both sides of the subject", Mappings: mappings},
			{ID: "correctness", Disposition: "passed", Evidence: "unit:u1"},
		})
}

func assertBlocker(t *testing.T, blockers []string, want string) {
	t.Helper()
	if !strings.Contains(strings.Join(blockers, "\n"), want) {
		t.Fatalf("missing blocker %q in %+v", want, blockers)
	}
}

func TestABMStructuralCausalityRequiresCoverageOfEveryMaterialFact(t *testing.T) {
	store := causalityFixture(t)
	blockers := causalityBlockers(t, store, []ReviewCriterionMapping{{Delta: "SD-aaaa1111", Basis: "D1"}}, "")
	assertBlocker(t, blockers, "design-causality passes without answering for SD-bbbb2222 (public-contract changed api/openapi.yaml)")
	full := causalityBlockers(t, store, []ReviewCriterionMapping{
		{Delta: "SD-aaaa1111", Basis: "D1"},
		{Delta: "SD-bbbb2222", Basis: "R1"},
	}, "")
	if len(full) != 0 {
		t.Fatalf("a fully mapped judgment was blocked: %+v", full)
	}
}

func TestABMStructuralCausalityRefusesInapplicabilityAndPhantomFacts(t *testing.T) {
	store := causalityFixture(t)
	assertBlocker(t, causalityBlockers(t, store, nil, "not-applicable"),
		"is not-applicable while the bundle seals 2 material structural facts (dependency, public-contract)")
	assertBlocker(t, causalityBlockers(t, store, []ReviewCriterionMapping{
		{Delta: "SD-aaaa1111", Basis: "D1"}, {Delta: "SD-bbbb2222", Basis: "R1"},
		{Delta: "SD-cccc3333", Basis: "R1"},
	}, ""), `maps "SD-cccc3333", which this subject does not observe as material`)
	assertBlocker(t, causalityBlockers(t, store, []ReviewCriterionMapping{
		{Delta: "SD-aaaa1111", Basis: "D1"}, {Delta: "SD-aaaa1111", Basis: "R1"},
		{Delta: "SD-bbbb2222", Basis: "R1"},
	}, ""), "maps SD-aaaa1111 twice")
	// A criterion that did not pass owes no mapping: the problem is on record.
	if blockers := causalityBlockers(t, store, nil, "finding"); len(blockers) != 0 {
		t.Fatalf("a criterion disposed as a finding owed a mapping: %+v", blockers)
	}
}

// The engine checks the namespace, the existence and the reach to a requirement.
// It never checks whether the reviewer's causal claim is true.
func TestABMStructuralCausalityBasisMustReachARequirement(t *testing.T) {
	store := causalityFixture(t)
	for _, tc := range []struct{ basis, want string }{
		{"D9", "to D9, which reaches no requirement or constraint"},
		{"A9", "to A9, which reaches no requirement or constraint"},
		{"R7", "to R7, which this scope's decision basis does not declare"},
		{"because it is needed", "which is not a decision-basis ref"},
		{"", "as mapped and names no basis"},
	} {
		t.Run(tc.basis, func(t *testing.T) {
			mapping := ReviewCriterionMapping{Delta: "SD-aaaa1111", Basis: tc.basis}
			if tc.basis == "" {
				mapping.Disposition = ReviewMappingMapped
			}
			assertBlocker(t, causalityBlockers(t, store, []ReviewCriterionMapping{
				mapping, {Delta: "SD-bbbb2222", Basis: "C1"},
			}, ""), tc.want)
		})
	}
	// A1 reaches R1, and C1 is a constraint: both are accepted.
	for _, basis := range []string{"A1", "C1", "D1", "R1"} {
		if blockers := causalityBlockers(t, store, []ReviewCriterionMapping{
			{Delta: "SD-aaaa1111", Basis: basis}, {Delta: "SD-bbbb2222", Basis: "R1"},
		}, ""); len(blockers) != 0 {
			t.Fatalf("basis %s was refused: %+v", basis, blockers)
		}
	}
}

// Missing evidence, inapplicability and accepted risk are three statements, not
// one blank. Each is available, each needs its own reason, and none may pretend
// to be a mapping at the same time.
func TestABMStructuralCausalityDistinguishesTheThreeNonMappings(t *testing.T) {
	store := causalityFixture(t)
	for _, state := range []string{ReviewMappingMissingEvidence, ReviewMappingNotApplicable, ReviewMappingAcceptedRisk} {
		t.Run(state, func(t *testing.T) {
			if blockers := causalityBlockers(t, store, []ReviewCriterionMapping{
				{Delta: "SD-aaaa1111", Disposition: state, Rationale: "stated reason"},
				{Delta: "SD-bbbb2222", Basis: "R1"},
			}, ""); len(blockers) != 0 {
				t.Fatalf("%s with a rationale was refused: %+v", state, blockers)
			}
			assertBlocker(t, causalityBlockers(t, store, []ReviewCriterionMapping{
				{Delta: "SD-aaaa1111", Disposition: state},
				{Delta: "SD-bbbb2222", Basis: "R1"},
			}, ""), "disposes SD-aaaa1111 as "+state+" with no rationale")
			assertBlocker(t, causalityBlockers(t, store, []ReviewCriterionMapping{
				{Delta: "SD-aaaa1111", Disposition: state, Rationale: "stated reason", Basis: "R1"},
				{Delta: "SD-bbbb2222", Basis: "R1"},
			}, ""), "and still names basis R1; state one or the other")
		})
	}
	assertBlocker(t, causalityBlockers(t, store, []ReviewCriterionMapping{
		{Delta: "SD-aaaa1111", Disposition: "handled"},
		{Delta: "SD-bbbb2222", Basis: "R1"},
	}, ""), `as "handled", which is not mapped, missing-evidence, not-applicable or accepted-risk`)
}

// Opt-in in both directions: no material fact, or no criterion that answers for
// structure, and the gate asks for nothing.
func TestABMStructuralCausalityIsOptInBothWays(t *testing.T) {
	store := causalityFixture(t)
	_, required := causalityInputs()
	dispositions := []ReviewCriterion{{ID: "design-causality", Disposition: "passed", Rationale: "examined"}}
	if blockers := store.reviewStructuralCausalityBlockers("spec:backend", nil, required, dispositions); len(blockers) != 0 {
		t.Fatalf("no observation still owed a mapping: %+v", blockers)
	}
	empty := &ReviewPlanStructure{Observed: true, Status: "observed"}
	if blockers := store.reviewStructuralCausalityBlockers("spec:backend", empty, required, dispositions); len(blockers) != 0 {
		t.Fatalf("an empty material set owed a mapping: %+v", blockers)
	}
	structure, _ := causalityInputs()
	unflagged := map[string]ReviewPlanCriterion{"design-causality": {ID: "design-causality", Required: true, Kind: ReviewCriterionKindJudgment}}
	if blockers := store.reviewStructuralCausalityBlockers("spec:backend", structure, unflagged, dispositions); len(blockers) != 0 {
		t.Fatalf("a criterion that never claimed the obligation owed a mapping: %+v", blockers)
	}
}

// A bundle sealed before the contract existed is never held to it.
func TestABMStructuralCausalityGovernsOnlySealedContracts(t *testing.T) {
	store := causalityFixture(t)
	structure, required := causalityInputs()
	planned := []ReviewPlanCriterion{}
	for _, criterion := range required {
		planned = append(planned, criterion)
	}
	bundle := ReviewBundle{Payload: ReviewBundlePayload{
		Scope: ReviewBundleScope{Ref: "spec:backend"},
		Plan:  ReviewBundlePlan{Criteria: planned, Structure: structure},
	}}
	att := ReviewAttestation{Criteria: []ReviewCriterion{{ID: "design-causality", Disposition: "passed", Rationale: "examined"}}}
	if governed, _ := BundleGovernedBy(bundle, "structural-causality"); governed {
		t.Fatal("a bundle with no governing contracts was reported governed")
	}
	bundle.Payload.GoverningContracts = governingContractsAtSeal()
	governed, _ := BundleGovernedBy(bundle, "structural-causality")
	if !governed {
		t.Fatal("a bundle sealed by this engine does not list the contract")
	}
	if blockers := store.reviewStructuralCausalityBlockers(bundle.Payload.Scope.Ref, bundle.Payload.Plan.Structure, required, att.Criteria); len(blockers) == 0 {
		t.Fatal("an unmapped judgment passed under the contract")
	}
}
