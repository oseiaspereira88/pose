package pose

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func componentReviewFixture(t *testing.T) (string, Store) {
	t.Helper()
	root := t.TempDir()
	writeReviewFixture(t, root, ".pose/policy/review.json", `{
  "schema_version": 2,
  "enabled": true,
  "adopted_at": "2026-08-02",
  "profiles": {"spec":"spec-closeout@2"},
  "reviewer_independence": {"spec":"same-actor-separate-execution"},
  "component_aware": true,
  "component_aware_adopted_at": "2026-08-13",
  "unmapped_component_behavior": "warning",
  "overlay_profiles": ["frontend-review@1", "backend-review@1"]
}`)
	writeReviewFixture(t, root, ".pose/review-profiles/spec-closeout.json", `{
  "schema_version":2,"id":"spec-closeout","version":2,"scope":"spec",
  "criteria":[{"id":"correctness","description":"Scope behavior is correct.","evidence_classes":["test"]}],
  "tools":[{"id":"review-check","requiredness":"required","criteria":["correctness"]}]
}`)
	writeReviewFixture(t, root, ".pose/review-profiles/frontend-review.json", `{
  "schema_version":2,"id":"frontend-review","version":1,"scope":"spec",
  "selectors":{"languages":["typescript"]},
  "criteria":[
    {"id":"frontend-accessibility","description":"The interface is accessible.","evidence_classes":["a11y"]},
    {"id":"frontend-failure-state","description":"Network and state failures remain usable.","evidence_classes":["test"]}
  ],
  "tools":[{"id":"surface-check","requiredness":"required","evidence_classes":["reachability"],"criteria":["frontend-accessibility"]}]
}`)
	writeReviewFixture(t, root, ".pose/review-profiles/backend-review.json", `{
  "schema_version":2,"id":"backend-review","version":1,"scope":"spec",
  "selectors":{"languages":["go"]},
  "criteria":[
    {"id":"backend-contracts","description":"Contracts and errors are compatible.","evidence_classes":["integration"]},
    {"id":"backend-observability","description":"Failures are diagnosable.","evidence_classes":["test"]}
  ],
  "tools":[{"id":"assess-integrate","requiredness":"recommended","criteria":["backend-contracts"]}]
}`)
	writeReviewFixture(t, root, ".pose/indexes/repo-map.json", `{
  "apps":[{"name":"web","path":"web","language":"typescript","owner":"@ui","domain":"frontend","criticality":"medium","metadataStatus":{"source":"declared"}}],
  "services":[{"name":"api","path":"api","language":"go","owner":"@api","domain":"backend","criticality":"high","metadataStatus":{"source":"declared"}}],
  "packages":[]
}`)
	writeReviewFixture(t, root, ".pose/indexes/delivery-integrity.json", `{"reverse":{}}`)
	writeReviewFixture(t, root, ".pose/specs/frontend/spec.md", `---
slug: frontend
status: in-progress
created_at: 2026-08-13
components: web
delivers: surface:dashboard
---

# Spec: frontend

### Artifacts
- modified: web/app.tsx

### Delivery targets
- surface:dashboard module:web profile:web-ui entrypoint:web/app.tsx
`)
	writeReviewFixture(t, root, ".pose/specs/backend/spec.md", `---
slug: backend
status: in-progress
created_at: 2026-08-13
components: api
---

# Spec: backend

### Artifacts
- modified: api/server.go
`)
	writeReviewFixture(t, root, ".pose/specs/fullstack/spec.md", `---
slug: fullstack
status: in-progress
created_at: 2026-08-13
components: web, api
---

# Spec: fullstack

### Artifacts
- modified: web/client.ts
- modified: api/server.go
`)
	return root, Store{Root: root}
}

func TestReviewPlanSelectsDistinctFrontendBackendAndBoundaryCoverage(t *testing.T) {
	_, store := componentReviewFixture(t)
	frontend, err := store.ReviewPlan("spec:frontend")
	if err != nil {
		t.Fatal(err)
	}
	if !planHasCriterion(frontend, "frontend-accessibility") || planHasCriterion(frontend, "backend-contracts") || !planHasTool(frontend, "surface-check") {
		t.Fatalf("unexpected frontend plan: %+v", frontend)
	}
	backend, err := store.ReviewPlan("spec:backend")
	if err != nil {
		t.Fatal(err)
	}
	if !planHasCriterion(backend, "backend-contracts") || planHasCriterion(backend, "frontend-accessibility") {
		t.Fatalf("unexpected backend plan: %+v", backend)
	}
	fullstack, err := store.ReviewPlan("spec:fullstack")
	if err != nil {
		t.Fatal(err)
	}
	if !planHasCriterion(fullstack, "frontend-accessibility") || !planHasCriterion(fullstack, "backend-contracts") || !planHasCriterion(fullstack, "cross-component-integration") || !planHasTool(fullstack, "assess-integrate") {
		t.Fatalf("multi-component coverage missing: %+v", fullstack)
	}
	if len(fullstack.Components) != 2 || len(fullstack.Blockers) != 0 {
		t.Fatalf("unexpected component projection: %+v", fullstack)
	}
}

func TestReviewPlanDigestIsDeterministicAndIgnoresUnconsumedOwner(t *testing.T) {
	root, store := componentReviewFixture(t)
	first, err := store.ReviewPlan("spec:backend")
	if err != nil {
		t.Fatal(err)
	}
	second, err := store.ReviewPlan("spec:backend")
	if err != nil || first.PlanDigest != second.PlanDigest {
		t.Fatalf("plan is not deterministic: first=%s second=%s err=%v", first.PlanDigest, second.PlanDigest, err)
	}
	path := filepath.Join(root, ".pose", "indexes", "repo-map.json")
	raw, _ := os.ReadFile(path)
	if err := os.WriteFile(path, []byte(strings.Replace(string(raw), `"@api"`, `"@new-owner"`, 1)), 0o644); err != nil {
		t.Fatal(err)
	}
	ownerOnly, err := store.ReviewPlan("spec:backend")
	if err != nil || ownerOnly.PlanDigest != first.PlanDigest || ownerOnly.Components[0].Owner != "@new-owner" {
		t.Fatalf("unconsumed owner changed digest or projection was stale: %+v err=%v", ownerOnly, err)
	}
	raw, _ = os.ReadFile(path)
	if err := os.WriteFile(path, []byte(strings.Replace(string(raw), `"language":"go"`, `"language":"typescript"`, 1)), 0o644); err != nil {
		t.Fatal(err)
	}
	reselected, err := store.ReviewPlan("spec:backend")
	if err != nil || reselected.PlanDigest == first.PlanDigest || !planHasCriterion(reselected, "frontend-accessibility") {
		t.Fatalf("consumed language did not reselect plan: %+v err=%v", reselected, err)
	}
}

func TestReviewPlanRejectsConflictingCriteriaUnknownToolsAndStrictUnmapped(t *testing.T) {
	root, store := componentReviewFixture(t)
	writeReviewFixture(t, root, ".pose/review-profiles/conflict.json", `{
  "schema_version":2,"id":"conflict","version":1,"scope":"spec",
  "selectors":{"languages":["go"]},
  "criteria":[{"id":"correctness","description":"Different contract."}],
  "tools":[{"id":"shell-command"}]
}`)
	writeReviewFixture(t, root, ".pose/review-profiles/command.json", `{
  "schema_version":2,"id":"command","version":1,"scope":"spec",
  "selectors":{"languages":["go"]},
  "criteria":[{"id":"command-safety","description":"No command surface."}],
  "tools":[{"id":"validate","command":"sh -c arbitrary"}]
}`)
	policyPath := filepath.Join(root, ".pose", "policy", "review.json")
	raw, _ := os.ReadFile(policyPath)
	policy := strings.Replace(string(raw), `"backend-review@1"]`, `"backend-review@1", "command@1", "conflict@1"]`, 1)
	if err := os.WriteFile(policyPath, []byte(policy), 0o644); err != nil {
		t.Fatal(err)
	}
	plan, err := store.ReviewPlan("spec:backend")
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(plan.Blockers, " ")
	if !strings.Contains(joined, "conflicting review criterion correctness") || !strings.Contains(joined, "unknown review tool shell-command") || !strings.Contains(joined, "unknown field \"command\"") {
		t.Fatalf("unsafe/conflicting overlay did not fail visible: %+v", plan)
	}

	writeReviewFixture(t, root, ".pose/specs/unmapped/spec.md", "---\nslug: unmapped\nstatus: in-progress\ncreated_at: 2026-08-13\ncomponents: unknown\n---\n\n# Spec\n")
	raw, _ = os.ReadFile(policyPath)
	var parsed map[string]any
	if err := json.Unmarshal(raw, &parsed); err != nil {
		t.Fatal(err)
	}
	parsed["unmapped_component_behavior"] = "blocker"
	updated, _ := json.Marshal(parsed)
	if err := os.WriteFile(policyPath, updated, 0o644); err != nil {
		t.Fatal(err)
	}
	unmapped, err := store.ReviewPlan("spec:unmapped")
	if err != nil || !strings.Contains(strings.Join(unmapped.Blockers, " "), "unmapped review component explicit:unknown") {
		t.Fatalf("strict unmapped policy did not block: %+v err=%v", unmapped, err)
	}
}

func TestReviewPlanRecommendationsAreClosedAndNonExecutable(t *testing.T) {
	_, store := componentReviewFixture(t)
	plan, err := store.ReviewPlan("spec:fullstack")
	if err != nil {
		t.Fatal(err)
	}
	for _, tool := range plan.Tools {
		if _, ok := reviewToolCatalog[tool.ID]; !ok || len(tool.Args) == 0 || tool.Args[0] != "pose" {
			t.Fatalf("recommendation escaped native catalog: %+v", tool)
		}
		for _, arg := range tool.Args {
			if strings.ContainsAny(arg, "\n\r;|&<>") {
				t.Fatalf("unsafe recommendation argument: %+v", tool)
			}
		}
	}
}

func TestReviewPlanPreservesStricterIndependenceAndRejectsAmbiguousSelectors(t *testing.T) {
	root, store := componentReviewFixture(t)
	policyPath := filepath.Join(root, ".pose", "policy", "review.json")
	raw, _ := os.ReadFile(policyPath)
	if err := os.WriteFile(policyPath, []byte(strings.Replace(string(raw), "same-actor-separate-execution", "mandatory-human", 1)), 0o644); err != nil {
		t.Fatal(err)
	}
	backendPath := filepath.Join(root, ".pose", "review-profiles", "backend-review.json")
	raw, _ = os.ReadFile(backendPath)
	if err := os.WriteFile(backendPath, []byte(strings.Replace(string(raw), `"criteria":`, `"independence":"same-actor-separate-execution","criteria":`, 1)), 0o644); err != nil {
		t.Fatal(err)
	}
	plan, err := store.ReviewPlan("spec:backend")
	if err != nil || plan.Independence != "mandatory-human" {
		t.Fatalf("overlay weakened reviewer independence: plan=%+v err=%v", plan, err)
	}

	mapPath := filepath.Join(root, ".pose", "indexes", "repo-map.json")
	raw, _ = os.ReadFile(mapPath)
	ambiguous := strings.Replace(string(raw), `"services":[`, `"services":[{"name":"api","path":"api-copy","language":"go","metadataStatus":{"source":"declared"}},`, 1)
	if err := os.WriteFile(mapPath, []byte(ambiguous), 0o644); err != nil {
		t.Fatal(err)
	}
	plan, err = store.ReviewPlan("spec:backend")
	if err != nil || !strings.Contains(strings.Join(plan.Blockers, " "), "ambiguous review component api") {
		t.Fatalf("ambiguous component selector did not block: plan=%+v err=%v", plan, err)
	}
}

func TestReviewPlanSchemaV1RemainsGenericAndResolutionIsReadOnly(t *testing.T) {
	root, store := componentReviewFixture(t)
	writeReviewFixture(t, root, ".pose/policy/review.json", `{"schema_version":1,"enabled":true,"adopted_at":"2026-08-02","profiles":{"spec":"legacy@1"}}`)
	writeReviewFixture(t, root, ".pose/review-profiles/legacy.json", `{"schema_version":1,"id":"legacy","version":1,"scope":"spec","criteria":[{"id":"legacy","description":"Generic legacy review."}]}`)
	before := reviewFixtureSnapshot(t, root)
	plan, err := store.ReviewPlan("spec:frontend")
	after := reviewFixtureSnapshot(t, root)
	if err != nil || plan.PolicySchemaVersion != 1 || len(plan.Components) != 0 || len(plan.SelectedProfiles) != 1 || !planHasCriterion(plan, "legacy") {
		t.Fatalf("schema-v1 plan changed generic behavior: plan=%+v err=%v", plan, err)
	}
	if strings.Join(before, "\n") != strings.Join(after, "\n") {
		t.Fatalf("review plan resolution mutated the repository: before=%v after=%v", before, after)
	}
}

func reviewFixtureSnapshot(t *testing.T, root string) []string {
	t.Helper()
	items := []string{}
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, path)
		items = append(items, rel+":"+info.Mode().String()+":"+fmt.Sprint(info.Size()))
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(items)
	return items
}

func TestReviewPlanRejectsComponentAndArtifactSymlinkEscapes(t *testing.T) {
	root, store := componentReviewFixture(t)
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(root, "escape")); err != nil {
		t.Fatal(err)
	}
	writeReviewFixture(t, root, ".pose/indexes/repo-map.json", `{
  "apps":[{"name":"escaped","path":"escape","language":"typescript","metadataStatus":{"source":"declared"}}],
  "services":[],"packages":[]
}`)
	writeReviewFixture(t, root, ".pose/specs/escaped/spec.md", "---\nslug: escaped\nstatus: in-progress\ncreated_at: 2026-08-13\ncomponents: escaped\n---\n\n# Spec\n")
	plan, err := store.ReviewPlan("spec:escaped")
	if err != nil || !strings.Contains(strings.Join(plan.Blockers, " "), "symlink escapes project root") {
		t.Fatalf("component symlink escape was accepted: plan=%+v err=%v", plan, err)
	}

	_, store = componentReviewFixture(t)
	root = store.Root
	if err := os.Symlink(outside, filepath.Join(root, "web", "escape")); err != nil {
		if os.IsNotExist(err) {
			if mkErr := os.MkdirAll(filepath.Join(root, "web"), 0o755); mkErr != nil {
				t.Fatal(mkErr)
			}
			if retryErr := os.Symlink(outside, filepath.Join(root, "web", "escape")); retryErr != nil {
				t.Fatal(retryErr)
			}
		} else {
			t.Fatal(err)
		}
	}
	writeReviewFixture(t, root, ".pose/specs/frontend/spec.md", `---
slug: frontend
status: in-progress
created_at: 2026-08-13
components: web
---

# Spec

### Artifacts
- modified: web/escape/file.ts
`)
	plan, err = store.ReviewPlan("spec:frontend")
	if err != nil || !strings.Contains(strings.Join(plan.Blockers, " "), "symlink escapes project root") {
		t.Fatalf("artifact symlink escape was accepted: plan=%+v err=%v", plan, err)
	}
}

func TestReviewCheckRequiresCurrentEffectivePlanDigestAndCoverage(t *testing.T) {
	root, store := componentReviewFixture(t)
	plan, err := store.ReviewPlan("spec:backend")
	if err != nil {
		t.Fatal(err)
	}
	var criteria strings.Builder
	for _, criterion := range plan.Criteria {
		if criterion.Required {
			class := "test"
			if len(criterion.EvidenceClasses) > 0 {
				class = criterion.EvidenceClasses[0]
			}
			criteria.WriteString("- " + criterion.ID + " [passed] evidence:" + class + ":review-plan\n")
		}
	}
	body := "---\nschema_version: 1\nreview_id: rvw-plan\nscope: spec:backend\nscope_digest: " + plan.ScopeDigest + "\nprofile: spec-closeout@2\nreviewer: agent:separate-review\ndecision: approved\nreviewed_at: 2026-08-13T12:00:00Z\nevidence_refs: [test:review-plan, integration:review-plan]\n---\n\n## Criteria\n" + criteria.String() + "\n## Findings\n"
	writeReviewFixture(t, root, ".pose/reviews/rvw-plan.md", body)
	stale, err := store.ReviewCheck("spec:backend")
	if err != nil || stale.Approved || stale.Fresh || !strings.Contains(strings.Join(stale.Blockers, " "), "effective plan digest is missing") {
		t.Fatalf("schema-v2 review without plan digest was accepted: %+v err=%v", stale, err)
	}
	body = strings.Replace(body, "profile: spec-closeout@2", "plan_digest: "+plan.PlanDigest+"\nprofile: spec-closeout@2", 1)
	if err := os.WriteFile(filepath.Join(root, ".pose", "reviews", "rvw-plan.md"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	current, err := store.ReviewCheck("spec:backend")
	if err != nil || !current.Approved || !current.Fresh || current.PlanDigest != plan.PlanDigest {
		t.Fatalf("current plan-bound review was rejected: %+v err=%v", current, err)
	}
}

func TestReviewCheckGrandfathersOnlyCompletedPreMigrationAttempts(t *testing.T) {
	root, store := componentReviewFixture(t)
	writeReviewFixture(t, root, ".pose/specs/backend/spec.md", `---
slug: backend
status: done
created_at: 2026-08-03
completed_at: 2026-08-12
components: api
---

# Spec: backend
`)
	writeReviewFixture(t, root, ".pose/reviews/rvw-legacy-component-plan.md", `---
schema_version: 1
review_id: rvw-legacy-component-plan
scope: spec:backend
scope_digest: sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
profile: spec-closeout@2
reviewer: agent:legacy-review
decision: approved
reviewed_at: 2026-08-12T12:00:00Z
evidence_refs: [test:legacy]
---

## Criteria
- correctness [passed] evidence:test:legacy

## Findings
`)
	eval, err := store.ReviewCheck("spec:backend")
	if err != nil || !eval.Approved || !eval.Fresh || !strings.Contains(strings.Join(eval.Warnings, " "), "pre-component-aware") {
		t.Fatalf("completed pre-migration review was invalidated: eval=%+v err=%v", eval, err)
	}

	path := filepath.Join(root, ".pose", "specs", "backend", "spec.md")
	raw, _ := os.ReadFile(path)
	if err := os.WriteFile(path, []byte(strings.Replace(string(raw), "status: done", "status: in-progress", 1)), 0o644); err != nil {
		t.Fatal(err)
	}
	eval, err = store.ReviewCheck("spec:backend")
	if err != nil || eval.Approved || !strings.Contains(strings.Join(eval.Blockers, " "), "effective plan digest is missing") {
		t.Fatalf("reopened scope retained migration exemption: eval=%+v err=%v", eval, err)
	}
}

func planHasCriterion(plan ReviewPlan, id string) bool {
	for _, criterion := range plan.Criteria {
		if criterion.ID == id {
			return true
		}
	}
	return false
}

func planHasTool(plan ReviewPlan, id string) bool {
	for _, tool := range plan.Tools {
		if tool.ID == id {
			return true
		}
	}
	return false
}
