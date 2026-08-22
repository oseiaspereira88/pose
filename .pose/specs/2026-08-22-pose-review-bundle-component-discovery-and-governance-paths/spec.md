---
slug: pose-review-bundle-component-discovery-and-governance-paths
status: done
created_at: 2026-08-22
supersedes:
depends_on: pose-contributor-mode-workflow-and-cli-hints
priority: 25
components: pose-mcp, cli, review
delivers: surface:review-bundle-classification
completed_at: 2026-08-22
---

# Spec: Review Bundle Component Discovery, Governance Paths, and Artifact-Check Attribution

## 1. Intent

### Goal
Fix issue #31 where `pose review bundle spec:<slug> --seal` fails with `unclassified review subject path` on valid project conventions (`docs/decisions/`, `.pose/roadmaps/`, root `.pose/*.json`, and component source trees like `agent/`, `conductor/`), and ensure `pose artifact-check` properly honors recorded change-set ranges from history reports.

### Business value
Unblocks seamless review bundle sealing and closing ceremonies across diverse project repository structures, ADR directory conventions (`docs/decisions/` alongside `.pose/adr/`), roadmap definitions, and custom component hierarchies without forcing manual overrides or recording workarounds.

### Constraints
- Zero security regressions; paths must remain confined and valid.
- Backward compatibility: existing `.pose/adr/`, `pose-mcp/`, `docs-site/` conventions remain 100% supported.
- Deterministic behavior in `ReviewPlan` and `ReviewBundle` generation.

### Non-goals
- Allowing unconfined path traversal outside repo root.

---

## 2. Requirements

### Functional
- R1: `loadReviewRepoEntries()` in `review_plan.go` shall gracefully synthesize repo entries from `.pose/indexes/repo-map.json` (if present), `.pose/indexes/module-metadata.json`, `.pose/state/components/*.json`, and top-level directory discovery so that declared spec components (e.g., `components: conductor, agent`) resolve to real `ReviewPlanComponent` paths even when `repo-map.json` is absent.
- R2: `reviewBundlePathClass()` in `review_bundle.go` shall classify `docs/decisions/` as `governance` and `docs/` as `documentation`.
- R3: `reviewBundlePathClass()` shall classify `.pose/roadmaps/`, `.pose/templates/`, `.pose/rules/`, `.pose/adr/`, `.pose/knowledge/`, and root governance manifests (`.pose/docs.json`, `.pose/docs-review.jsonl`, `.pose/release-policy.json`, `.pose/project.json`, `compatibility.json`) as `governance`.
- R4: `reviewBundlePathClass()` shall classify paths under any matched `ReviewPlanComponent` path root as `implementation`.
- R5: `cmdArtifactCheck` in `artifact_integrity.go` shall check recorded change sets from history reports (`.pose/reports/history/*.jsonl`) and use them when `--from`/`--to` are omitted, allowing re-scoped/corrected change-set reports to be validated.

### Non-functional
- Complete test coverage in `internal/pose` and `internal/cli`.
- Clean validation and lint passing in strict mode.

---

## 3. Technical Plan

### Affected areas
- `pose-mcp/internal/pose/review_plan.go`
- `pose-mcp/internal/pose/review_bundle.go`
- `pose-mcp/internal/cli/artifact_integrity.go`
- `pose-mcp/internal/pose/review_bundle_test.go`
- `pose-mcp/internal/cli/artifact_integrity_test.go`

### Delivery targets
- surface:review-bundle-classification module:pose-mcp profile:cli-surface entrypoint:pose-mcp/cmd/pose/main.go

---

## 4. Tasks

### Artifacts
- modified: pose-mcp/internal/pose/review_plan.go
- modified: pose-mcp/internal/pose/review_bundle.go
- modified: pose-mcp/internal/cli/artifact_integrity.go
- modified: pose-mcp/internal/pose/review_bundle_test.go
- modified: pose-mcp/internal/cli/artifact_integrity_test.go

### Increments
- [x] Increment 1: Enhance `loadReviewRepoEntries()` with multi-source component discovery (R1).
- [x] Increment 2: Update `reviewBundlePathClass()` with ADR, roadmap, and governance manifest conventions (R2, R3, R4).
- [x] Increment 3: Fix `cmdArtifactCheck` recorded change set precedence (R5).
- [x] Increment 4: Add unit tests, verify strict checks, seal review bundle, and attest.

---

## 5. Validation

### Automated
- `go test ./internal/pose -run ReviewBundle -count=1`
- `go test ./internal/pose -run ReviewPlan -count=1`
- `go test ./internal/cli -run Artifact -count=1`
- `pose check --strict`
- `pose validate --strict`

### Requirement trace
- R1 [satisfied] surface:review-bundle-classification check:delivery-integration test:TestReviewBundleComponentDiscoveryAndGovernancePaths evidence:integration
- R2 [satisfied] surface:review-bundle-classification check:delivery-integration test:TestReviewBundleComponentDiscoveryAndGovernancePaths evidence:integration
- R3 [satisfied] surface:review-bundle-classification check:delivery-integration test:TestReviewBundleComponentDiscoveryAndGovernancePaths evidence:integration
- R4 [satisfied] surface:review-bundle-classification check:delivery-integration test:TestReviewBundleComponentDiscoveryAndGovernancePaths evidence:integration
- R5 [satisfied] surface:review-bundle-classification check:delivery-integration test:TestArtifactCheckHonorsRecordedChangeSets evidence:integration

---

## 6. Delivery Evidence

### Artifact claims
- surface:review-bundle-classification -> pose-mcp/cmd/pose/main.go

### Known gaps
None.

---

## 7. Final Report

### Delivered scope
- Resolved issue #31: Enhanced `loadReviewRepoEntries()` in `pose-mcp/internal/pose/review_plan.go` to synthesize review components across `.pose/indexes/repo-map.json`, `.pose/indexes/module-metadata.json`, `.pose/state/components/*.json`, and top-level directory discovery.
- Expanded `reviewBundlePathClass()` in `pose-mcp/internal/pose/review_bundle.go` to properly classify ADR directories (`docs/decisions/`), roadmaps (`.pose/roadmaps/`), governance manifest files (`.pose/*.json`), and discovered component trees (`agent/`, `conductor/`, etc.).
- Updated `cmdArtifactCheck` in `pose-mcp/internal/cli/artifact_integrity.go` to honor recorded change-set reports from `.pose/reports/history/*.jsonl` when `--from`/`--to` are omitted.

### Follow-ups
- [done] All requirements delivered and verified.
