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
- created: pose-mcp/internal/pose/review_result_return_paths_test.go
- modified: .pose/indexes/validation-matrix.json

### Delivery targets

- capability:delegation-result-before-marker module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go

### API/contract changes

None. Internal test seam: `beforeRunResultPublished`, a package variable called with the run just before its result is published.

## 4. Tasks

- [x] Publish the result inside the marker's deferred removal
- [x] Regression shown failing before the fix
- [x] Independent review and closeout

## 5. Decisions

## 6. Validation

### Strategy

A dispatch whose hook asserts, at the moment its result is published, that the run's marker is still present — failing on the old order — for a first attempt and a continuation; a run refused before it is marked still publishes its result once.

### Requirement trace

- R1 [satisfied] check:delegation-result-before-marker-integration test:TestARunsResultIsPublishedBeforeItsMarkerGoes test:TestARunsReturnPathsPublishOnceAndKeepTheirMarker test:TestAContinuationsReturnPathsPublishOnceAndKeepTheirMarker report:.pose/results/delivery-validation.json commit:8cb9d00db53ef6d1678a1274f193395fefc6997b — The result is published inside the marker's deferred removal, before it; a failed publication keeps the marker. The regression failed for a first attempt and a handoff on the old order; the reviewer's one-second receipt-delay probe no longer makes a wait refuse.
- R2 [satisfied] check:delegation-result-before-marker-integration test:TestARunsReturnPathsPublishOnceAndKeepTheirMarker report:.pose/results/delivery-validation.json commit:8cb9d00db53ef6d1678a1274f193395fefc6997b — A run that ends before it is marked (marking fails) still publishes its result once; panics and record failures publish none and keep the marker.

## 7. Final Report

### Delivered scope

A delegated run's attempt result — attempt receipt or continuation receipt — is published before its active marker is removed, once, on every return path. Independently reviewed by Codex (approved in round 1), who found the defect.

### Residual risks

A heartbeat in flight when a run stops can recreate the marker it removed (F1, pre-existing), leaving a false running or stalled state.

### Follow-ups


- [open] F1: join a run's heartbeat watcher, or share its lock, before the active marker is removed, so an in-flight heartbeat cannot recreate a removed marker (review round 1, F04; reproduced with a scheduling probe on code before this spec). Next engine spec. (owner:@oseiaspereira88 crit:medium review:2026-11-06)
- [open] F2: decide whether a run's private live events are kept, bounded, when its record cannot be published, or document them as ephemeral even then (review round 1, F03). (owner:@oseiaspereira88 crit:low review:2026-11-27)
