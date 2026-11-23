---
slug: pose-review-run-reaps-lingering-reviewer-processes
status: draft        # draft | in-progress | done | superseded | abandoned; blocked = operational wait, not terminal
created_at: 2026-11-23
completed_at:        # stamped on the transition to status: done
supersedes:          # slug of the superseded spec (when applicable)
depends_on:          # prerequisites, inline list: other-spec, milestone:<roadmap>/<id>, roadmap:<slug>
remediates:          # optional explicit links: spec:slug@defect-fix, finding:rva-<16hex>/F1@revert (never inferred from depends_on)
priority: 2
components: pose-mcp
task_type: bugfix
surface: standard
delivers:            # optional typed refs: surface:id, contract:id, capability:id, infrastructure:id, governance:id
---

# Spec: pose-review-run-reaps-lingering-reviewer-processes

## 1. Intent

### Goal
Keep a delegated review run concludable when the reviewer ends while a process it started is still running, by terminating and reaping the run's remaining processes before the cleanup inventory.

### Business value
Round 2 of `pose-review-readiness-before-seal` (rrn-9d5816c780952f13) produced a complete draft with two valid findings, but its cleanup was `runtime cleanup inventory incomplete; resources retained` because a broad `go test` the reviewer started was still running. The run could not be concluded and its findings had to be carried by hand.

### Constraints
Cleanup keeps proving that the sealed checkout is unchanged and that no owned resource survives; only the order changes. A run whose resources truly cannot be reclaimed stays unconcludable.

### Non-goals
Changing review independence, round accounting or the attestation preflight.

## 2. Requirements

### Functional
- R1: When the adapter exits, the dispatch shall terminate every remaining process of the run's process group, wait for them within a bound, and only then take the cleanup inventory.
- R2: A run whose leftover processes were terminated and reaped, and whose inventory then completes, shall be concludable; the run record shall say which processes were terminated.
- R3: The review brief shall ask the reviewer to let every command it starts finish before ending.
- R4: A regression test shall dispatch an adapter that leaves a background process running and conclude the run.

### Non-functional
Existing behaviour outside the stated requirements is unchanged.

### Security
No new input crosses a trust boundary without the existing confinement.

### Compatibility
Records written by earlier engines keep their meaning.

## 3. Technical Plan

### Affected areas
`pose-mcp` (review dispatch and conclusion), the manual and its embedded copy.

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
- [ ] Confirm the cause against the current engine before starting.

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
