---
slug: pose-cli-test-wall-time
status: draft
created_at: 2026-10-30
supersedes:
depends_on:
remediates:
priority: 3
components: pose-mcp
task_type: refactor
delivers:
---

# Spec: The slowest CLI tests stop waiting on real time and run in parallel where they can

## 1. Intent

### Goal

Cut the wall time of `internal/cli` tests, which dominates every validation, by removing real-time waits from the slowest tests and running independent tests in parallel.

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
- R2: Tests that do not touch process-global state shall call `t.Parallel()`; those that do shall be listed with the reason.
- R3: A test helper shall refuse `t.Chdir`/`t.Setenv` in a parallel test, as the Go runtime already does, without masking it.

### Non-functional

- `go test ./internal/cli -count=1` shall take at most half its 2026-10-30 time on the same machine, and `-race` and `-count=3` runs shall pass.

### Security

- None.

### Compatibility

- None.

## 3. Technical Plan

### Affected areas

`internal/cli` delegation, MCP and review-dispatch tests and the helpers they share.

### Artifacts

### Delivery targets

### API/contract changes

None.

## 4. Tasks

- [ ] Inventory of real-time waits and global-state use in `internal/cli` tests
- [ ] Injectable waits in the slowest tests
- [ ] `t.Parallel()` where safe
- [ ] `-race` and `-count=3` stability
- [ ] Independent review and closeout

## 5. Decisions

## 6. Validation

### Strategy

Before/after `go test -json` timings of `internal/cli`; `-race -count=3` for flakiness; the list of tests left serial with reasons.

### Requirement trace

## 7. Final Report

### Delivered scope

Not started.

### Residual risks

None yet.

### Follow-ups
