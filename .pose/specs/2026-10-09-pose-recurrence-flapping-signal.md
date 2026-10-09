---
slug: pose-recurrence-flapping-signal
status: done
created_at: 2026-10-09
completed_at: 2026-10-09
supersedes:
depends_on: pose-recurrence-check-resolved-clusters
priority: 1
components: pose-mcp
task_type: feature
surface: minimal
delivers: capability:recurrence-flapping-signal
---

# Spec: recurrence-check reports a flapping task

## 1. Intent

### Goal

Report a task whose outcome keeps alternating between failing and passing in the window, as a warning that never gates.

### Business value

Origin: the open follow-up of `pose-recurrence-check-resolved-clusters` (crit medium), prioritized by the maintainer on 2026-10-09. Once a later pass settles a failure cluster, a task that fails and passes repeatedly reads clean, though its outcome is unstable.

### Constraints

Non-blocking in every mode: an unstable outcome is a reason to investigate, not a recurrence.

### Non-goals

Changing how recurrence is counted or which clusters resolve.

## 2. Requirements

### Functional

- R1: When consecutive runs of a task, across every `stable_hash`, switch between passing and not passing at least `--flap-threshold` times in the window (default 4), recurrence-check shall report a `flapping` warning and count it in `recurrence.flapping_keys`.
- R2: The warning shall never change the exit code, in `--strict` either.

## 3. Technical Plan

### Artifacts

- created: .pose/specs/2026-10-09-pose-recurrence-flapping-signal.md
- modified: pose-mcp/internal/cli/insights.go
- modified: pose-mcp/internal/cli/help_catalog.go
- modified: pose-mcp/internal/cli/recurrence_check_clusters_test.go
- modified: POSE.md
- modified: locales/pt-BR/POSE.md
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md
- modified: .pose/indexes/validation-matrix.json
- modified: .pose/specs/2026-10-07-pose-recurrence-check-resolved-clusters.md
- created: .pose/changelogs/unreleased/pose-recurrence-flapping-signal.md

### Delivery targets

- capability:recurrence-flapping-signal module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go

## 5. Decisions

### Decision D1
- Date: 2026-10-09
- Context: a pass settles its cluster, so alternation is invisible to the recurrence count.
- Options considered: (a) count transitions per task across hashes and warn above a threshold; (b) fold alternation into the recurrence count.
- Decision: (a).
- Rationale: (b) would gate on instability and undo the resolved-cluster semantics the source spec established.
- Consequences: the default threshold, two full fail-pass cycles, is tunable per run.

## 6. Validation

### Strategy

`TestRecurrenceFlappingIsReportedNotGated` alternates six runs and expects a warning with five transitions and exit 0 under `--strict`; a raised threshold and a single recovery report nothing. On the real histories it reports `validate-native` flapping in both repositories: 11 transitions in 50 runs on pose-dist and 10 in 40 on Harne8.

### Deterministic checks

#### Test
- Command: `cd pose-mcp && go test ./internal/cli -run TestRecurrence`
- Scope: pose-mcp
- Expected: pass

### Requirement trace

- R1 [satisfied] capability:recurrence-flapping-signal check:recurrence-flapping-integration evidence:integration test:TestRecurrenceFlappingIsReportedNotGated
- R2 [satisfied] capability:recurrence-flapping-signal check:recurrence-flapping-integration evidence:integration test:TestRecurrenceFlappingIsReportedNotGated

## 7. Final Report

### Delivered scope

`pose recurrence-check` reports `flapping` tasks and `recurrence.flapping_keys`, with `--flap-threshold`, documented in both manuals and the help. It found `validate-native` unstable in pose-dist and Harne8.

### Residual risks

- Reviewed in the same session that implemented it, with no separate reviewer execution: declared, not independent.

### Follow-ups

- [open] Investigate why validate-native alternates in both repositories (11 transitions in 50 runs on pose-dist, 10 in 40 on Harne8); environment-dependent checks such as the release-listing journey and disk-bound builds are the first suspects (owner:@pose-maintainers crit:medium review:2026-10-30)
