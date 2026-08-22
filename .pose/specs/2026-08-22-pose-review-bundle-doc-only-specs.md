---
slug: pose-review-bundle-doc-only-specs
status: done
created_at: 2026-08-22
supersedes:
depends_on: pose-review-overlay-uninstalled-rules-graceful-degradation
priority: 28
components: cli, pose-mcp
delivers: contract:review-bundle-doc-only-specs
completed_at: 2026-08-22
---

# Spec: Review Bundle and Seal Support for Doc-Only and Governance Specs

## 1. Intent

### Goal
Ensure `pose review bundle --explain`, `pose review bundle --seal`, `pose review auto-attest`, and `pose close` succeed seamlessly for documentation-only, governance, and ADR specs that have no code delivery targets:
1. Persist the updated delivery-integrity index when `pose artifact-check` reconciles a spec's change set and ensure `pose review bundle` dynamically creates `.pose/indexes/delivery-integrity.json` if absent.
2. Add precondition `delivery-target-declared` to the `validate` tool in the base review profile `spec-closeout.json` and in `ReviewPlan` tool construction, skipping required validation evidence when a spec has no code modules or delivery targets.
3. Allow `AutoAttestReviewBundle` to generate valid attestations for scopes without structured validation evidence.

### Business value
Unblocks initial repository setup, governance milestones, ADR reviews, and pure documentation deliverables in POSE without requiring dummy code modules or failing on missing validation evidence.

### Constraints
- Enforce validation evidence requirement whenever a spec has code components or declared delivery targets.
- Retain strict integrity and review verification gates across all commands.

---

## 2. Requirements

### Functional
- R1: `cmdArtifactCheck` and `cmdReviewBundle` shall persist and ensure `.pose/indexes/delivery-integrity.json` exists so subsequent review bundle operations resolve change sets cleanly.
- R2: `validate` tool in `spec-closeout.json` and `ReviewPlan` shall require precondition `delivery-target-declared` and be skipped when no components or delivery targets exist.
- R3: `PrepareReviewBundle` shall require structured validation evidence only when the scope actually declares deliveries, contains code components, or declares unconditioned required validation tools.
- R4: `AutoAttestReviewBundle` shall automatically attest and approve doc-only specs without requiring executed code test results.

### Non-functional
- Complete automated regression tests in `internal/cli` and `internal/pose`.
- Zero regressions in existing suite.

---

## 3. Technical Plan

### Affected areas
- `.pose/review-profiles/spec-closeout.json`
- `pose-mcp/internal/pose/review_plan.go`
- `pose-mcp/internal/pose/review_bundle.go`
- `pose-mcp/internal/cli/artifact_integrity.go`
- `pose-mcp/internal/cli/review_closeout.go`
- `pose-mcp/internal/cli/review_closeout_test.go`

### Delivery targets
- contract:review-bundle-doc-only-specs module:pose-mcp profile:api-contract entrypoint:pose-mcp/cmd/pose/main.go

---

## 4. Tasks

### Artifacts
- modified: .pose/review-profiles/spec-closeout.json
- modified: pose-mcp/internal/pose/review_plan.go
- modified: pose-mcp/internal/pose/review_bundle.go
- modified: pose-mcp/internal/cli/artifact_integrity.go
- modified: pose-mcp/internal/cli/review_closeout.go
- modified: pose-mcp/internal/cli/review_closeout_test.go

### Increments
- [x] Increment 1: Add `delivery-target-declared` precondition to `validate` tool and conditional evidence checks (R2, R3).
- [x] Increment 2: Update `AutoAttestReviewBundle` for doc-only scopes (R4).
- [x] Increment 3: Persist delivery integrity index in `cmdArtifactCheck` and `cmdReviewBundle` (R1).
- [x] Increment 4: Add regression test covering full lifecycle from `artifact-check` to `pose close` for doc-only spec with no delivery targets.

---

## 5. Validation

### Automated
- `TMPDIR=/home/go/.cache/tmp go test ./internal/cli -run TestReviewBundleSealAndCloseoutForDocOnlySpecWithNoDeliveryTargets -v`
- `TMPDIR=/home/go/.cache/tmp go test ./...`
- `pose validate --strict`
- `pose check --strict`

### Requirement trace
- R1 [satisfied] contract:review-bundle-doc-only-specs check:delivery-integration test:TestReviewBundleSealAndCloseoutForDocOnlySpecWithNoDeliveryTargets evidence:integration
- R2 [satisfied] contract:review-bundle-doc-only-specs check:delivery-integration test:TestReviewBundleSealAndCloseoutForDocOnlySpecWithNoDeliveryTargets evidence:integration
- R3 [satisfied] contract:review-bundle-doc-only-specs check:delivery-integration test:TestReviewBundleSealAndCloseoutForDocOnlySpecWithNoDeliveryTargets evidence:integration
- R4 [satisfied] contract:review-bundle-doc-only-specs check:delivery-integration test:TestReviewBundleSealAndCloseoutForDocOnlySpecWithNoDeliveryTargets evidence:integration

---

## 6. Delivery Evidence

### Artifact claims
- contract:review-bundle-doc-only-specs -> pose-mcp/cmd/pose/main.go

### Known gaps
None.

---

## 7. Final Report

### Delivered scope
- Supported review bundle sealing, auto-attestation, and spec closeout for doc-only and governance specs.
- Fixed change set persistence between artifact-check and review bundle.
- Preconditioned validate tool on delivery targets.

### Follow-ups
- [done] All requirements verified and delivered.
