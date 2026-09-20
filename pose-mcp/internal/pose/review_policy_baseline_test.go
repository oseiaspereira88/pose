package pose

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The strong contract: judgment required, a different reviewer, and an overlay
// that contributes a required judgment for Go components.
const protectedBaselinePolicy = `{
  "schema_version": 2,
  "enabled": true,
  "adopted_at": "2026-09-01",
  "profiles": {"spec":"spec-closeout@2"},
  "reviewer_independence": {"spec":"different-actor"},
  "component_aware": true,
  "component_aware_adopted_at": "2026-09-01",
  "unmapped_component_behavior": "warning",
  "overlay_profiles": ["backend-review@1"],
  "require_signed_attestations": true
}`

const protectedBaselineBase = `{
  "schema_version":2,"id":"spec-closeout","version":2,"scope":"spec",
  "criteria":[
    {"id":"correctness","description":"Scope behavior is correct.","evidence_classes":["unit"]},
    {"id":"contract-judgment","kind":"judgment","description":"State the contract change and who it binds."}
  ],
  "tools":[{"id":"review-check","requiredness":"required","criteria":["correctness"]}]
}`

const protectedBaselineOverlay = `{
  "schema_version":2,"id":"backend-review","version":1,"scope":"spec",
  "selectors":{"languages":["go"]},
  "criteria":[{"id":"backend-boundary","kind":"judgment","description":"State the boundary the change moves."}]
}`

// protectedBaselineFixture commits the strong contract, then leaves the working
// tree free for the test to weaken. The change set attributes the contract
// change to the spec under review, which is what arms the protection.
func protectedBaselineFixture(t *testing.T) (string, Store) {
	t.Helper()
	root := t.TempDir()
	designDeltaGit(t, root, "init", "-q")
	designDeltaGit(t, root, "config", "user.email", "pose@example.test")
	designDeltaGit(t, root, "config", "user.name", "POSE Test")
	writeReviewFixture(t, root, ".pose/policy/review.json", protectedBaselinePolicy)
	writeReviewFixture(t, root, ".pose/review-profiles/spec-closeout.json", protectedBaselineBase)
	writeReviewFixture(t, root, ".pose/review-profiles/backend-review.json", protectedBaselineOverlay)
	writeReviewFixture(t, root, ".pose/indexes/repo-map.json", `{"services":[{"name":"api","path":"api","language":"go","criticality":"high","metadataStatus":{"source":"declared"}}]}`)
	writeReviewFixture(t, root, "api/server.go", "package api\n")
	writeReviewFixture(t, root, ".pose/specs/governance/spec.md",
		"---\nslug: governance\nstatus: in-progress\ncreated_at: 2026-09-20\ncomponents: api\n---\n# Spec\n\n### Artifacts\n- modified: .pose/policy/review.json\n")
	designDeltaGit(t, root, "add", "--", ".")
	designDeltaGit(t, root, "commit", "-q", "-m", "protected contract")
	base := strings.TrimSpace(string(mustDesignDeltaGitOutput(t, root, "rev-parse", "HEAD")))
	graph := DeliveryIntegrityGraph{
		SchemaVersion: DeliveryIntegritySchemaVersion,
		ChangeSets: []ChangeSet{{
			ID: "cs-governance", Spec: "governance", ResolvedBase: base, ResolvedHead: base,
			Paths: []ObservedPath{{Action: "modified", Path: ".pose/policy/review.json"}},
		}},
	}
	raw, err := json.Marshal(graph)
	if err != nil {
		t.Fatal(err)
	}
	writeReviewFixture(t, root, ".pose/indexes/delivery-integrity.json", string(raw))
	return root, Store{Root: root}
}

func weakenProtectedBaseline(t *testing.T, root string, mutate func(policy map[string]any)) {
	t.Helper()
	var policy map[string]any
	if err := json.Unmarshal([]byte(protectedBaselinePolicy), &policy); err != nil {
		t.Fatal(err)
	}
	mutate(policy)
	raw, err := json.Marshal(policy)
	if err != nil {
		t.Fatal(err)
	}
	writeReviewFixture(t, root, ".pose/policy/review.json", string(raw))
}

func TestABMProtectedBaselineRestoresWhatTheDiffWeakened(t *testing.T) {
	root, store := protectedBaselineFixture(t)
	weakenProtectedBaseline(t, root, func(policy map[string]any) {
		policy["reviewer_independence"] = map[string]any{"spec": "same-actor-separate-execution"}
		policy["overlay_profiles"] = []string{}
		policy["require_signed_attestations"] = false
		policy["allow_approved_with_reservations"] = true
	})
	// The diff also softens a judged criterion into an optional collected one.
	writeReviewFixture(t, root, ".pose/review-profiles/spec-closeout.json", `{
  "schema_version":2,"id":"spec-closeout","version":2,"scope":"spec",
  "criteria":[
    {"id":"correctness","description":"Scope behavior is correct.","evidence_classes":["unit"]},
    {"id":"contract-judgment","kind":"mechanical","required":false,"description":"State the contract change and who it binds.","evidence_classes":["unit"]}
  ],
  "tools":[{"id":"review-check","requiredness":"required","criteria":["correctness"]}]
}`)
	plan, err := store.ReviewPlan("spec:governance")
	if err != nil {
		t.Fatalf("plan err=%v", err)
	}
	if !plan.PolicyBaseline.Protected || plan.PolicyBaseline.Revision == "" ||
		!containsFold(plan.PolicyBaseline.Paths, ".pose/policy/review.json") {
		t.Fatalf("contract change was not protected: %+v", plan.PolicyBaseline)
	}
	if plan.Independence != "different-actor" {
		t.Fatalf("diff lowered the floor that governs its own review: %q", plan.Independence)
	}
	for _, want := range []string{
		"independence:different-actor",
		"criterion:backend-boundary",
		"criterion:contract-judgment:required",
		"criterion:contract-judgment:judgment",
	} {
		if !containsFold(plan.PolicyBaseline.Restored, want) {
			t.Fatalf("missing restoration %q: %+v", want, plan.PolicyBaseline.Restored)
		}
	}
	for _, want := range []string{
		"allow_approved_with_reservations:true",
		"require_signed_attestations:false",
		"removed_criterion:backend-boundary",
		"reviewer_independence:same-actor-separate-execution",
	} {
		if !containsFold(plan.PolicyBaseline.Weakened, want) {
			t.Fatalf("unreported weakening %q: %+v", want, plan.PolicyBaseline.Weakened)
		}
	}
	restored := false
	for _, criterion := range plan.Criteria {
		switch criterion.ID {
		case "backend-boundary":
			restored = true
			if !criterion.Required || ReviewCriterionKind(criterion) != ReviewCriterionKindJudgment ||
				!strings.HasPrefix(criterion.Profiles[0], "protected-baseline:") {
				t.Fatalf("restored criterion lost its obligation or provenance: %+v", criterion)
			}
		case "contract-judgment":
			if !criterion.Required || ReviewCriterionKind(criterion) != ReviewCriterionKindJudgment {
				t.Fatalf("softened criterion was not restored: %+v", criterion)
			}
		}
	}
	if !restored {
		t.Fatalf("dropped criterion never came back: %+v", plan.Criteria)
	}
	// The restoration changes what the review owes, so it must change identity.
	if !strings.Contains(strings.Join(plan.Explain, "\n"), "protected policy baseline restored independence:different-actor") {
		t.Fatalf("restoration absent from the digested explain trail: %+v", plan.Explain)
	}
	if plan.Band != ReviewBandCritical {
		t.Fatalf("a self-weakening contract change is not critical: band=%q %+v", plan.Band, plan.Bands)
	}
}

// A diff that disables review must not become unreviewable. Before this, the
// plan refused to exist and the change landed with no plan at all.
func TestABMProtectedBaselineGovernsADisabledPolicy(t *testing.T) {
	root, store := protectedBaselineFixture(t)
	weakenProtectedBaseline(t, root, func(policy map[string]any) { policy["enabled"] = false })
	plan, err := store.ReviewPlan("spec:governance")
	if err != nil {
		t.Fatalf("disabling review made the contract change unreviewable: %v", err)
	}
	if !plan.PolicyBaseline.Protected || plan.Independence != "different-actor" ||
		!planHasCriterion(plan, "backend-boundary") || !planHasCriterion(plan, "contract-judgment") {
		t.Fatalf("baseline did not govern the disabled policy: %+v", plan.PolicyBaseline)
	}
}

// Deleting the profile the policy points at must not erase the obligation either.
func TestABMProtectedBaselineSurvivesADeletedProfile(t *testing.T) {
	root, store := protectedBaselineFixture(t)
	weakenProtectedBaseline(t, root, func(policy map[string]any) { policy["enabled"] = false })
	if err := os.Remove(filepath.Join(root, ".pose/review-profiles/spec-closeout.json")); err != nil {
		t.Fatal(err)
	}
	plan, err := store.ReviewPlan("spec:governance")
	if err != nil {
		t.Fatalf("deleting the profile made the contract change unreviewable: %v", err)
	}
	if !planHasCriterion(plan, "contract-judgment") || plan.Independence != "different-actor" {
		t.Fatalf("baseline profile was not used: %+v", plan)
	}
}

// Protection is claimed only when it exists. An unreachable base contract is
// reported, because an unprotected governance review is the failure to expose.
func TestABMProtectedBaselineUnavailableIsStatedNotAssumed(t *testing.T) {
	root, store := protectedBaselineFixture(t)
	writeReviewFixture(t, root, ".pose/indexes/delivery-integrity.json",
		`{"schema_version":1,"change_sets":[{"id":"cs-governance","spec":"governance","paths":[{"action":"modified","path":".pose/policy/review.json"}]}]}`)
	weakenProtectedBaseline(t, root, func(policy map[string]any) {
		policy["reviewer_independence"] = map[string]any{"spec": "same-actor-separate-execution"}
	})
	plan, err := store.ReviewPlan("spec:governance")
	if err != nil {
		t.Fatal(err)
	}
	if plan.PolicyBaseline.Protected || plan.PolicyBaseline.Reason == "" {
		t.Fatalf("protection claimed without a base revision: %+v", plan.PolicyBaseline)
	}
	if !strings.Contains(strings.Join(plan.Warnings, " "), "unprotected review contract change") {
		t.Fatalf("unprotected contract change was silent: %+v", plan.Warnings)
	}
	unknown := planBandsFor(plan, ReviewBandUnknown)
	if len(unknown) != 1 || unknown[0].Policy != "protected-baseline" {
		t.Fatalf("missing undecided band for the unprotected change: %+v", plan.Bands)
	}
}

// A scope that changes no contract path is untouched: no protection, no band,
// no warning, and the plan it always had.
func TestABMProtectedBaselineIgnoresOrdinaryScopes(t *testing.T) {
	root, store := protectedBaselineFixture(t)
	writeReviewFixture(t, root, ".pose/specs/governance/spec.md",
		"---\nslug: governance\nstatus: in-progress\ncreated_at: 2026-09-20\ncomponents: api\n---\n# Spec\n\n### Artifacts\n- modified: api/server.go\n")
	writeReviewFixture(t, root, ".pose/indexes/delivery-integrity.json",
		`{"schema_version":1,"change_sets":[{"id":"cs-governance","spec":"governance","resolved_base":"HEAD","paths":[{"action":"modified","path":"api/server.go"}]}]}`)
	plan, err := store.ReviewPlan("spec:governance")
	if err != nil {
		t.Fatal(err)
	}
	if plan.PolicyBaseline.Protected || len(plan.PolicyBaseline.Paths) != 0 || len(plan.PolicyBaseline.Restored) != 0 {
		t.Fatalf("ordinary scope was protected: %+v", plan.PolicyBaseline)
	}
	if strings.Contains(strings.Join(plan.Warnings, " "), "unprotected review contract change") {
		t.Fatalf("ordinary scope warned about the contract: %+v", plan.Warnings)
	}
	for _, band := range plan.Bands {
		if band.Policy == "protected-baseline" {
			t.Fatalf("ordinary scope gained a baseline band: %+v", band)
		}
	}
}

// The revision reaches a Git argument, and it comes from an index file in the
// repository. A symbolic or option-shaped value must be refused before the call,
// not after Git decides what to do with it.
func TestABMProtectedBaselineRefusesANonImmutableRevision(t *testing.T) {
	for _, revision := range []string{"HEAD", "--help", "main", "", "refs/heads/main", "-n"} {
		root, store := protectedBaselineFixture(t)
		raw, err := os.ReadFile(filepath.Join(root, ".pose/indexes/delivery-integrity.json"))
		if err != nil {
			t.Fatal(err)
		}
		var graph DeliveryIntegrityGraph
		if err := json.Unmarshal(raw, &graph); err != nil {
			t.Fatal(err)
		}
		graph.ChangeSets[0].ResolvedBase = revision
		patched, err := json.Marshal(graph)
		if err != nil {
			t.Fatal(err)
		}
		writeReviewFixture(t, root, ".pose/indexes/delivery-integrity.json", string(patched))
		weakenProtectedBaseline(t, root, func(policy map[string]any) {
			policy["reviewer_independence"] = map[string]any{"spec": "same-actor-separate-execution"}
		})
		plan, err := store.ReviewPlan("spec:governance")
		if err != nil {
			t.Fatalf("revision %q made the scope unplannable: %v", revision, err)
		}
		if plan.PolicyBaseline.Protected {
			t.Fatalf("revision %q was accepted as a protected baseline: %+v", revision, plan.PolicyBaseline)
		}
		if plan.Independence != "same-actor-separate-execution" {
			t.Fatalf("revision %q restored an obligation from an unread baseline", revision)
		}
		if !strings.Contains(strings.Join(plan.Warnings, " "), "unprotected review contract change") {
			t.Fatalf("revision %q was refused silently: %+v", revision, plan.Warnings)
		}
	}
}
