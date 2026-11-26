---
slug: pose-review-resume-keeps-the-sealed-checkout
status: in-progress        # draft | in-progress | done | superseded | abandoned; blocked = operational wait, not terminal
created_at: 2026-11-25
completed_at:        # stamped on the transition to status: done
supersedes:          # slug of the superseded spec (when applicable)
depends_on:          # prerequisites, inline list: other-spec, milestone:<roadmap>/<id>, roadmap:<slug>
remediates:          # optional explicit links: spec:slug@defect-fix, finding:rva-<16hex>/F1@revert (never inferred from depends_on)
priority: 2
components: pose-mcp
task_type: bugfix
surface: standard
delivers: capability:review-resume-keeps-the-sealed-checkout
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
- R2: A regression test shall resume an interrupted run whose first checkout was removed and see the session work in the recreated checkout.
- R3: The attempt's runtime directory shall be created exclusively: an existing path, a link included, shall fail the run with its reason, never be shared or followed.

### Non-functional
Existing behaviour outside the stated requirements is unchanged.

### Security
No new input crosses a trust boundary without the existing confinement.

### Compatibility
Runs recorded by earlier engines keep their meaning.

## 3. Technical Plan

### Affected areas
`pose-mcp` review dispatch: the runtime directory of a run (`reviewRunRuntimeDir` in `internal/pose/review_dispatch.go`).

### Artifacts
- created: .pose/specs/2026-11-25-pose-review-resume-keeps-the-sealed-checkout.md
- modified: pose-mcp/internal/pose/review_dispatch.go
- created: pose-mcp/internal/pose/review_run_runtime_dir_test.go
- modified: pose-mcp/internal/cli/review_run_continuity_test.go

### Delivery targets
- capability:review-resume-keeps-the-sealed-checkout module:pose-mcp profile:composed-capability entrypoint:pose-mcp/internal/cli/delegate.go

### Accepted limits
- L1: The path is derived from the temporary directory of the process that dispatches, so a resume run with another `TMPDIR` recreates the checkout elsewhere and the session meets the old failure. Accepted because the dispatcher's environment is the operator's and the failure is the same one this spec removes for the usual case, a resume from the same shell.
- L2: The path is predictable, so another local user can create it first and block the attempt's runs. Accepted because the directory is created exclusively and never followed, so such a path only fails the run with its reason and is never shared.

### API/contract changes
None: the run records, briefs and adapter commands are unchanged; only where the run directory is created changes.

### Data/storage changes
None.

### Technical risks
A run whose cleanup failed leaves the directory, so the next run of the attempt fails until it is removed; resume already refuses a predecessor whose cleanup is incomplete.

## 4. Tasks

### Planning
- [x] Confirm how the continuation prompt and the worktree path are built.

### Implementation
- [x] Implement each requirement with a test that fails before it.

### Validation
- [ ] `pose validate --carry` and `pose check --strict`.

## 5. Decisions

### Decision D1
- Basis: R1, R3
Keep the checkout where the session last saw it rather than telling the session a new path: the continuation sends the same sealed brief, whose wire digest the provenance checks, so adding a path notice would change what is verified; a path stable per attempt changes nothing the run records. Every run of an attempt uses `pose-review-run-<first 32 hex of the attempt id>` under the temporary directory, created exclusively; a run with no attempt keeps a random name.

## 6. Validation

### Strategy
Each requirement gets a regression test shown failing before its code.

### Deterministic checks

#### Test
- Command: `go test ./...` (in `pose-mcp`), then `pose validate --carry`.

### Execution log
- 2026-11-26: `reviewRunRuntimeDir` gives every run of an attempt the same runtime directory, created with `os.Mkdir` (no link followed, an existing path refused). `TestResumedReviewFindsItsSealedCheckoutAtTheSamePath` fails with the random name restored (the session was moved from the first checkout); `TestTheAttemptRuntimeDirectoryIsCreatedExclusively` fails with `MkdirAll`.

### Results summary
The full `pose-mcp` suite passes; the canonical result is refreshed with `pose validate --carry`.

### Requirement trace
- R1 [satisfied] test:TestResumedReviewFindsItsSealedCheckoutAtTheSamePath
- R2 [satisfied] test:TestResumedReviewFindsItsSealedCheckoutAtTheSamePath
- R3 [satisfied] test:TestTheAttemptRuntimeDirectoryIsCreatedExclusively

### Known gaps
None.

## 7. Final Report

### Delivered scope
A resumed review run recreates its sealed checkout at the path the interrupted run used, so the resumed session keeps working in it.

### Files and modules changed
`pose-mcp` review dispatch and its tests.

### Validation executed
`go test ./...` in `pose-mcp`; `pose check --strict`; `pose validate --carry --json-out .pose/results/delivery-validation.json`.

### Residual risks
L1 and L2 above.

### Follow-ups
None.
