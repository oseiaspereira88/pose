---
slug: pose-delegation-result-before-marker
status: in-progress
created_at: 2026-10-30
supersedes:
depends_on: pose-cli-test-wall-time
remediates:
priority: 1
components: pose-mcp
task_type: bugfix
delivers: capability:delegation-result-before-marker
---

# Spec: A run's active marker goes only after its attempt result is published

## 1. Intent

### Goal

A waiter never sees a delegated run end without the receipt of its result.

### Business value

`RunReviewDispatchContext` registers the deferred publication of the attempt result (the attempt receipt, or the continuation result receipt) before it marks the run active, and the deferred removal of the active marker after. Deferred calls run in reverse, so the marker was removed first. A `pose delegate wait` that read the run in that window found a terminal record without its receipt and refused: "the run history could not be read whole, so the run's end cannot be told (unread: missing continuation result receipt …)". Found by the independent review of `pose-cli-test-wall-time` (F01); reproduced on code before that spec by delaying the receipt by one second, and seen once in about twenty race runs of `TestDelegateToolsOverMCPContinueStoppedRuns`.

### Constraints

- No retry or loosened integrity check hides a missing receipt.
- A run whose result cannot be published keeps its marker, as a run without a record already does.

### Non-goals

- Other writers of delegation state.

## 2. Requirements

### Functional

- R1: The attempt result of a run — its attempt receipt, or the result receipt of a continuation — shall be published before the run's active marker is removed, and the marker shall be kept when that publication fails.
- R2: A run that ends before it is marked active shall still publish its attempt result, once.

### Non-functional

- No extra file operation on the normal path.

### Security

- None.

### Compatibility

- None: only the order of two writes changes.

## 3. Technical Plan

### Affected areas

`internal/pose/review_dispatch.go`.

### Artifacts

- created: .pose/specs/2026-10-30-pose-delegation-result-before-marker.md
- created: .pose/starts/pose-delegation-result-before-marker.json
- modified: pose-mcp/internal/pose/review_dispatch.go
- created: pose-mcp/internal/pose/review_result_before_marker_test.go
- modified: .pose/indexes/validation-matrix.json

### Delivery targets

- capability:delegation-result-before-marker module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go

### API/contract changes

None. Internal test seam: `beforeRunResultPublished`, a package variable called with the run just before its result is published.

## 4. Tasks

- [ ] Publish the result inside the marker's deferred removal
- [ ] Regression shown failing before the fix
- [ ] Independent review and closeout

## 5. Decisions

## 6. Validation

### Strategy

A dispatch whose hook asserts, at the moment its result is published, that the run's marker is still present — failing on the old order — for a first attempt and a continuation; a run refused before it is marked still publishes its result once.

### Requirement trace

## 7. Final Report

### Delivered scope

Not started.

### Residual risks

None yet.

### Follow-ups
