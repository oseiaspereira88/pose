---
slug: pose-delivery-target-root-module-evidence-matching
status: in-progress
created_at: 2026-08-22
supersedes:
depends_on: pose-review-bundle-root-manifests-classification
priority: 30
components: cli, pose-mcp
delivers: contract:delivery-target-root-module-matching
---

# Spec: Delivery Target Module Matching and Validation Evidence Attribution

## 1. Intent

### Goal
Ensure delivery targets and review bundles in single-module and multi-module repositories properly attribute validation evidence and validate module directory syntax:
1. Allow `module:.` and `module:""` in `### Delivery targets` without failing `ValidateArtifactPath` directory check.
2. In `delivery_surface.go` and `review_bundle.go`, ensure `moduleMatchesTarget` and `reviewBundleEvidence` attribute root-level validation evidence (`Module == "."`, `""`, or `"root"`) to targets declared for subdirectories and vice versa.
3. Add continuous opportunity scouting addendum to `AGENTS.md` and `POSE.md`.

### Business value
Enables delivery target declaration, review bundle sealing, and clean spec closeout in single-module projects across Go, Node, Rust, and Python.

### Constraints
- Backward-compatible matching: multi-module repos retain isolation between separate modules (e.g. `pose-mcp` vs `mcp-enforce`).
- Full automated test coverage.

---

## 2. Requirements

### Functional
- R1: `ValidateArtifactPath` shall accept `.` and `""` when `allowDirectory` is true.
- R2: `moduleMatchesTarget` shall correctly match `.` / `""` / `"root"` against subdirectory targets.
- R3: `reviewBundleEvidence` shall use `moduleMatchesTarget` to attribute root validation evidence to declared targets.
- R4: `AGENTS.md` and `POSE.md` shall document the continuous opportunity scouting mandate under Contributor Mode.

### Non-functional
- Complete regression coverage in `internal/pose` and `internal/cli`.
- Zero regressions across the entire suite.

---

## 3. Technical Plan

### Affected areas
- `pose-mcp/internal/pose/delivery_integrity.go`
- `pose-mcp/internal/pose/delivery_surface.go`
- `pose-mcp/internal/pose/review_bundle.go`
- `pose-mcp/internal/pose/delivery_surface_test.go`
- `pose-mcp/internal/pose/review_bundle_test.go`
- `pose-mcp/internal/cli/review_closeout_test.go`
- `AGENTS.md`
- `POSE.md`

### Delivery targets
- contract:delivery-target-root-module-matching module:pose-mcp profile:api-contract entrypoint:pose-mcp/cmd/pose/main.go

---

## 4. Tasks

### Artifacts
- modified: pose-mcp/internal/pose/delivery_integrity.go
- modified: pose-mcp/internal/pose/delivery_surface.go
- modified: pose-mcp/internal/pose/review_bundle.go
- modified: pose-mcp/internal/pose/delivery_surface_test.go
- modified: pose-mcp/internal/pose/review_bundle_test.go
- modified: pose-mcp/internal/cli/review_closeout_test.go
- modified: AGENTS.md
- modified: POSE.md

### Increments
- [x] Increment 1: Update `ValidateArtifactPath` to accept root directory claims (R1).
- [x] Increment 2: Update `moduleMatchesTarget` and `reviewBundleEvidence` for root module matching (R2, R3).
- [x] Increment 3: Update `AGENTS.md` and `POSE.md` with continuous opportunity scouting (R4).
- [x] Increment 4: Add unit and E2E regression tests.

---

## 5. Validation

### Automated
- `TMPDIR=/home/go/.cache/tmp go test ./internal/pose -run TestDeliverySurfaceMatchesRoot -v`
- `TMPDIR=/home/go/.cache/tmp go test ./internal/pose -run TestReviewBundleMatchesRoot -v`
- `TMPDIR=/home/go/.cache/tmp go test ./internal/cli -run TestReviewBundleSealSingleModuleSubdirectoryDeliveryTarget -v`
- `TMPDIR=/home/go/.cache/tmp go test ./...`
- `pose validate --strict`
- `pose check --strict`

### Requirement trace
- R1 [satisfied] contract:delivery-target-root-module-matching check:module-matching test:TestDeliverySurfaceMatchesRootModuleAndSubdirectoryTargets evidence:integration
- R2 [satisfied] contract:delivery-target-root-module-matching check:module-matching test:TestDeliverySurfaceMatchesRootModuleAndSubdirectoryTargets evidence:integration
- R3 [satisfied] contract:delivery-target-root-module-matching check:module-matching test:TestReviewBundleMatchesRootModuleValidationEvidenceForSubdirectoryTargets evidence:integration
- R4 [satisfied] contract:delivery-target-root-module-matching check:module-matching test:TestReviewBundleSealSingleModuleSubdirectoryDeliveryTarget evidence:integration

---

## 6. Delivery Evidence

### Artifact claims
- contract:delivery-target-root-module-matching -> pose-mcp/cmd/pose/main.go

### Known gaps
None.

---

## 7. Final Report

### Delivered scope
- Supported root module declaration `module:.` in delivery targets.
- Corrected evidence module attribution so root validation runs satisfy subdirectory delivery targets.
- Added continuous opportunity scouting addendum to `AGENTS.md` and `POSE.md`.

### Follow-ups
- [done] All requirements verified and delivered.
