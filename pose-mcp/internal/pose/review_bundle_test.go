package pose

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"os"
	"os/exec"
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
		Deliveries: []DeliveryTarget{{Spec: "backend", Ref: "contract:backend-api", Kind: "contract", ID: "backend-api", Module: "api", Profile: "api-contract", Entrypoint: "api/server.go"}},
		ValidationResults: []DeliveryValidationResult{
			{ID: "validate-backend", Module: "api", Check: "go-test", EvidenceClass: "integration", Severity: "required", Outcome: "pass", GitHead: "head-resolved", ProvenanceDigest: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
			// The plan's `backend-observability` and `correctness` criteria ask
			// for class `test`. Without a result that emits it the fixture is a
			// repository whose own plan cannot be satisfied, and every
			// attestation built on it can only cite evidence that is not there.
			{ID: "unit-backend", Module: "api", Check: "go-unit", EvidenceClass: "unit", Severity: "required", Outcome: "pass", GitHead: "head-resolved", ProvenanceDigest: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
		},
		Reverse: map[string][]string{"api/server.go": {"backend"}}, Nodes: []DeliveryIntegrityNode{}, Edges: []DeliveryIntegrityEdge{}, Claims: []ArtifactClaim{}, Findings: []DeliveryIntegrityFinding{},
	}
	raw, err := json.MarshalIndent(graph, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	writeReviewFixture(t, root, ".pose/indexes/delivery-integrity.json", string(raw))
	return root, store
}

// approvedBundleAttestation builds the attestation a reviewer who did the work
// would record: every criterion cites evidence the bundle actually seals, of a
// class the criterion asks for. It used to cite a fixed `integration:bundle-test`
// that appeared in no bundle, so the whole suite proved only that a signature
// was well-formed — never that it was supported.
func approvedBundleAttestation(bundle ReviewBundle, reviewer string) ReviewAttestation {
	sealed := []string{}
	byClass := map[string][]string{}
	for _, ev := range bundle.Payload.Evidence {
		ref := ev.EvidenceClass + ":" + ev.ID
		sealed = append(sealed, ref)
		byClass[ev.EvidenceClass] = append(byClass[ev.EvidenceClass], ref)
	}
	pick := func(classes []string) string {
		for _, class := range classes {
			if refs := byClass[class]; len(refs) > 0 {
				return refs[0]
			}
		}
		if len(classes) == 0 && len(sealed) > 0 {
			return sealed[0]
		}
		return ""
	}
	criteria := []ReviewCriterion{}
	for _, criterion := range bundle.Payload.Plan.Criteria {
		if !criterion.Required {
			continue
		}
		if evidence := pick(criterion.EvidenceClasses); evidence != "" {
			criteria = append(criteria, ReviewCriterion{ID: criterion.ID, Disposition: "passed", Evidence: evidence})
			continue
		}
		criteria = append(criteria, ReviewCriterion{ID: criterion.ID, Disposition: "not-applicable", Rationale: "the fixture seals no evidence of a class this criterion asks for"})
	}
	tools := []ReviewToolDisposition{}
	for _, tool := range bundle.Payload.Plan.Tools {
		disposition := ReviewToolDisposition{ID: tool.ID, Component: tool.Component}
		if containsFold(tool.Preconditions, "review-complete") {
			disposition.Disposition, disposition.Rationale = "deferred", "post-review gate"
		} else if tool.Requiredness == "recommended" {
			disposition.Disposition, disposition.Rationale = "not-used", "not needed for fixture"
		} else if ev := pick(tool.EvidenceClasses); ev != "" {
			disposition.Disposition, disposition.Evidence = "passed", ev
		} else if len(tool.EvidenceClasses) == 0 {
			// A tool that declares no class is supported by having run, which
			// naming it records. This used to cite `integration:bundle-test`,
			// which appeared in no bundle — the same fabrication the criteria
			// half of this helper carried.
			disposition.Disposition, disposition.Evidence = "passed", "check:"+tool.ID
		} else {
			disposition.Disposition, disposition.Rationale = "not-used", "the fixture seals no evidence of a class this tool reports"
		}
		tools = append(tools, disposition)
	}
	return ReviewAttestation{BundleID: bundle.BundleID, Reviewer: reviewer, Decision: "approved", Criteria: criteria, Tools: tools, EvidenceRefs: sealed, Findings: []ReviewFinding{}}
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

func TestReviewBundleClassifiesSubmodulePath(t *testing.T) {
	root, store := reviewBundleFixture(t)
	for _, args := range [][]string{
		{"init", "-q"},
		{"config", "user.email", "fixture@example.test"},
		{"config", "user.name", "fixture"},
		// `git commit` can spawn `gc --auto`, which keeps writing into
		// .git/objects after the test returns and races t.TempDir's cleanup:
		// `unlinkat .git/objects: directory not empty`, intermittently and only
		// under load. The assertions had already passed.
		{"config", "gc.auto", "0"},
		{"add", "-A"},
		{"commit", "-q", "-m", "fixture"},
		// A gitlink without a checked-out submodule: enough for the index to
		// record mode 160000, and no network.
		{"update-index", "--add", "--cacheinfo", "160000,0000000000000000000000000000000000000001,vendor/dep"},
	} {
		if out, err := exec.Command("git", append([]string{"-C", root}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
	graph, err := store.GetDeliveryIntegrity("")
	if err != nil {
		t.Fatal(err)
	}
	graph.ChangeSets[0].Paths = append(graph.ChangeSets[0].Paths, ObservedPath{Action: "modified", Path: "vendor/dep"})
	raw, _ := json.Marshal(graph)
	writeReviewFixture(t, root, ".pose/indexes/delivery-integrity.json", string(raw))
	bundle, err := store.PrepareReviewBundle("spec:backend")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(strings.Join(bundle.Blockers, " "), "unclassified review subject path vendor/dep") {
		t.Fatalf("submodule was treated as unclassified: %+v", bundle.Blockers)
	}
	var entry ReviewBundleSubjectEntry
	for _, candidate := range bundle.Payload.Subject.Entries {
		if candidate.Path == "vendor/dep" {
			entry = candidate
		}
	}
	if entry.Class != "submodule" {
		t.Fatalf("submodule class = %q, want submodule", entry.Class)
	}
	if entry.Digest == "" {
		t.Fatal("submodule entry carries no digest")
	}
	if !strings.Contains(entry.Reason, "0000000000000000000000000000000000000001") {
		t.Fatalf("reason does not name the pinned commit: %q", entry.Reason)
	}
}

func TestReviewBundleClassifiesSubmoduleUnderAMappedComponent(t *testing.T) {
	root, store := reviewBundleFixture(t)
	for _, args := range [][]string{
		{"init", "-q"},
		{"config", "user.email", "fixture@example.test"},
		{"config", "user.name", "fixture"},
		// `git commit` can spawn `gc --auto`, which keeps writing into
		// .git/objects after the test returns and races t.TempDir's cleanup:
		// `unlinkat .git/objects: directory not empty`, intermittently and only
		// under load. The assertions had already passed.
		{"config", "gc.auto", "0"},
		{"add", "-A"},
		{"commit", "-q", "-m", "fixture"},
	} {
		if out, err := exec.Command("git", append([]string{"-C", root}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
	// A real nested repository, so the gitlink is genuine and the working tree
	// is clean — `update-index --cacheinfo` alone leaves the path added in the
	// index and absent on disk, which is not what a checked-out submodule looks
	// like. It sits under the mapped component `api`, so the path-shape rules
	// classify it before anything asks Git whether it is a gitlink.
	writeReviewFixture(t, root, "api/dep/README.md", "# dep\n")
	for _, args := range [][]string{
		{"init", "-q"},
		{"config", "user.email", "fixture@example.test"},
		{"config", "user.name", "fixture"},
		// `git commit` can spawn `gc --auto`, which keeps writing into
		// .git/objects after the test returns and races t.TempDir's cleanup:
		// `unlinkat .git/objects: directory not empty`, intermittently and only
		// under load. The assertions had already passed.
		{"config", "gc.auto", "0"},
		{"add", "-A"},
		{"commit", "-q", "-m", "dep"},
	} {
		if out, err := exec.Command("git", append([]string{"-C", filepath.Join(root, "api", "dep")}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("git dep %v: %v: %s", args, err, out)
		}
	}
	for _, args := range [][]string{{"add", "api/dep"}, {"commit", "-q", "-m", "vendor dep"}} {
		if out, err := exec.Command("git", append([]string{"-C", root}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
	graph, err := store.GetDeliveryIntegrity("")
	if err != nil {
		t.Fatal(err)
	}
	graph.ChangeSets[0].Paths = append(graph.ChangeSets[0].Paths, ObservedPath{Action: "modified", Path: "api/dep"})
	raw, _ := json.Marshal(graph)
	writeReviewFixture(t, root, ".pose/indexes/delivery-integrity.json", string(raw))
	bundle, err := store.PrepareReviewBundle("spec:backend")
	if err != nil {
		t.Fatal(err)
	}
	if joined := strings.Join(bundle.Blockers, " "); strings.Contains(joined, "api/dep") {
		t.Fatalf("submodule under a mapped component blocked the bundle: %s", joined)
	}
	var entry ReviewBundleSubjectEntry
	for _, candidate := range bundle.Payload.Subject.Entries {
		if candidate.Path == "api/dep" {
			entry = candidate
		}
	}
	if entry.Class != "submodule" {
		t.Fatalf("class = %q, want submodule: a gitlink is a gitlink wherever it sits", entry.Class)
	}
	if entry.Digest == "" {
		t.Fatal("submodule entry carries no digest")
	}
}

func TestReviewBundleDoesNotBlockOnAnUnclassifiedRemoval(t *testing.T) {
	root, store := reviewBundleFixture(t)
	graph, err := store.GetDeliveryIntegrity("")
	if err != nil {
		t.Fatal(err)
	}
	// A root-level dotfile the shape rules do not recognise. As a creation it
	// blocks, and should; as a removal there is nothing left to read and the
	// deletion is the reviewable fact.
	graph.ChangeSets[0].Paths = append(graph.ChangeSets[0].Paths, ObservedPath{Action: "removed", Path: ".agent-sync-cache.json"})
	raw, _ := json.Marshal(graph)
	writeReviewFixture(t, root, ".pose/indexes/delivery-integrity.json", string(raw))
	bundle, err := store.PrepareReviewBundle("spec:backend")
	if err != nil {
		t.Fatal(err)
	}
	if joined := strings.Join(bundle.Blockers, " "); strings.Contains(joined, ".agent-sync-cache.json") {
		t.Fatalf("an unclassified removal blocked the bundle: %s", joined)
	}
	var entry ReviewBundleSubjectEntry
	for _, candidate := range bundle.Payload.Subject.Entries {
		if candidate.Path == ".agent-sync-cache.json" {
			entry = candidate
		}
	}
	if entry.Class != "removed" {
		t.Fatalf("class = %q, want removed", entry.Class)
	}
	if entry.Digest != "" {
		t.Fatalf("a removal carries a digest: %q", entry.Digest)
	}
}

func TestReviewBundleStillBlocksOnAnUnclassifiedCreation(t *testing.T) {
	root, store := reviewBundleFixture(t)
	writeReviewFixture(t, root, ".agent-sync-cache.json", "{}")
	graph, err := store.GetDeliveryIntegrity("")
	if err != nil {
		t.Fatal(err)
	}
	graph.ChangeSets[0].Paths = append(graph.ChangeSets[0].Paths, ObservedPath{Action: "created", Path: ".agent-sync-cache.json"})
	raw, _ := json.Marshal(graph)
	writeReviewFixture(t, root, ".pose/indexes/delivery-integrity.json", string(raw))
	bundle, err := store.PrepareReviewBundle("spec:backend")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.Join(bundle.Blockers, " "), "unclassified review subject path .agent-sync-cache.json") {
		t.Fatalf("an unclassified creation stopped failing closed: %+v", bundle.Blockers)
	}
}

func TestReviewBundleClassifiesRootReleaseFiles(t *testing.T) {
	root, store := reviewBundleFixture(t)
	writeReviewFixture(t, root, "README.md", "# root readme\n")
	writeReviewFixture(t, root, "compatibility.json", `{"engine_version":"1.2.1"}`)
	graph, err := store.GetDeliveryIntegrity("")
	if err != nil {
		t.Fatal(err)
	}
	graph.ChangeSets[0].Paths = append(graph.ChangeSets[0].Paths,
		ObservedPath{Action: "modified", Path: "README.md"},
		ObservedPath{Action: "modified", Path: "compatibility.json"},
	)
	raw, _ := json.Marshal(graph)
	writeReviewFixture(t, root, ".pose/indexes/delivery-integrity.json", string(raw))
	bundle, err := store.PrepareReviewBundle("spec:backend")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(strings.Join(bundle.Blockers, " "), "unclassified review subject path") {
		t.Fatalf("known root release files were treated as unclassified: %+v", bundle.Blockers)
	}
	classes := map[string]string{}
	for _, entry := range bundle.Payload.Subject.Entries {
		classes[entry.Path] = entry.Class
	}
	if classes["README.md"] != "documentation" {
		t.Fatalf("README.md classified as %q, want documentation", classes["README.md"])
	}
	if classes["compatibility.json"] != "governance" {
		t.Fatalf("compatibility.json classified as %q, want governance", classes["compatibility.json"])
	}
}

func TestReviewBundleClassifiesRootManifestsAndProjectFiles(t *testing.T) {
	root, store := reviewBundleFixture(t)
	writeReviewFixture(t, root, "go.mod", "module example.com/test\n\ngo 1.22\n")
	writeReviewFixture(t, root, "PROJECT.md", "# Project Overview\n")
	writeReviewFixture(t, root, "package.json", `{"name": "test"}`)
	writeReviewFixture(t, root, "Cargo.toml", `[package]
name = "test"
version = "0.1.0"
`)
	writeReviewFixture(t, root, ".gitignore", "*.o\n")
	writeReviewFixture(t, root, "cmd/main.go", "package main\nfunc main() {}\n")

	graph, err := store.GetDeliveryIntegrity("")
	if err != nil {
		t.Fatal(err)
	}
	graph.ChangeSets[0].Paths = append(graph.ChangeSets[0].Paths,
		ObservedPath{Action: "modified", Path: "go.mod"},
		ObservedPath{Action: "created", Path: "PROJECT.md"},
		ObservedPath{Action: "created", Path: "package.json"},
		ObservedPath{Action: "created", Path: "Cargo.toml"},
		ObservedPath{Action: "created", Path: ".gitignore"},
		ObservedPath{Action: "created", Path: "cmd/main.go"},
	)
	raw, _ := json.Marshal(graph)
	writeReviewFixture(t, root, ".pose/indexes/delivery-integrity.json", string(raw))
	bundle, err := store.PrepareReviewBundle("spec:backend")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(strings.Join(bundle.Blockers, " "), "unclassified review subject path") {
		t.Fatalf("root manifests and project files were treated as unclassified: %+v", bundle.Blockers)
	}
	classes := map[string]string{}
	for _, entry := range bundle.Payload.Subject.Entries {
		classes[entry.Path] = entry.Class
	}
	if classes["go.mod"] != "governance" {
		t.Fatalf("go.mod classified as %q, want governance", classes["go.mod"])
	}
	if classes["PROJECT.md"] != "documentation" {
		t.Fatalf("PROJECT.md classified as %q, want documentation", classes["PROJECT.md"])
	}
	if classes["package.json"] != "governance" {
		t.Fatalf("package.json classified as %q, want governance", classes["package.json"])
	}
	if classes["Cargo.toml"] != "governance" {
		t.Fatalf("Cargo.toml classified as %q, want governance", classes["Cargo.toml"])
	}
	if classes[".gitignore"] != "governance" {
		t.Fatalf(".gitignore classified as %q, want governance", classes[".gitignore"])
	}
	if classes["cmd/main.go"] != "implementation" {
		t.Fatalf("cmd/main.go classified as %q, want implementation", classes["cmd/main.go"])
	}
}

func TestReviewBundleMatchesRootModuleValidationEvidenceForSubdirectoryTargets(t *testing.T) {
	root, store := reviewBundleFixture(t)
	writeReviewFixture(t, root, "internal/mypkg/lib.go", "package mypkg\n")
	graph, err := store.GetDeliveryIntegrity("")
	if err != nil {
		t.Fatal(err)
	}

	// Change set for target spec
	graph.ChangeSets = []ChangeSet{{
		ID: "cs-sub", Spec: "backend", Selector: "range:base..head", Base: "base", Head: "head", ResolvedBase: "base", ResolvedHead: "head",
		Paths: []ObservedPath{{Action: "modified", Path: "internal/mypkg/lib.go"}},
	}}
	// Delivery target declared with subdirectory module: internal/mypkg
	graph.Deliveries = []DeliveryTarget{{
		Spec: "backend", Ref: "contract:my-contract", Kind: "contract", ID: "my-contract",
		Module: "internal/mypkg", Profile: "api-contract", Entrypoint: "internal/mypkg/lib.go",
	}}
	// Validation result emitted at root module "."
	graph.ValidationResults = []DeliveryValidationResult{{
		ID: "val-root", Module: ".", Check: "go-test", EvidenceClass: "unit", Severity: "required", Outcome: "pass",
		GitHead: "head", ProvenanceDigest: graph.ProvenanceDigest,
	}}
	raw, _ := json.Marshal(graph)
	writeReviewFixture(t, root, ".pose/indexes/delivery-integrity.json", string(raw))

	bundle, err := store.PrepareReviewBundle("spec:backend")
	if err != nil {
		t.Fatal(err)
	}
	if len(bundle.Blockers) > 0 {
		t.Fatalf("expected no blockers, got: %+v", bundle.Blockers)
	}
	if len(bundle.Payload.Evidence) == 0 {
		t.Fatalf("expected root validation evidence to be attributed to spec:backend, got 0")
	}
	if bundle.Payload.Evidence[0].ID != "val-root" {
		t.Fatalf("expected evidence val-root, got: %+v", bundle.Payload.Evidence)
	}
}

func TestListReviewBundlesScopeIsolationFromUnrelatedCorruptedBundle(t *testing.T) {
	root, store := reviewBundleFixture(t)

	// Bundle A for spec:backend (valid)
	sealedA, err := store.SealReviewBundle("spec:backend", time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}

	// Corrupted Bundle B for spec:unrelated (simulate hand-edit / corruption)
	corruptedJSON := `{
  "schema_version": 2,
  "bundle_id": "rvb-0000000000000000",
  "bundle_digest": "sha256:0000000000000000000000000000000000000000000000000000000000000000",
  "sealed_at": "2026-08-22T00:00:00Z",
  "payload": {
    "scope": {"ref": "spec:unrelated", "kind": "spec", "slug": "unrelated"}
  }
}`
	writeReviewFixture(t, root, ".pose/review-bundles/rvb-0000000000000000.json", corruptedJSON)

	// Listing bundles for spec:backend must succeed and return sealedA despite corrupted rvb-0000000000000000.json
	bundles, err := store.ListReviewBundles("spec:backend")
	if err != nil {
		t.Fatalf("expected ListReviewBundles(spec:backend) to succeed, got: %v", err)
	}
	if len(bundles) != 1 || bundles[0].BundleID != sealedA.BundleID {
		t.Fatalf("expected 1 bundle with ID %s, got: %+v", sealedA.BundleID, bundles)
	}

	// CurrentReviewBundle for spec:backend must also succeed
	curr, err := store.CurrentReviewBundle("spec:backend")
	if err != nil || curr == nil || curr.BundleID != sealedA.BundleID {
		t.Fatalf("expected CurrentReviewBundle to return sealedA, got: %+v, err=%v", curr, err)
	}

	// But listing for spec:unrelated must surface the corruption error
	_, errUnrelated := store.ListReviewBundles("spec:unrelated")
	if errUnrelated == nil {
		t.Fatal("expected ListReviewBundles(spec:unrelated) to return error for corrupted bundle")
	}
}

// TestReviewBundleClassifiesExtensionsDirectory guards spec
// pose-domain-rule-extension-migration: extensions/ predates review-bundle
// path classification (pose-rule-kubernetes shipped 2026-08-07, before
// component_aware/review_bundles were adopted on 2026-08-13/14), so no spec
// touching it had ever exercised this path — discovered when migrating
// backend-go.md/frontend-react.md into extensions/pose-rule-*.
func TestReviewBundleClassifiesExtensionsDirectory(t *testing.T) {
	root, store := reviewBundleFixture(t)
	writeReviewFixture(t, root, "extensions/pose-rule-backend-go/extension.json", `{"id":"pose-rule-backend-go"}`)
	writeReviewFixture(t, root, "extensions/pose-rule-backend-go/files/.pose/rules/backend-go.md", "# Rule: Backend Go\n")
	graph, err := store.GetDeliveryIntegrity("")
	if err != nil {
		t.Fatal(err)
	}
	graph.ChangeSets[0].Paths = append(graph.ChangeSets[0].Paths,
		ObservedPath{Action: "created", Path: "extensions/pose-rule-backend-go/extension.json"},
		ObservedPath{Action: "created", Path: "extensions/pose-rule-backend-go/files/.pose/rules/backend-go.md"},
	)
	raw, _ := json.Marshal(graph)
	writeReviewFixture(t, root, ".pose/indexes/delivery-integrity.json", string(raw))
	bundle, err := store.PrepareReviewBundle("spec:backend")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(strings.Join(bundle.Blockers, " "), "unclassified review subject path") {
		t.Fatalf("extensions/ paths were treated as unclassified: %+v", bundle.Blockers)
	}
	classes := map[string]string{}
	for _, entry := range bundle.Payload.Subject.Entries {
		classes[entry.Path] = entry.Class
	}
	if classes["extensions/pose-rule-backend-go/extension.json"] != "governance" {
		t.Fatalf("extension.json classified as %q, want governance", classes["extensions/pose-rule-backend-go/extension.json"])
	}
	if classes["extensions/pose-rule-backend-go/files/.pose/rules/backend-go.md"] != "governance" {
		t.Fatalf("extension file classified as %q, want governance", classes["extensions/pose-rule-backend-go/files/.pose/rules/backend-go.md"])
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
	replaced := from.Payload.Evidence[0].EvidenceClass
	to.Payload.Evidence[0].EvidenceClass = "e2e"
	delta := ReviewBundleDiff(from, to)
	wantClasses := "e2e," + replaced
	if strings.Join(delta.ChangedComponents, ",") != from.Payload.Plan.Components[0].ID {
		t.Fatalf("changed components = %v", delta.ChangedComponents)
	}
	if strings.Join(delta.ChangedEvidence, ",") != from.Payload.Evidence[0].ID || strings.Join(delta.ChangedEvidenceClasses, ",") != wantClasses {
		t.Fatalf("changed evidence = %v classes=%v, want %s", delta.ChangedEvidence, delta.ChangedEvidenceClasses, wantClasses)
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

func TestAutoAttestReviewBundle(t *testing.T) {
	_, store := reviewBundleFixture(t)
	bundle, err := store.SealReviewBundle("spec:backend", time.Date(2026, 8, 14, 10, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	att, err := store.AutoAttestReviewBundle(bundle.BundleID, "agent:test-subagent", true, time.Date(2026, 8, 14, 10, 5, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("AutoAttestReviewBundle failed: %v", err)
	}
	if att.Reviewer != "agent:test-subagent" || att.Decision != "approved" {
		t.Fatalf("unexpected attestation: %+v", att)
	}
	if len(att.Criteria) == 0 {
		t.Fatalf("no criteria in auto-attestation: %+v", att.Criteria)
	}
	for _, c := range att.Criteria {
		if c.Disposition != "passed" {
			t.Fatalf("criterion %s not passed: %+v", c.ID, c)
		}
	}
	verification, err := store.VerifyReviewBundle("spec:backend")
	if err != nil {
		t.Fatal(err)
	}
	if !verification.Approved || !verification.Fresh {
		t.Fatalf("auto-attested bundle did not verify: %+v", verification)
	}
}

func TestReviewBundleComponentDiscoveryAndGovernancePaths(t *testing.T) {
	root, store := componentReviewFixture(t)

	// Create directories matching custom component roots without repo-map.json
	writeReviewFixture(t, root, "agent/runner.go", "package agent\n")
	writeReviewFixture(t, root, "conductor/orchestrator.go", "package conductor\n")
	writeReviewFixture(t, root, "docs/decisions/ADR-001-custom.md", "# ADR 001\n")
	writeReviewFixture(t, root, ".pose/roadmaps/vision.md", "---\nslug: vision\nstatus: active\n---\n# Roadmap\n")
	writeReviewFixture(t, root, ".pose/docs.json", "{\"schema_version\": 1}\n")

	writeReviewFixture(t, root, ".pose/specs/agent-feature/spec.md", `---
slug: agent-feature
status: in-progress
created_at: 2026-08-22
components: conductor, agent
delivers: capability:agent-runner
---

# Spec: agent-feature

## 1. Intent
### Goal
Enhance agent.

## 2. Requirements
- R1: Agent runner shall work.

## 3. Technical Plan
### Artifacts
- modified: agent/runner.go
- modified: conductor/orchestrator.go
- modified: docs/decisions/ADR-001-custom.md
- modified: .pose/roadmaps/vision.md
- modified: .pose/docs.json

### Delivery targets
- capability:agent-runner module:agent profile:composed-capability entrypoint:agent/runner.go

### Technical risks
None.

## 4. Tasks
- [x] Done.

## 5. Validation
### Automated
- go test ./...

### Requirement trace
- R1 [satisfied] capability:agent-runner check:delivery-integration test:TestAgent evidence:integration

## 6. Delivery Evidence
### Artifact claims
- capability:agent-runner -> agent/runner.go

## 7. Final Report
Delivered.
`)

	graph := DeliveryIntegrityGraph{
		SchemaVersion:    1,
		ProvenanceDigest: "sha256:1111111111111111111111111111111111111111111111111111111111111111",
		ChangeSets: []ChangeSet{{
			ID: "cs-agent-feature", Spec: "agent-feature", Selector: "range:base..head",
			Base: "base", Head: "head", ResolvedBase: "base-resolved", ResolvedHead: "head-resolved",
			Paths: []ObservedPath{
				{Action: "modified", Path: "agent/runner.go"},
				{Action: "modified", Path: "conductor/orchestrator.go"},
				{Action: "modified", Path: "docs/decisions/ADR-001-custom.md"},
				{Action: "modified", Path: ".pose/roadmaps/vision.md"},
				{Action: "modified", Path: ".pose/docs.json"},
			},
			DiffDigest: "sha256:2222222222222222222222222222222222222222222222222222222222222222",
		}},
		Deliveries: []DeliveryTarget{{Spec: "agent-feature", Ref: "capability:agent-runner", Kind: "capability", ID: "agent-runner", Module: "agent", Profile: "composed-capability", Entrypoint: "agent/runner.go"}},
		ValidationResults: []DeliveryValidationResult{{
			ID: "val-1", Module: "agent", Check: "delivery-integration", EvidenceClass: "integration", Severity: "required", Outcome: "pass", GitHead: "head-resolved", ProvenanceDigest: "sha256:1111111111111111111111111111111111111111111111111111111111111111",
		}},
		Reverse: map[string][]string{
			"agent/runner.go":                  {"agent-feature"},
			"conductor/orchestrator.go":        {"agent-feature"},
			"docs/decisions/ADR-001-custom.md": {"agent-feature"},
			".pose/roadmaps/vision.md":         {"agent-feature"},
			".pose/docs.json":                  {"agent-feature"},
		},
	}
	raw, err := json.MarshalIndent(graph, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	writeReviewFixture(t, root, ".pose/indexes/delivery-integrity.json", string(raw))

	// Verify plan resolves the components
	plan, err := store.ReviewPlan("spec:agent-feature")
	if err != nil {
		t.Fatalf("ReviewPlan failed: %v", err)
	}
	if len(plan.Components) != 2 {
		t.Fatalf("expected 2 components resolved, got %d: %+v", len(plan.Components), plan.Components)
	}

	// Verify review bundle prepares and seals cleanly without unclassified paths
	bundle, err := store.PrepareReviewBundle("spec:agent-feature")
	if err != nil {
		t.Fatalf("PrepareReviewBundle failed: %v", err)
	}
	if len(bundle.Blockers) > 0 {
		t.Fatalf("unexpected blockers: %+v", bundle.Blockers)
	}

	sealed, err := store.SealReviewBundle("spec:agent-feature", time.Now().UTC())
	if err != nil {
		t.Fatalf("SealReviewBundle failed: %v", err)
	}
	if sealed.State != "sealed" {
		t.Fatalf("expected bundle state sealed, got %s", sealed.State)
	}
}

// An attestation is a claim that criteria were judged against a subject. Until
// now nothing tied the claim to the subject: `passed` could cite evidence the
// bundle never sealed, evidence of a class the criterion did not ask for, or
// nothing at all, and the attestation verified. Three closeouts in an adopting
// repository were approved that way — one against a bundle carrying zero
// evidence — and the gate reported them clean.
func TestAttestationCriterionMustCiteEvidenceTheBundleSeals(t *testing.T) {
	_, store := reviewBundleFixture(t)
	now := time.Date(2026, 9, 8, 12, 0, 0, 0, time.UTC)
	bundle, err := store.SealReviewBundle("spec:backend", now)
	if err != nil {
		t.Fatal(err)
	}
	if len(bundle.Payload.Evidence) == 0 {
		t.Fatal("the fixture must seal evidence for this test to mean anything")
	}

	supported := approvedBundleAttestation(bundle, "agent:did-the-work")
	supported.BundleDigest = bundle.BundleDigest
	supported.AttestedAt = now.Add(time.Minute).Format(time.RFC3339)
	if blockers := store.validateBundleAttestation(bundle, supported); len(blockers) != 0 {
		t.Fatalf("a supported attestation was rejected: %v", blockers)
	}

	for _, tc := range []struct {
		name     string
		evidence string
		want     string
	}{
		{"absent from the bundle", "integration:never-ran", "absent from the sealed bundle"},
		{"nothing at all", "", "passed with no evidence"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			att := approvedBundleAttestation(bundle, "agent:stamped")
			att.BundleDigest = bundle.BundleDigest
			att.AttestedAt = now.Add(time.Minute).Format(time.RFC3339)
			att.Criteria[0].Disposition = "passed"
			att.Criteria[0].Evidence = tc.evidence
			blockers := strings.Join(store.validateBundleAttestation(bundle, att), " ")
			if !strings.Contains(blockers, tc.want) {
				t.Fatalf("blockers = %q, want one mentioning %q", blockers, tc.want)
			}
		})
	}
}

func TestAttestationCriterionMustCiteEvidenceOfARequiredClass(t *testing.T) {
	// The failure that let `backend-integration-impact` pass on a `go vet`
	// result: the reference was real and in the bundle, but of a class the
	// criterion did not ask for.
	_, store := reviewBundleFixture(t)
	now := time.Date(2026, 9, 8, 12, 0, 0, 0, time.UTC)
	bundle, err := store.SealReviewBundle("spec:backend", now)
	if err != nil {
		t.Fatal(err)
	}
	var classed ReviewPlanCriterion
	for _, criterion := range bundle.Payload.Plan.Criteria {
		if criterion.Required && len(criterion.EvidenceClasses) > 0 {
			classed = criterion
			break
		}
	}
	if classed.ID == "" {
		t.Fatal("the fixture plan has no criterion demanding a class")
	}
	wrong := ""
	for _, ev := range bundle.Payload.Evidence {
		if !containsFold(classed.EvidenceClasses, ev.EvidenceClass) {
			wrong = ev.EvidenceClass + ":" + ev.ID
			break
		}
	}
	if wrong == "" {
		t.Fatalf("the fixture seals no evidence of a class %s does not ask for", classed.ID)
	}

	att := approvedBundleAttestation(bundle, "agent:wrong-class")
	att.BundleDigest = bundle.BundleDigest
	att.AttestedAt = now.Add(time.Minute).Format(time.RFC3339)
	for i := range att.Criteria {
		if att.Criteria[i].ID == classed.ID {
			att.Criteria[i].Disposition, att.Criteria[i].Evidence = "passed", wrong
		}
	}
	blockers := strings.Join(store.validateBundleAttestation(bundle, att), " ")
	if !strings.Contains(blockers, "requires evidence class") || !strings.Contains(blockers, classed.ID) {
		t.Fatalf("blockers = %q, want one naming %s and the required class", blockers, classed.ID)
	}
}

func TestAutoAttestRefusesRatherThanInventingAReference(t *testing.T) {
	// Auto-attest used to fabricate `<class>:auto-attest` whenever the bundle
	// sealed nothing of a required class. Nothing downstream checked the
	// reference, so the command produced a passed criterion supported by a
	// string. It must refuse instead, and name what is missing.
	root, store := reviewBundleFixture(t)
	graph, err := store.GetDeliveryIntegrity("")
	if err != nil {
		t.Fatal(err)
	}
	kept := []DeliveryValidationResult{}
	for _, result := range graph.ValidationResults {
		if result.EvidenceClass != "integration" {
			kept = append(kept, result)
		}
	}
	if len(kept) == len(graph.ValidationResults) {
		t.Fatal("the fixture seals no integration evidence to remove")
	}
	graph.ValidationResults = kept
	raw, _ := json.Marshal(graph)
	writeReviewFixture(t, root, ".pose/indexes/delivery-integrity.json", string(raw))

	now := time.Date(2026, 9, 8, 12, 0, 0, 0, time.UTC)
	sealed, err := store.SealReviewBundle("spec:backend", now)
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.AutoAttestReviewBundle(sealed.BundleID, "", false, now.Add(time.Minute))
	if err == nil {
		t.Fatal("auto-attest invented a reference for a class the bundle does not seal")
	}
	if !strings.Contains(err.Error(), "integration") || !strings.Contains(err.Error(), "not-applicable") {
		t.Fatalf("error = %q, want it to name the missing class and the way out", err)
	}
}

func TestReviewCriterionReuseIsInvalidatedByAnUnclassifiedRemoval(t *testing.T) {
	// A criterion that is not subject-sensitive sees only the documentation and
	// governance slice of the subject, so an unrelated implementation edit
	// legitimately leaves its digest alone and its prior verdict reusable. A
	// `removed` entry cannot be dismissed that way: the path carries no governed
	// classification and no content survives to prove it belonged to neither
	// category. It must reach every criterion, or the deletion is reviewed by
	// reusing a verdict issued before it existed.
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
	now := time.Date(2026, 9, 8, 12, 0, 0, 0, time.UTC)
	first, err := store.SealReviewBundle("spec:backend", now)
	if err != nil {
		t.Fatal(err)
	}
	prior, err := store.RecordReviewAttestation(approvedBundleAttestation(first, "agent:first-review"), now.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}

	graph, err := store.GetDeliveryIntegrity("")
	if err != nil {
		t.Fatal(err)
	}
	graph.ChangeSets[0].Paths = append(graph.ChangeSets[0].Paths, ObservedPath{Action: "removed", Path: ".agent-sync-cache.json"})
	raw, _ := json.Marshal(graph)
	writeReviewFixture(t, root, ".pose/indexes/delivery-integrity.json", string(raw))
	second, err := store.SealReviewBundle("spec:backend", now.Add(2*time.Minute))
	if err != nil {
		t.Fatal(err)
	}

	criterion := second.Payload.Plan.Criteria[0]
	priorCriterion := first.Payload.Plan.Criteria[0]
	found := false
	for i, candidate := range second.Payload.Plan.Criteria {
		if !reviewCriterionSubjectSensitive(candidate) {
			criterion, priorCriterion, found = candidate, first.Payload.Plan.Criteria[i], true
			break
		}
	}
	if !found {
		t.Fatalf("fixture has no reusable criterion: %+v", second.Payload.Plan.Criteria)
	}
	if reviewCriterionInputDigest(first, priorCriterion) == reviewCriterionInputDigest(second, criterion) {
		t.Fatalf("criterion %s digest survived an unclassified removal", criterion.ID)
	}

	attestation := approvedBundleAttestation(second, "agent:targeted-review")
	attestation.BundleDigest = second.BundleDigest
	attestation.AttestedAt = now.Add(3 * time.Minute).Format(time.RFC3339)
	attestation.ReusedFrom = []ReviewAttestationReuse{{Criterion: criterion.ID, FromAttestation: prior.AttestationID, InputDigest: reviewCriterionInputDigest(first, priorCriterion)}}
	if blockers := store.validateBundleAttestation(second, attestation); !strings.Contains(strings.Join(blockers, " "), "input digest changed") {
		t.Fatalf("a verdict issued before the deletion was reused over it: %v", blockers)
	}
}

func TestEvidenceSupportIsWaivedOnlyForApprovalsPredatingTheRule(t *testing.T) {
	// The exemption that lets 52 completed closeouts survive this engine change
	// must not become a way in. It is dated: an attestation recorded after the
	// reconciliation gets no waiver, and one recorded before gets it only while
	// the scope is done.
	_, store := reviewBundleFixture(t)
	now := time.Date(2026, 9, 8, 12, 0, 0, 0, time.UTC)
	bundle, err := store.SealReviewBundle("spec:backend", now)
	if err != nil {
		t.Fatal(err)
	}
	att := approvedBundleAttestation(bundle, "agent:unsupported")
	att.BundleDigest = bundle.BundleDigest
	att.AttestedAt = now.Format(time.RFC3339)
	att.Criteria[0].Disposition, att.Criteria[0].Evidence = "passed", "integration:never-ran"

	if blockers := store.validateBundleAttestationWith(bundle, att, false); len(blockers) == 0 {
		t.Fatal("an unsupported criterion passed without the waiver")
	}
	waived := store.validateBundleAttestationWith(bundle, att, true)
	for _, blocker := range waived {
		if strings.Contains(blocker, "absent from the sealed bundle") || strings.Contains(blocker, "requires evidence class") {
			t.Fatalf("the waiver did not cover the evidence-support blocker: %v", waived)
		}
	}

	// It waives only that. A malformed attestation is still rejected.
	broken := att
	broken.Criteria = append([]ReviewCriterion{}, att.Criteria...)
	broken.Criteria[0].Disposition = "definitely-fine"
	if blockers := store.validateBundleAttestationWith(bundle, broken, true); len(blockers) == 0 {
		t.Fatal("the waiver suppressed unrelated validation")
	}
}

func TestToolDispositionMustCiteEvidenceTheBundleSeals(t *testing.T) {
	// The criteria half of this was fixed one spec ago; the tool half kept the
	// same hole. A disposition could name a class the tool asks for and an id
	// that appears nowhere, and the attestation verified.
	root, store := reviewBundleFixture(t)
	// The shipped profile's `validate` declares evidence classes; the fixture's
	// does not, and the presence check only applies to a tool that demands one.
	// Seeding it here is what makes this test exercise the rule rather than the
	// absence of a demand.
	profilePath := filepath.Join(root, ".pose/review-profiles/spec-closeout.json")
	profileRaw, _ := os.ReadFile(profilePath)
	profile := strings.Replace(string(profileRaw), `"tools":[`, `"tools":[{"id":"validate","requiredness":"required","evidence_classes":["integration"],"criteria":["correctness"]},`, 1)
	if err := os.WriteFile(profilePath, []byte(profile), 0o644); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 9, 8, 12, 0, 0, 0, time.UTC)
	bundle, err := store.SealReviewBundle("spec:backend", now)
	if err != nil {
		t.Fatal(err)
	}
	var classed ReviewPlanTool
	for _, tool := range bundle.Payload.Plan.Tools {
		if len(tool.EvidenceClasses) > 0 {
			classed = tool
			break
		}
	}
	if classed.ID == "" {
		t.Fatal("the fixture plan has no tool demanding a class")
	}

	att := approvedBundleAttestation(bundle, "agent:did-the-work")
	att.BundleDigest = bundle.BundleDigest
	att.AttestedAt = now.Add(time.Minute).Format(time.RFC3339)
	if blockers := store.validateBundleAttestation(bundle, att); len(blockers) != 0 {
		t.Fatalf("a supported attestation was rejected: %v", blockers)
	}

	for i := range att.Tools {
		if att.Tools[i].ID == classed.ID && att.Tools[i].Component == classed.Component {
			// Right class, and an id the bundle does not carry.
			att.Tools[i].Disposition = "passed"
			att.Tools[i].Evidence = classed.EvidenceClasses[0] + ":never-ran"
		}
	}
	blockers := strings.Join(store.validateBundleAttestation(bundle, att), " ")
	if !strings.Contains(blockers, "absent from the sealed bundle") {
		t.Fatalf("blockers = %q, want one naming the absent tool evidence", blockers)
	}
}

func TestAutoAttestDoesNotInventToolEvidence(t *testing.T) {
	// `validation:auto-attest`, `<class>:auto-attest`, `docs:auto-attest`: the
	// same invention the criteria path stopped doing, on the other half of the
	// attestation.
	//
	// The demand is `reachability`, which no criterion in this fixture asks for
	// and the bundle does not seal — so the refusal can only come from the tool
	// half. Asserting it on a class a criterion also demands would pass whether
	// or not the tool path was fixed.
	root, store := reviewBundleFixture(t)
	profilePath := filepath.Join(root, ".pose/review-profiles/spec-closeout.json")
	profileRaw, _ := os.ReadFile(profilePath)
	profile := strings.Replace(string(profileRaw), `"tools":[`, `"tools":[{"id":"validate","requiredness":"required","evidence_classes":["reachability"],"criteria":["correctness"]},`, 1)
	if err := os.WriteFile(profilePath, []byte(profile), 0o644); err != nil {
		t.Fatal(err)
	}

	now := time.Date(2026, 9, 8, 12, 0, 0, 0, time.UTC)
	sealed, err := store.SealReviewBundle("spec:backend", now)
	if err != nil {
		t.Fatal(err)
	}
	for _, ev := range sealed.Payload.Evidence {
		if ev.EvidenceClass == "reachability" {
			t.Fatal("the fixture seals reachability, so this test cannot show a refusal")
		}
	}
	_, err = store.AutoAttestReviewBundle(sealed.BundleID, "", false, now.Add(time.Minute))
	if err == nil {
		t.Fatal("auto-attest invented tool evidence for a class the bundle does not seal")
	}
	if !strings.Contains(err.Error(), "review tool") || !strings.Contains(err.Error(), "reachability") {
		t.Fatalf("error = %q, want it to name the tool and the missing class", err)
	}
}
