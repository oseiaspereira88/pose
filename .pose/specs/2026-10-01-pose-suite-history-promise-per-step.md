---
slug: pose-suite-history-promise-per-step
status: done
created_at: 2026-10-01
completed_at: 2026-10-01
components: pose-mcp
task_type: bugfix
changelog: none
---

# Spec: Full-suite history promise per workflow step

## 1. Intent

Prevent one step's history declaration from authorizing another full-suite run.

## 2. Requirements

- R1: Every full-suite invocation shall declare its own release-history signal.
- R2: Comments and declarations in unrelated steps shall not satisfy the guard.

## 3. Technical Plan

### Artifacts

- modified: pose-mcp/internal/version/workflow_history_depth_test.go
- created: .pose/specs/2026-10-01-pose-suite-history-promise-per-step.md

## 4. Tasks

- [x] Track declarations and suite invocations within each step.
- [x] Test duplicate suite runs, unrelated declarations and commented promises.

## 5. Decisions

Keep the existing workflow scanner's supported YAML forms; full YAML parsing is
separate scope. This change narrows an incorrect job-wide declaration boundary.

## 6. Validation

Synthetic two-step workflow with only the first declaration must fail detection;
two explicit declarations must pass. Run the guard on every actual workflow.

### Execution log

- 2026-10-01: targeted synthetic guards and production-source/workflow guards
  passed. No native runner or release execution was required.

### Requirement trace

- R1 [satisfied] test:TestJobsRunningTheGoSuiteCheckOutFullHistory test:TestHistoryPromiseBelongsToEachSuiteStep
- R2 [satisfied] test:TestHistoryPromiseBelongsToEachSuiteStep

## 7. Final Report

Implementation validated and closed through the governed review gate on 2026-10-01. Canonical strict validation passed 48/48 checks at f41b5aa; review bundle `rvb-4c25f3ec247a586d` was fresh and approved before `pose close`.
