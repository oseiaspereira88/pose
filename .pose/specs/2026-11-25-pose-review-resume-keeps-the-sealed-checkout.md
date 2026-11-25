---
slug: pose-review-resume-keeps-the-sealed-checkout
status: draft        # draft | in-progress | done | superseded | abandoned; blocked = operational wait, not terminal
created_at: 2026-11-25
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

# Spec: pose-review-resume-keeps-the-sealed-checkout

## 1. Intent

### Goal
Make `pose delegate resume` give the resumed reviewer session a sealed checkout it can reach: either at the path the interrupted run used, or with the new path stated to the session, so a resume never points the reviewer at a removed directory.

### Business value
On 2026-11-25 round 11 of `pose-review-run-reaps-lingering-reviewer-processes` hit the Codex usage limit; `pose delegate resume rrn-7e7520ad4fac9345` started a new run whose session still referred to the first attempt's sealed checkout, which the cleanup had removed. The reviewer correctly stopped (`the sealed checkout ... has disappeared`), the resume produced no draft, and the round had to be dispatched again from scratch.

### Constraints
The resumed run still reviews exactly the sealed subject; no path outside the run's runtime is trusted.

### Non-goals
Changing round accounting or the conclusion step.

## 2. Requirements

### Functional
- R1: A resumed run shall materialize its sealed checkout where the resumed session can reach it, or tell the session its new path in the continuation prompt.
- R2: A regression test shall resume an interrupted run whose first checkout was removed and see the session work in the new checkout.

### Non-functional
Existing behaviour outside the stated requirements is unchanged.

### Security
No new input crosses a trust boundary without the existing confinement.

### Compatibility
Runs recorded by earlier engines keep their meaning.

## 3. Technical Plan

### Affected areas
`pose-mcp` review dispatch and continuation.

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
- [ ] Confirm how the continuation prompt and the worktree path are built.

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
