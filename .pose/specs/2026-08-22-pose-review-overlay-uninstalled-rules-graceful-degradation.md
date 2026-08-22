---
slug: pose-review-overlay-uninstalled-rules-graceful-degradation
status: in-progress
created_at: 2026-08-22
supersedes:
depends_on: pose-skills-harmonization
priority: 27
components: cli, pose-mcp
delivers: contract:review-overlay-uninstalled-rules-graceful-degradation
---

# Spec: Graceful Degradation for Review Overlay Profiles Referencing Uninstalled Extension Rules

## 1. Intent

### Goal
Prevent `pose review bundle --explain` and `pose review bundle --seal` from failing with hard errors when overlay review profiles reference rule IDs provided by uninstalled extensions (such as `backend-go` or `frontend-react`):
1. In `validateReviewContractRefs`, validate rule identifier syntax and directory traversal safety without requiring local presence of rule files on disk during profile loading.
2. In `ReviewPlan`, skip unmatched overlay profiles without blocking or warning, and degrade uninstalled rules in matched profiles to warnings in `plan.Warnings` instead of hard blockers in `plan.Blockers`.
3. Allow bundle sealing, review attestation, and spec closeout to proceed unimpeded on projects without all optional rule extensions installed.

### Business value
Eliminates blocking failures on fresh projects (`pose init` on empty repositories or repositories without all language extensions) where default overlay profiles reference extension rules, restoring smooth review bundle workflows.

### Constraints
- Retain syntax validation for rule slugs and strict safety validation against directory escapes.
- Retain closed evidence class catalog validation.
- Zero regressions in existing review bundle and plan test suites.

---

## 2. Requirements

### Functional
- R1: `validateReviewContractRefs` shall validate rule slug format and safety without requiring `.pose/rules/<rule>.md` to exist on disk.
- R2: Overlay profiles whose selectors do not match the review scope context shall be skipped without evaluating or blocking on missing rules.
- R3: Overlay profiles whose selectors match the review scope context shall compose criteria normally and record uninstalled rules as warnings in `plan.Warnings`.
- R4: Review bundle preparation and sealing shall succeed when matched criteria reference uninstalled extension rules.

### Non-functional
- Deterministic regression test coverage in `internal/pose`.
- Strict validation pass across all modules.

---

## 3. Technical Plan

### Affected areas
- `pose-mcp/internal/pose/review_closeout.go`
- `pose-mcp/internal/pose/review_plan.go`
- `pose-mcp/internal/pose/review_plan_test.go`

### Delivery targets
- contract:review-overlay-uninstalled-rules-graceful-degradation module:pose-mcp profile:api-contract entrypoint:pose-mcp/cmd/pose/main.go

---

## 4. Tasks

### Artifacts
- modified: pose-mcp/internal/pose/review_closeout.go
- modified: pose-mcp/internal/pose/review_plan.go
- modified: pose-mcp/internal/pose/review_plan_test.go

### Increments
- [x] Increment 1: Update `validateReviewContractRefs` to validate syntax and path safety without requiring local rule presence (R1).
- [x] Increment 2: Add uninstalled rule inspection in `ReviewPlan` composing criteria and emitting plan warnings (R2, R3, R4).
- [x] Increment 3: Add unit and regression tests in `review_plan_test.go` covering unmatched and matched overlay profiles with uninstalled extension rules.

---

## 5. Validation

### Automated
- `TMPDIR=/home/go/.cache/tmp go test ./internal/pose ./internal/cli ./internal/scaffold -v`
- `pose validate --strict`
- `pose check --strict`

### Requirement trace
- R1 [satisfied] contract:review-overlay-uninstalled-rules-graceful-degradation check:delivery-integration test:TestReviewPlanOverlayProfilesWithUninstalledExtensionRulesDegradeToWarnings evidence:integration
- R2 [satisfied] contract:review-overlay-uninstalled-rules-graceful-degradation check:delivery-integration test:TestReviewPlanOverlayProfilesWithUninstalledExtensionRulesDegradeToWarnings evidence:integration
- R3 [satisfied] contract:review-overlay-uninstalled-rules-graceful-degradation check:delivery-integration test:TestReviewPlanOverlayProfilesWithUninstalledExtensionRulesDegradeToWarnings evidence:integration
- R4 [satisfied] contract:review-overlay-uninstalled-rules-graceful-degradation check:delivery-integration test:TestReviewPlanOverlayProfilesWithUninstalledExtensionRulesDegradeToWarnings evidence:integration

---

## 6. Delivery Evidence

### Artifact claims
- contract:review-overlay-uninstalled-rules-graceful-degradation -> pose-mcp/cmd/pose/main.go

### Known gaps
None.

---

## 7. Final Report

### Delivered scope
- Enabled graceful degradation for uninstalled rule extensions in review profiles.
- Verified review bundle planning and sealing on projects with uninstalled extension rules.

### Follow-ups
- [done] All requirements delivered and verified.
