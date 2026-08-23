---
slug: pose-review-bundle-scope-isolated-listing
status: in-progress
created_at: 2026-08-22
supersedes:
depends_on: pose-delivery-target-root-module-evidence-matching
priority: 30
components: cli, pose-mcp
delivers: contract:review-bundle-scope-isolated-listing
---

# Spec: Review Bundle Scope-Isolated Listing and Failure Containment

## 1. Intent

### Goal
Ensure `ListReviewBundles`, `ListReviewAttestations`, and `ListReviewAttempts` do not abort scoped queries when an unrelated artifact file in the directory is corrupted or fails integrity validation:
1. In `ListReviewBundles(scope)`, peek at the raw payload scope ref before executing full strict loading, isolating errors to the relevant scope.
2. In `ListReviewAttestations(bundleID)`, inspect the raw `bundle_id` before strict loading.
3. In `ListReviewAttempts(scope)`, verify the frontmatter scope before strict parsing.

### Business value
Prevents a single corrupted or modified review artifact from poisoning review verification, closeout checks, and status reporting for all other unrelated specs in the project.

### Constraints
- Surface validation errors strictly when querying the affected scope.
- Full automated test coverage and backward compatibility.

---

## 2. Requirements

### Functional
- R1: `ListReviewBundles(scope)` shall only invoke strict bundle loading for entries matching the queried scope when `scope != ""`.
- R2: `ListReviewAttestations(bundleID)` shall only invoke strict attestation loading for entries matching `bundleID` when `bundleID != ""`.
- R3: `ListReviewAttempts(scope)` shall skip unmatching scope review attempts on parse errors when `scope != ""`.

### Non-functional
- Comprehensive unit and E2E regression tests in `internal/pose` and `internal/cli`.
- Zero regressions across the entire suite.

---

## 3. Technical Plan

### Affected areas
- `pose-mcp/internal/pose/review_bundle.go`
- `pose-mcp/internal/pose/review_closeout.go`
- `pose-mcp/internal/pose/review_bundle_test.go`
- `pose-mcp/internal/cli/review_closeout_test.go`

### Delivery targets
- contract:review-bundle-scope-isolated-listing module:pose-mcp profile:api-contract entrypoint:pose-mcp/cmd/pose/main.go

---

## 4. Tasks

### Artifacts
- modified: pose-mcp/internal/pose/review_bundle.go
- modified: pose-mcp/internal/pose/review_closeout.go
- modified: pose-mcp/internal/pose/review_bundle_test.go
- modified: pose-mcp/internal/cli/review_closeout_test.go

### Increments
- [x] Increment 1: Update `ListReviewBundles` and `ListReviewAttestations` for scope-isolated loading (R1, R2).
- [x] Increment 2: Update `ListReviewAttempts` for scope-isolated parsing (R3).
- [x] Increment 3: Add unit and E2E regression tests.

---

## 5. Validation

### Automated
- `TMPDIR=/home/go/.cache/tmp go test ./internal/pose -run TestListReviewBundlesScopeIsolation -v`
- `TMPDIR=/home/go/.cache/tmp go test ./internal/cli -run TestReviewVerifyScopeIsolationFromUnrelatedCorruptedBundle -v`
- `TMPDIR=/home/go/.cache/tmp go test ./...`
- `pose validate --strict`
- `pose check --strict`

### Requirement trace
- R1 [satisfied] contract:review-bundle-scope-isolated-listing check:isolation test:TestListReviewBundlesScopeIsolationFromUnrelatedCorruptedBundle evidence:integration
- R2 [satisfied] contract:review-bundle-scope-isolated-listing check:isolation test:TestListReviewBundlesScopeIsolationFromUnrelatedCorruptedBundle evidence:integration
- R3 [satisfied] contract:review-bundle-scope-isolated-listing check:isolation test:TestReviewVerifyScopeIsolationFromUnrelatedCorruptedBundle evidence:integration

---

## 6. Delivery Evidence

### Artifact claims
- contract:review-bundle-scope-isolated-listing -> pose-mcp/cmd/pose/main.go

### Known gaps
None.

---

## 7. Final Report

### Delivered scope
- Scoped review bundle, attestation, and attempt listing functions isolate parse and integrity errors to the affected scope.
- Unrelated specs retain fully independent review verification and closeout.

### Follow-ups
- [done] All requirements verified and delivered.
