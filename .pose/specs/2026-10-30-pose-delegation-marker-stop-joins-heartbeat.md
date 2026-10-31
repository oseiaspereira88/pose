---
slug: pose-delegation-marker-stop-joins-heartbeat
status: in-progress
created_at: 2026-10-30
supersedes:
depends_on: pose-delegation-result-before-marker
remediates:
priority: 2
components: pose-mcp
task_type: bugfix
delivers: capability:delegation-marker-stop-joins-heartbeat
---

# Spec: A stopped run's heartbeat cannot bring its marker back

## 1. Intent

### Goal

Once a run's active marker is removed, nothing of that run writes it again.

### Business value

`markDelegationActive` starts a goroutine that rewrites the marker's heartbeat; `stop` closes the goroutine's channel and removes the marker without waiting for it. A heartbeat already past its byte check when stop runs renames its new marker into place after the removal, and a finished run reads as running, then stalled, and can block its continuation. Found by the independent review of `pose-delegation-result-before-marker` (F04), reproduced there with a scheduling probe on the code before it.

### Constraints

- The marker's compare-and-swap heartbeat is unchanged.

### Non-goals

- Other writers of the active directory.

## 2. Requirements

### Functional

- R1: `stop` shall return only after the run's heartbeat goroutine has ended, and remove the marker after that, so a heartbeat in flight cannot recreate it.
- R2: A stop after the goroutine already ended — on a cancel request — shall not block.

### Non-functional

- A stop waits at most for one heartbeat write.

### Security

- None.

### Compatibility

- None.

## 3. Technical Plan

### Affected areas

`internal/pose/delegation_active.go`.

### Artifacts

- created: .pose/specs/2026-10-30-pose-delegation-marker-stop-joins-heartbeat.md
- created: .pose/starts/pose-delegation-marker-stop-joins-heartbeat.json
- modified: pose-mcp/internal/pose/delegation_active.go
- modified: pose-mcp/internal/pose/writer_publication.go
- created: pose-mcp/internal/pose/delegation_marker_stop_test.go
- modified: .pose/indexes/validation-matrix.json

### Delivery targets

- capability:delegation-marker-stop-joins-heartbeat module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go

### API/contract changes

None. Internal test seam: `PublishBeforeReplace`, beside `PublishFault` and `PublishSyncHook`, called by an ExpectDigest publication after its byte check and before its rename.

## 4. Tasks

- [ ] Join the heartbeat goroutine in stop
- [ ] Regression shown failing before the fix
- [ ] Independent review and closeout

## 5. Decisions

## 6. Validation

### Strategy

A heartbeat held between its read and its publication while stop runs: on the old code the marker is back after stop; with the fix it is gone. A stop after a cancel request returns.

### Requirement trace

## 7. Final Report

### Delivered scope

Not started.

### Residual risks

None yet.

### Follow-ups
