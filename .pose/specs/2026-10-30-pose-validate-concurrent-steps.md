---
slug: pose-validate-concurrent-steps
status: done
created_at: 2026-10-30
supersedes:
depends_on:
remediates:
priority: 2
components: pose-mcp
task_type: feature
delivers: surface:validate-jobs
completed_at: 2026-10-31
---

# Spec: `pose validate` runs independent checks concurrently and says where its time went

## 1. Intent

### Goal

Cut the wall time of a strict validation by running independent checks of the matrix at the same time, within a bound the machine can carry, and make the time each check took visible.

### Business value

A strict validation of pose-dist runs 162 checks one after another: 453 s with the full-suite result cached, about 700 s (the "~13 min" a maintainer sees) without it — the uncached `go test ./...` alone is 280 s of package time. Every closeout pays it at least once, often twice, and a reviewer waits on it. The checks are independent processes; the machine that ran them has 16 cores, and the run used one at a time.

Measured on 2026-10-30, HEAD `8aaa408c` (`.pose/results/delivery-validation.json`, `duration_seconds` per check): 157 of 162 checks are `go test`; median 1.06 s (mostly compile and link of the test binary); the five slowest take 229 s (delegation-surface 105 s, full suite 56 s cached, delegated-review-capability `-race` 36.5 s, cli-mcp-operation-results 11.5 s, git-dispatch-snapshot 11.1 s).

### Constraints

- Results, their order in the report, and the outcome of each check do not depend on the concurrency used.
- A check that cannot share the machine with another (port, global state, heavy memory) can say so and runs alone.
- The bound is explicit and lowerable: memory, not cores, is what limited this machine (about 4–7 GB free).

### Non-goals

- Changing which tests a check runs or how its evidence is attributed (spec `pose-validate-shared-go-test-run`).
- Making individual tests faster (spec `pose-cli-test-wall-time`).

## 2. Requirements

### Functional

- R1: `pose validate` shall run up to `--jobs N` checks at once (default: a bound derived from the cores and memory available to the process, cgroup limits included, never below 1, at most 8, and at most 4 when memory cannot be read), starting them in matrix order and recording every result, changed-scope skips included, in matrix order.
- R2: A matrix check marked `exclusive: true`, a boolean key `pose check` accepts, shall run with no other check running, after every earlier check has ended.
- R3: `--jobs 1` shall reproduce today's serial execution exactly, and the report shall record the concurrency used.
- R4: At the end of a run, validate shall print the slowest checks with their durations and the run's wall time next to the sum of check durations.
- R5: A failing, timed-out or output-limited check shall be reported exactly as in a serial run; cancelling the run stops every running check's process group and its preparatory commands, starts nothing more, records those checks `limit_state: cancelled`, makes the run `fail` with exit 1 whatever their severity, and leaves a second signal its default effect.
- R6: A check's `when` shall be read as the check is about to start, so at `--jobs 1`, with `--verbose`, or past an exclusive barrier an earlier check can decide it.

### Non-functional

- On pose-dist, a strict run with `--jobs 4` shall take about half the wall time of `--jobs 1` on the same machine, cache state and Go settings, measured and recorded in the final report; the slowest single check (the full suite) is its floor.

### Security

- None: the same commands run with the same environment.

### Compatibility

- The results schema gains the concurrency used and the wall time; existing readers ignore unknown fields.

## 3. Technical Plan

### Affected areas

`internal/cli/validate.go` (step loop, renderer), the matrix schema (`exclusive`), the manuals.

### Artifacts

- created: .pose/specs/2026-10-30-pose-validate-concurrent-steps.md
- created: .pose/starts/pose-validate-concurrent-steps.json
- created: .pose/specs/2026-10-30-pose-validate-concurrent-steps.amendments.jsonl
- modified: pose-mcp/internal/cli/check.go
- modified: pose-mcp/internal/cli/validate_root_and_nodemodules_test.go
- created: .pose/changelogs/unreleased/pose-validate-concurrent-steps.md
- modified: .pose/indexes/validation-matrix.json
- modified: pose-mcp/internal/cli/validate.go
- modified: pose-mcp/internal/cli/validate_results.go
- modified: pose-mcp/internal/cli/help_catalog.go
- created: pose-mcp/internal/cli/validate_concurrent_test.go
- modified: POSE.md
- modified: locales/pt-BR/POSE.md
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md

### Delivery targets

- surface:validate-jobs module:pose-mcp profile:cli-surface entrypoint:pose-mcp/cmd/pose/main.go

### API/contract changes

`--jobs`; matrix key `exclusive`; result fields `jobs` and `wall_seconds`; `limit_state: cancelled` for a check a cancelled run stopped or kept from starting. An interrupt or SIGTERM now stops running checks' process groups, serial runs included.

## 4. Tasks

- [x] Bounded worker pool with ordered results
- [x] `exclusive` checks
- [x] Timing summary
- [x] Measure serial vs concurrent on pose-dist
- [x] Independent review and closeout

## 5. Decisions

## 6. Validation

### Strategy

Fixture matrices with sleeping and failing checks prove ordering, exclusivity, timeout and cancellation under concurrency; `--jobs 1` output compared byte for byte with the serial engine; the real pose-dist matrix timed at `--jobs 1` and `--jobs 4`.

### Requirement trace

- R1 [satisfied] surface:validate-jobs evidence:integration check:validate-concurrent-steps-integration check:validate-jobs-reachability test:TestValidateRunsChecksConcurrentlyWithinJobs test:TestValidateKeepsMatrixOrderAndOutcomes test:TestChangedScopeSkipsKeepMatrixOrder test:TestDefaultValidationJobsFollowsTheBudget report:.pose/results/delivery-validation.json commit:b4da8dc222cee4957b705b4c237e2a5a34ae06c6 — Checks run up to --jobs at once, every result keeps matrix order, and the default follows the cores and memory available to the process through its mounted cgroups.
- R2 [satisfied] check:validate-concurrent-steps-integration test:TestValidateExclusiveCheckRunsAlone test:TestCheckAcceptsExclusiveAsABoolean report:.pose/results/delivery-validation.json commit:b4da8dc222cee4957b705b4c237e2a5a34ae06c6 — An exclusive check runs after every earlier check has ended, with nothing else running, and `pose check` accepts the key as a boolean.
- R3 [satisfied] check:validate-concurrent-steps-integration test:TestValidateRunsChecksConcurrentlyWithinJobs test:TestValidateKeepsMatrixOrderAndOutcomes report:.pose/results/delivery-validation.json commit:b4da8dc222cee4957b705b4c237e2a5a34ae06c6 — --jobs 1 runs serially, outcomes match a concurrent run, and the report records jobs and wall_seconds.
- R4 [satisfied] check:validate-concurrent-steps-integration test:TestValidateRunsChecksConcurrentlyWithinJobs report:.pose/results/delivery-validation.json commit:b4da8dc222cee4957b705b4c237e2a5a34ae06c6 — The run ends with its wall time, the sum of its checks and the slowest checks.
- R5 [satisfied] check:validate-concurrent-steps-integration check:validate-jobs-reachability test:TestValidateKeepsMatrixOrderAndOutcomes test:TestValidateCancelStopsRunningChecks test:TestValidateCancelStopsThePreparatoryGoList test:TestACancelledRunIsNeverASuccess report:.pose/results/delivery-validation.json commit:b4da8dc222cee4957b705b4c237e2a5a34ae06c6 — Failures, timeouts and output limits read as in a serial run; a cancelled run stops its process groups and preparatory commands, starts nothing more and fails with exit 1 whatever the severity.
- R6 [satisfied] check:validate-concurrent-steps-integration test:TestValidateWhenIsReadBeforeTheCheckStarts report:.pose/results/delivery-validation.json commit:b4da8dc222cee4957b705b4c237e2a5a34ae06c6 — when is read as a check starts, so an earlier check decides it serially, with --verbose and past an exclusive barrier.

## 7. Final Report

### Delivered scope

`pose validate` runs independent checks concurrently within `--jobs`, keeps matrix order, runs `exclusive` checks alone, reports its timing and stops cleanly on a signal. Measured before the review's corrections, uncached, GOMAXPROCS=2 and GOFLAGS=-p=1: 825 s at --jobs 1, 400 s and 447 s at --jobs 4, 166/166 each, at least 6.4 GB free; 285 s at the default without those limits, a different setting and not a controlled comparison. The floor is the slowest check, the full suite. Independently reviewed by Codex over four rounds, which found no interference between pose-dist's checks.

### Residual risks

A project's own checks can share mutable state that pose-dist's do not (generated files, coverage, build directories): `exclusive` is the remedy, and `--jobs 1` the old behaviour. The memory bound is a heuristic, not a reservation. Signal ownership is process-wide for in-process callers, and descendants that leave a process group can outlive a cancel.

### Follow-ups

- [open] F1: discover resources natively off Linux and qualify other platforms; the memory bound stays a heuristic (review round 2, follow-up 1). (owner:@oseiaspereira88 crit:medium review:2026-11-27)
- [open] F2: give in-process callers an explicit context instead of a process-wide signal subscription, and bound shutdown against escaped descendants and inherited pipes (review round 2, follow-up 2). (owner:@oseiaspereira88 crit:medium review:2026-11-27)
- [open] F3: show live progress for the checks running at once, say that jobs is a maximum and that a cancelled check never started counts as executed, state what wall_seconds covers, and add --jobs to the website reference (review round 2, follow-up 3). (owner:@oseiaspereira88 crit:low review:2026-11-27)
- [open] F4: add repository regressions for cancellation around slots and barriers, concurrent output limits, the projection writers, a second signal and a comparison with the serial engine's output (review round 2, follow-up 4). (owner:@oseiaspereira88 crit:medium review:2026-11-27)
- [open] F5: record Go settings, effective resources, binary identity and cache policy with benchmark timings (review round 2, follow-up 5). (owner:@oseiaspereira88 crit:low review:2026-11-27)
