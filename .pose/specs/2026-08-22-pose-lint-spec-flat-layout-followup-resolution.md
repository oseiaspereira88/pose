---
slug: pose-lint-spec-flat-layout-followup-resolution
status: in-progress
created_at: 2026-08-22
supersedes:
depends_on: pose-review-bundle-doc-only-specs
priority: 29
components: cli, pose-mcp
delivers: contract:lint-spec-flat-layout-resolution
---

# Spec: lint-spec Flat Layout Followup and Sibling Spec Resolution

## 1. Intent

### Goal
Ensure `pose lint-spec` properly discovers and resolves spec slugs across both flat layout (`.pose/specs/<date>-<slug>.md`, `.pose/specs/<slug>.md`) and nested directory layout (`.pose/specs/<slug>/spec.md`, `.pose/specs/<date>-<slug>/spec.md`):
1. Fix `collectSpecSlugs` and `siblingSpecStatus` so that followup dispositions (`[covered: <slug>]`, `[spawned: <slug>]`, `[duplicate: <slug>]`) and `depends_on:` checks correctly recognize target specs in flat dated files.
2. Fix `specsDir` calculation in `lintSpecFile` so that checking a flat spec file does not navigate two directory levels up to `.pose`.
3. Update `cmdLintSpec --all` to iterate over all specs discovered via `Store.ListSpecs`.
4. Update `repointSpecClaimsOnFragmentArchive` in release lifecycle to support flat spec files.

### Business value
Prevents false-positive `points to a missing spec` errors when closing specs that reference modern flat dated specs produced by default by `pose new-spec`.

### Constraints
- Zero breaking changes to existing nested directory specs.
- Retain strict linting and lifecycle verification.

---

## 2. Requirements

### Functional
- R1: `collectSpecSlugs` and `siblingSpecStatus` shall discover specs from both flat dated files and nested directories.
- R2: `lintSpecFile` shall calculate `specsDir` safely for both flat files and nested directory structures.
- R3: `cmdLintSpec --all` shall lint all specs regardless of layout.
- R4: `repointSpecClaimsOnFragmentArchive` shall process claims in flat spec files.

### Non-functional
- Full automated test coverage in `internal/cli`.
- Zero regressions in existing suite.

---

## 3. Technical Plan

### Affected areas
- `pose-mcp/internal/cli/lintspec.go`
- `pose-mcp/internal/cli/release_lifecycle.go`
- `pose-mcp/internal/cli/trace_lint_test.go`

### Delivery targets
- contract:lint-spec-flat-layout-resolution module:pose-mcp profile:api-contract entrypoint:pose-mcp/cmd/pose/main.go

---

## 4. Tasks

### Artifacts
- modified: pose-mcp/internal/cli/lintspec.go
- modified: pose-mcp/internal/cli/release_lifecycle.go
- modified: pose-mcp/internal/cli/trace_lint_test.go

### Increments
- [x] Increment 1: Update `collectSpecSlugs` and `siblingSpecStatus` for flat and nested layouts (R1).
- [x] Increment 2: Update `specsDir` resolution in `lintSpecFile` and `cmdLintSpec` (R2, R3).
- [x] Increment 3: Update `repointSpecClaimsOnFragmentArchive` for flat files (R4).
- [x] Increment 4: Add regression test covering covered/spawned dispositions pointing to flat dated specs.

---

## 5. Validation

### Automated
- `TMPDIR=/home/go/.cache/tmp go test ./internal/cli -run TestLintSpecCoveredDispositionRecognizesFlatAndNestedSpecs -v`
- `TMPDIR=/home/go/.cache/tmp go test ./...`
- `pose validate --strict`
- `pose check --strict`

### Requirement trace
- R1 [satisfied] contract:lint-spec-flat-layout-resolution check:lintspec-resolution test:TestLintSpecCoveredDispositionRecognizesFlatAndNestedSpecs evidence:integration
- R2 [satisfied] contract:lint-spec-flat-layout-resolution check:lintspec-resolution test:TestLintSpecCoveredDispositionRecognizesFlatAndNestedSpecs evidence:integration
- R3 [satisfied] contract:lint-spec-flat-layout-resolution check:lintspec-resolution test:TestLintSpecCoveredDispositionRecognizesFlatAndNestedSpecs evidence:integration
- R4 [satisfied] contract:lint-spec-flat-layout-resolution check:lintspec-resolution test:TestLintSpecCoveredDispositionRecognizesFlatAndNestedSpecs evidence:integration

---

## 6. Delivery Evidence

### Artifact claims
- contract:lint-spec-flat-layout-resolution -> pose-mcp/cmd/pose/main.go

### Known gaps
None.

---

## 7. Final Report

### Delivered scope
- Supported flat dated specs in `collectSpecSlugs`, `siblingSpecStatus`, and `cmdLintSpec`.
- Fixed false-positive missing spec error in follow-up dispositions.

### Follow-ups
- [done] All requirements verified and delivered.
