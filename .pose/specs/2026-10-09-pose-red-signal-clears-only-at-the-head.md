---
slug: pose-red-signal-clears-only-at-the-head
status: done
created_at: 2026-10-09
completed_at: 2026-10-09
supersedes:
depends_on: pose-red-signals-reach-a-person
remediates: spec:pose-red-signals-reach-a-person@defect-fix
priority: 0
components: ci, release
task_type: bugfix
surface: minimal
changelog: none
delivers: governance:red-signal-clears-only-at-the-head
---

# Spec: A red signal clears only at main's head

## 1. Intent

### Goal

Keep a red-signal alert open until the workflow succeeds at the current head of main, so a late success of an older commit cannot clear the alert of a newer failure.

### Business value

Found on its first real alert: the freeze commit `2cf8d4b1` turned CI red on main and the alert opened issue #134, assigned to the owner; minutes later the CI of the previous commit `4b833300` finished green and closed it, while main was still red. The person the alert exists to reach was told the problem was gone.

### Constraints

Release tags keep their behaviour: Release runs only on a tag, so its next success on a tag clears its alert.

## 2. Requirements

### Functional

- R1: A success on main shall close the open alert only when its commit is main's current head; otherwise the alert shall stay open.
- R2: A success on a release tag and a failure on main shall behave as before.

## 3. Technical Plan

### Artifacts

- created: .pose/specs/2026-10-09-pose-red-signal-clears-only-at-the-head.md
- modified: .github/workflows/failure-alert.yml
- modified: pose-mcp/internal/version/failure_alert_test.go

### Delivery targets

- governance:red-signal-clears-only-at-the-head module:. profile:release-governance entrypoint:.github/workflows/failure-alert.yml

## 5. Decisions

No material decision: the head check uses the job token already granted to the step, with no new permission.

## 6. Validation

### Strategy

`TestFailureAlertClearsOnlyAtMainsHead` runs the workflow step as the runner would, with a recording `gh`: a success of an older commit leaves the alert open, a success at main's head and a release success on its tag close it, and a failure on main opens one. It fails on the workflow before this change.

### Deterministic checks

#### Test
- Command: `cd pose-mcp && go test ./internal/version -run FailureAlert`
- Scope: workflows
- Expected: pass

### Requirement trace

- R1 [satisfied] governance:red-signal-clears-only-at-the-head check:red-signal-contract evidence:integration test:TestFailureAlertClearsOnlyAtMainsHead
- R2 [satisfied] governance:red-signal-clears-only-at-the-head check:red-signal-contract evidence:integration test:TestFailureAlertClearsOnlyAtMainsHead

## 7. Final Report

### Delivered scope

The alert step compares a successful run's commit with main's head before closing; the step itself is now exercised by a test.

### Residual risks

- Reviewed in the same session that implemented it, with no separate reviewer execution: declared, not independent.

### Follow-ups
