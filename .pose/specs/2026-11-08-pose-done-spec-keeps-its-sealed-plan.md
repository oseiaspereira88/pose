---
slug: pose-done-spec-keeps-its-sealed-plan
status: in-progress
created_at: 2026-11-08
completed_at:
supersedes:
depends_on: pose-milestone-review-proportional
remediates:
priority: 1
components: pose-mcp
task_type: bugfix
delivers: capability:done-spec-keeps-its-sealed-plan
---

# Spec: A done spec keeps the plan it was sealed with

## 1. Intent

### Goal

The approval of a spec that is the only member of a milestone stays valid when the spec turns done, and a closeout that stops because its inputs were not proven stable leaves no checkpoint behind.

### Business value

Closing B36 and B33 cost an extra, empty review round each: the review plan of an open spec carries the milestone exit gate, the plan of the same spec once done does not, the plan is an input of the sealed bundle, so the bundle reads superseded and the approval no longer matches. The reviewer then confirmed only that the status, the derived indexes and the review artifacts had changed. On top of it, the apply that made the spec done stopped at `verify` with "governance read view is unknown" and committed a checkpoint recording that failure, which the next reviewer took for an open defect. Four extra reviews were spent on B33 alone.

### Constraints

- The decision D2 of `pose-milestone-review-proportional` says a done spec keeps the plan it was sealed with. The engine does not do that for the milestone criterion, and this spec makes it do so.
- No approval may be created or widened: an approval stands only when the attestation answers every criterion of the plan it was sealed with, as it does today.
- Specs closed before the exit gate existed must read exactly as they do.

### Non-goals

- Dropping the review of a milestone of several members, or any review of an open spec.
- Changing what a reviewer must judge for the exit gate.
- Repairing history: a checkpoint already committed is not rewritten.

## 2. Requirements

### Functional

- R1: When a spec is done and is the only member of a milestone of a live roadmap whose exit gate is declared, and a sealed bundle of that spec carries the criterion of that gate, the review plan the spec is prepared with shall carry the same criterion, so that the bundle it was sealed with still represents its semantic inputs and the approval attested over it stays fresh.
- R2: A done spec that no sealed bundle carries the criterion for, a spec of several members, a milestone without an exit gate and a spec of a terminal roadmap shall be prepared with the plan they have today.
- R3: An open spec shall be prepared as today: the criterion is required and judged, one per gate.
- R4: When the lifecycle of a scope is done and its terminal verification stops because the governance read view is unknown, `pose close --apply` and `--resume` shall write no checkpoint for that stop and shall not keep one a previous call wrote for it, and the result shall say that the repository, not a checkpoint, carries the state and that verification is to be repeated.
- R5: A step blocked or failed for any other reason shall be recorded in the checkpoint as before.
- R6: The attestation rules shall not change: a criterion of the sealed plan that the attestation does not answer, or answers without the evidence it demands, still blocks the approval, for an open or a done spec.

### Non-functional

- The sealed bundles of the scope are read once per preparation, only for a done spec that is the sole member of a milestone with a gate.

### Security

- Keeping a criterion in a plan can only add a requirement. The approval of a done spec is still the one attested over its sealed bundle, and `pose review verify` reports where it comes from.

### Compatibility

- A done spec that was sealed with the criterion reads fresh instead of superseded after this change; a done spec sealed without it is unchanged. Bundles and attestations are not rewritten.

## 3. Technical Plan

### Affected areas

`internal/pose/review_plan.go` (the milestone gate criterion), `internal/cli/closeout_plan.go` (the checkpoint of a blocked step), their tests, the closeout skill and the ADR of the one-member milestone.

### Artifacts

- created: .pose/specs/2026-11-08-pose-done-spec-keeps-its-sealed-plan.md
- created: .pose/starts/pose-done-spec-keeps-its-sealed-plan.json
- created: .pose/changelogs/unreleased/pose-done-spec-keeps-its-sealed-plan.md
- modified: .pose/adr/2026-11-02-a-milestone-of-one-member-is-that-member.md
- modified: .pose/indexes/validation-matrix.json
- modified: pose-mcp/internal/pose/review_plan.go
- modified: pose-mcp/internal/cli/closeout_plan.go
- created: pose-mcp/internal/pose/done_spec_sealed_plan_test.go
- modified: pose-mcp/internal/pose/closeout_plan.go
- created: pose-mcp/internal/cli/closeout_unstable_inputs_test.go

### Delivery targets

- capability:done-spec-keeps-its-sealed-plan module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go

### API/contract changes

None to a schema. The review plan of a done spec in that position carries the criterion it was sealed with; `pose close` leaves no checkpoint at a stop on an unknown read view.

## 4. Tasks

- [ ] Reproduce the superseded approval of a done sole-member spec
- [ ] Keep the criterion for a spec sealed with it
- [ ] Do not checkpoint a stop on an unknown read view
- [ ] Measure every done spec of this repository, old reader against new
- [ ] Amend the ADR and the closeout skill
- [ ] Independent review and closeout

## 5. Decisions

### Decision D1
- Basis: R1, R2
- Minimal option: add the criterion to every done spec in that position
- Selected option: keep it only where a sealed bundle of the spec carries it
- Rationale: the plan is an input of the bundle; adding a criterion to specs that were sealed without it would supersede every approved done spec of a live roadmap, and the code and the decision it implements already say a done spec keeps the plan it was sealed with
- Consequences: the plan of a done spec depends on its sealed bundles, which are read for it
- Falsifier: a done spec sealed with the criterion that still reads superseded, or a done spec sealed without it whose approval falls

### Decision D2
- Basis: R4, R5
- Minimal option: delete the checkpoint after the fact
- Selected option: write none for a stop on an unknown read view of a done scope
- Rationale: the transition has been made, the repository carries the state, and the checkpoint is progress and never authority; one that records "unknown" is a failure the next plan recomputes, and reviewers read it as an open defect
- Consequences: `--resume` of such a stop re-plans from the repository, as it already does without a checkpoint
- Falsifier: a stop on an unknown read view whose progress cannot be recovered from the repository

## 6. Validation

### Strategy

Integration tests over real bundles seen failing first: a sole-member spec sealed with the criterion, approved, then done must read approved and fresh; a done spec sealed without it must read as before. Every done spec of this repository is verified by the old and the new engine: nothing approved may fall and nothing superseded may rise.

### Deterministic checks

`go test -race` of the tests named in the trace; the strict matrix in a clean clone.

### Requirement trace
- R1 [satisfied] test:TestADoneSpecSealedWithTheGateKeepsItsPlanAndItsApproval check:done-spec-keeps-its-sealed-plan-integration
- R2 [satisfied] test:TestADoneSpecSealedWithoutTheGateGainsNone test:TestTheSpecAnswersForTheExitGate
- R3 [satisfied] test:TestAnOpenSpecStillOwesTheGate test:TestEachExitGateHasItsOwnCriterion
- R4 [satisfied] test:TestTheStopOnUnstableInputsClearsAnOlderCheckpointAndSaysWhereTheStateIs test:TestAStopOnAnUnknownViewOfADoneScopeLeavesNoCheckpoint
- R5 [satisfied] test:TestEveryOtherStopStillRecordsItsCheckpoint
- R6 [satisfied] test:TestADoneSpecSealedWithTheGateKeepsItsPlanAndItsApproval

### Known gaps
- Retained approvals backed by a delegated run: `ReviewRunStillGoverns` requires the run's bundle to equal what the scope prepares today, so a done spec whose inputs drifted after sealing, whatever the cause, cannot retain a run-backed approval. This spec removes the cause that every sole-member spec hit at the transition to done; routine drift later is left to the follow-up below.
- The no-checkpoint stop is covered at the decision (`closeoutStopLeavesNoCheckpoint`) and at the helper both entry paths share, which clears an older checkpoint and names where the state is (`stopOnUnstableInputs`, the path of a call made after the stop and the path inside the apply loop); an end-to-end apply that produces an unknown read view is not covered, because no fixture of this repository can provoke one without a Git failure mid-apply (a Git that cannot be read at all fails the plan instead).
- Measured: 17 done specs of this repository gain the criterion; `review verify` gives the same answer for each with the engine before and after (closed, fresh, approved).

---

## 7. Final Report

### Delivered scope
A done spec that is the only member of a milestone with a declared exit gate keeps the gate criterion in its review plan when the governing sealed bundle carries it, so the bundle it was sealed with still represents its inputs and the attested approval stays fresh across the transition to done (R1). A done spec sealed without the criterion, a spec of several members, a milestone without a gate and a spec of a terminal roadmap are prepared as before (R2); an open spec still owes the criterion (R3). `pose close --apply` and `--resume` write no checkpoint, and clear the one they hold, when the terminal verification of a done scope stops on an unknown governance read view (R4); every other stop records as before (R5). The attestation rules did not change (R6). Left out: retaining run-backed approvals over later drift, and any change to how a reviewer judges the gate.

### Files and modules changed
- pose-mcp/internal/pose (review_plan.go, closeout_plan.go, done_spec_sealed_plan_test.go)
- pose-mcp/internal/cli (closeout_plan.go, closeout_unstable_inputs_test.go)
- ADR a-milestone-of-one-member-is-that-member (amendment); the validation matrix gains `done-spec-keeps-its-sealed-plan-integration`

### Validation executed
- Command: `pose validate --strict --jobs 4` on a clean clone of the reviewed head, and the suites of `internal/pose`, `internal/cli` and `internal/mcpserver`
- Result: recorded by the sealed validation evidence; independent delegated review by a different vendor, concluded with `pose review conclude`.

### Residual risks
- The criterion is kept by the criterion id found in the governing bundle's plan: a bundle sealed with a gate that the roadmap later changed keeps a plan whose description is the current one, not the sealed one, and reads as drifted, which is what it would have done before.
- The checkpoint rule is keyed on the plan's read view not being current; a plan that reports another reason for a blocked verify still records.

### Follow-ups

- [open] Let a done spec retain an approval backed by a delegated run over routine drift after sealing: `ReviewRunStillGoverns` compares the run's bundle with what the scope prepares today, which a completed scope never matches once anything moved. (owner:@pose-maintainers crit:medium review:2027-01-08)
