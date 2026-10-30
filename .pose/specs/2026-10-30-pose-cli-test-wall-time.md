---
slug: pose-cli-test-wall-time
status: in-progress
created_at: 2026-10-30
supersedes:
depends_on:
remediates:
priority: 3
components: pose-mcp
task_type: refactor
delivers: capability:cli-test-wall-time
---

# Spec: The slowest CLI tests stop waiting on real time

## 1. Intent

### Goal

Cut the wall time of `internal/cli` tests, which dominates every validation, by removing real-time waits from the slowest tests.

### Business value

`go test ./internal/cli` takes 192 s of the uncached 280 s suite. Its 131 delegation tests alone take 100 s, strictly one after another (sum of test times 109 s ≈ package time); the fifteen slowest take 63 s: MCP-over-stdio journeys (`TestDelegateToolsOverMCP*`, 4.7–10.9 s), detach and wait loops (`TestDelegateDetachAndWait` 5.2 s, `TestDelegateWaitsForTheFirstReset` 4.1 s, `TestDelegateWatchFollowsLive` 3.5 s), and review-dispatch runs (`TestReviewDispatch*` 2.6–5.4 s). Measured 2026-10-30 with `go test -json`.

### Constraints

- No test loses an assertion; timing changes go through injectable clocks or poll intervals, not shorter sleeps that can flake.
- A test marked parallel must not share process-global state (working directory, environment, package variables) with another.

### Non-goals

- Changing product behaviour.

## 2. Requirements

### Functional

- R1: Waits in the slowest delegation tests shall be driven by an injectable clock or poll interval that tests set, so no test sleeps longer than it needs to observe the state it asserts.
- R2: The tests shall stay serial, and the reason shall be recorded: `runPose` and `runPoseSplit` call `Main` through `inDir`, which changes the process's working directory, so nearly every test of the package shares process-global state; making `Main` take its directory is a follow-up.

### Non-functional

- `go test ./internal/cli -count=1` shall take measurably less than its 2026-10-30 baseline (191 s package time) on the same machine, the before and after per-test timings kept as evidence, and the changed tests shall pass under `-race -count=3`.

### Security

- None.

### Compatibility

- None.

## 3. Technical Plan

### Affected areas

`internal/cli` delegation, MCP and review-dispatch tests and the helpers they share.

### Artifacts

- created: .pose/specs/2026-10-30-pose-cli-test-wall-time.md
- created: .pose/starts/pose-cli-test-wall-time.json
- created: .pose/specs/2026-10-30-pose-cli-test-wall-time.amendments.jsonl
- modified: pose-mcp/internal/cli/delegate_wait.go
- modified: pose-mcp/internal/pose/review_dispatch.go
- modified: pose-mcp/internal/cli/delegate_mcp_test.go
- modified: pose-mcp/internal/cli/delegate_observation_review_test.go
- modified: pose-mcp/internal/cli/delegate_wait_test.go
- modified: pose-mcp/internal/cli/delegate_watch_test.go
- modified: pose-mcp/internal/cli/review_dispatch_test.go
- modified: .pose/indexes/validation-matrix.json

### Delivery targets

- capability:cli-test-wall-time module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go

### API/contract changes

No public surface. Internal: `delegate wait` polls the run's marker every 100 ms instead of 500 ms; `posemodel.AdapterPipeGrace` (default 5 s, unchanged) holds the adapter pipe grace so a test can shorten it; and `POSE_TEST_DELEGATE_WAIT_SIGNAL`, an unsupported test seam, makes a waiter create — exclusively, once, empty — an entry named after its pid in the named directory when it starts waiting on a running run. It controls nothing about the run, and an existing entry is never truncated, followed or opened.

## 4. Tasks

- [x] Inventory of real-time waits and global-state use in `internal/cli` tests
- [x] Gates and events instead of waits in the slowest tests
- [x] Why the package stays serial (R2)
- [x] `-race` and `-count=3` stability
- [x] Independent review and closeout

## 5. Decisions

## 6. Validation

### Strategy

Before/after `go test -json` timings of `internal/cli`; `-race -count=3` for flakiness; the list of tests left serial with reasons.

### Requirement trace

- R1 [satisfied] check:cli-test-wall-time-integration test:TestDelegateToolsOverMCPHoldAgainstTheFirstReview test:TestDelegateDetachAndWait test:TestADetachedRunThatDoesNotHandBackIsStopped test:TestDelegateWatchFollowsLive test:TestTheWaitAcknowledgementNeverTouchesAnExistingEntry report:.pose/results/delivery-validation.json commit:1dc147707c59f910c2cc8bbae2e745cd041e714c — The slowest delegation tests hold their runs on gates and release them on observed events (live events, waiter acknowledgements, process exit, watch output) instead of sleeping; the reviewer's privacy and waiter mutants fail them.
- R2 [satisfied] check:cli-test-wall-time-integration report:.pose/results/delivery-validation.json commit:1dc147707c59f910c2cc8bbae2e745cd041e714c — The package stays serial; the reason (inDir changes the process working directory) is recorded, and making Main take its directory is a follow-up.

## 7. Final Report

### Delivered scope

Real-time waits removed from the slowest `internal/cli` delegation tests: package time 191 s → 156 s on 1085 tests, serial, same machine (per-test timings kept outside the checkout). `delegate wait` polls every 100 ms. Independently reviewed by Codex over three rounds.

### Residual risks

The package remains serial, so most of its time is the sum of many short tests. A pre-existing race in review dispatch (the active marker is removed before the continuation receipt is written) can make a wait refuse; it was observed once in about twenty race runs (F1).

### Follow-ups

- [open] F1: write the continuation result receipt before removing a run's active marker, keeping the marker when the receipt fails, so a waiter never reads a terminal run without its receipt (review round 1, F01; reproduced on code before this spec). Next engine bugfix spec. (owner:@oseiaspereira88 crit:high review:2026-11-06)
- [open] F2: make `Main` take its working directory instead of tests changing the process's, so `internal/cli` tests can run in parallel. (owner:@oseiaspereira88 crit:medium review:2026-11-27)
