---
slug: pose-review-resume-residuals
status: in-progress        # draft | in-progress | done | superseded | abandoned; blocked = operational wait, not terminal
created_at: 2026-11-27
completed_at:        # stamped on the transition to status: done
supersedes:          # slug of the superseded spec (when applicable)
depends_on:          # prerequisites, inline list: other-spec, milestone:<roadmap>/<id>, roadmap:<slug>
remediates:          # optional explicit links: spec:slug@defect-fix, finding:rva-<16hex>/F1@revert (never inferred from depends_on)
priority: 3
components: pose-mcp
task_type: bugfix
surface: standard
delivers: capability:review-resume-residuals
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
- created: .pose/specs/2026-11-27-pose-review-resume-residuals.md
- modified: pose-mcp/internal/pose/review_dispatch.go
- modified: pose-mcp/internal/pose/review_rounds.go
- modified: pose-mcp/internal/pose/review_run_continuity.go
- modified: pose-mcp/internal/pose/delegation_handoff.go
- modified: pose-mcp/internal/pose/review_run_runtime_dir_test.go
- modified: pose-mcp/internal/cli/delegate_rounds_test.go
- created: pose-mcp/internal/pose/review_resume_race_test.go
- modified: POSE.md
- modified: locales/pt-BR/POSE.md
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md

### Delivery targets
- capability:review-resume-residuals module:pose-mcp profile:composed-capability entrypoint:pose-mcp/internal/cli/delegate.go

### Accepted limits
- L1: Two guards have no failing test: an lstat error other than not-exist counting as held, and a continued round whose ledger read fails. Accepted because each needs a permission or ledger fault the fixtures cannot make deterministically, and each fails toward refusing or recording, never toward sharing a directory.

### API/contract changes
A continued round whose checkout could not follow its session records it among the run's `safety.limitations`. A resume or handoff that meets another's recorded continuation returns that record.

### Data/storage changes
None.

### Technical risks
A resume waits up to two seconds for another resume's claim, now only while a claim file exists.

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
A continuation that loses its take or its claim to another answers with what that other one recorded, under the same rules as the plans: a resume returns another resume's record and refuses a handoff's ("handed off; it is not resumed"); a handoff returns its own agent's handoff and refuses anything else ("already continued"). A refused plan's answer is returned instead of the raw publication conflict, and a cancelled wait returns its context's error.

### Decision D2
- Basis: R3
A thread path taken only after the round was planned keeps the continued session: the session was chosen at plan time, and failing now would waste the reserved attempt. The round works in its own directory and records the move among `safety.limitations`, which the manual states; a path held at plan time still opens a new session.

## 6. Validation

### Strategy
Each requirement gets a regression test shown failing before its code.

### Deterministic checks

#### Test
- Command: `go test ./...` (in `pose-mcp`), then `pose validate --carry`.

### Execution log
- 2026-11-27: R1 `awaitContinuationClaim` returns a recorded continuation as the result, reads the ledger only once a claim file exists and stops with its context; a claim conflict replans to the recorded or pending continuation. R2/R3 `runtimeSourceFor` says why a continued round cannot take the previous name, and a path taken after the plan falls back to the round's own directory; both are recorded in `safety.limitations`. R4 the reason names the held path, and an lstat error other than not-exist counts as held. R5 the rounds paragraph in both locales and the embedded copies. `TestRunContinuityTheClaimWaitStopsAndHeldPathsAreNamed` and `TestReviewRoundsKeepTheCheckoutPathOfTheirThread` fail with the cancellation, the named path or the fallback mutated out; L1 lists what has no test.
- 2026-11-27: review round 1's first run timed out running the full suite (the note had lost its time-budget paragraph); the rerun (rrn-612ac36bfe01cb24) requested changes. N1: recorded continuations are checked for kind (D1, `continuationKindMismatch`). N2: test hooks before the take and the claim make the races deterministic: `TestRunContinuityALosingResumeReturnsTheRecordedContinuation`, `TestRunContinuityALostClaimReleasesTheDirectory` and `TestRunContinuityARecordedContinuationOfAnotherKindIsRefused` fail with the recorded return, the directory release or either kind rule mutated out. N3: a cancelled wait returns its context's error. N4: a refused replan's answer is returned. N5: D2 and the manual. N6: the held path is evaluated once.

### Results summary
The full `pose-mcp` suite passes; the canonical result is refreshed with `pose validate --carry`.

### Requirement trace
- R1 [satisfied] test:TestRunContinuityARecordedContinuationOfAnotherKindIsRefused, test:TestRunContinuityALosingResumeReturnsTheRecordedContinuation, test:TestRunContinuityTheClaimWaitStopsAndHeldPathsAreNamed, test:TestRunContinuityConcurrentResumeLaunchesOnce
- R2 [satisfied] (accepted limit L1)
- R3 [satisfied] test:TestReviewRoundsKeepTheCheckoutPathOfTheirThread
- R4 [satisfied] test:TestRunContinuityTheClaimWaitStopsAndHeldPathsAreNamed, test:TestReviewRoundsKeepTheCheckoutPathOfTheirThread
- R5 [satisfied] (manual, both locales)
- R6 [satisfied] test:TestRunContinuityALosingResumeReturnsTheRecordedContinuation, test:TestRunContinuityALostClaimReleasesTheDirectory, test:TestRunContinuityTakesTheRuntimeDirectoryBeforeTheClaim, test:TestReviewRoundsKeepTheCheckoutPathOfTheirThread

### Known gaps
The random-name end-to-end resume and the handoff plan's pending branch remain untested; see Follow-ups.

## 7. Final Report

### Delivered scope
The claim wait returns recorded continuations and stops with its context; a continued round that cannot keep its checkout path works in its own directory and says so; held paths are named; the manual lists the condition.

### Files and modules changed
`pose-mcp` review dispatch, rounds, resume and handoff plans; the manual in both locales and its embedded copies.

### Validation executed
`go test ./...` in `pose-mcp`; `pose check --strict`; `pose validate --carry --json-out .pose/results/delivery-validation.json`.

### Residual risks
L1 above.

### Follow-ups
Origin: `pose-review-resume-runtime-dir-followups` (done 2026-11-27).
- [open] Resume end to end a run whose recorded runtime name is a random one of an earlier engine, and reach the handoff plan's pending branch in a test (R6 in part). (owner:@pose-maintainers crit:low review:2026-12-27)
