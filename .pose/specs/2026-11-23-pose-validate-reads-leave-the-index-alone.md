---
slug: pose-validate-reads-leave-the-index-alone
status: draft
created_at: 2026-11-23
completed_at:
supersedes:
depends_on: 
remediates:
priority: 2
components: pose-mcp
task_type: bugfix
surface: standard
delivers:
---

# Spec: pose-validate-reads-leave-the-index-alone

## 1. Intent

### Goal
Make the engine's own Git reads during `pose validate` and `pose check`, such as the provenance status of `validate`, never rewrite `.git/index` or take index locks in the validated checkout.

### Business value
In Harne8 the governance read view of `instance-health-composite` failed intermittently with `review inputs changed during operation: .git` because concurrent reads refreshed the index stat cache. The Harne8 bugfix `harne8-validation-reads-leave-root-index-alone` fixed the instance's own scripts and left this engine part as its follow-up F1.

### Constraints
Reads keep their results. Stale stat caches may cost time, never correctness.

### Non-goals
Changing which files are compared.

## 2. Requirements

### Functional
- R1: Every status, diff and ls-files read the engine makes for validation provenance, material cleanliness and readiness checks shall run with optional locks disabled and `diff.autoRefreshIndex=false`.
- R2: A regression test shall run those reads over a checkout with a stale stat cache and prove `.git/index` byte-identical afterwards.
- R3: A concurrency test shall run `validate` and `check --strict` together on one checkout and show no `governance read view changed during capture` error.

### Non-functional
Existing behaviour outside the stated requirements is unchanged.

### Security
No new input crosses a trust boundary without the existing confinement.

### Compatibility
Records and indexes written by earlier engines keep their meaning.

## 3. Technical Plan

### Affected areas
`pose-mcp` (CLI and engine), the manual and its embedded copy.

### Artifacts
Declared when the spec starts.

### Delivery targets
Declared when the spec starts.

### API/contract changes
To be detailed when the spec starts.

### Data/storage changes
To be detailed when the spec starts.

### Technical risks
To be detailed when the spec starts.

## 4. Tasks

### Planning
- [ ] Confirm the requirements against the current engine before starting.

### Implementation
- [ ] Implement each requirement with a test that fails before it.

### Validation
- [ ] `pose validate --carry` and `pose check --strict`.

## 5. Decisions

## 6. Validation

### Strategy
Each requirement gets a regression test shown failing before its code.

### Deterministic checks

#### Test
- Command: `go test ./...` (in `pose-mcp`), then `pose validate --carry`.

### Execution log

### Results summary

### Requirement trace

### Known gaps

## 7. Final Report

### Delivered scope

### Files and modules changed

### Validation executed

### Residual risks

### Follow-ups
