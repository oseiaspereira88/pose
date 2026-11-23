---
slug: pose-review-conclude-ignores-unrelated-local-edits
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

# Spec: pose-review-conclude-ignores-unrelated-local-edits

## 1. Intent

### Goal
Make `pose review conclude --apply` judge a run against the committed review inputs, so uncommitted edits unrelated to the scope neither refuse the conclusion nor leave a permanent refusal record.

### Business value
During `pose-review-readiness-before-seal`, `review conclude rrn-fa6a6e8e1df0a0ba --apply` was refused with `persisted review inputs no longer materialize: sealed bytes changed for <ADR>` because round-2 fixes were being edited in the working tree. It wrote a permanent refusal record; recovering meant deleting that uncommitted record, stashing the edits and concluding again. Implementers fix findings while the conclusion is pending, so the conflict is routine.

### Constraints
A run whose committed inputs changed still loses governance. Nothing weakens the check that the reviewed subject is what was sealed.

### Non-goals
Changing review independence, round accounting or the attestation preflight.

## 2. Requirements

### Functional
- R1: The governance check of a conclusion shall read the sealed inputs from the committed tree (or the sealed commit), not from uncommitted working-tree bytes.
- R2: A refusal caused only by the local working tree shall not be recorded as the run's conclusion; it is reported, and the same run can be concluded once the tree is clean.
- R3: A regression test shall edit a sealed input in the working tree without committing, conclude, and get the recorded conclusion; and a committed change to a sealed input shall still refuse.

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
