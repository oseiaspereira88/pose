---
slug: pose-trace-test-refs-resolve
status: in-progress
created_at: 2026-10-09
completed_at:
supersedes:
depends_on: pose-requirement-evidence-traceability
priority: 1
components: pose-mcp
task_type: feature
surface: minimal
delivers: capability:trace-test-refs-resolve
---

# Spec: Trace test refs resolve to tests that exist

## 1. Intent

### Goal

Make `lint-spec` and `pose close` resolve every `test:` ref of a requirement trace against the tests that exist, so an invented test name can no longer close a spec.

### Business value

Origin: the open follow-up of `pose-abm-design-basis` (crit medium), prioritized by the maintainer on 2026-10-09. Its R7 trace cited `TestABMDesignBasisDigestStable`, which never existed, and the gate passed because it counted refs without resolving them: the formal-compliance shape the ABM program exists to catch.

### Constraints

History is not rewritten. Measured before the change with a resolver of Go functions only: 80 of 980 refs in pose-dist and 162 of 321 in Harne8 did not resolve, all in done specs; an error there would turn the `lint-spec --all` gate red. The maintainer chose to block only specs that can still change.

### Non-goals

Judging whether the cited test is adequate to the requirement; that remains the review.

## 2. Requirements

### Functional

- R1: A `test:` ref shall resolve when it names a Go test function or subtest, a JavaScript or TypeScript test title (exact or slug), a Python `test_` or Rust `#[test]` function, a tracked test file, or a `go test -run` pattern that selects a Go test, in the repository, its submodules or a nested repository the validation matrix registers as a module.
- R2: An unresolved ref shall fail `lint-spec` and `pose close` while the spec is open, and shall be reported as a warning on a done, superseded or abandoned spec.
- R3: Without Git, nothing shall be reported as unresolved.

### Compatibility

- Measured with the final resolver: 14 warnings in pose-dist and 28 in Harne8, all on done specs, no error; `lint-spec --all` passes in both. Five CLI fixtures that cited `test:TestFixture` in a Git tree without it now declare it.

## 3. Technical Plan

### Artifacts

- created: .pose/specs/2026-10-09-pose-trace-test-refs-resolve.md
- created: pose-mcp/internal/pose/test_catalog.go
- created: pose-mcp/internal/pose/test_catalog_test.go
- created: pose-mcp/internal/cli/trace_test_refs_test.go
- modified: pose-mcp/internal/cli/lintspec.go
- modified: pose-mcp/internal/cli/review_closeout.go
- modified: pose-mcp/internal/cli/review_closeout_test.go
- modified: pose-mcp/internal/version/failure_alert_test.go
- modified: .pose/indexes/validation-matrix.json
- modified: .pose/specs/2026-09-19-pose-abm-design-basis.md
- modified: POSE.md
- modified: locales/pt-BR/POSE.md
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md

The change to `failure_alert_test.go` is gofmt only.

### Delivery targets

- capability:trace-test-refs-resolve module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go

## 5. Decisions

### Decision D1
- Date: 2026-10-09
- Context: 242 historical refs do not resolve, all in done specs.
- Options considered: (a) error only while the spec is open, warning when closed; (b) always a warning; (c) always an error.
- Decision: (a), chosen by the maintainer.
- Rationale: an invented name stops closing from now on, history stays visible without being rewritten, and the `--all` gate keeps passing.
- Consequences: a spec that cites a test in a repository the catalog cannot see must cite the test file or register the repository as a module.

## 6. Validation

### Strategy

`TestTraceTestRefsResolveAgainstTheTrackedTests` resolves each ecosystem's names, a subtest path, a slug title, a test file and a `-run` pattern, refuses an invented name, a non-test function and a missing file, and reports nothing without Git. `TestTraceTestRefsResolveInNestedModuleRepositories` reads a registered nested repository and ignores an unregistered one. `TestTraceTestRefsBlockAnOpenSpecAndWarnOnAClosedOne` fails without the lint and close changes: an invented ref fails lint and close on an open spec, warns on a done one, and a real ref passes.

### Deterministic checks

#### Test
- Command: `cd pose-mcp && go test ./internal/pose ./internal/cli -run TraceTestRefs`
- Scope: pose-mcp
- Expected: pass

### Requirement trace

- R1 [satisfied] capability:trace-test-refs-resolve check:trace-test-refs-integration evidence:integration test:TestTraceTestRefsResolveAgainstTheTrackedTests test:TestTraceTestRefsResolveInNestedModuleRepositories
- R2 [satisfied] capability:trace-test-refs-resolve check:trace-test-refs-integration evidence:integration test:TestTraceTestRefsBlockAnOpenSpecAndWarnOnAClosedOne
- R3 [satisfied] capability:trace-test-refs-resolve check:trace-test-refs-integration evidence:integration test:TestTraceTestRefsResolveAgainstTheTrackedTests

## 7. Final Report

### Delivered scope

A per-process test catalog built from `git ls-files --recurse-submodules` plus the nested repositories the matrix registers resolves `test:` refs; `lint-spec` and `pose close` refuse an unresolved ref on an open spec and report it on a closed one. The manuals state the rule.

### Residual risks

- Ecosystems outside Go, JS/TS, Python and Rust resolve only by test file.
- Reviewed in the same session that implemented it, with no separate reviewer execution: declared, not independent.

### Follow-ups
