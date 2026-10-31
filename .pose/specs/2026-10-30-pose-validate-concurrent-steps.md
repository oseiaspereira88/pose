---
slug: pose-validate-concurrent-steps
status: in-progress
created_at: 2026-10-30
supersedes:
depends_on:
remediates:
priority: 2
components: pose-mcp
task_type: feature
delivers: surface:validate-jobs
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

- [ ] Bounded worker pool with ordered results
- [ ] `exclusive` checks
- [ ] Timing summary
- [ ] Measure serial vs concurrent on pose-dist
- [ ] Independent review and closeout

## 5. Decisions

## 6. Validation

### Strategy

Fixture matrices with sleeping and failing checks prove ordering, exclusivity, timeout and cancellation under concurrency; `--jobs 1` output compared byte for byte with the serial engine; the real pose-dist matrix timed at `--jobs 1` and `--jobs 4`.

### Requirement trace

## 7. Final Report

### Delivered scope

Not started.

### Residual risks

None yet.

### Follow-ups
