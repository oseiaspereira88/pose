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
- R3: The attempt's runtime directory shall be created exclusively: an existing path, a link included, shall fail the run with its reason, never be shared or followed; a resume beside it shall be refused before the continuation is claimed.

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
- modified: pose-mcp/internal/pose/review_run_continuity.go
- modified: pose-mcp/internal/pose/delegation_handoff.go
- modified: POSE.md
- modified: locales/pt-BR/POSE.md
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md

### Delivery targets
- capability:review-resume-keeps-the-sealed-checkout module:pose-mcp profile:composed-capability entrypoint:pose-mcp/internal/cli/delegate.go

### Accepted limits
- L1: The path is derived from the temporary directory of the process that dispatches, so a resume run with another `TMPDIR` recreates the checkout elsewhere and the session meets the old failure. Accepted because the dispatcher's environment is the operator's and the failure is the same one this spec removes for the usual case, a resume from the same shell.
- L2: The path is predictable, so another local user can create it first and block the attempt's runs. Accepted because the directory is created exclusively and never followed, so such a path only fails the run with its reason and is never shared.
- L3: A process that escaped an earlier run of the attempt knows the next run's paths, including its output file outside the checkout, which random names used to hide. Accepted because such a process already runs as the dispatching user and can reach any of that user's paths; on Linux the review run shim ends the run's subtree, and elsewhere the escape is the limit `pose-review-run-reaping-beyond-linux` addresses.

### API/contract changes
None: the run records, briefs and adapter commands are unchanged; only where the run directory is created changes.

### Data/storage changes
None.

### Technical risks
A run whose cleanup failed leaves the directory, and resume already refuses such a predecessor. A directory present although the last run cleaned up (another process holding the path) makes resume and handoff refuse before they claim the run, naming the directory, so removing it restores them; a run that fails before anything is in the directory removes it.

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
Keep the checkout where the session last saw it rather than telling the session a new path: the continuation sends the same sealed brief, whose wire digest the provenance checks, so adding a path notice would change what is verified; a path stable per attempt changes nothing the run records. Every run of an attempt uses `pose-review-run-<first 10 hex of the attempt id>` under the temporary directory, created exclusively; a run with no attempt keeps a random name. The name is no longer than the random one, because the reviewer's `TMPDIR` is this directory and a sandboxed reviewer creates Unix sockets under it, whose paths are bounded.

## 6. Validation

### Strategy
Each requirement gets a regression test shown failing before its code.

### Deterministic checks

#### Test
- Command: `go test ./...` (in `pose-mcp`), then `pose validate --carry`.

### Execution log
- 2026-11-26: `reviewRunRuntimeDir` gives every run of an attempt the same runtime directory, created with `os.Mkdir` (no link followed, an existing path refused). `TestResumedReviewFindsItsSealedCheckoutAtTheSamePath` fails with the random name restored (the session was moved from the first checkout); `TestTheAttemptRuntimeDirectoryIsCreatedExclusively` fails with `MkdirAll`.
- 2026-11-26: review round 1 (rrn-7adbffa551b9acf2, Claude Sonnet 5.5) ended without a draft: its Bash sandbox did not start. From its reading: N1, resume refuses a leftover runtime directory before claiming the continuation (the test fails with the check mutated out); N2, the R2 adapter fails unless the recreated checkout holds the spec; N3, the manual in both locales states the stable path, the `TMPDIR` condition and the remedy; N4, a run that fails right after creating the directory removes it; N5 is declared as L3.
- 2026-11-26: the full suite caught a repeated resume refused beside the directory; the check now runs only for a continuation still to start. Review round 2 (rrn-c5d57ff07d69ec1a) requested changes: N7, the manual, the technical risks and the messages no longer promise that removing a directory left by an unfinished cleanup restores resume (that predecessor is refused anyway); N6, a handoff checks the directory before it claims the run (the test fails with the check mutated out); N9, the regression tests are named `TestRunContinuity…`, so the matrix's continuity pattern runs them; N10, the exclusivity test isolates `TMP` and `TEMP` as well. N8 is a follow-up.
- 2026-11-26: review round 3 (rrn-95a0d363d2041e1d) approved, but its conclusion was refused: its required tools were `deferred`, because, as in round 1, the reviewer's Bash sandbox failed to start ("Failed to create bridge sockets"). Cause: this spec's 32-hex directory name made the reviewer's `TMPDIR` 70 bytes long, and the Claude sandbox's Unix socket paths under it passed the ~108-byte bound; the random name was 48. The name now uses 10 hex digits, as long as the random suffix, and `TestRunContinuityAttemptRuntimeDirectoryIsCreatedExclusively` fails when it is longer than a random one. The round's follow-ups N11–N15 are recorded with its attestation once a round concludes.

### Results summary
The full `pose-mcp` suite passes; the canonical result is refreshed with `pose validate --carry`.

### Requirement trace
- R1 [satisfied] test:TestRunContinuityResumedReviewFindsItsSealedCheckoutAtTheSamePath
- R2 [satisfied] test:TestRunContinuityResumedReviewFindsItsSealedCheckoutAtTheSamePath
- R3 [satisfied] test:TestRunContinuityAttemptRuntimeDirectoryIsCreatedExclusively, test:TestRunContinuityResumedReviewFindsItsSealedCheckoutAtTheSamePath

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
L1, L2 and L3 above.

### Follow-ups
- [open] Make the continuation claim and the runtime-directory check one step, so a resume that loses a race is told the continuation is pending instead of being asked to remove a live run's directory (review round 2, N8). (owner:@pose-maintainers crit:low review:2026-12-26)
