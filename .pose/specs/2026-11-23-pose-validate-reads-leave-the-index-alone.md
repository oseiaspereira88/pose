---
slug: pose-validate-reads-leave-the-index-alone
status: draft
created_at: 2026-11-23
completed_at:
supersedes:
depends_on: 
remediates:
priority: 2
components: pose-mcp
task_type: bugfix
surface: standard
delivers: capability:validate-reads-leave-the-index-alone
---

# Spec: pose-validate-reads-leave-the-index-alone

## 1. Intent

### Goal
Make the engine's own Git reads during `pose validate` and `pose check`, such as the provenance status of `validate`, never rewrite `.git/index` or take index locks in the validated checkout.

### Business value
In Harne8 the governance read view of `instance-health-composite` failed intermittently with `review inputs changed during operation: .git` because concurrent reads refreshed the index stat cache. The Harne8 bugfix `harne8-validation-reads-leave-root-index-alone` fixed the instance's own scripts and left this engine part as its follow-up F1.

### Constraints
Reads keep their results. Stale stat caches may cost time, never correctness.

### Non-goals
Changing which files are compared.

## 2. Requirements

### Functional
- R1: Every status, diff and ls-files read the engine makes for validation provenance, material cleanliness and readiness checks shall run with optional locks disabled, and a diff against the work tree shall run over a private copy of the index, so none rewrites `.git/index` or takes its lock (D1).
- R2: A regression test shall run those reads over a checkout with a stale stat cache and prove `.git/index` byte-identical afterwards.
- R3: A concurrency test shall run those reads together on one checkout while a reader watches `.git/index` and its lock, and see neither change (D2).

### Non-functional
Existing behaviour outside the stated requirements is unchanged.

### Security
No new input crosses a trust boundary without the existing confinement.

### Compatibility
Records and indexes written by earlier engines keep their meaning.

## 3. Technical Plan

### Affected areas
`pose-mcp`: the engine's Git command constructor (`gitCommandContext`), a private-index helper (`git_private_index.go`), and the work-tree diffs of `check` (ready transitions), `validate` scope selection, `report` and the review history ledger.

### Artifacts
- created: .pose/specs/2026-11-23-pose-validate-reads-leave-the-index-alone.md
- modified: pose-mcp/internal/pose/git_snapshot_integrity.go
- created: pose-mcp/internal/pose/git_private_index.go
- created: pose-mcp/internal/pose/git_private_index_test.go
- modified: pose-mcp/internal/pose/review_attempt_ledger.go
- modified: pose-mcp/internal/cli/check.go
- modified: pose-mcp/internal/cli/validate_scope.go
- modified: pose-mcp/internal/cli/report.go
- created: pose-mcp/internal/cli/validate_reads_index_test.go

### Delivery targets
- capability:validate-reads-leave-the-index-alone module:pose-mcp profile:composed-capability entrypoint:pose-mcp/internal/cli/validate.go

### Accepted limits
- L1: A Git command the engine runs through another constructor, or a tool a validation step runs, can still refresh the index. Accepted because every engine read goes through `gitCommandContext`, and what a step runs is the project's command, which the Harne8 bugfix of its own scripts already covers.
- L2: A diff against the work tree now reads a copy of the index, which costs one copy of up to 512 MiB per read. Accepted because indexes are far smaller in practice and the bound refuses an oversized one rather than reading it.

### API/contract changes
None: the reads answer as before; only where their stat refresh is written changes.

### Data/storage changes
None.

### Technical risks
A split index would make the private copy write a shared index into `.git`; the copy is written whole (`core.splitIndex=false`).

## 4. Tasks

### Planning
- [x] Confirm the requirements against the current engine before starting.

### Implementation
- [x] Implement each requirement with a test that fails before it.

### Validation
- [ ] `pose validate --carry` and `pose check --strict`.

## 5. Decisions

### Decision D1
- Basis: R1
R1 as drafted asked for `diff.autoRefreshIndex=false`. Measured on a stale stat cache, `git diff --name-only HEAD` rewrites the index even with `GIT_OPTIONAL_LOCKS=0`, and with `diff.autoRefreshIndex=false` it lists a file whose stat alone changed, which breaks the constraint that reads keep their results. `git status` with `GIT_OPTIONAL_LOCKS=0` refreshes in memory and writes nothing. So every engine Git command carries `GIT_OPTIONAL_LOCKS=0`, and a diff against the work tree runs with `GIT_INDEX_FILE` set to a private copy of the index: Git refreshes the copy and answers as before.

### Decision D2
- Basis: R3
R3 as drafted ran `validate` and `check --strict` together. The property they broke is that a reader saw `.git` change under it; the test runs the reads those commands make, concurrently, while a reader watches `.git/index` and `index.lock`, which shows the property directly and runs in under a second instead of a full validation.

## 6. Validation

### Strategy
Each requirement gets a regression test shown failing before its code.

### Deterministic checks

#### Test
- Command: `go test ./...` (in `pose-mcp`), then `pose validate --carry`.

### Execution log
- 2026-11-26: `GIT_OPTIONAL_LOCKS=0` in `gitCommandContext`; `WithPrivateIndex` and `RawGitOutputWithPrivateIndex` for the work-tree diffs of the ready-transition read, scope selection, `report --since` and the review history ledger. `TestValidationReadsLeaveTheIndexAlone`, `TestConcurrentValidationReadsNeverChangeTheIndex` and `TestReviewHistoryDiffLeavesTheIndexAlone` fail with each guard mutated out (the optional-locks setting, the private index, and each caller's use of it).

### Results summary
The full `pose-mcp` suite passes; the canonical result is refreshed with `pose validate --carry`.

### Requirement trace
- R1 [satisfied] test:TestValidationReadsLeaveTheIndexAlone, test:TestReviewHistoryDiffLeavesTheIndexAlone
- R2 [satisfied] test:TestValidationReadsLeaveTheIndexAlone
- R3 [satisfied] test:TestConcurrentValidationReadsNeverChangeTheIndex

### Known gaps
None.

## 7. Final Report

### Delivered scope
The engine's Git reads during `validate`, `check` and `report` no longer rewrite `.git/index` or take its lock, and keep their answers.

### Files and modules changed
`pose-mcp` Git command construction and the work-tree diffs of check, validate scope, report and the review history ledger.

### Validation executed
`go test ./...` in `pose-mcp`; `pose check --strict`; `pose validate --carry --json-out .pose/results/delivery-validation.json`.

### Residual risks
L1 and L2 above.

### Follow-ups
None.
