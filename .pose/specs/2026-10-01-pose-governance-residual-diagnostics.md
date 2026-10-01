---
slug: pose-governance-residual-diagnostics
status: in-progress
created_at: 2026-10-01
completed_at:
components: pose-mcp
task_type: bugfix
priority: 3
---

# Spec: Governance residual diagnostics

## 1. Intent

Expose retained compatibility boundaries and contradictory policy declarations
without changing which policy is authoritative or invalidating completed reviews.

## 2. Requirements

- R1: Doctor shall report conflicting map and legacy contract adoption dates,
  preserving map-first authority, including explicit empty map values.
- R2: Doctor shall count sealed bundles missing governing contracts or gates;
  an empty repository shall not receive a legacy-bundle warning.
- R3: Stale profile findings shall name the first declaration schema v2 refuses,
  using the actual profile parser without mutating the profile.
- R4: Roadmap checks with no cut criteria shall explicitly say so while retaining
  member acceptance and current exit semantics.
- R5: Review verification shall report observation warnings from the selected
  sealed bundle, including when a completed scope's review is retained.

## 3. Technical Plan

### Artifacts

- modified: pose-mcp/internal/cli/doctor.go
- created: pose-mcp/internal/cli/doctor_governance_residuals_test.go
- modified: pose-mcp/internal/cli/surface_check.go
- modified: pose-mcp/internal/cli/surface_check_test.go
- modified: pose-mcp/internal/pose/review_closeout.go
- modified: pose-mcp/internal/pose/review_bundle.go
- modified: pose-mcp/internal/pose/review_bundle_test.go
- created: .pose/specs/2026-10-01-pose-governance-residual-diagnostics.md
- created: .pose/changelogs/unreleased/pose-governance-residual-diagnostics.md

## 4. Tasks

- [x] Add diagnostic-only findings and retain prior validation decisions.
- [x] Cover empty, valid, contradictory and historical inputs.
- [ ] Run required module checks.

## 5. Decisions

No automatic policy/profile rewrites. Diagnostics are warnings rather than new
acceptance obligations. Use existing JSON finding/warning containers and schema.

## 6. Validation

Before implementation: test contradictory/equal/absent adoption values, explicit
empty map precedence, old/current/absent bundles, stale invalid evidence classes,
empty versus populated roadmap criteria and selected sealed evidence warnings.
Capture profile bytes before/after to prove diagnostic projection does not write.

### Execution log

- Reused knowledge:cli-output-design-taxonomy: emit new warnings through cliout.
- Fixed the direct-print regression in the human roadmap warning; added a
  regression assertion for its stdout channel and unchanged successful exit.
- Targeted doctor conflict/count/profile and roadmap warning tests passed.
- Selected retained-evidence warning regression passed while approval stays closed.

### Requirement trace

- R1 [satisfied] test:TestDoctorReportsConflictingAdoptionDates
- R2 [satisfied] test:TestDoctorCountsLegacySealedBundleFields
- R3 [satisfied] test:TestDoctorNamesStaleProfileDeclarationWithoutRewriting
- R4 [satisfied] test:TestRoadmapCheckNamesAbsentCutCriteria
- R5 [satisfied] test:TestRetainedVerifyReportsSelectedEvidenceObservation

## 7. Final Report

In progress.
