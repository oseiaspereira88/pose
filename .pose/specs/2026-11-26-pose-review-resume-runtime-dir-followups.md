---
slug: pose-review-resume-runtime-dir-followups
status: draft        # draft | in-progress | done | superseded | abandoned; blocked = operational wait, not terminal
created_at: 2026-11-26
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

# Spec: pose-review-resume-runtime-dir-followups

## 1. Intent

### Goal
Close the follow-ups left by `pose-review-resume-keeps-the-sealed-checkout`: the claim and the runtime-directory check as one step, a resume of a run interrupted under an earlier engine, the session continued into a new round, and a sharper refusal test.

### Business value
The spec closed on 2026-11-26 after five review rounds, with follow-ups recorded in its Final Report and in the approving attestation (round 5, N22–N25).

### Constraints
Each item keeps the behaviour its origin spec delivered; a fix that changes an answer states it as a decision.

### Non-goals
Reopening the decisions of the origin spec.

## 2. Requirements

### Functional
- R1: The continuation claim and the runtime-directory check shall be one step: a resume or handoff that loses a race shall be told the continuation is pending, never asked to remove a live run's directory; a collision found at creation shall leave the predecessor resumable (rounds 2 N8, 3 N12, 4 N17, 5 N22).
- R2: A run interrupted under an engine that named its runtime directory at random (its `safety.runtime_id` differs from the attempt's stable name) shall not be resumed into a checkout its session cannot find: the resume shall refuse and point to a handoff (rounds 3 N11, 4 N18, 5 N23).
- R3: A session continued into a new review round shall be given the checkout path it saw, or the brief shall state the new path, so a continued thread does not meet the same failure (rounds 3 N15, 4 N21, 5 N24).
- R4: The resume-refusal regression test shall assert the resume's own refusal, not any 'still exists' text in the output (round 5, N25).

### Non-functional
Existing behaviour outside the stated requirements is unchanged.

### Security
No new input crosses a trust boundary without the existing confinement.

### Compatibility
Records and indexes written by earlier engines keep their meaning.

## 3. Technical Plan

### Affected areas
`pose-mcp`: `review_dispatch.go` (runtime directory), `review_run_continuity.go`, `delegation_handoff.go`, the round-continuation path, and `review_run_continuity_test.go`.

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
Origin: `pose-review-resume-keeps-the-sealed-checkout` (done 2026-11-26).
