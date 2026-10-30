---
slug: pose-validate-shared-go-test-run
status: draft
created_at: 2026-10-30
supersedes:
depends_on: pose-validate-concurrent-steps
remediates:
priority: 2
components: pose-mcp
task_type: feature
delivers:
---

# Spec: Go test checks of a module are answered from one test run instead of one run each

## 1. Intent

### Goal

Run each module's Go tests once per validation and answer every `go test` check of the matrix from that run's per-test results, so a test is compiled and executed once instead of once per check that selects it.

### Business value

In pose-dist every integration and reachability check is a `go test <packages> -run <pattern> -count=1` over tests that the `test` check (`go test ./...`) already runs. On 2026-10-30 those checks took 393 s on top of the full suite (280 s uncached): more than half of a strict validation re-runs tests that already passed minutes earlier, and each of the 157 Go checks re-links its package's test binary (median check 1.06 s).

The same change closes a consistency gap: a check whose `-run` pattern matches no test passes today (Go prints "no tests to run" and exits 0), so a renamed or deleted test silently turns its evidence into nothing. On 2026-10-30 no check of the 154 with `-run` matched nothing, but four name tests that no longer exist in an alternative of their pattern: `review-bundle-convergence` (`ReviewPlanActionableToolPhases`), `spec-authority-transfer-negative-gates` (`TestSpecTransferBoundary`), `material-reuse-integration` (`reuse`, which Go's case-sensitive match never hits) and `atomic-start-cutoff-integration` (`PreviewStart`, `ApplyStart`).

### Constraints

- Each check keeps its own name, evidence class, severity and outcome in the results; what changes is that the outcome is derived from recorded test events, and the result says so.
- A check's outcome is pass only when at least one test of its packages matches its pattern and every matching test passed in the shared run; build failures of a package fail every check over it.
- Checks with flags that change how tests run (`-race`, `-tags`, `-short`, `-timeout`, environment) are grouped only with checks that use the same flags.

### Non-goals

- Non-Go checks; Go checks with arguments the attribution cannot model run as today.

## 2. Requirements

### Functional

- R1: For each module, validate shall group `go test` checks by their execution flags, run `go test -json` once per group over the union of their packages, and record the run.
- R2: Each grouped check's outcome shall be derived from the test events of its packages whose names match its `-run` pattern (Go's matching rules, subtests included), and its result shall name the shared run it was derived from.
- R3: A check whose pattern matches no test in its packages, or whose pattern is a plain alternation (`A|B|…`) with an alternative that matches no test, shall fail with a message naming the pattern or the alternative.
- R6: The four checks found with dead alternatives on 2026-10-30 shall be corrected before R3 is enforced — done by spec `pose-validation-matrix-dead-test-patterns`, whose source-level guard keeps new dead alternatives out of plain-alternation patterns; non-plain patterns, subtests, `-skip` and run-time outcomes are left to R3.
- R4: A check the grouping cannot model (unknown flags, `-c`, `-o`, custom `-exec`) shall run on its own as today.
- R5: The full-suite check and the derived checks shall report the same per-test outcome for every test they share.

### Non-functional

- On pose-dist, a strict validation shall run the Go tests of `internal/cli` once per flag group, and its wall time shall drop by at least the duplicated 393 s minus the shared run's own cost, measured and recorded.

### Security

- None.

### Compatibility

- Evidence attribution changes from "the check's command passed" to "the check's tests passed in a shared run"; this needs an ADR before implementation, and the results schema records the derivation.

## 3. Technical Plan

### Affected areas

`internal/cli/validate.go`, Go check modelling, results schema, the manuals, an ADR.

### Artifacts

### Delivery targets

### API/contract changes

Result field `derived_from` (shared run id); outcome rule of R2–R3.

## 4. Tasks

- [ ] ADR: evidence of a Go check derived from a shared test run
- [ ] Model and group Go checks
- [ ] Shared `go test -json` run and per-check derivation
- [ ] Empty-pattern and dead-alternative refusal at run time
- [ ] Measure on pose-dist
- [ ] Independent review and closeout

## 5. Decisions

## 6. Validation

### Strategy

Fixture modules with passing, failing, skipped, subtest and panicking tests; patterns that match nothing; mixed flags; a package that does not build. Each derived outcome is compared with the outcome of running that check alone. Then the real matrix, timed before and after.

### Requirement trace

## 7. Final Report

### Delivered scope

Not started.

### Residual risks

None yet.

### Follow-ups
