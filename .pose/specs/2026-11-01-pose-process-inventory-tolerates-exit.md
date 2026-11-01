---
slug: pose-process-inventory-tolerates-exit
status: in-progress
created_at: 2026-11-01
supersedes:
depends_on: pose-delegated-adapter-containment
remediates:
priority: 1
components: pose-mcp
task_type: bugfix
delivers: capability:process-inventory-tolerates-exit
---

# Spec: A process that exits during the inventory is not a failure

## 1. Intent

### Goal

Confirming that an adapter's process group has ended is not disturbed by unrelated processes ending at the same moment.

### Business value

After killing an adapter's group, the engine walks `/proc` to confirm nothing of it remains. A process that exits between opening its `stat` and reading it answers `ESRCH`, not "not found", and the walk took that for an incomplete inventory. The run then failed as "process group termination failed; descendants may remain" and its cleanup was withheld, though nothing remained. Security failed on `main` at `ff50c49b` this way: `TestAdapterContainmentCallerCancellation`, 84 ms into a run, with a real reviewer dispatch exposed to the same race on any busy machine.

### Constraints

- A process that is really alive in the group still holds the run; the check only stops misreading a process that is gone.
- Any other read error still reports an incomplete inventory.

### Non-goals

- Other Unix ports, which report signal delivery only.
- Changing the one-second wait or the inventory bound.

## 2. Requirements

### Functional

- R1: When a process exits before its `stat` is read, whether the open or the read reports it, the group inventory shall treat it as gone and continue, and a lock's owner shall read as not running.
- R2: A read error that does not mean the process is gone shall still report the inventory incomplete.

### Non-functional

- No additional reads per process.

### Security

- The containment guarantee stands: a live member of the group is never skipped by this change.

### Compatibility

- None: a run that spuriously failed now completes as what it was.

## 3. Technical Plan

### Affected areas

`internal/pose/review_dispatch_unix.go`, `internal/pose/start_lock_linux.go`.

### Artifacts

- created: .pose/specs/2026-11-01-pose-process-inventory-tolerates-exit.md
- created: .pose/starts/pose-process-inventory-tolerates-exit.json
- created: .pose/changelogs/unreleased/pose-process-inventory-tolerates-exit.md
- modified: .pose/indexes/validation-matrix.json
- modified: pose-mcp/internal/pose/review_dispatch_unix.go
- modified: pose-mcp/internal/pose/start_lock_linux.go
- created: pose-mcp/internal/pose/process_inventory_unix_test.go

### Delivery targets

- capability:process-inventory-tolerates-exit module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go

### API/contract changes

None.

## 4. Tasks

- [ ] Treat a vanished process as gone
- [ ] Independent review and closeout

## 5. Decisions

### Decision D1
- Basis: R1
- Minimal option: retry the read on any error
- Selected option: name the two answers that mean the process is gone, `ErrNotExist` from the open and `ESRCH` from the read
- Rationale: a process that vanished is a fact about that process, while a retry would hide errors that do mean the inventory is incomplete
- Consequences: other read errors keep failing the run closed
- Falsifier: an inventory that reports incomplete for a process that exited, or skips one that is alive

## 6. Validation

### Strategy

The real sequence on a real process: open its `stat`, kill and reap it, read. The inventory run while processes exit continuously. The gate fails on the previous predicate.

### Requirement trace

- R1 [planned] capability:process-inventory-tolerates-exit evidence:integration check:process-inventory-tolerates-exit-integration test:TestProcessThatExitedBeforeItsReadIsGone test:TestInventoryHoldsWhileProcessesExit — A process gone before its read does not fail the inventory.
- R2 [planned] check:process-inventory-tolerates-exit-integration test:TestProcessGoneIsOnlyThoseTwoAnswers — Other errors still fail it.

## 7. Final Report

### Delivered scope

### Residual risks

### Follow-ups
