---
slug: pose-review-run-reaping-beyond-linux
status: draft        # draft | in-progress | done | superseded | abandoned; blocked = operational wait, not terminal
created_at: 2026-11-25
completed_at:        # stamped on the transition to status: done
supersedes:          # slug of the superseded spec (when applicable)
depends_on: pose-review-run-reaps-lingering-reviewer-processes
remediates:          # optional explicit links: spec:slug@defect-fix, finding:rva-<16hex>/F1@revert (never inferred from depends_on)
priority: 3
components: pose-mcp
task_type: feature
surface: standard
delivers:            # optional typed refs: surface:id, contract:id, capability:id, infrastructure:id, governance:id
---

# Spec: pose-review-run-reaping-beyond-linux

## 1. Intent

### Goal
Carry the guarantees `pose-review-run-reaps-lingering-reviewer-processes` gives on Linux (through the review run shim) and on Windows (through a job object) to the remaining cases: other Unix platforms, the identities of what Windows and Unix group kills ended, and the environment-search fallback of an embedder without the shim.

### Business value
Review rounds 13 and 14 of that spec found these gaps; by the maintainer's decision on 2026-11-25 they were moved here so that spec could close on its declared scope (Linux and Windows termination).

### Constraints
No new guarantee is claimed before it is enforced and tested on the platform it names.

### Non-goals
Changing the Linux shim or the Windows job object.

## 2. Requirements

### Functional
- R1: On Unix platforms other than Linux, after the process-group kill the dispatch shall wait, within a bound, until the group is proven empty, and leave the cleanup incomplete otherwise (round 14, N2).
- R2: When the Windows job object or a Unix group kill ends processes, the run record shall name them, as the Linux shim's report does (round 14, N3).
- R3: The environment-search fallback of an embedder without the shim shall not end a process it cannot prove belongs to the run, for example by constraining it to the run's descendants (round 14, N1; limit L6 of the parent spec).
- R4: Each requirement shall have a test on the platform it concerns, run in CI where that platform is available.

### Non-functional
Existing behaviour outside the stated requirements is unchanged.

### Security
No new input crosses a trust boundary without the existing confinement.

### Compatibility
Records written by earlier engines keep their meaning.

## 3. Technical Plan

### Affected areas
`pose-mcp` review dispatch, its platform files and the manual.

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
- [ ] Confirm CI availability for macOS and Windows before starting.

### Implementation
- [ ] Implement each requirement with a test that fails before it.

### Validation
- [ ] `pose validate --carry` and `pose check --strict`.

## 5. Decisions

## 6. Validation

### Strategy
Each requirement gets a regression test on its platform.

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
