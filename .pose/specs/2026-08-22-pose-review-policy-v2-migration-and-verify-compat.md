---
slug: pose-review-policy-v2-migration-and-verify-compat
status: done
created_at: 2026-08-22
supersedes:
depends_on: pose-new-spec-default-flat-format
priority: 25
components: cli, pose, review
delivers: surface:review-policy-v2-migration
completed_at: 2026-08-22
---

# Spec: Review Policy V2 Migration, Component Discovery Fallback, and Review Verify Compatibility

## 1. Intent

### Goal
Resolve Issue #34:
1. Ensure `pose update` idempotently migrates managed review policy (`.pose/policy/review.json`) and review profiles from schema v1 to schema v2 (`component_aware: true`, `review_bundles: true`, typed overlay profiles).
2. Ensure engine component discovery remains active for path classification during review bundle preparation even when a repository is on a schema v1 policy.
3. Ensure `pose review verify` honors `review_bundles=false` and recorded reviews (`pose review record`) consistently with `pose review-check` and `pose close`.

### Business value
Prevents legitimate subproject paths from being rejected as unclassified on older repositories or when review bundles are disabled, and ensures smooth, automated upgrades to schema-v2 review policies.

### Constraints
- Zero breaking changes to existing review commands.
- Full parity across English and Portuguese.

---

## 2. Requirements

### Functional
- R1: `cmdUpdate` and `seedAbsentInstanceConfig` shall idempotently upgrade `.pose/policy/review.json` from schema_version 1 to 2, enabling `component_aware` and `review_bundles`.
- R2: `cmdUpdate --dry-run` shall report when a review policy migration from v1 to v2 is pending.
- R3: `reviewBundleSubject` shall resolve discovered repository components when `plan.Components` is empty, ensuring legitimate subproject paths are classified as `implementation`.
- R4: `VerifyReviewBundle` shall check recorded reviews via `ReviewCheck` when `policy.ReviewBundles == false` and no sealed bundles exist, reporting state `ready-to-close` / `closed` appropriately.

### Non-functional
- Complete test coverage in `internal/cli` and `internal/pose`.
- Zero check/validation warnings or errors.

---

## 3. Technical Plan

### Affected areas
- `pose-mcp/internal/cli/maintenance.go`
- `pose-mcp/internal/cli/stack_seed.go`
- `pose-mcp/internal/pose/review_bundle.go`
- `pose-mcp/internal/cli/review_closeout_test.go`

### Delivery targets
- surface:review-policy-v2-migration module:pose-mcp profile:cli-surface entrypoint:pose-mcp/cmd/pose/main.go

---

## 4. Tasks

### Artifacts
- modified: pose-mcp/internal/cli/maintenance.go
- modified: pose-mcp/internal/cli/stack_seed.go
- modified: pose-mcp/internal/pose/review_bundle.go
- modified: pose-mcp/internal/cli/review_closeout_test.go

### Increments
- [x] Increment 1: Keep component discovery active in `reviewBundleSubject` (R3).
- [x] Increment 2: Support `review_bundles=false` and recorded reviews in `VerifyReviewBundle` (R4).
- [x] Increment 3: Add `migrateInstanceReviewPolicy` to `cmdUpdate` / `seedAbsentInstanceConfig` with dry-run support (R1, R2).
- [x] Increment 4: Add regression tests for policy migration and recorded review verification (R1, R4).

---

## 5. Validation

### Automated
- `go test ./internal/pose ./internal/cli ./internal/mcpserver -count=1`
- `pose check --strict`
- `pose validate --strict`

### Requirement trace
- R1 [satisfied] surface:review-policy-v2-migration check:delivery-integration test:TestUpdateMigratesReviewPolicySchemaV1ToV2 evidence:integration
- R2 [satisfied] surface:review-policy-v2-migration check:delivery-integration test:TestUpdateMigratesReviewPolicySchemaV1ToV2 evidence:integration
- R3 [satisfied] surface:review-policy-v2-migration check:delivery-integration test:TestReviewBundleSubjectClassification evidence:integration
- R4 [satisfied] surface:review-policy-v2-migration check:delivery-integration test:TestPoseCloseWithLiveGitTrailerNoReport evidence:integration

---

## 6. Delivery Evidence

### Artifact claims
- surface:review-policy-v2-migration -> pose-mcp/cmd/pose/main.go

### Known gaps
None.

---

## 7. Final Report

### Delivered scope
- Implemented `migrateInstanceReviewPolicy` in `cmdUpdate` and `seedAbsentInstanceConfig` to upgrade schema v1 review policies to v2.
- Added dry-run notification for pending review policy migrations.
- Kept component discovery active in `reviewBundleSubject` for path classification on v1 policies.
- Updated `VerifyReviewBundle` to evaluate recorded reviews via `ReviewCheck` when bundles are disabled.
- Added comprehensive unit tests and verified strict compliance.

### Follow-ups
- [done] All requirements delivered and verified.
