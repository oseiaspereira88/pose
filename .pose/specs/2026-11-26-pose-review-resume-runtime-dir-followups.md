---
slug: pose-review-resume-runtime-dir-followups
status: in-progress        # draft | in-progress | done | superseded | abandoned; blocked = operational wait, not terminal
created_at: 2026-11-26
completed_at:        # stamped on the transition to status: done
supersedes:          # slug of the superseded spec (when applicable)
depends_on:          # prerequisites, inline list: other-spec, milestone:<roadmap>/<id>, roadmap:<slug>
remediates:          # optional explicit links: spec:slug@defect-fix, finding:rva-<16hex>/F1@revert (never inferred from depends_on)
priority: 2
components: pose-mcp
task_type: bugfix
surface: standard
delivers: capability:review-resume-runtime-dir-followups
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
- R2: A run interrupted under an engine that named its runtime directory at random (its `safety.runtime_id` differs from the attempt's stable name) shall be resumed into a checkout its session can find: the continuation reuses the recorded name, or the resume refuses and points to a handoff (rounds 3 N11, 4 N18, 5 N23).
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
- created: .pose/specs/2026-11-26-pose-review-resume-runtime-dir-followups.md
- modified: pose-mcp/internal/pose/review_dispatch.go
- modified: pose-mcp/internal/pose/review_run_continuity.go
- modified: pose-mcp/internal/pose/delegation_handoff.go
- modified: pose-mcp/internal/pose/review_run_runtime_dir_test.go
- modified: pose-mcp/internal/cli/review_run_continuity_test.go
- modified: pose-mcp/internal/pose/review_rounds.go
- modified: pose-mcp/internal/cli/delegate_rounds_test.go

### Delivery targets
- capability:review-resume-runtime-dir-followups module:pose-mcp profile:composed-capability entrypoint:pose-mcp/internal/cli/delegate.go

### Accepted limits
- L1: A recorded runtime name is a path component taken from the run ledger, which the repository holds. Accepted because only names of the form `pose-review-run-<1–40 letters or digits>` are used, always below the temporary directory, and the directory is still created exclusively and never followed.

### API/contract changes
None: run records and briefs are unchanged; a continuation now reads the runtime name its predecessor recorded.

### Data/storage changes
None.

### Technical risks
A resume waits up to two seconds for another resume's claim before it reports an existing directory.

## 4. Tasks

### Planning
- [x] Confirm each requirement against the current engine before starting.

### Implementation
- [x] Implement each requirement with a test that fails before it.

### Validation
- [ ] `pose validate --carry` and `pose check --strict`.

## 5. Decisions

### Decision D1
- Basis: R1
The attempt's runtime directory is the lock: a continuation creates it, exclusively, before it claims the predecessor, and removes it if the claim fails. A resume that cannot create it has claimed nothing, so the predecessor stays resumable; when the directory is there, a claim that appears within two seconds means another resume holds it, and the continuation is reported pending. The plan-time check waits for a claim the same way.

### Decision D2
- Basis: R2, R3
The runtime directory of a continuation is named by what its predecessor recorded in `safety.runtime_id`, so a session finds the checkout where it saw it, also when an earlier engine named it at random; a round that continues its thread takes the name of the previous round's run, unless that path is already there (a leftover or a live run): then the round opens a new session in its own directory, recording why, before its attempt is reserved. Only an attempt's first run derives the name from the attempt id. This answers R2 by resuming instead of refusing.

## 6. Validation

### Strategy
Each requirement gets a regression test shown failing before its code.

### Deterministic checks

#### Test
- Command: `go test ./...` (in `pose-mcp`), then `pose validate --carry`.

### Execution log
- 2026-11-27: D1 in `RunReviewDispatch` (`takeContinuationRuntimeDir`) and in the plans (`awaitContinuationClaim`); D2 by `runtimeSourceOf` and `runtimeSourceAmong`; the refusal test asserts the resume's own wording (`then resume again`). `TestRunContinuityTakesTheRuntimeDirectoryBeforeTheClaim`, `TestRunContinuityRuntimeNameFollowsTheSession` and `TestRunContinuityResumedReviewFindsItsSealedCheckoutAtTheSamePath` fail with each guard mutated out (the recorded name, the thread's source, the name's path, the exclusive take, the verb).
- 2026-11-27: review round 1 (rrn-5be6bc059866ee70) requested changes. N1: a continuation that fails before it owns its cleanup removes the directory it took (`TestRunContinuityAFailedContinuationLeavesNoRuntimeDirectory` fails with the removal mutated out). N2: a round does not continue a session whose path is held; it opens a new session with the reason, before reservation (D2). `TestReviewRoundsKeepTheCheckoutPathOfTheirThread` dispatches a continued round, compares `safety.runtime_id` with the previous round's (N3, R3 end to end), and holds the path for the next round; it fails with the held-path check or the dispatch's use of the thread's name mutated out. N4 and N5 are follow-ups.

### Results summary
The full `pose-mcp` suite passes; the canonical result is refreshed with `pose validate --carry`.

### Requirement trace
- R1 [satisfied] test:TestRunContinuityAFailedContinuationLeavesNoRuntimeDirectory, test:TestRunContinuityTakesTheRuntimeDirectoryBeforeTheClaim, test:TestRunContinuityResumedReviewFindsItsSealedCheckoutAtTheSamePath
- R2 [satisfied] test:TestRunContinuityRuntimeNameFollowsTheSession
- R3 [satisfied] test:TestRunContinuityRuntimeNameFollowsTheSession, test:TestReviewRoundsKeepTheCheckoutPathOfTheirThread
- R4 [satisfied] test:TestRunContinuityResumedReviewFindsItsSealedCheckoutAtTheSamePath

### Known gaps
None.

## 7. Final Report

### Delivered scope
A continuation takes the attempt's runtime directory before its claim, and every continuation, including a run from an earlier engine and a round continuing its thread, recreates the checkout under the name its session saw.

### Files and modules changed
`pose-mcp` review dispatch, resume and handoff plans, and their tests.

### Validation executed
`go test ./...` in `pose-mcp`; `pose check --strict`; `pose validate --carry --json-out .pose/results/delivery-validation.json`.

### Residual risks
L1 above.

### Follow-ups
Origin: `pose-review-resume-keeps-the-sealed-checkout` (done 2026-11-26).
- [open] Let the wait for another resume's claim report a completed continuation as such (and to a handoff as already continued), honour cancellation, read the ledger once per poll cheaply, and turn a claim conflict after the winner finished into the recorded result (review round 1, N4). (owner:@pose-maintainers crit:low review:2026-12-27)
- [open] Record, in the run, a continued round whose runtime name could not be read from the ledger, instead of moving its checkout silently (review round 1, N5). (owner:@pose-maintainers crit:low review:2026-12-27)
