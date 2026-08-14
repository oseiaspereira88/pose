package pose

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func reviewBundleFixture(t *testing.T) (string, Store) {
	t.Helper()
	root, store := componentReviewFixture(t)
	writeReviewFixture(t, root, "api/server.go", "package api\n\nfunc Ready() bool { return true }\n")
	writeReviewFixture(t, root, ".pose/specs/backend/spec.md", `---
slug: backend
status: in-progress
created_at: 2026-08-13
components: api
delivers: contract:backend-api
---

# Spec: backend

## 1. Intent

### Goal
Ship a stable backend contract.

## 2. Requirements

- R1: The backend shall remain compatible.

## 3. Technical Plan

### Artifacts
- modified: api/server.go

### Delivery targets
- contract:backend-api module:api profile:api-contract entrypoint:api/server.go

### Technical risks
No additional runtime dependency.

## 4. Tasks
- [ ] Implement incrementally.

## 5. Decisions
Use typed errors.

## 6. Validation
### Execution log
Not run yet.

## 7. Final Report
Pending.
`)
	graph := DeliveryIntegrityGraph{
		SchemaVersion:    1,
		ProvenanceDigest: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		ChangeSets: []ChangeSet{{
			ID: "cs-backend", Spec: "backend", Selector: "range:base..head",
			Base: "base", Head: "head", ResolvedBase: "base-resolved", ResolvedHead: "head-resolved",
			Paths:      []ObservedPath{{Action: "modified", Path: "api/server.go"}},
			DiffDigest: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		}},
		Deliveries:        []DeliveryTarget{{Spec: "backend", Ref: "contract:backend-api", Kind: "contract", ID: "backend-api", Module: "api", Profile: "api-contract", Entrypoint: "api/server.go"}},
		ValidationResults: []DeliveryValidationResult{{ID: "validate-backend", Module: "api", Check: "go-test", EvidenceClass: "integration", Severity: "required", Outcome: "pass", GitHead: "head-resolved", ProvenanceDigest: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}},
		Reverse:           map[string][]string{"api/server.go": {"backend"}}, Nodes: []DeliveryIntegrityNode{}, Edges: []DeliveryIntegrityEdge{}, Claims: []ArtifactClaim{}, Findings: []DeliveryIntegrityFinding{},
	}
	raw, err := json.MarshalIndent(graph, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	writeReviewFixture(t, root, ".pose/indexes/delivery-integrity.json", string(raw))
	return root, store
}

func approvedBundleAttestation(bundle ReviewBundle, reviewer string) ReviewAttestation {
	criteria := []ReviewCriterion{}
	for _, criterion := range bundle.Payload.Plan.Criteria {
		if criterion.Required {
			criteria = append(criteria, ReviewCriterion{ID: criterion.ID, Disposition: "passed", Evidence: "integration:bundle-test"})
		}
	}
	tools := []ReviewToolDisposition{}
	for _, tool := range bundle.Payload.Plan.Tools {
		disposition := ReviewToolDisposition{ID: tool.ID, Component: tool.Component}
		if containsFold(tool.Preconditions, "review-complete") {
			disposition.Disposition, disposition.Rationale = "deferred", "post-review gate"
		} else if tool.Requiredness == "recommended" {
			disposition.Disposition, disposition.Rationale = "not-used", "not needed for fixture"
		} else {
			evidenceClass := "integration"
			if len(tool.EvidenceClasses) > 0 {
				evidenceClass = tool.EvidenceClasses[0]
			}
			disposition.Disposition, disposition.Evidence = "passed", evidenceClass+":bundle-test"
		}
		tools = append(tools, disposition)
	}
	return ReviewAttestation{BundleID: bundle.BundleID, Reviewer: reviewer, Decision: "approved", Criteria: criteria, Tools: tools, EvidenceRefs: []string{"integration:bundle-test"}, Findings: []ReviewFinding{}}
}

func TestReviewBundleCanonicalAndDigestStable(t *testing.T) {
	_, store := reviewBundleFixture(t)
	first, err := store.PrepareReviewBundle("spec:backend")
	if err != nil {
		t.Fatal(err)
	}
	second, err := store.PrepareReviewBundle("spec:backend")
	if err != nil {
		t.Fatal(err)
	}
	a, _ := json.Marshal(first.Payload)
	b, _ := json.Marshal(second.Payload)
	if string(a) != string(b) || first.BundleDigest != second.BundleDigest || first.BundleID != second.BundleID {
		t.Fatalf("bundle is not stable:\n%s\n%s", a, b)
	}
	if len(first.Blockers) != 0 {
		t.Fatalf("unexpected blockers: %v", first.Blockers)
	}
}

func TestReviewBundleVerifiesSyntheticMergeByPatchAndTree(t *testing.T) {
	_, store := reviewBundleFixture(t)
	bundle, err := store.SealReviewBundle("spec:backend", time.Date(2026, 8, 13, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if bundle.Payload.Subject.Head != "head-resolved" {
		t.Fatalf("advisory head = %q, want deliberately non-fetchable fixture ref", bundle.Payload.Subject.Head)
	}
	if bundle.Payload.Subject.PatchDigest == "" || bundle.Payload.Subject.TreeDigest == "" {
		t.Fatalf("synthetic subject lacks stable identities: %+v", bundle.Payload.Subject)
	}
	loaded, err := store.LoadReviewBundle(bundle.BundleID)
	if err != nil {
		t.Fatalf("sealed synthetic subject did not verify without provider ref: %v", err)
	}
	if loaded.BundleDigest != bundle.BundleDigest || loaded.Payload.Subject.PatchDigest != bundle.Payload.Subject.PatchDigest || loaded.Payload.Subject.TreeDigest != bundle.Payload.Subject.TreeDigest {
		t.Fatalf("synthetic subject identities changed after verification: sealed=%+v loaded=%+v", bundle.Payload.Subject, loaded.Payload.Subject)
	}
}

func TestReviewBundleSemanticProjectionAndDerivedChangesDoNotStale(t *testing.T) {
	root, store := reviewBundleFixture(t)
	before, err := store.PrepareReviewBundle("spec:backend")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, ".pose/specs/backend/spec.md")
	raw, _ := os.ReadFile(path)
	derived := strings.Replace(string(raw), "Not run yet.", "Tests passed at 12:00.", 1)
	derived = strings.Replace(derived, "Pending.", "Delivered.", 1)
	derived = strings.Replace(derived, "status: in-progress", "status: done", 1)
	if err := os.WriteFile(path, []byte(derived), 0o644); err != nil {
		t.Fatal(err)
	}
	afterDerived, err := store.PrepareReviewBundle("spec:backend")
	if err != nil {
		t.Fatal(err)
	}
	if before.BundleDigest != afterDerived.BundleDigest {
		t.Fatalf("derived closeout content changed bundle: %s != %s", before.BundleDigest, afterDerived.BundleDigest)
	}
	semantic := strings.Replace(derived, "remain compatible", "return typed stable errors", 1)
	if err := os.WriteFile(path, []byte(semantic), 0o644); err != nil {
		t.Fatal(err)
	}
	afterSemantic, err := store.PrepareReviewBundle("spec:backend")
	if err != nil {
		t.Fatal(err)
	}
	if before.BundleDigest == afterSemantic.BundleDigest {
		t.Fatal("semantic requirement change did not supersede bundle")
	}
}

func TestReviewBundleDerivedOnlyChangeSetDoesNotStale(t *testing.T) {
	root, store := reviewBundleFixture(t)
	before, err := store.PrepareReviewBundle("spec:backend")
	if err != nil {
		t.Fatal(err)
	}
	graph, err := store.GetDeliveryIntegrity("")
	if err != nil {
		t.Fatal(err)
	}
	graph.ChangeSets[0].ID = "cs-backend-with-closeout"
	graph.ChangeSets[0].ResolvedHead = "head-after-closeout"
	graph.ChangeSets[0].DiffDigest = "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
	graph.ChangeSets[0].Paths = append(graph.ChangeSets[0].Paths,
		ObservedPath{Action: "created", Path: ".pose/review-attestations/rva-derived.json"},
		ObservedPath{Action: "modified", Path: ".pose/specs/backend/spec.md"},
		ObservedPath{Action: "modified", Path: ".pose/state/project-state.md"},
	)
	raw, _ := json.Marshal(graph)
	writeReviewFixture(t, root, ".pose/indexes/delivery-integrity.json", string(raw))
	after, err := store.PrepareReviewBundle("spec:backend")
	if err != nil {
		t.Fatal(err)
	}
	if before.BundleDigest != after.BundleDigest {
		t.Fatalf("derived-only change-set update changed bundle: %s != %s", before.BundleDigest, after.BundleDigest)
	}
	if len(after.ExcludedInputs) < len(before.ExcludedInputs)+2 {
		t.Fatalf("derived paths were not explained: %+v", after.ExcludedInputs)
	}
}

func TestReviewBundleSealIsAtomicIdempotentAndDetectsSubjectChange(t *testing.T) {
	root, store := reviewBundleFixture(t)
	now := time.Date(2026, 8, 13, 12, 0, 0, 0, time.UTC)
	first, err := store.SealReviewBundle("spec:backend", now)
	if err != nil {
		t.Fatal(err)
	}
	second, err := store.SealReviewBundle("spec:backend", now.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if first.BundleID != second.BundleID || first.SealedAt != second.SealedAt {
		t.Fatalf("seal replay created a different artifact: first=%+v second=%+v", first, second)
	}
	if err := os.WriteFile(filepath.Join(root, "api/server.go"), []byte("package api\n\nfunc Ready() bool { return false }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	changed, err := store.PrepareReviewBundle("spec:backend")
	if err != nil {
		t.Fatal(err)
	}
	if changed.BundleID == first.BundleID {
		t.Fatal("implementation content change did not create a new bundle identity")
	}
	verification, err := store.VerifyReviewBundle("spec:backend")
	if err != nil {
		t.Fatal(err)
	}
	if verification.State != "superseded" || verification.Delta == nil || len(verification.Delta.ChangedPaths) == 0 {
		t.Fatalf("missing supersession delta: %+v", verification)
	}
}

func TestReviewAttestationDoesNotMutateBundleAndConverges(t *testing.T) {
	root, store := reviewBundleFixture(t)
	policyPath := filepath.Join(root, ".pose/policy/review.json")
	policyRaw, _ := os.ReadFile(policyPath)
	policy := strings.Replace(string(policyRaw), `"component_aware": true,`, `"component_aware": true,
  "review_bundles": true,
  "review_bundles_adopted_at": "2026-08-13",`, 1)
	if err := os.WriteFile(policyPath, []byte(policy), 0o644); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 13, 12, 0, 0, 0, time.UTC)
	bundle, err := store.SealReviewBundle("spec:backend", now)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(store.Root, filepath.FromSlash(bundle.Path))
	before, _ := os.ReadFile(path)
	att, err := store.RecordReviewAttestation(approvedBundleAttestation(bundle, "agent:test-review"), now.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if att.BundleDigest != bundle.BundleDigest {
		t.Fatal("attestation did not bind exact bundle")
	}
	after, _ := os.ReadFile(path)
	if string(before) != string(after) {
		t.Fatal("recording attestation mutated sealed bundle bytes")
	}
	verification, err := store.VerifyReviewBundle("spec:backend")
	if err != nil {
		t.Fatal(err)
	}
	if !verification.Fresh || !verification.Approved || verification.State != "ready-to-close" || len(verification.Blockers) != 0 {
		t.Fatalf("bundle did not converge: %+v", verification)
	}
	evaluation, err := store.ReviewCheck("spec:backend")
	if err != nil || !evaluation.Fresh || !evaluation.Approved || evaluation.BundleID != bundle.BundleID || evaluation.AttestationID != att.AttestationID {
		t.Fatalf("review check did not consume bundle attestation: eval=%+v err=%v", evaluation, err)
	}
}

func TestReviewAttestationCriterionReuseRequiresExactUnchangedContract(t *testing.T) {
	root, store := reviewBundleFixture(t)
	policyPath := filepath.Join(root, ".pose/policy/review.json")
	policyRaw, _ := os.ReadFile(policyPath)
	policy := strings.Replace(string(policyRaw), `"component_aware": true,`, `"component_aware": true,
  "allow_criterion_reuse": true,`, 1)
	if err := os.WriteFile(policyPath, []byte(policy), 0o644); err != nil {
		t.Fatal(err)
	}
	profilePath := filepath.Join(root, ".pose/review-profiles/spec-closeout.json")
	profileRaw, _ := os.ReadFile(profilePath)
	profile := strings.Replace(string(profileRaw), `"criteria":[`, `"criteria":[{"id":"documentation","description":"Docs stay aligned.","rules":["documentation-style"]},`, 1)
	if err := os.WriteFile(profilePath, []byte(profile), 0o644); err != nil {
		t.Fatal(err)
	}
	writeReviewFixture(t, root, ".pose/rules/documentation-style.md", "# Documentation Style\n")
	now := time.Date(2026, 8, 13, 12, 0, 0, 0, time.UTC)
	first, err := store.SealReviewBundle("spec:backend", now)
	if err != nil {
		t.Fatal(err)
	}
	prior, err := store.RecordReviewAttestation(approvedBundleAttestation(first, "agent:first-review"), now.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "api/server.go"), []byte("package api\n\nfunc Ready() bool { return false }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	second, err := store.SealReviewBundle("spec:backend", now.Add(2*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	criterion := second.Payload.Plan.Criteria[0]
	priorCriterion := first.Payload.Plan.Criteria[0]
	for i, candidate := range second.Payload.Plan.Criteria {
		if !reviewCriterionSubjectSensitive(candidate) {
			criterion = candidate
			priorCriterion = first.Payload.Plan.Criteria[i]
			break
		}
	}
	if reviewCriterionSubjectSensitive(criterion) {
		t.Fatalf("fixture has no reusable criterion: %+v", second.Payload.Plan.Criteria)
	}
	attestation := approvedBundleAttestation(second, "agent:targeted-review")
	attestation.BundleDigest = second.BundleDigest
	attestation.AttestedAt = now.Add(3 * time.Minute).Format(time.RFC3339)
	if reviewCriterionInputDigest(first, priorCriterion) != reviewCriterionInputDigest(second, criterion) {
		t.Fatalf("criterion %s slice unexpectedly changed: first=%s second=%s", criterion.ID, reviewCriterionInputDigest(first, priorCriterion), reviewCriterionInputDigest(second, criterion))
	}
	attestation.ReusedFrom = []ReviewAttestationReuse{{Criterion: criterion.ID, FromAttestation: prior.AttestationID, InputDigest: reviewCriterionInputDigest(second, criterion)}}
	if blockers := store.validateBundleAttestation(second, attestation); len(blockers) != 0 {
		t.Fatalf("exact unchanged criterion reuse rejected: %v", blockers)
	}
	attestation.ReusedFrom[0].InputDigest = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	if blockers := store.validateBundleAttestation(second, attestation); !strings.Contains(strings.Join(blockers, " "), "input digest changed") {
		t.Fatalf("changed criterion reuse accepted: %v", blockers)
	}
}

func TestReviewAttestationCriterionReuseRejectsChangedSubjectSlice(t *testing.T) {
	_, store := reviewBundleFixture(t)
	root := store.Root
	policyPath := filepath.Join(root, ".pose/policy/review.json")
	policyRaw, _ := os.ReadFile(policyPath)
	policy := strings.Replace(string(policyRaw), `"component_aware": true,`, `"component_aware": true,
  "allow_criterion_reuse": true,`, 1)
	if err := os.WriteFile(policyPath, []byte(policy), 0o644); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 13, 12, 0, 0, 0, time.UTC)
	first, _ := store.SealReviewBundle("spec:backend", now)
	prior, _ := store.RecordReviewAttestation(approvedBundleAttestation(first, "agent:first-review"), now.Add(time.Minute))
	if err := os.WriteFile(filepath.Join(root, "api/server.go"), []byte("package api\n\nfunc Ready() bool { return false }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	second, _ := store.SealReviewBundle("spec:backend", now.Add(2*time.Minute))
	criterion := second.Payload.Plan.Criteria[0]
	priorCriterion := first.Payload.Plan.Criteria[0]
	for i, candidate := range second.Payload.Plan.Criteria {
		if candidate.ID == "backend-concurrency" {
			criterion = candidate
			priorCriterion = first.Payload.Plan.Criteria[i]
			break
		}
	}
	att := approvedBundleAttestation(second, "agent:targeted-review")
	att.AttestedAt = now.Add(3 * time.Minute).Format(time.RFC3339)
	att.ReusedFrom = []ReviewAttestationReuse{{Criterion: criterion.ID, FromAttestation: prior.AttestationID, InputDigest: reviewCriterionInputDigest(first, priorCriterion)}}
	if blockers := store.validateBundleAttestation(second, att); !strings.Contains(strings.Join(blockers, " "), "input digest changed") {
		t.Fatalf("changed subject slice was reused: %v", blockers)
	}
}

func TestReviewAttestationSealIsIdempotentAndRejectsIdentityCollision(t *testing.T) {
	_, store := reviewBundleFixture(t)
	now := time.Date(2026, 8, 13, 12, 0, 0, 0, time.UTC)
	bundle, err := store.SealReviewBundle("spec:backend", now)
	if err != nil {
		t.Fatal(err)
	}
	first, err := store.RecordReviewAttestation(approvedBundleAttestation(bundle, "agent:review"), now.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	replay := approvedBundleAttestation(bundle, "agent:review")
	replay.AttestedAt = first.AttestedAt
	second, err := store.RecordReviewAttestation(replay, now.Add(2*time.Minute))
	if err != nil || second.AttestationID != first.AttestationID {
		t.Fatalf("identical replay did not reuse attestation: att=%+v err=%v", second, err)
	}
	conflict := replay
	conflict.AttestationID = first.AttestationID
	conflict.EvidenceRefs = []string{"integration:different"}
	if _, err := store.RecordReviewAttestation(conflict, now.Add(2*time.Minute)); err == nil || (!strings.Contains(err.Error(), "identity collision") && !strings.Contains(err.Error(), "content digest")) {
		t.Fatalf("conflicting immutable replay was accepted: %v", err)
	}
}

func TestReviewBundleRejectsUnclassifiedSubjectPath(t *testing.T) {
	root, store := reviewBundleFixture(t)
	writeReviewFixture(t, root, "mystery.data", "opaque")
	graph, err := store.GetDeliveryIntegrity("")
	if err != nil {
		t.Fatal(err)
	}
	graph.ChangeSets[0].Paths = append(graph.ChangeSets[0].Paths, ObservedPath{Action: "created", Path: "mystery.data"})
	raw, _ := json.Marshal(graph)
	writeReviewFixture(t, root, ".pose/indexes/delivery-integrity.json", string(raw))
	bundle, err := store.PrepareReviewBundle("spec:backend")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.Join(bundle.Blockers, " "), "unclassified review subject path mystery.data") {
		t.Fatalf("unknown path did not fail closed: %+v", bundle.Blockers)
	}
	if _, err := store.SealReviewBundle("spec:backend", time.Now()); err == nil {
		t.Fatal("unclassified subject was sealable")
	}
}

func TestReviewBundleSubjectUsesCurrentRenameAndClassifiesReleaseMetadata(t *testing.T) {
	root, store := reviewBundleFixture(t)
	writeReviewFixture(t, root, ".pose/changelogs/v1.1.0/backend.md", "released fragment\n")
	writeReviewFixture(t, root, ".pose/releases/v1.1.0/manifest.json", "{}\n")
	writeReviewFixture(t, root, ".pose/specs/sibling/spec.md", "---\nslug: sibling\nstatus: done\n---\n")
	graph, err := store.GetDeliveryIntegrity("")
	if err != nil {
		t.Fatal(err)
	}
	graph.ChangeSets[0].Paths = append(graph.ChangeSets[0].Paths,
		ObservedPath{Action: "created", Path: ".pose/changelogs/unreleased/backend.md"},
	)
	graph.ChangeSets = append(graph.ChangeSets, ChangeSet{
		ID: "cs-release", Spec: "backend", ResolvedBase: "head-resolved", ResolvedHead: "release-head",
		Paths: []ObservedPath{
			{Action: "renamed", OldPath: ".pose/changelogs/unreleased/backend.md", NewPath: ".pose/changelogs/v1.1.0/backend.md"},
			{Action: "created", Path: ".pose/releases/v1.1.0/manifest.json"},
			{Action: "modified", Path: ".pose/specs/sibling/spec.md"},
		},
	})
	scope, err := ParseScopeRef("spec:backend")
	if err != nil {
		t.Fatal(err)
	}
	subject, excluded, blockers, err := store.reviewBundleSubject(scope, []ReviewPlanComponent{{Path: "api"}}, graph, nil)
	if err != nil || len(blockers) != 0 {
		t.Fatalf("release rename subject blockers=%v err=%v", blockers, err)
	}
	if len(subject.Entries) != 3 {
		t.Fatalf("subject entries=%+v, want implementation, archived fragment and manifest", subject.Entries)
	}
	excludedText := ""
	for _, input := range excluded {
		excludedText += input.Kind + ":" + input.Path + "\n"
	}
	if !strings.Contains(excludedText, "superseded-path:.pose/changelogs/unreleased/backend.md") ||
		!strings.Contains(excludedText, "semantic-scope:.pose/specs/sibling/spec.md") {
		t.Fatalf("release exclusions=%s", excludedText)
	}
}

func TestReviewBundleRejectsWorkingTreeOnlySubjectContent(t *testing.T) {
	root, store := reviewBundleFixture(t)
	runStateTestGit(t, root, "init")
	runStateTestGit(t, root, "config", "user.name", "POSE Test")
	runStateTestGit(t, root, "config", "user.email", "pose@example.invalid")
	runStateTestGit(t, root, "add", ".")
	runStateTestGit(t, root, "commit", "-m", "test: seed review fixture")
	if err := os.WriteFile(filepath.Join(root, "api/server.go"), []byte("package api\n\nfunc Ready() bool { return false }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	bundle, err := store.PrepareReviewBundle("spec:backend")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.Join(bundle.Blockers, " "), "working-tree-only content") {
		t.Fatalf("working-tree-only subject was sealable: blockers=%v", bundle.Blockers)
	}
}

func TestReviewBundleMilestoneSubjectIsConfinedAndChildOrderIsDeclared(t *testing.T) {
	root, store := reviewBundleFixture(t)
	writeReviewFixture(t, root, "api/unrelated.go", "package api\n")
	writeReviewFixture(t, root, ".pose/roadmaps/delivery.md", `---
slug: delivery
status: active
---

# Roadmap

## Milestone: core
- specs: backend, zeta

## Milestone: later
- specs: alpha
`)
	graph, err := store.GetDeliveryIntegrity("")
	if err != nil {
		t.Fatal(err)
	}
	graph.ChangeSets = append(graph.ChangeSets, ChangeSet{ID: "cs-unrelated", Spec: "unrelated", ResolvedBase: "base", ResolvedHead: "head", Paths: []ObservedPath{{Action: "created", Path: "api/unrelated.go"}}})
	subject, _, blockers, err := store.reviewBundleSubject(ScopeRef{Kind: "milestone", Roadmap: "delivery", Milestone: "core"}, []ReviewPlanComponent{{ID: "api", Path: "api"}}, graph, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(blockers) != 0 || len(subject.ChangeSets) != 1 || subject.ChangeSets[0] != "cs-backend" {
		t.Fatalf("milestone subject escaped declared specs: subject=%+v blockers=%v", subject, blockers)
	}
	refs, err := store.reviewBundleChildRefs(ScopeRef{Kind: "milestone", Roadmap: "delivery", Milestone: "core"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(refs, ",") != "spec:backend,spec:zeta" {
		t.Fatalf("child order = %v, want declaration order", refs)
	}
	refs, err = store.reviewBundleChildRefs(ScopeRef{Kind: "roadmap", Slug: "delivery"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(refs, ",") != "milestone:delivery/core,milestone:delivery/later" {
		t.Fatalf("milestone order = %v, want declaration order", refs)
	}
}

func TestReviewBundleDeltaIncludesChangedComponentsAndEvidenceClasses(t *testing.T) {
	_, store := reviewBundleFixture(t)
	from, err := store.PrepareReviewBundle("spec:backend")
	if err != nil {
		t.Fatal(err)
	}
	to := from
	to.BundleID = "rvb-ffffffffffffffff"
	to.Payload.Plan.Components = append([]ReviewPlanComponent{}, from.Payload.Plan.Components...)
	to.Payload.Plan.Components[0].Owner = "@new-owner"
	to.Payload.Evidence = append([]ReviewBundleEvidence{}, from.Payload.Evidence...)
	to.Payload.Evidence[0].EvidenceClass = "e2e"
	delta := ReviewBundleDiff(from, to)
	if strings.Join(delta.ChangedComponents, ",") != from.Payload.Plan.Components[0].ID {
		t.Fatalf("changed components = %v", delta.ChangedComponents)
	}
	if strings.Join(delta.ChangedEvidence, ",") != from.Payload.Evidence[0].ID || strings.Join(delta.ChangedEvidenceClasses, ",") != "e2e,integration" {
		t.Fatalf("changed evidence = %v classes=%v", delta.ChangedEvidence, delta.ChangedEvidenceClasses)
	}
}

func TestReviewBundleDropsSubsumedChangeSets(t *testing.T) {
	sets := []ChangeSet{
		{ID: "cs-narrow", Commits: []string{"a"}, ResolvedHead: "a"},
		{ID: "cs-wide", Commits: []string{"a", "b"}, ResolvedHead: "b"},
		{ID: "cs-disjoint", Commits: []string{"c"}, ResolvedHead: "c"},
	}
	reduced := reduceReviewBundleChangeSets(sets)
	if len(reduced) != 2 || reduced[0].ID != "cs-wide" || reduced[1].ID != "cs-disjoint" {
		t.Fatalf("reduced change sets = %+v", reduced)
	}
}

func TestReviewBundleRejectsManagedDirectorySymlinkEscape(t *testing.T) {
	root, store := reviewBundleFixture(t)
	outside := t.TempDir()
	link := filepath.Join(root, ".pose", "review-bundles")
	if err := os.Symlink(outside, link); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	if _, err := store.SealReviewBundle("spec:backend", time.Now()); err == nil || !strings.Contains(err.Error(), "refusing to follow review artifact symlink") {
		t.Fatalf("review bundle followed managed-directory symlink: %v", err)
	}
}

func TestReviewAttestationEnvelopeTrustPolicy(t *testing.T) {
	root, store := reviewBundleFixture(t)
	bundle, err := store.SealReviewBundle("spec:backend", time.Date(2026, 8, 13, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	seed := make([]byte, ed25519.SeedSize)
	for i := range seed {
		seed[i] = byte(i + 1)
	}
	privateKey := ed25519.NewKeyFromSeed(seed)
	publicKey := privateKey.Public().(ed25519.PublicKey)
	issuer := "conductor:test"
	policyPath := filepath.Join(root, ".pose/policy/review.json")
	policyRaw, _ := os.ReadFile(policyPath)
	policy := strings.Replace(string(policyRaw), `"overlay_profiles": ["frontend-review@1", "backend-review@1"]`, `"overlay_profiles": ["frontend-review@1", "backend-review@1"],
  "require_signed_attestations": true,
  "trusted_attestation_issuers": ["`+issuer+`#`+digestBytes(publicKey)+`"]`, 1)
	if err := os.WriteFile(policyPath, []byte(policy), 0o644); err != nil {
		t.Fatal(err)
	}
	attestation := ReviewAttestation{SchemaVersion: 1, BundleID: bundle.BundleID, BundleDigest: bundle.BundleDigest, Reviewer: "agent:independent-envelope", Decision: "approved", Criteria: []ReviewCriterion{}, Findings: []ReviewFinding{}, AttestedAt: "2026-08-13T12:01:00Z"}
	attestation.AttestationID = reviewAttestationID(attestation)
	raw, _ := json.Marshal(attestation)
	envelope := ReviewAttestationEnvelope{SchemaVersion: 1, Issuer: issuer, Subject: bundle.BundleID, Algorithm: "ed25519", PublicKey: base64.StdEncoding.EncodeToString(publicKey), Signature: base64.StdEncoding.EncodeToString(ed25519.Sign(privateKey, raw)), Attestation: attestation}
	verified, err := store.VerifyReviewAttestationEnvelope(envelope)
	if err != nil || verified.BundleID != bundle.BundleID {
		t.Fatalf("valid envelope rejected: att=%+v err=%v", verified, err)
	}
	if _, err := store.RecordReviewAttestation(attestation, time.Now()); err == nil || !strings.Contains(err.Error(), "requires a trusted signed") {
		t.Fatalf("unsigned local attestation bypassed signed policy: %v", err)
	}
	envelopeRaw, _ := json.Marshal(envelope)
	writeReviewFixture(t, root, "signed-attestation.json", string(envelopeRaw))
	imported, err := store.ImportReviewAttestationEnvelope("signed-attestation.json", true)
	if err != nil || imported.AttestationID != attestation.AttestationID {
		t.Fatalf("trusted signed import failed: att=%+v err=%v", imported, err)
	}
	loaded, err := store.LoadReviewAttestation(imported.AttestationID)
	if err != nil || loaded.Envelope == nil {
		t.Fatalf("signed proof was not retained: att=%+v err=%v", loaded, err)
	}
	if blockers := store.validateBundleAttestation(bundle, loaded); strings.Contains(strings.Join(blockers, " "), "signature") || strings.Contains(strings.Join(blockers, " "), "untrusted") {
		// Signature revalidation must not be the blocker; this deliberately sparse
		// fixture still lacks the full review criteria.
		t.Fatalf("stored signed attestation was not revalidated as expected: %v", blockers)
	}
	envelope.Signature = base64.StdEncoding.EncodeToString(make([]byte, ed25519.SignatureSize))
	if _, err := store.VerifyReviewAttestationEnvelope(envelope); err == nil || !strings.Contains(err.Error(), "signature") {
		t.Fatalf("invalid signature accepted: %v", err)
	}
	envelope.Signature = base64.StdEncoding.EncodeToString(ed25519.Sign(privateKey, raw))
	envelope.Issuer = "conductor:untrusted"
	if _, err := store.VerifyReviewAttestationEnvelope(envelope); err == nil || !strings.Contains(err.Error(), "untrusted") {
		t.Fatalf("untrusted issuer accepted: %v", err)
	}
}

func TestReviewBundleRejectsTraversal(t *testing.T) {
	_, store := reviewBundleFixture(t)
	if _, err := store.ImportReviewAttestationEnvelope("../outside.json", false); err == nil || !strings.Contains(err.Error(), "invalid attestation envelope path") {
		t.Fatalf("traversal was not rejected: %v", err)
	}
}

func TestReviewBundleRejectsSymlinkEscape(t *testing.T) {
	root, store := reviewBundleFixture(t)
	outside := filepath.Join(t.TempDir(), "envelope.json")
	if err := os.WriteFile(outside, []byte(`{}`), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "envelope-link.json")
	if err := os.Symlink(outside, link); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	if _, err := store.ImportReviewAttestationEnvelope("envelope-link.json", false); err == nil || !strings.Contains(err.Error(), "invalid attestation envelope path") {
		t.Fatalf("symlink escape was not rejected: %v", err)
	}
}

func TestReviewBundleRejectsMalformedInput(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bundle.json")
	cases := []struct {
		name string
		raw  string
		want string
	}{
		{"duplicate", `{"schema_version":1,"schema_version":1}`, "duplicate JSON field"},
		{"unknown", `{"schema_version":1,"unexpected":true}`, "unknown field"},
		{"trailing", `{"schema_version":1} {}`, "trailing JSON content"},
		{"control", `{"schema_version":1,"bundle_id":"rvb-0123456789ab\u000a"}`, "control character"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := os.WriteFile(path, []byte(tc.raw), 0o600); err != nil {
				t.Fatal(err)
			}
			var bundle ReviewBundle
			if err := strictJSONFile(path, &bundle); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("malformed input result = %v, want %q", err, tc.want)
			}
		})
	}
}

func TestReviewAttestationRejectsMalformedInput(t *testing.T) {
	path := filepath.Join(t.TempDir(), "attestation.json")
	raw := []byte(`{"schema_version":1,"attestation_id":"rva-0123456789abcdef","attestation_id":"rva-fedcba9876543210"}`)
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	var attestation ReviewAttestation
	if err := strictJSONFile(path, &attestation); err == nil || !strings.Contains(err.Error(), "duplicate JSON field") {
		t.Fatalf("duplicate attestation identity was accepted: %v", err)
	}
	oversized := make([]byte, maxReviewBundleBytes+1)
	if err := os.WriteFile(path, oversized, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := strictJSONFile(path, &attestation); err == nil || !strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("oversized attestation was accepted: %v", err)
	}
}
