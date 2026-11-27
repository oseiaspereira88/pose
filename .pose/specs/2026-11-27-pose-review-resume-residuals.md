---
slug: pose-review-resume-residuals
status: draft        # draft | in-progress | done | superseded | abandoned; blocked = operational wait, not terminal
created_at: 2026-11-27
completed_at:        # stamped on the transition to status: done
supersedes:          # slug of the superseded spec (when applicable)
depends_on:          # prerequisites, inline list: other-spec, milestone:<roadmap>/<id>, roadmap:<slug>
remediates:          # optional explicit links: spec:slug@defect-fix, finding:rva-<16hex>/F1@revert (never inferred from depends_on)
priority: 3
components: pose-mcp
task_type: bugfix
surface: standard
delivers:            # optional typed refs: surface:id, contract:id, capability:id, infrastructure:id, governance:id
---

# Spec: pose-review-resume-residuals

## 1. Intent

### Goal
Close what `pose-review-resume-runtime-dir-followups` left open: the claim wait, a continued round whose ledger read fails, the held-path window between plan and dispatch, the recorded reason, the rounds documentation and the remaining test gaps.

### Business value
The spec closed on 2026-11-27 with follow-ups in its Final Report (round 1, N4–N5) and its approving attestation (round 2, N6–N11).

### Constraints
Each item keeps the behaviour its origin spec delivered; a fix that changes an answer states it as a decision.

### Non-goals
Reopening the decisions of the origin spec.

## 2. Requirements

### Functional
- R1: The wait for another resume's claim shall report a completed continuation as such (and to a handoff as already continued), honour cancellation, read the ledger cheaply, and turn a claim conflict after the winner finished into the recorded result (round 1 N4, round 2 N9).
- R2: A continued round whose previous runtime name cannot be read from the ledger shall record that in the run instead of moving its checkout silently (round 1 N5, round 2 N10).
- R3: A thread path taken between the plan and the dispatch shall not fail the run after its attempt is reserved, nor ask to remove a directory another run may hold (round 2, N6).
- R4: The recorded reason shall name the held path, and an lstat error other than not-exist shall not count as a free path (round 2, N7).
- R5: The review rounds documentation, in both locales and the embedded copies, shall list the held-path condition for opening a new session (round 2, N11).
- R6: Tests shall reach the exclusive take after a replan, resume end to end a run whose recorded random name differs from the stable one, and cover the claim-failure removal and the handoff plan's pending branch (rounds 1 N3, 2 N8).

### Non-functional
Existing behaviour outside the stated requirements is unchanged.

### Security
No new input crosses a trust boundary without the existing confinement.

### Compatibility
Records and indexes written by earlier engines keep their meaning.

## 3. Technical Plan

### Affected areas
`pose-mcp`: `review_dispatch.go`, `review_rounds.go`, `delegation_handoff.go`, the manual in both locales, and their tests.

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
- [ ] Confirm each requirement against the current engine before starting.

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
Origin: `pose-review-resume-runtime-dir-followups` (done 2026-11-27).
