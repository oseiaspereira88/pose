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
delivers:
---

# Spec: The slowest CLI tests stop waiting on real time and run in parallel where they can

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

### Delivery targets

### API/contract changes

None.

## 4. Tasks

- [x] Inventory of real-time waits and global-state use in `internal/cli` tests
- [x] Gates and events instead of waits in the slowest tests
- [x] Why the package stays serial (R2)
- [x] `-race` and `-count=3` stability
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
