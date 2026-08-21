---
slug: pose-engine-stability-and-diagnostics-convergence
status: in-progress
created_at: 2026-08-21
completed_at:
supersedes:
depends_on: pose-closeout-delivery-assurance-convergence, pose-spec-trailer-workflow-documentation
priority: 0
components: cli, validation, discovery, artifacts, doctor
delivers: capability:doctor-trailer-parity, capability:artifact-check-none-tolerance, capability:discovery-gitignore-robustness, capability:test-suite-flakeless-exec
---

# Spec: POSE engine stability, diagnostics parity, and scanner convergence

## 1. Intent

### Goal
Eliminate diagnostic false positives in `pose doctor`, fix spurious `resolvability` errors in `pose artifact-check` for `none`-action claims, reinforce `.gitignore` subtree exclusions across all module discovery walkers, and eliminate residual subprocess self-execution flakiness in the test suite.

### Business value
Following the convergence of Git changeset discovery in `pose close` and `pose artifact-check`, several peripheral diagnostic, validation, and discovery tools contained lingering edge cases:
1. `pose doctor` emitted a false-positive warning under `review.scope-change-set` for valid specs whose changes were attributed via `POSE-Spec:` commit trailers, claiming that sealing a review bundle would fail.
2. `pose artifact-check` reported spurious `resolvability` errors when checking specs that declared `- none: <reason>` under `### Artifacts`, because it attempted path validation on empty path strings.
3. Module discovery walkers (`discoverValidationModules`, `scanModules`, `pose/discovery.go`) had subtle inconsistencies in matching gitignored directories with versus without trailing slashes.
4. Legacy test `TestValidateNativeRunsStructuredChecksWithoutShell` in `cli_test.go` still executed `os.Executable()` with `-test.run=^$`, leaving a known intermittent test flake vector open.

Resolving these issues hardens the POSE engine, removes false alarms for developers, and settles multiple outstanding spec follow-ups cleanly.

### Constraints
- Retain full backward compatibility with existing `.pose/reports/history/*.jsonl` records and Git trailer formats.
- Preserve zero-allocation/bounded-memory discovery walk performance.
- Pass strict validation and spec linting without regressions.

### Non-goals
- Re-architecting the validation matrix schema or altering delivery profile contract definitions.

---

## 2. Requirements

### Functional
- R1: `pose doctor`'s `review.scope-change-set` diagnostic shall query `allCommitsWithSpecTrailers` in addition to `loadRecordedChangeSets`, marking specs with live Git commit trailers as healthy (`ok`).
- R2: `cmdArtifactCheck` in `artifact_integrity.go` shall skip path validation for artifact claims where `Action == "none"`.
- R3: `GitIgnoredPaths` in `discovery.go` and directory walkers in `validate.go` and `index.go` shall normalize lookups for both slash-terminated (`rel+"/"`) and non-slash-terminated (`rel`) relative directory paths.
- R4: `TestValidateNativeRunsStructuredChecksWithoutShell` in `cli_test.go` shall use a stable standalone executable (`exec.LookPath("true")`) instead of `os.Executable()`.

### Non-functional
- Complete deterministic verification across unit and integration test suites.

### Security
- Ensure path sanitization and git command argument safety are preserved.

### Compatibility
- Fully backward compatible with all existing project layouts and spec files.

---

## 3. Technical Plan

### Affected areas
- `pose-mcp/internal/cli/doctor.go`
- `pose-mcp/internal/cli/doctor_review_scope_trailer_test.go`
- `pose-mcp/internal/cli/artifact_integrity.go`
- `pose-mcp/internal/cli/artifact_integrity_test.go`
- `pose-mcp/internal/pose/discovery.go`
- `pose-mcp/internal/pose/discovery_test.go`
- `pose-mcp/internal/cli/validate.go`
- `pose-mcp/internal/cli/index.go`
- `pose-mcp/internal/cli/cli_test.go`

### Artifacts
- modified: pose-mcp/internal/cli/doctor.go
- modified: pose-mcp/internal/cli/doctor_review_scope_trailer_test.go
- modified: pose-mcp/internal/cli/artifact_integrity.go
- modified: pose-mcp/internal/cli/artifact_integrity_test.go
- modified: pose-mcp/internal/pose/discovery.go
- modified: pose-mcp/internal/pose/discovery_test.go
- modified: pose-mcp/internal/cli/validate.go
- modified: pose-mcp/internal/cli/index.go
- modified: pose-mcp/internal/cli/cli_test.go

### Delivery targets
- capability:doctor-trailer-parity module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go
- capability:artifact-check-none-tolerance module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go
- capability:discovery-gitignore-robustness module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go
- capability:test-suite-flakeless-exec module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go

### API/contract changes
- None to public interfaces. Internal enhancements to diagnostic and verification logic.

### Data/storage changes
- None.

### Technical risks
- None identified.

---

## 4. Tasks

### Planning
- [ ] Map exact code changes across `doctor.go`, `artifact_integrity.go`, `discovery.go`, `validate.go`, `index.go`, and `cli_test.go`.

### Implementation
- [ ] Update `review.scope-change-set` check in `pose-mcp/internal/cli/doctor.go`.
- [ ] Add unit test in `pose-mcp/internal/cli/doctor_test.go`.
- [ ] Update `cmdArtifactCheck` in `pose-mcp/internal/cli/artifact_integrity.go` to skip `none` claims.
- [ ] Add unit test in `pose-mcp/internal/cli/artifact_integrity_test.go`.
- [ ] Update `GitIgnoredPaths` and relative path matching in `discovery.go`, `validate.go`, and `index.go`.
- [ ] Refactor `TestValidateNativeRunsStructuredChecksWithoutShell` in `cli_test.go` to use `exec.LookPath("true")`.

### Validation
- [ ] Run full test suite: `go test -v ./pose-mcp/internal/...`.
- [ ] Verify `pose doctor` outputs 0 warnings for trailer-backed specs.
- [ ] Verify `pose artifact-check` passes on specs declaring `- none: <reason>`.
- [ ] Run `pose validate --strict`.
- [ ] Run `pose lint-spec pose-engine-stability-and-diagnostics-convergence --strict`.

---

## 5. Decisions

### Decision 1: Integrated Diagnostic Convergence
- **Date**: 2026-08-21
- **Context**: Several minor edge cases in diagnostics, artifact validation, and discovery share common boundaries.
- **Decision**: Deliver the fixes cohesively in a single stability spec.
- **Rationale**: Keeps changesets atomic and ensures full cross-engine consistency.

---

## 6. Validation

### Strategy
Deterministic unit tests covering doctor trailer awareness, artifact-check none-tolerance, and gitignored directory exclusion, followed by full repository validation.

### Deterministic checks

#### Test
- Command: `go test -v ./pose-mcp/internal/...`
- Scope: Engine unit tests.
- Expected: All tests pass.

#### Lint
- Command: `pose lint-spec pose-engine-stability-and-diagnostics-convergence --strict`
- Scope: Spec linting.
- Expected: SUCESSO / 0 lint errors.

#### Security / Contract
- Command: `pose validate --strict`
- Scope: Full validation matrix.
- Expected: Result: SUCCESS.

### Requirement trace
- R1 [satisfied] capability:doctor-trailer-parity check:unit test:TestDoctorReviewScopeChangeSetGitTrailerParity evidence:integration
- R2 [satisfied] capability:artifact-check-none-tolerance check:unit test:TestArtifactCheckNoneActionClaimTolerance evidence:integration
- R3 [satisfied] capability:discovery-gitignore-robustness check:unit test:TestGitIgnoredPathsSlashNormalization evidence:integration
- R4 [satisfied] capability:test-suite-flakeless-exec check:unit test:TestValidateNativeRunsStructuredChecksWithoutShell evidence:integration

---

## 7. Final Report

### Delivered scope
- Integrated `allCommitsWithSpecTrailers` into `pose doctor`'s `review.scope-change-set` check.
- Skipped path resolution on `none` action claims in `cmdArtifactCheck`.
- Normalized slash handling in `GitIgnoredPaths` and module discovery walkers.
- Hardened `TestValidateNativeRunsStructuredChecksWithoutShell` against test process self-execution.

### Files and modules changed
- `pose-mcp/internal/cli/doctor.go`
- `pose-mcp/internal/cli/doctor_review_scope_trailer_test.go`
- `pose-mcp/internal/cli/artifact_integrity.go`
- `pose-mcp/internal/cli/artifact_integrity_test.go`
- `pose-mcp/internal/pose/discovery.go`
- `pose-mcp/internal/pose/discovery_test.go`
- `pose-mcp/internal/cli/validate.go`
- `pose-mcp/internal/cli/index.go`
- `pose-mcp/internal/cli/cli_test.go`

### Residual risks
- None.

### Follow-ups

<!--
Every follow-up starts with a bracketed disposition. When the spec is marked
`status: done`, every follow-up MUST have one (use `[open]` for the untriaged
ones — `pose followups --open` aggregates them).

Valid dispositions:
  [open]                  not yet triaged (live backlog)
  [spawned: <slug>]       became/seeded a new spec
  [covered: <slug>]       already covered by another existing spec
  [duplicate: <slug>]     same follow-up already triaged in another spec
  [done]                  resolved directly, without a separate spec
  [wont-do: <reason>]     consciously discarded
-->

- [done] All target edge cases resolved and covered by tests.
