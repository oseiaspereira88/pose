---
slug: pose-validation-matrix-dead-test-patterns
status: in-progress
created_at: 2026-10-30
supersedes:
depends_on:
remediates:
priority: 2
components: pose-mcp
task_type: bugfix
delivers:
---

# Spec: No validation check names a test that does not exist

## 1. Intent

### Goal

Remove the alternatives of `-run` patterns in the validation matrix that match no test, and keep any new one from entering it.

### Business value

`go test -run` exits 0 when a pattern, or one alternative of it, matches nothing, so a check that names a missing test reports evidence it never produced. On 2026-10-30 four checks carried such alternatives: `review-bundle-convergence` (`ReviewPlanActionableToolPhases`), `spec-authority-transfer-negative-gates` (`TestSpecTransferBoundary`), `material-reuse-integration` (`reuse`, which Go's case-sensitive match never hits) and `atomic-start-cutoff-integration` (`PreviewStart`, `ApplyStart`, names of production methods, not tests). Four of them were never test names; their intended coverage is already matched by other alternatives of the same checks.

### Constraints

- No check loses a test it runs today.

### Non-goals

- Enforcing this at validation time from a shared test run (spec `pose-validate-shared-go-test-run` R3).

## 2. Requirements

### Functional

- R1: The five dead alternatives shall be removed from their four checks; each check shall run the same tests as before.
- R2: A test shall read every Go test check of the matrix — `-run X` and `-run=X`, packages as relative paths, import paths or the module directory by default, `-tags` honoured — and fail, naming the module, check and alternative, when an alternative of a plain-alternation pattern matches no test Go would run under `-run` in the files `go/build` selects for those packages (tests and fuzz targets by Go's name and signature rule, examples with an output comment; not benchmarks; `./...` stopping at `testdata`, `vendor`, `_`/`.` directories and nested modules).
- R3: The guard shall also fail on a check it cannot judge: an unmodelled flag, a package outside the module, a stack check that selects tests by name, or a non-plain pattern not listed with its reason.

### Non-functional

- The guard reads source declarations; it starts no `go` process.

### Security

- None.

### Compatibility

- None.

## 3. Technical Plan

### Affected areas

`.pose/indexes/validation-matrix.json`, a test in `internal/cli`.

### Artifacts

- created: .pose/specs/2026-10-30-pose-validation-matrix-dead-test-patterns.md
- created: .pose/starts/pose-validation-matrix-dead-test-patterns.json
- modified: .pose/indexes/validation-matrix.json
- created: pose-mcp/internal/cli/validation_matrix_patterns_test.go
- modified: pose-mcp/internal/cli/testdata/adversarial/README.md
- created: .pose/specs/2026-10-30-pose-validation-matrix-dead-test-patterns.amendments.jsonl
- modified: .pose/specs/2026-10-30-pose-validate-shared-go-test-run.md

### Delivery targets

### API/contract changes

None.

## 4. Tasks

- [ ] Remove the dead alternatives
- [ ] Guard test, shown failing before the removal
- [ ] Independent review and closeout

## 5. Decisions

## 6. Validation

### Strategy

The guard first runs against the matrix with the dead alternatives and fails naming all five; after the removal it passes. Each changed check's matched test list is compared before and after.

### Requirement trace

## 7. Final Report

### Delivered scope

Not started.

### Residual risks

None yet.

### Follow-ups
