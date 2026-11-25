---
slug: pose-review-conclude-ignores-unrelated-local-edits
status: done
created_at: 2026-11-23
completed_at: 2026-11-25
supersedes:          # slug of the superseded spec (when applicable)
depends_on:          # prerequisites, inline list: other-spec, milestone:<roadmap>/<id>, roadmap:<slug>
remediates:          # optional explicit links: spec:slug@defect-fix, finding:rva-<16hex>/F1@revert (never inferred from depends_on)
priority: 2
components: pose-mcp
task_type: bugfix
surface: standard
delivers: capability:review-conclude-ignores-unrelated-local-edits
---

# Spec: pose-review-conclude-ignores-unrelated-local-edits

## 1. Intent

### Goal
Make `pose review conclude --apply` record a run's verdict while the implementer is already editing the sealed subject for the next round, and never leave a permanent refusal caused only by the working tree.

### Business value
During `pose-review-readiness-before-seal`, `review conclude rrn-fa6a6e8e1df0a0ba --apply` was refused with `persisted review inputs no longer materialize: sealed bytes changed for <ADR>` because round-2 fixes were being edited in the working tree. It wrote a permanent refusal record; recovering meant deleting that uncommitted record, stashing the edits and concluding again. Implementers fix findings while the conclusion is pending, so the conflict is routine.

### Constraints
An approval still requires its bundle to be the current subject, and closeout and verification still judge the current bundle. A verdict that does not approve stays bound to the bundle it reviewed.

### Non-goals
Changing review independence, round accounting or the attestation preflight.

## 2. Requirements

### Functional
- R1: A conclusion that does not approve (changes requested or rejected) shall be recorded whatever the working tree holds: it needs the run's own record to stand, its proof to verify against the brief the run was sent, and its draft to account for its round, but not its bundle to be the current subject.
- R2: An approval refused only because the working tree no longer holds its bundle's subject shall not be recorded as the run's conclusion; it is reported, and the same run concludes once the tree matches its bundle again.
- R3: Regression tests shall conclude a run that requests changes over an uncommitted edit of a sealed input, and an approval over such an edit (not recorded), then after the edit is undone (recorded).

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
- created: .pose/specs/2026-11-23-pose-review-conclude-ignores-unrelated-local-edits.md
- modified: pose-mcp/internal/pose/review_conclusions.go
- modified: pose-mcp/internal/pose/review_run_provenance.go
- modified: pose-mcp/internal/pose/review_brief.go
- created: pose-mcp/internal/pose/review_conclusion_local_edits_test.go
- modified: POSE.md
- modified: locales/pt-BR/POSE.md
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md


### Delivery targets

- capability:review-conclude-ignores-unrelated-local-edits module:pose-mcp profile:composed-capability entrypoint:pose-mcp/internal/cli/review_conclude.go


### API/contract changes
`pose review conclude` records a non-approving conclusion over local edits; an approval refused only for the working tree is reported without a record.


### Data/storage changes
None.


### Technical risks
A non-approving verdict of a bundle that is no longer current is recorded; it approves nothing, and closeout and verification still read only the current bundle's attestations.


## 4. Tasks

### Planning
- [ ] Confirm the cause against the current engine before starting.

### Implementation
- [ ] Implement each requirement with a test that fails before it.

### Validation
- [ ] `pose validate --carry` and `pose check --strict`.

## 5. Decisions

### Decision D1
- Basis: R1, R2
Currency (the bundle is the current subject, and the sealed inputs still materialize from the working tree) matters only to an approval. A verdict that requests changes or rejects is a fact about the bundle it reviewed, so the conclusion, the run proof and the re-rendered brief of a non-approving run check the run's own record and what it was sent, not the working tree. The handoff and resume paths keep requiring the current bundle.

## 6. Validation

### Strategy
Each requirement gets a regression test shown failing before its code.

### Deterministic checks

#### Test
- Command: `go test ./...` (in `pose-mcp`), then `pose validate --carry`.

### Execution log
- 2026-11-25: implemented R1–R3. The conclusion checks currency only for approvals; the run proof of a completed run that does not approve needs its record to stand; the provenance check re-renders a run's brief from its bundle whether or not it is current. An approval refused only for currency returns its reasons without recording them. Each change has a test that fails with it mutated out.


### Results summary
The full `pose-mcp` suite passes; the canonical result is refreshed with `pose validate --carry`.


### Requirement trace
- R1 [satisfied] test:TestARequestForChangesConcludesOverLocalEdits
- R2 [satisfied] test:TestAnApprovalOverLocalEditsIsNotRecordedAndConcludesLater
- R3 [satisfied] test:TestARequestForChangesConcludesOverLocalEdits, test:TestAnApprovalOverLocalEditsIsNotRecordedAndConcludesLater


### Known gaps
None.


## 7. Final Report

### Delivered scope
A conclusion that does not approve is recorded over local edits; an approval refused only because the working tree changed is reported without a record and concludes once the tree matches.


### Files and modules changed
`pose-mcp` (review conclusion, run proof, brief rendering), the manual in both locales and its embedded copies.


### Validation executed
`go test ./...` in `pose-mcp`; `pose check --strict`; `pose validate --carry --json-out .pose/results/delivery-validation.json`.


### Residual risks
None beyond the stated design: a recorded non-approving verdict of an old bundle approves nothing.


### Follow-ups
None.


