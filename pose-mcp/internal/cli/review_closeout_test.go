package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	posemodel "github.com/harne8/pose-mcp/internal/pose"
)

func writeCloseoutCLIFile(t *testing.T, root, rel, body string) {
	t.Helper()
	path := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func closeoutCLIFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	writeCloseoutCLIFile(t, root, ".pose/policy/review.json", `{"schema_version":1,"enabled":true,"adopted_at":"2026-08-02","profiles":{"spec":"spec-closeout@1"},"continuous_closeout":true}`)
	writeCloseoutCLIFile(t, root, ".pose/review-profiles/spec-closeout.json", `{"schema_version":1,"id":"spec-closeout","version":1,"scope":"spec","criteria":[{"id":"correctness","description":"reviewed"}]}`)
	writeCloseoutCLIFile(t, root, ".pose/specs/alpha/spec.md", "---\nslug: alpha\nstatus: in-progress\ncreated_at: 2026-08-02\ncompleted_at:\n---\n\n# Spec: alpha\n\n## 2. Requirements\n- R1: works\n")
	return root
}

func TestReviewRecordIsDryRunByDefaultAndCloseIsReviewGated(t *testing.T) {
	root := closeoutCLIFixture(t)
	var out, errOut bytes.Buffer
	args := []string{"spec:alpha", "--reviewer", "agent:review-pass", "--decision", "approved", "--evidence", "check:unit"}
	if code := cmdReviewRecord(root, args, &out, &errOut); code != 0 {
		t.Fatalf("dry run code=%d err=%s", code, errOut.String())
	}
	if entries, _ := os.ReadDir(filepath.Join(root, ".pose", "reviews")); len(entries) != 0 {
		t.Fatalf("dry run wrote reviews: %v", entries)
	}
	out.Reset()
	args = append(args, "--apply")
	if code := cmdReviewRecord(root, args, &out, &errOut); code != 0 {
		t.Fatalf("apply code=%d err=%s", code, errOut.String())
	}
	state, err := (posemodel.Store{Root: root}).GetCloseoutState("spec:alpha")
	if err != nil || !state.Review.Approved || state.Terminal {
		t.Fatalf("unexpected pre-close state=%+v err=%v", state, err)
	}
	out.Reset()
	if code := cmdClose(root, []string{"spec:alpha"}, &out, &errOut); code != 0 {
		t.Fatalf("close code=%d err=%s", code, errOut.String())
	}
	state, err = (posemodel.Store{Root: root}).GetCloseoutState("spec:alpha")
	if err != nil || !state.Terminal {
		t.Fatalf("unexpected terminal state=%+v err=%v", state, err)
	}
}

func TestCloseoutCheckJSONUsesTheSameProjection(t *testing.T) {
	root := closeoutCLIFixture(t)
	var out, errOut bytes.Buffer
	if code := cmdCloseoutCheck(root, []string{"spec:alpha", "--json"}, &out, &errOut); code != 1 {
		t.Fatalf("expected blocked code, got %d", code)
	}
	var state posemodel.CloseoutState
	if err := json.Unmarshal(out.Bytes(), &state); err != nil {
		t.Fatal(err)
	}
	if state.Scope != "spec:alpha" || state.Terminal || !strings.Contains(state.NextAction, "review") {
		t.Fatalf("unexpected JSON projection: %+v", state)
	}
}

func TestContinuousCloseoutPersistsTerminalScopeAndRefusesEarlyCompletion(t *testing.T) {
	root := closeoutCLIFixture(t)
	var out, errOut bytes.Buffer
	if code := cmdContinuousCloseout(root, []string{"start", "spec:alpha", "--apply"}, &out, &errOut); code != 0 {
		t.Fatalf("start code=%d err=%s", code, errOut.String())
	}
	out.Reset()
	if code := cmdContinuousCloseout(root, []string{"status"}, &out, &errOut); code != 1 || !strings.Contains(out.String(), "continuous.next_action=record or remediate") {
		t.Fatalf("status code/output mismatch: code=%d out=%s err=%s", code, out.String(), errOut.String())
	}
	if code := cmdContinuousCloseout(root, []string{"complete", "--apply"}, &out, &errOut); code != 1 {
		t.Fatalf("early complete code=%d", code)
	}
	if _, err := os.Stat(filepath.Join(root, ".pose", "continuous-closeout.json")); err != nil {
		t.Fatalf("early completion removed selection: %v", err)
	}
}

func TestReviewPlanCLIProjectsJSONAndPinsReviewRecord(t *testing.T) {
	root := closeoutCLIFixture(t)
	var out, errOut bytes.Buffer
	if code := cmdReviewPlan(root, []string{"spec:alpha", "--json", "--explain"}, &out, &errOut); code != 0 {
		t.Fatalf("review-plan code=%d err=%s", code, errOut.String())
	}
	var plan posemodel.ReviewPlan
	if err := json.Unmarshal(out.Bytes(), &plan); err != nil {
		t.Fatal(err)
	}
	if plan.Scope != "spec:alpha" || plan.PlanDigest == "" || len(plan.Criteria) != 1 || !strings.Contains(strings.Join(plan.Tools[0].Args, " "), "pose") {
		t.Fatalf("unexpected review plan: %+v", plan)
	}
	out.Reset()
	args := []string{"spec:alpha", "--reviewer", "agent:review-pass", "--decision", "approved", "--evidence", "check:unit", "--plan-digest", "sha256:stale"}
	if code := cmdReviewRecord(root, args, &out, &errOut); code != 1 || !strings.Contains(errOut.String(), "current is "+plan.PlanDigest) {
		t.Fatalf("stale plan pin was accepted: code=%d out=%s err=%s", code, out.String(), errOut.String())
	}
	errOut.Reset()
	args[len(args)-1] = plan.PlanDigest
	if code := cmdReviewRecord(root, args, &out, &errOut); code != 0 || !strings.Contains(out.String(), "review.plan_digest="+plan.PlanDigest) {
		t.Fatalf("current plan pin failed: code=%d out=%s err=%s", code, out.String(), errOut.String())
	}
}

func TestReviewRecordRequiresRequiredToolDispositions(t *testing.T) {
	root := t.TempDir()
	writeCloseoutCLIFile(t, root, ".pose/policy/review.json", `{"schema_version":2,"enabled":true,"adopted_at":"2026-08-02","profiles":{"spec":"spec-closeout@2"},"reviewer_independence":{"spec":"same-actor-separate-execution"},"component_aware":true,"component_aware_adopted_at":"2026-08-13","unmapped_component_behavior":"warning"}`)
	writeCloseoutCLIFile(t, root, ".pose/review-profiles/spec-closeout.json", `{"schema_version":2,"id":"spec-closeout","version":2,"scope":"spec","criteria":[{"id":"correctness","description":"reviewed","evidence_classes":["test"]}]}`)
	writeCloseoutCLIFile(t, root, ".pose/indexes/repo-map.json", `{"apps":[],"services":[],"packages":[{"name":"module","path":"module space","language":"go","owner":"@team","domain":"backend","criticality":"high","validationProfile":"baseline","metadataStatus":{"source":"declared","isComplete":true,"missingFields":[]}}]}`)
	writeCloseoutCLIFile(t, root, ".pose/indexes/delivery-integrity.json", `{"reverse":{}}`)
	writeCloseoutCLIFile(t, root, ".pose/specs/alpha/spec.md", "---\nslug: alpha\nstatus: in-progress\ncreated_at: 2026-08-13\ncomponents: module\n---\n\n# Spec: alpha\n\n### Artifacts\n- modified: module space/main.go\n")

	baseArgs := []string{"spec:alpha", "--reviewer", "agent:review-pass", "--decision", "approved", "--evidence", "test:unit"}
	var out, errOut bytes.Buffer
	if code := cmdReviewRecord(root, baseArgs, &out, &errOut); code != 2 || !strings.Contains(errOut.String(), "required review tool") {
		t.Fatalf("missing required tools were accepted: code=%d out=%s err=%s", code, out.String(), errOut.String())
	}
	errOut.Reset()
	out.Reset()
	failedArgs := append([]string{}, baseArgs...)
	failedArgs[4] = "changes-requested"
	failedArgs = append(failedArgs,
		"--tool", "artifact-check|-|failed|check:artifact-failure|",
		"--tool", "validate|module+space|passed|validation:module|",
	)
	if code := cmdReviewRecord(root, failedArgs, &out, &errOut); code != 0 {
		t.Fatalf("failed required tool could not be recorded for remediation: code=%d out=%s err=%s", code, out.String(), errOut.String())
	}
	errOut.Reset()
	out.Reset()
	args := append([]string{}, baseArgs...)
	args = append(args,
		"--tool", "artifact-check|-|passed|check:artifact|",
		"--tool", "validate|module+space|passed|validation:module|",
		"--apply",
	)
	if code := cmdReviewRecord(root, args, &out, &errOut); code != 0 {
		t.Fatalf("explicit required tools were rejected: code=%d out=%s err=%s", code, out.String(), errOut.String())
	}
	eval, err := (posemodel.Store{Root: root}).ReviewCheck("spec:alpha")
	if err != nil || !eval.Approved || eval.Current == nil || len(eval.Current.Tools) != len(mustReviewPlan(t, root).Tools) {
		t.Fatalf("recorded tool coverage did not approve: eval=%+v err=%v", eval, err)
	}
}

func reviewBundleCLIFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	writeCloseoutCLIFile(t, root, ".pose/policy/review.json", `{
  "schema_version":2,"enabled":true,"adopted_at":"2026-08-13",
  "profiles":{"spec":"spec-closeout@2"},
  "reviewer_independence":{"spec":"same-actor-separate-execution"},
  "component_aware":true,"component_aware_adopted_at":"2026-08-13",
  "unmapped_component_behavior":"warning",
  "review_bundles":true,"review_bundles_adopted_at":"2026-08-13"
}`)
	writeCloseoutCLIFile(t, root, ".pose/review-profiles/spec-closeout.json", `{
  "schema_version":2,"id":"spec-closeout","version":2,"scope":"spec",
  "criteria":[{"id":"correctness","description":"reviewed","evidence_classes":["test"]}]
}`)
	writeCloseoutCLIFile(t, root, ".pose/indexes/repo-map.json", `{
  "apps":[],"services":[],
  "packages":[{"name":"pose-mcp","path":"pose-mcp","language":"go","owner":"@team","domain":"backend","criticality":"high","validationProfile":"baseline","metadataStatus":{"source":"declared","isComplete":true,"missingFields":[]}}]
}`)
	writeCloseoutCLIFile(t, root, "pose-mcp/main.go", "package main\n")
	writeCloseoutCLIFile(t, root, ".pose/specs/bundle/spec.md", `---
slug: bundle
status: in-progress
created_at: 2026-08-13
components: pose-mcp
delivers: contract:bundle
---

# Spec: bundle

## 1. Intent
Ship review bundles.

## 2. Requirements
- R1: The bundle shall converge.

## 3. Technical Plan
### Artifacts
- modified: pose-mcp/main.go
### Delivery targets
- contract:bundle module:pose-mcp profile:api-contract entrypoint:pose-mcp/main.go
### Risks
Keep compatibility.

## 4. Tasks
- [ ] Implement.

## 5. Decisions
Use immutable JSON.

## 6. Validation
Pending.

## 7. Final Report
Pending.
`)
	writeCloseoutCLIFile(t, root, ".pose/indexes/delivery-integrity.json", `{
  "schema_version":1,"provenance_digest":"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
  "nodes":[],"edges":[],"claims":[],"reverse":{"pose-mcp/main.go":["bundle"]},"findings":[],
  "change_sets":[{"id":"cs-bundle","spec":"bundle","selector":"range:a..b","base":"a","head":"b","resolved_base":"a","resolved_head":"b","paths":[{"action":"modified","path":"pose-mcp/main.go"}],"diff_digest":"sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"}],
  "deliveries":[{"spec":"bundle","ref":"contract:bundle","kind":"contract","id":"bundle","module":"pose-mcp","profile":"api-contract","entrypoint":"pose-mcp/main.go"}],
  "validation_results":[{"id":"bundle-test","module":"pose-mcp","check":"test","evidence_class":"test","severity":"required","outcome":"pass","provenance_digest":"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}]
}`)
	return root
}

func TestReviewBundleCLISealsAttestsAndVerifies(t *testing.T) {
	root := reviewBundleCLIFixture(t)
	var out, errOut bytes.Buffer
	if code := cmdReview(root, []string{"bundle", "spec:bundle", "--json"}, &out, &errOut); code != 0 {
		t.Fatalf("prepare code=%d err=%s", code, errOut.String())
	}
	var prepared posemodel.ReviewBundle
	if err := json.Unmarshal(out.Bytes(), &prepared); err != nil {
		t.Fatal(err)
	}
	if prepared.State != "prepared" || prepared.BundleID == "" || len(prepared.Blockers) != 0 {
		t.Fatalf("unexpected prepared bundle: %+v", prepared)
	}
	out.Reset()
	errOut.Reset()
	if code := cmdReview(root, []string{"bundle", "spec:bundle", "--explain"}, &out, &errOut); code != 0 {
		t.Fatalf("explain code=%d err=%s", code, errOut.String())
	}
	if text := out.String(); !strings.Contains(text, "include.subject=") || !strings.Contains(text, " class:implementation digest:sha256:") || !strings.Contains(text, " reason:attributed implementation path") {
		t.Fatalf("subject classification missing from explain output:\n%s", text)
	}
	out.Reset()
	errOut.Reset()
	if code := cmdReview(root, []string{"bundle", "spec:bundle", "--seal", "--json"}, &out, &errOut); code != 0 {
		t.Fatalf("seal code=%d err=%s", code, errOut.String())
	}
	var sealed posemodel.ReviewBundle
	if err := json.Unmarshal(out.Bytes(), &sealed); err != nil {
		t.Fatal(err)
	}
	if sealed.State != "sealed" || sealed.Path == "" {
		t.Fatalf("bundle was not sealed: %+v", sealed)
	}
	out.Reset()
	errOut.Reset()
	attest := []string{"attest", sealed.BundleID, "--reviewer", "agent:bundle-review", "--decision", "approved", "--evidence", "test:bundle", "--tool", "artifact-check|-|passed|check:artifact|", "--tool", "validate|pose-mcp|passed|validation:module|", "--apply"}
	if code := cmdReview(root, attest, &out, &errOut); code != 0 {
		t.Fatalf("attest code=%d out=%s err=%s", code, out.String(), errOut.String())
	}
	out.Reset()
	errOut.Reset()
	if code := cmdReview(root, []string{"verify", "spec:bundle", "--json"}, &out, &errOut); code != 0 {
		t.Fatalf("verify code=%d out=%s err=%s", code, out.String(), errOut.String())
	}
	var verification posemodel.ReviewBundleVerification
	if err := json.Unmarshal(out.Bytes(), &verification); err != nil {
		t.Fatal(err)
	}
	if !verification.Approved || !verification.Fresh || verification.State != "ready-to-close" {
		t.Fatalf("unexpected verification: %+v", verification)
	}
}

func TestReviewRecordDelegatesToBundleAttestationWhenAdopted(t *testing.T) {
	root := reviewBundleCLIFixture(t)
	var out, errOut bytes.Buffer
	if code := cmdReview(root, []string{"bundle", "spec:bundle", "--seal"}, &out, &errOut); code != 0 {
		t.Fatalf("seal code=%d err=%s", code, errOut.String())
	}
	out.Reset()
	errOut.Reset()
	args := []string{"record", "spec:bundle", "--reviewer", "agent:bundle-review", "--decision", "approved", "--evidence", "test:bundle", "--tool", "artifact-check|-|passed|check:artifact|", "--tool", "validate|pose-mcp|passed|validation:module|", "--apply"}
	if code := cmdReview(root, args, &out, &errOut); code != 0 {
		t.Fatalf("record adapter code=%d out=%s err=%s", code, out.String(), errOut.String())
	}
	entries, err := os.ReadDir(filepath.Join(root, ".pose", "review-attestations"))
	if err != nil || len(entries) != 1 {
		t.Fatalf("record did not write one bundle attestation: entries=%v err=%v", entries, err)
	}
}

func TestReviewPlanGroupsRepeatedWarningsAndPresentsActionableToolPhases(t *testing.T) {
	warnings := groupedReviewPlanWarnings([]string{
		"unmapped review component path:a", "unmapped review component path:b",
		"unmapped review component path:c", "unmapped review component path:d", "another warning",
	})
	joined := strings.Join(warnings, "\n")
	if !strings.Contains(joined, "unmapped-review-component count=4") || strings.Contains(joined, "path:d") || !strings.Contains(joined, "another warning") {
		t.Fatalf("warnings were not grouped: %v", warnings)
	}
	tools := []posemodel.ReviewPlanTool{
		{ID: "validate", Requiredness: "required", Args: []string{"pose", "validate"}},
		{ID: "validate-duplicate", Requiredness: "required", Args: []string{"pose", "validate"}},
		{ID: "suggest", Requiredness: "recommended", Args: []string{"pose", "suggest"}},
		{ID: "review-check", Requiredness: "required", Args: []string{"pose", "review-check"}, Preconditions: []string{"review-complete"}},
	}
	actionable := actionableReviewPlanTools(tools)
	if len(actionable) != 3 || actionable[0].Phase != "required" || actionable[1].Phase != "recommended" || actionable[2].Phase != "completion-deferred" {
		t.Fatalf("unexpected actionable phases: %+v", actionable)
	}
}

func mustReviewPlan(t *testing.T, root string) posemodel.ReviewPlan {
	t.Helper()
	plan, err := (posemodel.Store{Root: root}).ReviewPlan("spec:alpha")
	if err != nil {
		t.Fatal(err)
	}
	return plan
}

func TestReviewAutoAttestCLI(t *testing.T) {
	root := t.TempDir()
	writeCloseoutCLIFile(t, root, ".pose/policy/review.json", `{"schema_version":2,"enabled":true,"adopted_at":"2026-08-02","profiles":{"spec":"spec-closeout@1"},"review_bundles":true,"review_bundles_adopted_at":"2026-08-14"}`)
	writeCloseoutCLIFile(t, root, ".pose/review-profiles/spec-closeout.json", `{"schema_version":1,"id":"spec-closeout","version":1,"scope":"spec","criteria":[{"id":"correctness","description":"reviewed"}]}`)
	writeCloseoutCLIFile(t, root, ".pose/specs/alpha/spec.md", "---\nslug: alpha\nstatus: in-progress\ncreated_at: 2026-08-02\ncompleted_at:\n---\n\n# Spec: alpha\n\n## 2. Requirements\n- R1: works\n")
	writeCloseoutCLIFile(t, root, "pose-mcp/lib.go", "package posemcp\n")
	graph := posemodel.DeliveryIntegrityGraph{
		SchemaVersion:    1,
		ProvenanceDigest: "sha256:1111111111111111111111111111111111111111111111111111111111111111",
		ChangeSets: []posemodel.ChangeSet{{
			ID: "cs-alpha", Spec: "alpha", Selector: "range:base..head",
			Base: "base", Head: "head", ResolvedBase: "base-resolved", ResolvedHead: "head-resolved",
			Paths:      []posemodel.ObservedPath{{Action: "modified", Path: "pose-mcp/lib.go"}},
			DiffDigest: "sha256:2222222222222222222222222222222222222222222222222222222222222222",
		}},
		Deliveries:        []posemodel.DeliveryTarget{{Spec: "alpha", Ref: "contract:alpha-api", Kind: "contract", ID: "alpha-api", Module: "pose-mcp", Profile: "api-contract", Entrypoint: "pose-mcp/lib.go"}},
		ValidationResults: []posemodel.DeliveryValidationResult{{ID: "val-alpha", Module: "pose-mcp", Check: "go-test", EvidenceClass: "integration", Severity: "required", Outcome: "pass", GitHead: "head-resolved", ProvenanceDigest: "sha256:1111111111111111111111111111111111111111111111111111111111111111"}},
		Reverse:           map[string][]string{"pose-mcp/lib.go": {"alpha"}},
	}
	rawGraph, _ := json.Marshal(graph)
	writeCloseoutCLIFile(t, root, ".pose/indexes/delivery-integrity.json", string(rawGraph))

	var out, errOut bytes.Buffer
	if code := cmdReviewBundle(root, []string{"spec:alpha", "--seal"}, &out, &errOut); code != 0 {
		t.Fatalf("bundle seal failed: code=%d out=%s err=%s", code, out.String(), errOut.String())
	}
	out.Reset()
	if code := cmdReviewAutoAttest(root, []string{"spec:alpha", "--apply"}, &out, &errOut); code != 0 {
		t.Fatalf("auto-attest failed: code=%d out=%s err=%s", code, out.String(), errOut.String())
	}
	out.Reset()
	if code := cmdReviewVerify(root, []string{"spec:alpha"}, &out, &errOut); code != 0 {
		t.Fatalf("review verify failed: code=%d out=%s err=%s", code, out.String(), errOut.String())
	}
}

func TestPoseCloseWithLiveGitTrailerNoReport(t *testing.T) {
	root := t.TempDir()
	artifactGit(t, root, "init", "-q")
	artifactGit(t, root, "config", "user.email", "pose@example.invalid")
	artifactGit(t, root, "config", "user.name", "POSE Tests")
	writeCloseoutCLIFile(t, root, ".pose/policy/review.json", `{"schema_version":1,"enabled":true,"adopted_at":"2026-08-02","profiles":{"spec":"spec-closeout@1"}}`)
	writeCloseoutCLIFile(t, root, ".pose/review-profiles/spec-closeout.json", `{"schema_version":1,"id":"spec-closeout","version":1,"scope":"spec","criteria":[{"id":"correctness","description":"reviewed"}]}`)
	writeCloseoutCLIFile(t, root, ".pose/policy/artifacts.json", `{"schema_version":1,"enabled":true,"adopted_at":"2026-08-02","governed_roots":["internal"],"severities":{"action-mismatch":"error","undeclared":"error"}}`)
	writeCloseoutCLIFile(t, root, ".pose/policy/delivery.json", `{"schema_version":1,"enabled":true,"adopted_at":"2026-08-02","results_path":".pose/results/current.json"}`)
	writeCloseoutCLIFile(t, root, ".pose/specs/alpha/spec.md", "---\nslug: alpha\nstatus: in-progress\ncreated_at: 2026-08-03\ncompleted_at:\n---\n\n# Spec: alpha\n\n## 2. Requirements\n- R1: works\n\n## 3. Technical Plan\n\n### Artifacts\n- created: internal/feature.go\n\n## 4. Tasks\nwork\n")
	writeCloseoutCLIFile(t, root, "README.md", "baseline\n")
	artifactGit(t, root, "add", "--", ".")
	artifactGit(t, root, "commit", "-q", "-m", "baseline")

	writeCloseoutCLIFile(t, root, "internal/feature.go", "package internal\n")
	artifactGit(t, root, "add", "--", "internal/feature.go")
	artifactGit(t, root, "commit", "-q", "-m", "implement feature", "-m", "POSE-Spec: alpha")

	var out, errOut bytes.Buffer
	if code := cmdReviewRecord(root, []string{"spec:alpha", "--reviewer", "agent:test", "--decision", "approved", "--evidence", "check:unit", "--apply"}, &out, &errOut); code != 0 {
		t.Fatalf("review record failed: code=%d err=%s", code, errOut.String())
	}
	out.Reset()
	errOut.Reset()
	if code := cmdClose(root, []string{"spec:alpha"}, &out, &errOut); code != 0 {
		t.Fatalf("pose close should succeed with live git trailer without prior report history: code=%d out=%s err=%s", code, out.String(), errOut.String())
	}
	state, err := (posemodel.Store{Root: root}).GetCloseoutState("spec:alpha")
	if err != nil || !state.Terminal {
		t.Fatalf("expected terminal closeout, got state=%+v err=%v", state, err)
	}
}
