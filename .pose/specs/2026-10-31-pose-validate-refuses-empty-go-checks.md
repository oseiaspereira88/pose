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

- Proof that a test ran: the rule reads Go's own summary. A test that prints a line in the exact form of a result, or Go's warning that no tests ran, can mislead it, as it misleads Go's summary.
- Judging each alternative of a pattern at run time, skipped tests or dynamic subtests.
- Rerunning checks with `-json`, which would change how every check runs and what it prints.

## 2. Requirements

### Functional

- R1: A `go test` check that selects tests by name — `-run` or `-skip` in any spelling Go accepts, in the plain form the manual lists, packages then a short list of flags, or unquoted `GOFLAGS` — and whose command succeeded shall fail, with `limit_state: no-tests` and a message naming the pattern, when Go's result lines report packages and every one of them as having run nothing (`[no tests to run]`, `[no test files]`, or the coverage line of a package without test files), cached or not.
- R2: A check where a package reported running tests, a check that selects nothing by name, a non-Go check, a check whose output has no result line, a mode whose success says nothing about tests (`-bench`, `-list`, `-c`, `-o`, `-exec`, `-json`), and any other form (another flag, a missing value, `-args`, `--`, a positional argument after the packages, quoted `GOFLAGS`) shall keep their outcome: a check that cannot be read for certain is not judged.
- R3: The check's severity decides as for any failure: an optional check counts as an optional failure.

### Non-functional

- The output is read line by line in bounded memory; a line of a test's output that is not in the exact form of a result does not count.

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
- modified: pose-mcp/internal/cli/testdata/adversarial/README.md
- modified: pose-mcp/internal/cli/validation_matrix_patterns_test.go
- created: pose-mcp/internal/cli/validate_empty_go_checks_test.go
- created: .pose/specs/2026-10-31-pose-validate-refuses-empty-go-checks.amendments.jsonl
- modified: POSE.md
- modified: locales/pt-BR/POSE.md
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md

### Delivery targets

- capability:validate-refuses-empty-go-checks module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go

### API/contract changes

`limit_state: no-tests`.

## 4. Tasks

- [x] Read the check's output for packages that ran tests
- [x] Fail a check that ran none
- [x] Independent review and closeout

## 5. Decisions

## 6. Validation

### Strategy

Real `go test` fixtures: a pattern matching nothing fails, with and without `-v`, `-race` and a cached result; a pattern matching one of two packages passes; a package without test files alone fails; an optional check is an optional failure; a check with no `-run` and a non-Go check printing the marker keep their outcome. Then the real matrix.

### Requirement trace

- R1 [satisfied] capability:validate-refuses-empty-go-checks evidence:integration check:validate-refuses-empty-go-checks-integration test:TestValidateRefusesAGoCheckThatRanNoTest test:TestGoTestSelection test:TestGoTestPackagesReadsLinesInBoundedMemory report:.pose/results/delivery-validation.json commit:78c8a1b35b1269d594525ec7041367af281fe00c — A go test check in the plain form that selects by name fails with limit_state no-tests when Go's result lines report every package as having run nothing, including with coverage, cached, and through GOFLAGS.
- R2 [satisfied] check:validate-refuses-empty-go-checks-integration test:TestValidateRefusesAGoCheckThatRanNoTest test:TestGoTestSelection report:.pose/results/delivery-validation.json commit:78c8a1b35b1269d594525ec7041367af281fe00c — A package reporting tests, no selection, a non-Go check, no result line and every form outside the plain grammar keep their outcome.
- R3 [satisfied] check:validate-refuses-empty-go-checks-integration test:TestValidateRefusesAGoCheckThatRanNoTest report:.pose/results/delivery-validation.json commit:78c8a1b35b1269d594525ec7041367af281fe00c — An optional check is an optional failure.

## 7. Final Report

### Delivered scope

`pose validate` fails a `go test` check of the plain form that selects by name when Go reports that no package ran a test, instead of passing on Go's exit 0. pose-dist's matrix passes with the rule on. Independently reviewed by Codex over four rounds; it replaces the abandoned shared test run.

### Residual risks

The rule reads Go's text summary: a test printing a line in the exact form of a result, or Go's own warning, can mislead it, and a check outside the plain grammar can pass empty. Both are stated in the manuals. The grammar's flag list follows the Go toolchain by hand.

### Follow-ups
