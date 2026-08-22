---
slug: pose-review-bundle-root-manifests-classification
status: done
created_at: 2026-08-22
supersedes:
depends_on: pose-lint-spec-flat-layout-followup-resolution
priority: 30
components: cli, pose-mcp
delivers: contract:review-bundle-root-manifests
completed_at: 2026-08-22
---

# Spec: Review Bundle Root Manifests and Project Configuration Classification

## 1. Intent

### Goal
Ensure `pose review bundle` properly classifies root-level project manifests, lockfiles, toolchain configs, documentation files, and single-module root components without raising unclassified path blockers:
1. Classify standard project manifests and toolchain configurations (`go.mod`, `go.sum`, `package.json`, `Cargo.toml`, `pyproject.toml`, `tsconfig.json`, `.gitignore`, `Makefile`, etc.) as `governance`.
2. Classify any root documentation file (`PROJECT.md`, `CLAUDE.md`, `CONTRIBUTING.md`, `LICENSE`, etc.) as `documentation`.
3. Allow components rooted at `.` or `""` to classify their direct implementation files rather than skipping root components.
4. Support standard code directories (`cmd/`, `internal/`, `pkg/`, `src/`, `lib/`, `app/`, `api/`) as `implementation`.

### Business value
Unblocks single-module repositories across Go, Rust, Node, Python, and other ecosystems from sealing review bundles and closing specs when modifying root-level manifests.

### Constraints
- Retain fail-closed blocker behavior on truly unclassified files (e.g. `mystery.data`).
- Full automated test coverage and backward compatibility.

---

## 2. Requirements

### Functional
- R1: `reviewBundlePathClass` shall classify standard project manifests and toolchain configs as `governance`.
- R2: `reviewBundlePathClass` shall classify root `.md` and `.txt` documents and license files as `documentation`.
- R3: `reviewBundlePathClass` shall support root components (`Path == "."` or `Path == ""`) matching direct implementation files.
- R4: `reviewBundlePathClass` shall recognize common implementation source directories (`cmd/`, `internal/`, `pkg/`, `src/`, `lib/`, `app/`, `api/`).

### Non-functional
- Complete regression coverage in `internal/pose` and `internal/cli`.
- Zero regressions in existing suite.

---

## 3. Technical Plan

### Affected areas
- `pose-mcp/internal/pose/review_bundle.go`
- `pose-mcp/internal/pose/review_bundle_test.go`
- `pose-mcp/internal/cli/review_closeout_test.go`

### Delivery targets
- contract:review-bundle-root-manifests module:pose-mcp profile:api-contract entrypoint:pose-mcp/cmd/pose/main.go

---

## 4. Tasks

### Artifacts
- modified: pose-mcp/internal/pose/review_bundle.go
- modified: pose-mcp/internal/pose/review_bundle_test.go
- modified: pose-mcp/internal/cli/review_closeout_test.go

### Increments
- [x] Increment 1: Add manifest and root documentation patterns to `reviewBundlePathClass` (R1, R2).
- [x] Increment 2: Support root components and common source directory prefixes (R3, R4).
- [x] Increment 3: Add unit tests in `review_bundle_test.go` and E2E test in `review_closeout_test.go`.

---

## 5. Validation

### Automated
- `TMPDIR=/home/go/.cache/tmp go test ./internal/pose -run TestReviewBundleClassifiesRoot -v`
- `TMPDIR=/home/go/.cache/tmp go test ./internal/cli -run TestReviewBundleSealSingleModuleRootFilesAndManifests -v`
- `TMPDIR=/home/go/.cache/tmp go test ./...`
- `pose validate --strict`
- `pose check --strict`

### Requirement trace
- R1 [satisfied] contract:review-bundle-root-manifests check:root-classification test:TestReviewBundleClassifiesRootManifestsAndProjectFiles evidence:integration
- R2 [satisfied] contract:review-bundle-root-manifests check:root-classification test:TestReviewBundleClassifiesRootManifestsAndProjectFiles evidence:integration
- R3 [satisfied] contract:review-bundle-root-manifests check:root-classification test:TestReviewBundleSealSingleModuleRootFilesAndManifests evidence:integration
- R4 [satisfied] contract:review-bundle-root-manifests check:root-classification test:TestReviewBundleClassifiesRootManifestsAndProjectFiles evidence:integration

---

## 6. Delivery Evidence

### Artifact claims
- contract:review-bundle-root-manifests -> pose-mcp/cmd/pose/main.go

### Known gaps
None.

---

## 7. Final Report

### Delivered scope
- Classified root-level manifests, project docs, and root components in review bundles.
- Resolved false-positive blocker for single-module projects.

### Follow-ups
- [done] All requirements verified and delivered.
