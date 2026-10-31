---
slug: pose-validation-matrix-dead-test-patterns
status: done
created_at: 2026-10-30
supersedes:
depends_on:
remediates:
priority: 2
components: pose-mcp
task_type: bugfix
delivers: capability:validation-matrix-run-patterns
completed_at: 2026-10-30
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

- Enforcing this at validation time (spec `pose-validate-refuses-empty-go-checks`).

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

- capability:validation-matrix-run-patterns module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go

### API/contract changes

None.

## 4. Tasks

- [x] Remove the dead alternatives
- [x] Guard test, shown failing before the removal
- [x] Independent review and closeout

## 5. Decisions

## 6. Validation

### Strategy

The guard first runs against the matrix with the dead alternatives and fails naming all five; after the removal it passes. Each changed check's matched test list is compared before and after.

### Requirement trace

- R1 [satisfied] check:test test:TestValidationMatrixRunPatternsNameExistingTests report:.pose/results/delivery-validation.json commit:5d2e5d87e45dcaec699f986b663336b448e88f90 — The five dead alternatives are gone; each of the four checks selects the same tests as before (52, 9, 17, 3), confirmed independently by the reviewer.
- R2 [satisfied] check:test test:TestValidationMatrixRunPatternsNameExistingTests test:TestDeadRunPatternGuardFollowsGoSelection report:.pose/results/delivery-validation.json commit:5d2e5d87e45dcaec699f986b663336b448e88f90 — The guard reads both flag spellings, final flag values, tags and race, module packages and go/build's file selection, and counts only what -run executes; it failed naming the five alternatives before their removal.
- R3 [satisfied] check:test test:TestDeadRunPatternGuardFollowsGoSelection report:.pose/results/delivery-validation.json commit:5d2e5d87e45dcaec699f986b663336b448e88f90 — Unmodelled flags and values, packages outside the module or in excluded directories, Go build environment, stack checks selecting by name and unlisted or changed non-plain patterns are refused.

## 7. Final Report

### Delivered scope

Five -run alternatives that named no test removed from four checks, and a source-level guard that keeps new ones out under a narrow, explicitly modelled grammar and refuses the rest. Independently reviewed by Codex over eight rounds (203 differential cases against `go test` in the last sweeps; reports kept outside the checkout).

### Residual risks

The guard judges declarations, not runs: skips, `-short`, `TestMain` control flow, dynamic subtests and the operator's inherited environment are judged at run time by `pose-validate-refuses-empty-go-checks`. Two review fixtures fail outside a Git repository ancestry (F1).

### Follow-ups

- [open] F1: make the review-bundle fixtures (`componentReviewFixture`, `reviewBundleFixture`) initialise their own Git repository, so `review-bundle-convergence` and `material-reuse-integration` do not depend on an enclosing repository of the temporary directory (reviewer F4, rounds 1–2). (owner:@oseiaspereira88 crit:medium review:2026-11-13)
- [covered: pose-validate-refuses-empty-go-checks] F2: judge at run time what the declaration guard cannot — non-plain patterns, subtests, skips and the inherited environment.
