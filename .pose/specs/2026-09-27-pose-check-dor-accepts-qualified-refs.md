---
slug: pose-check-dor-accepts-qualified-refs
status: in-progress
created_at: 2026-09-27
supersedes:
depends_on: pose-check-qualified-roadmap-members
priority: 0
components: pose-mcp
task_type: bugfix
delivers: contract:check-dor-accepts-qualified-refs
completed_at:
---

# Spec: `pose check` reads Definition of Ready dependencies with the lint grammar

## 1. Intent

### Goal
Make the Definition of Ready that `pose check` applies to a transition into
`in-progress` accept the same `depends_on` references as
`pose lint-spec --ready-check`, including qualified `xref:` dependencies.

### Business value
Harne8's `harne8-abm-governed-execution` depends on three pose-dist specs
through `xref:proj.pose-dist/spec:*`. Moving it to `in-progress` made
`pose check --strict` fail with "transition to in-progress without Definition
of Ready", while `pose lint-spec --ready-check` on the same file reported
`spec.ready=true`. The two gates disagreed because `specReady` in
`check.go` matched `depends_on` against local slug, milestone and roadmap
patterns only, and the lint gate uses `ParseArtifactRef`.

### Constraints
A malformed reference must still make the spec not ready. Whether a qualified
dependency resolves stays with the dependency graph checks and federated
acceptance, as it already does for lint.

### Non-goals
Do not change section or requirement rules of the Definition of Ready.

## 2. Requirements

- R1: A spec whose `depends_on` holds a well-formed `xref:` reference is ready under `pose check` when it is ready under `pose lint-spec --ready-check`.
- R2: A malformed `depends_on` reference still makes the spec not ready.

## 3. Technical Plan

### Affected areas
`pose-mcp/internal/cli/check.go` (`specReady`).

### Artifacts
- created: .pose/specs/2026-09-27-pose-check-dor-accepts-qualified-refs.md
- modified: pose-mcp/internal/cli/check.go
- modified: pose-mcp/internal/cli/remaining_surfaces_coverage_test.go
- created: .pose/changelogs/unreleased/pose-check-dor-accepts-qualified-refs.md

### Delivery targets
- contract:check-dor-accepts-qualified-refs module:pose-mcp profile:api-contract entrypoint:pose-mcp/cmd/pose/main.go

### Approach and rollback
Validate each `depends_on` entry with `pose.ParseArtifactRef`, the parser
`lint-spec --ready-check` uses. Rollback is a revert.

## 4. Tasks

- [x] Reproduce the disagreement on Harne8.
- [x] Write the regression and prove it fails without the change.
- [x] Use the lint grammar in `specReady`.
- [ ] Run the matrix, review and close.

## 5. Decisions

One grammar for one question: the lint gate's parser is already the one that
knows qualified references, so `check` reuses it instead of growing a second
pattern list.

## 6. Validation

| Scenario | Command | Expected evidence |
| --- | --- | --- |
| R1/R2 | `go test ./internal/cli -run SpecReady` | Qualified refs ready; malformed refs not ready. |
| Harne8 consumer | `pose check --strict` in Harne8 with `harne8-abm-governed-execution` in progress | No DoR error with the candidate; the DoR error with the previous pin. |

### Execution log
2026-09-27: on Harne8, `pose check --strict` with the pinned `680764d`
binary failed with the DoR error for `harne8-abm-governed-execution`, and
`pose lint-spec --ready-check` passed on the same file. The regression
`TestSpecReadyAcceptsTheReferencesLintAccepts` failed on two qualified cases
before the change and passes after it.

### Requirement trace
- R1 [satisfied] test:TestSpecReadyAcceptsTheReferencesLintAccepts
- R2 [satisfied] test:TestSpecReadyAcceptsTheReferencesLintAccepts

## 7. Final Report

### Delivered scope
`pose check` accepts well-formed qualified dependencies in the Definition of
Ready and still rejects malformed ones.

### Residual risks
None identified.

### Follow-ups
