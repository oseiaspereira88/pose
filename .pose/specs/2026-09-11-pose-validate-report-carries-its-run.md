---
slug: pose-validate-report-carries-its-run
status: in-progress
completed_at:
created_at: 2026-09-11
supersedes:
depends_on: pose-one-follow-up-format
priority: 0
components: pose-mcp
task_type: bugfix
delivers:
---

# Spec: `pose validate --report` records the run it made

## 1. Intent

### Goal
The report `pose validate --report` writes shall record that run's commands and
result, and derive its outcome from them.

### Business value
A review of pose#93 pointed at the validation report its evidence commit
carried: an outcome of `pass`, while the commands read `_Fill manually_`, the
results `_No validation output detected_`, and the outcome's source `manual`.
It could not substantiate the pass it recorded.

It was not that report. Every `validate-native` report in this repository since
2026-08-13 says the same. `pose validate --report` called `pose report` with its
verdict as `--outcome` and no validation output, and `pose report` then looked
for `.pose/reports/pose-validate.latest.log` — a file the shell-based validation
used to write and native validation never has. The report found nothing, every
time, and recorded a verdict nothing beside it supported.

Regenerating the report, which is what the review asked for, would have
produced the same empty report. The generator had to be fixed first.

### Constraints
- The run's output is handed to the report in memory: writing the log into the
  tree would leave an untracked file after every run.
- `pose report --validate-output` keeps working as before.

### Non-goals
- Rewriting the reports already recorded. They are history, and say what the
  generator produced at the time.

---

## 2. Requirements

### Functional
- R1: The report shall list the commands the run executed and its Result line.
- R2: When the run's Result line agrees with the verdict, the outcome shall be
  recorded as derived, not manual.
- R3: No log file shall be written to the tree.

### Non-functional
- `pose report`'s own flags and defaults are unchanged.

---

## 3. Technical Plan

### Affected areas
- `pose-mcp/internal/cli/validate.go` — captures its own stdout when `--report`
- `pose-mcp/internal/cli/report.go` — accepts that output in place of a log file

### Artifacts
- created: .pose/specs/2026-09-11-pose-validate-report-carries-its-run.md
- renamed: .pose/changelogs/unreleased/pose-validate-report-carries-its-run.md -> .pose/changelogs/v5.0.2/pose-validate-report-carries-its-run.md
- modified: pose-mcp/internal/cli/validate.go
- modified: pose-mcp/internal/cli/report.go
- modified: pose-mcp/internal/cli/cli_test.go

### Technical risks
- The captured output is what validation prints to stdout, the format the log
  parser was written for. A change to that output's shape would empty the
  report again; the test pins the command and the Result line.

---

## 4. Tasks

### Implementation
- [x] Increment 1: Hand the run's output to the report (R1, R3)
- [x] Increment 2: Derive the outcome from it (R2)

### Validation
- [x] The test fails against the previous code with the symptoms the review saw

---

## 5. Decisions

### Decision 1
- Date: 2026-09-11
- Context: the report could read a log validation writes, or take the output in
  memory. POSE.md already names `pose-validate.latest.log` as a per-run artifact.
- Decision: in memory.
- Rationale: the file would appear untracked after every `--report` run in every
  instance. The documented artifact is one a pipeline captures and attaches, and
  a pipeline that wants it can still tee the output and pass `--validate-output`.

---

## 6. Validation

### Strategy
Reproduce the empty report, fix it, and regenerate this branch's evidence with
the fixed engine in strict mode.

### Deterministic checks

#### Test
- Command: `go test -count=1 ./...`
- Scope: `pose-mcp`
- Expected: exit 0

#### Gate
- Command: `pose check --strict`
- Scope: this instance
- Expected: exit 0

### Execution log
- Date: 2026-09-11
- Environment: local, Go 1.26
- Notes: against the previous code, TestValidateReportRecordsTheRunItMade fails
  on the command, the Result line and the derived outcome, and finds `_Fill
  manually_` and `_No validation output detected_` — the review's symptoms. With
  the fix it passes, and no log file is written.

### Results summary
- Successes: R1, R2, R3 verified.
- Failures: none.

### Requirement trace
- R1 [satisfied] <TestValidateReportRecordsTheRunItMade requires the check's command and `Result: SUCCESS` in the report, and neither placeholder>
- R2 [satisfied] <the same test requires `Outcome: pass (source: derived)`>
- R3 [satisfied] <the same test requires no `.pose/reports/pose-validate.latest.log`>

### Known gaps
- Reports recorded before this fix keep their empty sections.

---

## 7. Final Report

### Summary
A validation report now carries the run that justifies its outcome.

### Follow-ups
