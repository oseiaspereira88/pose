---
slug: pose-validate-refuses-empty-go-checks
status: in-progress
created_at: 2026-10-31
supersedes:
depends_on: pose-validate-concurrent-steps
remediates:
priority: 2
components: pose-mcp
task_type: bugfix
delivers: capability:validate-refuses-empty-go-checks
---

# Spec: A Go check that ran no test does not pass

## 1. Intent

### Goal

A validation check that selects Go tests by name and ran none of them fails, instead of passing on Go's exit 0.

### Business value

`go test -run <pattern>` exits 0 when the pattern matches nothing, printing `[no tests to run]`, so a renamed or deleted test silently turns its check's evidence into nothing. `pose-validation-matrix-dead-test-patterns` keeps dead names out of plain patterns at the source; patterns it does not judge, and projects without that guard, are only caught at run time. The check's own output says what happened; validate ignores it. This was the part of `pose-validate-shared-go-test-run` with value of its own (its decision D1).

### Constraints

- Each check stays an execution of its own, and only its output is read.
- A check over several packages passes when any of them ran tests: a pattern usually selects tests of some of its packages.

### Non-goals

- Judging each alternative of a pattern at run time, skipped tests or dynamic subtests.
- Checks that do not select by name, and `go test -json` output.

## 2. Requirements

### Functional

- R1: A `go test` check with a `-run` pattern whose command succeeded shall fail, with `limit_state: no-tests` and a message naming the pattern, when its output reports packages and none of them ran a test (`[no tests to run]` or `[no test files]`), whatever the flags and whether the result was cached.
- R2: A check where at least one package ran tests, a check with no `-run`, a non-Go check and a check whose output reports no package shall keep their outcome.
- R3: The check's severity decides as for any failure: an optional check counts as an optional failure.

### Non-functional

- The output is read line by line in bounded memory.

### Security

- None.

### Compatibility

- A project with a Go check that selects nothing starts failing, which is the point; the changelog says so.

## 3. Technical Plan

### Affected areas

`internal/cli/validate.go`, the manuals.

### Artifacts

- created: .pose/specs/2026-10-31-pose-validate-refuses-empty-go-checks.md
- created: .pose/starts/pose-validate-refuses-empty-go-checks.json
- created: .pose/changelogs/unreleased/pose-validate-refuses-empty-go-checks.md
- modified: .pose/specs/2026-10-30-pose-validate-shared-go-test-run.md
- modified: .pose/specs/2026-10-30-pose-validation-matrix-dead-test-patterns.md
- modified: .pose/indexes/validation-matrix.json
- modified: pose-mcp/internal/cli/validate.go
- modified: pose-mcp/internal/cli/validate_results.go
- modified: pose-mcp/internal/cli/validation_matrix_patterns_test.go
- created: pose-mcp/internal/cli/validate_empty_go_checks_test.go
- modified: POSE.md
- modified: locales/pt-BR/POSE.md
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md

### Delivery targets

- capability:validate-refuses-empty-go-checks module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go

### API/contract changes

`limit_state: no-tests`.

## 4. Tasks

- [ ] Read the check's output for packages that ran tests
- [ ] Fail a check that ran none
- [ ] Independent review and closeout

## 5. Decisions

## 6. Validation

### Strategy

Real `go test` fixtures: a pattern matching nothing fails, with and without `-v`, `-race` and a cached result; a pattern matching one of two packages passes; a package without test files alone fails; an optional check is an optional failure; a check with no `-run` and a non-Go check printing the marker keep their outcome. Then the real matrix.

### Requirement trace

## 7. Final Report

### Delivered scope

Not started.

### Residual risks

None yet.

### Follow-ups
