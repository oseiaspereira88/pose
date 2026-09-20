package pose

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/harne8/pose-mcp/internal/scaffold"
)

// Consume the shipped assets so the corpus also catches distribution drift.
func progressiveReviewFixture(t *testing.T, criticality, kind, entrypoint string) Store {
	t.Helper()
	root, store := componentReviewFixture(t)
	for _, name := range []string{"engineering-judgment", "high-criticality-review"} {
		path := ".pose/review-profiles/" + name + ".json"
		raw, err := fs.ReadFile(scaffold.Dist(), path)
		if err != nil {
			t.Fatal(err)
		}
		writeReviewFixture(t, root, path, string(raw))
	}
	writeReviewFixture(t, root, ".pose/indexes/repo-map.json", fmt.Sprintf(`{"services":[{"name":"api","path":"api","language":"go","criticality":%q,"metadataStatus":{"source":"declared"}}]}`, criticality))
	body := "---\nslug: backend\nstatus: in-progress\ncreated_at: 2026-09-19\ncomponents: api\n"
	if kind != "" {
		body += "delivers: " + kind + ":example\n"
	}
	body += "---\n# Spec\n\n### Artifacts\n- modified: " + entrypoint + "\n"
	if kind != "" {
		body += "\n### Delivery targets\n- " + kind + ":example module:api profile:example entrypoint:" + entrypoint + "\n"
	}
	writeReviewFixture(t, root, ".pose/specs/backend/spec.md", body)
	return store
}

func adoptProgressiveReview(t *testing.T, store Store, floor string) {
	t.Helper()
	policy, _, err := store.loadReviewPolicy()
	if err != nil {
		t.Fatal(err)
	}
	policy.OverlayProfiles = append(policy.OverlayProfiles, "engineering-judgment@1", "high-criticality-review@1")
	if floor != "" {
		policy.ReviewerIndependence["spec"] = floor
	}
	raw, err := json.Marshal(policy)
	if err != nil {
		t.Fatal(err)
	}
	writeReviewFixture(t, store.Root, ".pose/policy/review.json", string(raw))
}

func progressivePlan(t *testing.T, store Store) ReviewPlan {
	t.Helper()
	plan, err := store.ReviewPlan("spec:backend")
	if err != nil || len(plan.Blockers) != 0 {
		t.Fatalf("plan=%+v err=%v", plan, err)
	}
	return plan
}

func TestABMProgressiveReviewOptInAndDeduplicatedJudgment(t *testing.T) {
	store := progressiveReviewFixture(t, "high", "contract", "api/server.go")
	before := progressivePlan(t, store)
	if planHasCriterion(before, "assumption-integrity") || before.Independence != "same-actor-separate-execution" {
		t.Fatalf("installing profiles activated them: %+v", before)
	}
	adoptProgressiveReview(t, store, "")
	snapshot := reviewFixtureSnapshot(t, store.Root)
	after := progressivePlan(t, store)
	if after.Independence != "different-actor" || after.PlanDigest == before.PlanDigest {
		t.Fatalf("adoption did not raise the effective obligation: %+v", after)
	}
	if !reflect.DeepEqual(before.Tools, after.Tools) {
		t.Fatalf("judgment overlays added tools: before=%+v after=%+v", before.Tools, after.Tools)
	}
	for _, id := range []string{"assumption-integrity", "design-causality", "solution-proportionality"} {
		count := 0
		for _, criterion := range after.Criteria {
			if criterion.ID != id {
				continue
			}
			count++
			if !criterion.Required || ReviewCriterionKind(criterion) != ReviewCriterionKindJudgment || len(criterion.EvidenceClasses) != 0 || !reflect.DeepEqual(criterion.Profiles, []string{"engineering-judgment@1", "high-criticality-review@1"}) {
				t.Fatalf("lost judgment or profile provenance: %+v", criterion)
			}
		}
		if count != 1 {
			t.Fatalf("criterion %s occurs %d times", id, count)
		}
	}
	if !reflect.DeepEqual(snapshot, reviewFixtureSnapshot(t, store.Root)) || !reflect.DeepEqual(after, progressivePlan(t, store)) {
		t.Fatal("planning mutated the repository or was nondeterministic")
	}
}

func TestABMProgressiveReviewTrivialCorpusHasNoAdditionalObligations(t *testing.T) {
	for _, criticality := range []string{"low", "medium", "high", "critical"} {
		for _, path := range []string{"api/README.md", "api/server.go", "api/server_test.go"} {
			t.Run(criticality+"/"+path, func(t *testing.T) {
				store := progressiveReviewFixture(t, criticality, "", path)
				before := progressivePlan(t, store)
				adoptProgressiveReview(t, store, "")
				after := progressivePlan(t, store)
				// Policy adoption changes scope identity even when no overlay
				// matches. Compare obligations, preserving that freshness rule.
				before.ScopeDigest, after.ScopeDigest = "", ""
				before.PlanDigest, after.PlanDigest = "", ""
				if !reflect.DeepEqual(before, after) {
					t.Fatal("target-free scope gained ABM obligations")
				}
			})
		}
	}
}

func TestABMProgressiveReviewAuthorityMarkdownRemainsMaterial(t *testing.T) {
	store := progressiveReviewFixture(t, "critical", "governance", "api/AGENTS.md")
	adoptProgressiveReview(t, store, "")
	plan := progressivePlan(t, store)
	if !planHasCriterion(plan, "design-causality") || plan.Independence != "different-actor" {
		t.Fatalf("Markdown extension suppressed authority review: %+v", plan)
	}
}

func TestABMCriticalityExplicitHighAndCritical(t *testing.T) {
	for _, level := range []string{"low", "medium", "high", "critical", "higher", "unknown", ""} {
		t.Run(level, func(t *testing.T) {
			store := progressiveReviewFixture(t, level, "capability", "api/server.go")
			adoptProgressiveReview(t, store, "")
			plan := progressivePlan(t, store)
			want := "same-actor-separate-execution"
			if level == "high" || level == "critical" {
				want = "different-actor"
			}
			if plan.Independence != want || !planHasCriterion(plan, "assumption-integrity") {
				t.Fatalf("criticality %q: %+v", level, plan)
			}
		})
	}
}

func TestABMCriticalityIncompleteMetadataIsVisible(t *testing.T) {
	store := progressiveReviewFixture(t, "high", "governance", "api/AGENTS.md")
	adoptProgressiveReview(t, store, "mandatory-human")
	writeReviewFixture(t, store.Root, ".pose/indexes/repo-map.json", `{"services":[{"name":"api","path":"api","criticality":"high","metadataStatus":{"source":"defaulted","isComplete":false,"missingFields":["criticality"]}}]}`)
	plan := progressivePlan(t, store)
	if plan.Components[0].Criticality != "" || !strings.Contains(strings.Join(plan.Warnings, " "), "metadata-incomplete") || plan.Independence != "mandatory-human" || !planHasCriterion(plan, "assumption-integrity") {
		t.Fatalf("unknown metadata silently weakened the floor: %+v", plan)
	}
}

func TestABMPolicyDowngradeOverlayCannotLowerHumanFloor(t *testing.T) {
	store := progressiveReviewFixture(t, "high", "contract", "api/server.go")
	adoptProgressiveReview(t, store, "mandatory-human")
	if plan := progressivePlan(t, store); plan.Independence != "mandatory-human" {
		t.Fatalf("different-actor overlay lowered mandatory-human: %+v", plan)
	}
	// An author's metadata downgrade must not remove the policy's own floor.
	path := filepath.Join(store.Root, ".pose/indexes/repo-map.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	writeReviewFixture(t, store.Root, ".pose/indexes/repo-map.json", strings.ReplaceAll(string(raw), `"high"`, `"low"`))
	if plan := progressivePlan(t, store); plan.Independence != "mandatory-human" {
		t.Fatalf("metadata lowered mandatory-human: %+v", plan)
	}
}
