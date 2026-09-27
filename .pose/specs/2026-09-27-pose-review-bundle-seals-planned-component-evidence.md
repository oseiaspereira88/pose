---
slug: pose-review-bundle-seals-planned-component-evidence
status: in-progress
created_at: 2026-09-27
supersedes:
depends_on: pose-abm-review-soundness
priority: 0
components: pose-mcp
task_type: bugfix
delivers: contract:review-bundle-seals-planned-component-evidence
completed_at:
---

# Spec: The review bundle seals evidence for the components its plan validates

## 1. Intent

### Goal
Make a sealed review bundle carry the current validation evidence of every
module its plan asks a per-component `validate` tool for, not only the modules
of the spec's delivery targets.

### Business value
Harne8's `harne8-abm-governed-execution` delivers one capability in
`conductor` and also changes `harness`, `contracts` and `scripts`. Its review
plan requires `validate` for each of the four components and refuses a
component tool that cites a sibling's evidence, which is right. But the bundle
sealed evidence only from the target module (and the repository root), so the
attestation could not cite anything for `harness`, `contracts` or `scripts`, and
`review verify` blocked on "cites evidence from conductor". A multi-component
spec with one target could not be reviewed honestly, and the only way out was
declaring delivery targets that do not exist.

### Constraints
A scope without delivery targets keeps sealing every module, as before. A tool
for a component still must cite that component's own evidence. Evidence
currency rules are unchanged.

### Non-goals
Do not change which tools the plan requires or how dispositions are judged.

## 2. Requirements

- R1: When a spec has delivery targets, the bundle also seals current, passing, required evidence from each module a per-component `validate` tool of the plan names.
- R2: A module no `validate` tool names is still not sealed, and a scope without targets keeps its previous sealing.

## 3. Technical Plan

### Affected areas
`pose-mcp/internal/pose/review_bundle.go` (`reviewBundleEvidence`).

### Artifacts
- created: .pose/specs/2026-09-27-pose-review-bundle-seals-planned-component-evidence.md
- modified: pose-mcp/internal/pose/review_bundle.go
- modified: pose-mcp/internal/pose/review_bundle_test.go
- created: .pose/changelogs/unreleased/pose-review-bundle-seals-planned-component-evidence.md

### Delivery targets
- contract:review-bundle-seals-planned-component-evidence module:pose-mcp profile:api-contract entrypoint:pose-mcp/cmd/pose/main.go

### Approach and rollback
Pass the plan to `reviewBundleEvidence` and add the components of its
`validate` tools to the sealed modules when targets exist. Rollback is a
revert. Bundles sealed before the change keep their evidence; resealing a scope
may add evidence, which supersedes its review until it is reattested.

## 4. Tasks

- [x] Reproduce the blocked attestation on Harne8.
- [x] Write the regression and prove it fails without the change.
- [x] Seal the planned components' evidence.
- [ ] Run the matrix, review and close.

## 5. Decisions

Sealing follows the plan rather than a new declaration: the plan already names
the modules it will hold the reviewer to, so those are the modules whose
evidence the reviewer must be able to cite.

## 6. Validation

| Scenario | Command | Expected evidence |
| --- | --- | --- |
| R1/R2 | `go test ./internal/pose -run TestReviewBundleSealsEvidenceForPlannedComponentModules` | Planned component sealed; unplanned module and target-less scope unchanged. |
| Harne8 consumer | `pose review bundle spec:harne8-abm-governed-execution --json` | Evidence from conductor, contracts, harness and scripts. |

### Execution log
2026-09-27: on Harne8, attesting `harne8-abm-governed-execution` with the
pinned `1bcdd97` binary blocked on three component tools citing conductor
evidence, because the bundle sealed only `conductor` and root results. The
regression failed before the change (only the target module sealed) and passes
after it. A candidate binary's bundle for the same spec carries evidence from
all four components.

### Requirement trace
- R1 [satisfied] test:TestReviewBundleSealsEvidenceForPlannedComponentModules
- R2 [satisfied] test:TestReviewBundleSealsEvidenceForPlannedComponentModules

## 7. Final Report

### Delivered scope
A review bundle seals the current evidence of the modules its plan validates
per component, so multi-component specs can be attested without invented
targets.

### Residual risks
Resealing any scope whose plan names extra components adds evidence and
supersedes its review until reattested.

### Follow-ups
