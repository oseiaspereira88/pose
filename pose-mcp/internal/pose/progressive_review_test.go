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

func planBandsFor(plan ReviewPlan, band string) []ReviewPlanBand {
	found := []ReviewPlanBand{}
	for _, item := range plan.Bands {
		if item.Band == band {
			found = append(found, item)
		}
	}
	return found
}

func planBandFor(t *testing.T, plan ReviewPlan, policy string) ReviewPlanBand {
	t.Helper()
	for _, item := range plan.Bands {
		if item.Policy == policy {
			return item
		}
	}
	t.Fatalf("no band explained by %s: %+v", policy, plan.Bands)
	return ReviewPlanBand{}
}

func TestABMProgressiveReviewBandsExplainTheEffectivePlan(t *testing.T) {
	store := progressiveReviewFixture(t, "high", "contract", "api/server.go")
	adoptProgressiveReview(t, store, "")
	plan := progressivePlan(t, store)
	if plan.Band != ReviewBandCritical {
		t.Fatalf("band=%q bands=%+v", plan.Band, plan.Bands)
	}
	base := planBandFor(t, plan, "spec-closeout@2")
	if base.Band != ReviewBandBaseline || base.Basis != reviewBasisPolicy || base.Source != ".pose/policy/review.json" ||
		!containsFold(base.Obligations, "independence:same-actor-separate-execution") || !containsFold(base.Obligations, "criterion:correctness") {
		t.Fatalf("baseline band lost trigger, source, policy or obligation: %+v", base)
	}
	judgment := planBandFor(t, plan, "engineering-judgment@1")
	if judgment.Band != ReviewBandElevated || judgment.Trigger != "delivery_kind=contract" || judgment.Basis != reviewBasisDeclared ||
		!strings.Contains(judgment.Source, "delivery-target:backend") || len(judgment.Obligations) != 3 {
		t.Fatalf("elevated band is not explainable: %+v", judgment)
	}
	for _, obligation := range judgment.Obligations {
		if strings.HasPrefix(obligation, "independence:") {
			t.Fatalf("elevated band claimed an escalation it did not cause: %+v", judgment)
		}
	}
	criticality := planBandFor(t, plan, "high-criticality-review@1")
	if criticality.Band != ReviewBandCritical || !strings.Contains(criticality.Trigger, "criticality=high") ||
		!strings.Contains(criticality.Trigger, "delivery_kind=contract") || !strings.Contains(criticality.Source, "component:api") ||
		!containsFold(criticality.Obligations, "independence:different-actor") {
		t.Fatalf("critical band did not name the escalating fact: %+v", criticality)
	}
	if len(planBandsFor(plan, ReviewBandUnknown)) != 0 {
		t.Fatalf("declared metadata reported as undecided: %+v", plan.Bands)
	}
}

// The summary is a projection of digested fields, so it must not change plan
// identity. A band that moved the digest would be indistinguishable, to a
// verifier, from a real change in obligations.
func TestABMProgressiveReviewBandsDoNotChangePlanIdentity(t *testing.T) {
	store := progressiveReviewFixture(t, "critical", "governance", "api/AGENTS.md")
	adoptProgressiveReview(t, store, "")
	plan := progressivePlan(t, store)
	if len(plan.Bands) == 0 || plan.Projection.Basis == "" {
		t.Fatalf("fixture produced no summary to compare: %+v", plan)
	}
	stripped := plan
	stripped.Band, stripped.Bands, stripped.Projection = "", nil, ReviewPlanProjection{}
	stripped.Components = append([]ReviewPlanComponent{}, plan.Components...)
	for i := range stripped.Components {
		stripped.Components[i].Origin = ""
	}
	withSummary, err := digestReviewPlan(plan)
	if err != nil {
		t.Fatal(err)
	}
	withoutSummary, err := digestReviewPlan(stripped)
	if err != nil {
		t.Fatal(err)
	}
	if withSummary != withoutSummary || withSummary != plan.PlanDigest {
		t.Fatalf("summary entered plan identity: %s vs %s (plan %s)", withSummary, withoutSummary, plan.PlanDigest)
	}
}

func TestABMProgressiveReviewUnknownSelectorIsVisibleWithoutEscalating(t *testing.T) {
	for _, metadata := range []string{
		`{"services":[{"name":"api","path":"api","language":"go","criticality":"high","metadataStatus":{"source":"defaulted","isComplete":false,"missingFields":["criticality"]}}]}`,
		`{"services":[{"name":"api","path":"api","language":"go","criticality":"","metadataStatus":{"source":"declared"}}]}`,
	} {
		store := progressiveReviewFixture(t, "high", "governance", "api/AGENTS.md")
		adoptProgressiveReview(t, store, "")
		writeReviewFixture(t, store.Root, ".pose/indexes/repo-map.json", metadata)
		plan := progressivePlan(t, store)
		unknown := planBandsFor(plan, ReviewBandUnknown)
		if len(unknown) != 1 || unknown[0].Policy != "high-criticality-review@1" ||
			unknown[0].Trigger != "criticality=unknown" || len(unknown[0].Obligations) != 0 {
			t.Fatalf("undecided selector not reported once and unpriced: %+v", plan.Bands)
		}
		if plan.Band != ReviewBandElevated {
			t.Fatalf("unknown criticality moved the band to %q: %+v", plan.Band, plan.Bands)
		}
		if plan.Independence != "same-actor-separate-execution" {
			t.Fatalf("unknown criticality changed the floor: %+v", plan)
		}
	}
}

// R5: a preflight forecast and a final observation are different claims. The
// plan must say which obligations exist only because provenance attributed more
// scope than the author declared.
func TestABMProgressiveReviewObservedScopeExpansionIsLabelled(t *testing.T) {
	store := progressiveReviewFixture(t, "medium", "contract", "api/server.go")
	adoptProgressiveReview(t, store, "")
	writeReviewFixture(t, store.Root, ".pose/indexes/repo-map.json",
		`{"services":[{"name":"api","path":"api","language":"go","criticality":"medium","metadataStatus":{"source":"declared"}},`+
			`{"name":"web","path":"web","language":"typescript","criticality":"critical","metadataStatus":{"source":"declared"}}]}`)
	forecast := progressivePlan(t, store)
	if forecast.Band != ReviewBandElevated || forecast.Projection.ScopeExpanded || forecast.Projection.RaisedBand != "" {
		t.Fatalf("declared-only scope already reported an expansion: %+v", forecast.Projection)
	}
	for _, component := range forecast.Components {
		if component.Origin != reviewBasisDeclared {
			t.Fatalf("declared component labelled %q: %+v", component.Origin, component)
		}
	}
	writeReviewFixture(t, store.Root, ".pose/indexes/delivery-integrity.json",
		`{"reverse":{"web/app.ts":["backend"]}}`)
	observed := progressivePlan(t, store)
	projection := observed.Projection
	if projection.Basis != ReviewProjectionBasis || !projection.ScopeExpanded {
		t.Fatalf("observed expansion not labelled as such: %+v", projection)
	}
	if !reflect.DeepEqual(projection.ObservedComponents, []string{"web"}) {
		t.Fatalf("observed component not attributed: %+v", projection)
	}
	if projection.Band != forecast.Band || projection.Independence != forecast.Independence {
		t.Fatalf("forecast was recomputed from observed scope: %+v", projection)
	}
	if observed.Band != ReviewBandCritical || projection.RaisedBand != ReviewBandElevated+" -> "+ReviewBandCritical {
		t.Fatalf("expansion did not surface the raised band: band=%q %+v", observed.Band, projection)
	}
	if projection.RaisedIndependence != "same-actor-separate-execution -> different-actor" {
		t.Fatalf("expansion did not surface the raised floor: %+v", projection)
	}
	if !containsFold(projection.AddedProfiles, "high-criticality-review@1") ||
		!containsFold(projection.AddedCriteria, "cross-component-integration") ||
		!containsFold(projection.AddedProfiles, "frontend-review@1") {
		t.Fatalf("expansion did not attribute the added obligations: %+v", projection)
	}
	if len(projection.AddedTools) == 0 {
		t.Fatalf("expansion added components without any scoped tool: %+v", projection)
	}
	for _, component := range observed.Components {
		want := reviewBasisDeclared
		if component.Path == "web" {
			want = reviewBasisObserved
		}
		if component.Origin != want {
			t.Fatalf("component %s origin=%q want=%q", component.Path, component.Origin, want)
		}
	}
}

// Criticality alone escalates the band. Without this case the rule would be
// satisfied by the independence raise that the shipped high profile happens to
// carry, and an overlay that escalates criticality without touching the floor
// would be summarized as merely elevated.
func TestABMProgressiveReviewCriticalityEscalatesBandWithoutRaisingTheFloor(t *testing.T) {
	store := progressiveReviewFixture(t, "critical", "contract", "api/server.go")
	writeReviewFixture(t, store.Root, ".pose/review-profiles/criticality-only.json", `{
  "schema_version":2,"id":"criticality-only","version":1,"scope":"spec",
  "selectors":{"criticalities":["high","critical"]},
  "criteria":[{"id":"criticality-only-judgment","kind":"judgment","description":"State the blast radius the criticality implies."}]
}`)
	policy, _, err := store.loadReviewPolicy()
	if err != nil {
		t.Fatal(err)
	}
	policy.OverlayProfiles = append(policy.OverlayProfiles, "criticality-only@1")
	raw, err := json.Marshal(policy)
	if err != nil {
		t.Fatal(err)
	}
	writeReviewFixture(t, store.Root, ".pose/policy/review.json", string(raw))
	plan := progressivePlan(t, store)
	band := planBandFor(t, plan, "criticality-only@1")
	if band.Band != ReviewBandCritical || band.Trigger != "criticality=critical" {
		t.Fatalf("criticality did not escalate the band on its own: %+v", band)
	}
	for _, obligation := range band.Obligations {
		if strings.HasPrefix(obligation, "independence:") {
			t.Fatalf("band claimed a floor raise the overlay did not make: %+v", band)
		}
	}
	if plan.Independence != "same-actor-separate-execution" || plan.Band != ReviewBandCritical {
		t.Fatalf("escalating the band changed the floor: independence=%q band=%q", plan.Independence, plan.Band)
	}
}
