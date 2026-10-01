---
slug: pose-history-quiet-tolerated-verdict
status: in-progress
created_at: 2026-10-01
completed_at:
components: pose-mcp
task_type: bugfix
---

# Spec: One quiet verdict for tolerated history failures

## 1. Intent

Fix the machine-channel contract exposed when the active validation history has
unstaged changes: quiet output printed an initial failure and final tolerated verdict.

## 2. Requirements

- R1: history-check --quiet shall print one verdict in strict and tolerant modes
  when history JSONL is untracked or modified; exit semantics shall remain unchanged.

## 3. Technical Plan

### Artifacts

- modified: pose-mcp/internal/cli/historycheck.go
- created: pose-mcp/internal/cli/history_quiet_test.go
- created: .pose/specs/2026-10-01-pose-history-quiet-tolerated-verdict.md
- created: .pose/changelogs/unreleased/pose-history-quiet-tolerated-verdict.md

## 4. Tasks

- [x] Suppress the intermediate human line in quiet mode.
- [x] Verify both severities against synthetic untracked and modified history.

## 5. Decisions

Preserve ordinary human and JSON output, strict failure and tolerated success.

## 6. Validation

Observed full-suite failure: two quiet lines with one unstaged JSONL. Reproduce
using a temporary Git fixture, then assert exactly one line and the existing exit
code in each mode. Run the actual whole-repository machine-channel regression.

### Execution log

- Synthetic untracked/modified JSONL in strict and tolerant modes passed.
- Whole-repository machine-channel regression passed with unstaged history.

### Requirement trace

- R1 [satisfied] test:TestHistoryQuietPrintsOneToleratedOrStrictVerdict

## 7. Final Report

In progress.
