---
slug: check-worker-count-is-the-machines
status: in-progress
created_at: 2026-09-21
completed_at:
supersedes:
depends_on: parse-the-delivery-index-once-per-content
priority: 2
components: pose-mcp
task_type: feature
delivers: surface:check-worker-count
---

# Spec: the gate's worker count is the machine's to state

## 1. Intent

### Goal

`pose check` sizes its worker pool from `runtime.NumCPU()` with no way to say
otherwise. Add `POSE_CHECK_WORKERS`, and leave the default where measurement puts it.

### Business value

The right pool size is a property of the machine, not of the gate. A container pinned
to fewer cores than the host reports will oversubscribe; an operator who wants the gate
to leave room for other work has no way to ask. `parse-the-delivery-index-once-per-content`
recorded that gap as a residual risk: "the worker count is `runtime.NumCPU()` with no
flag: a machine where that is wrong has no way to say so yet."

### Constraints

The default must not change on a difference that noise explains. A gate is the wrong
place to fail over an environment variable, so an unusable value is ignored rather than
rejected. Adding an environment variable changes the composition contract and has to be
declared there.

### Non-goals

Changing the default. Eight workers measured faster than sixteen over two runs and does
not over five; see the trace. Making the pool adaptive, or exposing it as a command flag
— the environment is where a machine states its own shape, and a flag would invite
per-run tuning of something that is not a tuning knob.

Reducing the 7,248 Git subprocesses one run spawns. That is the real remaining lever and
it is a different change: batching object reads through `git cat-file --batch` touches
the structural detector's correctness guards, and is out of scope here.

## 2. Requirements

- R1: `POSE_CHECK_WORKERS` sets the pool size, and the default stays the core count.
- R2: An unusable value — zero, negative, non-numeric, empty — is ignored and the
  default stands; surrounding whitespace is tolerated.
- R3: The pool never exceeds the item count and is never below one, with or without the
  override.
- R4: The new variable is declared in the composition contract, so a consumer composing
  the service sees it.

## 3. Technical Plan

`checkWorkerCount(count)` resolves the pool: `runtime.NumCPU()`, overridden by the
environment when the value parses as a positive integer, then clamped by the item count.
`composition-contract.json` is regenerated.

### Artifacts

- created: .pose/specs/2026-09-21-check-worker-count-is-the-machines.md
- created: .pose/changelogs/unreleased/check-worker-count-is-the-machines.md
- created: pose-mcp/internal/cli/check_workers_test.go
- modified: pose-mcp/internal/cli/check_parallel.go
- modified: composition-contract.json
- modified: .pose/indexes/validation-matrix.json
- modified: .pose/results/delivery-validation.json
- modified: .pose/indexes/delivery-integrity.json
- modified: .pose/indexes/spec-graph.json

### Delivery targets

- surface:check-worker-count module:pose-mcp/internal/cli profile:cli-surface entrypoint:pose-mcp/cmd/pose/main.go

### Rollout and reversal

Absent variable, unchanged behaviour. Reverting removes the override and the contract
line.

## 4. Tasks

- [x] Measure whether the default should move before touching it.
- [x] Add the override, ignore unusable values, keep the clamps.
- [x] Declare the variable in the composition contract.

## 5. Decisions

The default stays `runtime.NumCPU()`. Two runs suggested eight was about a second
faster; five runs put the medians 0.49 seconds apart with overlapping ranges, so the
difference is noise at this sample size. Baking in a three-percent constant that noise
explains would be worse than shipping the knob, and it would have been a performance
claim the measurement does not support.

An unusable value is ignored rather than refused. A structural gate that fails because
`POSE_CHECK_WORKERS=many` was exported in a shell profile would be a worse failure than
running with the default.

## 6. Validation

| Scenario | Command (from pose-mcp) | Expected evidence |
| --- | --- | --- |
| Default and clamps | `go test ./internal/cli -run CheckWorkerCountDefaults -count=1` | Core count by default; never above the item count, never below one |
| Override | `go test ./internal/cli -run CheckWorkerCountOverride -count=1` | The value is used, and still clamped by the item count |
| Unusable values | `go test ./internal/cli -run CheckWorkerCountIgnores -count=1` | Zero, negative, non-numeric and blank all fall back to the default |
| Contract declared | `go test ./internal/version -run TestCompositionContract -count=1` | The contract on disk matches the repository |
| Registered producer | `pose validate --strict --module pose-mcp` | The dedicated check runs the worker-count corpus |

### Execution log

2026-09-21. The default was measured before being touched, five runs per point on a
sixteen-core machine:

| workers | min | median | max |
| --- | --- | --- | --- |
| 4 | 17.5s | 18.3s | 18.5s |
| 8 | 16.1s | 16.3s | 16.6s |
| 16 | 16.5s | 16.8s | 17.2s |

Four workers is clearly worse. Eight against sixteen is 0.49 seconds of median with
overlapping ranges — sixteen's fastest run beat eight's slowest — so the default is left
alone. An earlier two-run comparison had suggested eight was a second faster; it did not
survive five runs, and that correction is the reason this spec changes no default.

Above the core count it degrades rather than plateaus, measured earlier in
`parse-the-delivery-index-once-per-content`: the wall is flat at 18.7 seconds for
thirty-two and sixty-four workers while the summed item time nearly triples between
sixteen and sixty-four and the slowest item grows from 4.65 to 5.5 seconds. The work is
process spawning and CPU, not idle waiting, so extra workers contend.

Adding the variable broke `TestCompositionContract`, which is the contract doing its
job: the composition contract enumerates every `POSE_*` variable the service reads. The
reported diff was inspected before regenerating and is exactly one line,
`POSE_CHECK_WORKERS`. No manual documents environment variables, so nothing else needed
updating — checked rather than assumed, because a documented path that does not exist
fails the post-install gate in this repository.

Defect injection: accepting a non-positive override fails the unusable-value case with
`POSE_CHECK_WORKERS="0" = 1, want the default 16`, and removing the item clamp fails the
default case with `with 3 items = 16, want 3`.

### Requirement trace

- R1 [satisfied] surface:check-worker-count evidence:integration
  check:check-workers-integration test:TestCheckWorkerCountDefaultsToCoresAndClamps
  test:TestCheckWorkerCountOverride — the default is the core count and the variable
  replaces it
- R2 [satisfied] surface:check-worker-count evidence:integration
  check:check-workers-integration test:TestCheckWorkerCountIgnoresUnusableValues — zero,
  negative, non-numeric, blank and padded values all behave as stated
- R3 [satisfied] surface:check-worker-count evidence:integration
  check:check-workers-integration test:TestCheckWorkerCountDefaultsToCoresAndClamps
  test:TestCheckWorkerCountOverride — clamped by item count in both paths
- R4 [satisfied] surface:check-worker-count evidence:integration
  check:check-workers-integration test:TestCompositionContract — the contract on disk
  matches the repository, with the one added line inspected before regenerating

## 7. Final Report

### Scope delivered

A machine can state its own pool size. The default is unchanged, and the measurement
that decided not to change it is recorded rather than the guess that nearly did.

### Residual risks

The measurement is one machine, sixteen cores, one repository. The numbers say where the
plateau is here and nothing about a container with two cores or a repository ten times
this size — which is the argument for the override rather than for a new default.

### Follow-ups

- [open] Batch Git object reads through cat-file --batch: 7,248 spawns per run, about a
  quarter of the remaining item work, and the only lever left that does not depend on
  cores (owner:@pose-maintainers crit:medium review:2026-11-21)
