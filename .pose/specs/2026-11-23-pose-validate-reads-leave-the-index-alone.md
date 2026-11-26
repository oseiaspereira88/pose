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
- R1: Every status, diff and ls-files read the engine makes for validation provenance, material cleanliness and readiness checks shall run with optional locks disabled, and a diff against the work tree shall read the index without being able to write it, so none rewrites `.git/index` or takes its lock (D1).
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
`pose-mcp`: the engine's Git command constructor (`gitCommandContext`), work-tree read helpers (`git_worktree_reads.go`), and the work-tree diffs of `check` (ready transitions), `validate` scope selection, `report` and the review history ledger.

### Artifacts
- created: .pose/specs/2026-11-23-pose-validate-reads-leave-the-index-alone.md
- modified: pose-mcp/internal/pose/git_snapshot_integrity.go
- created: pose-mcp/internal/pose/git_worktree_reads.go
- created: pose-mcp/internal/pose/git_worktree_reads_test.go
- created: pose-mcp/internal/pose/git_worktree_reads_linux.go
- created: pose-mcp/internal/pose/git_worktree_reads_other.go
- modified: pose-mcp/internal/pose/review_attempt_ledger.go
- modified: pose-mcp/internal/cli/check.go
- modified: pose-mcp/internal/cli/validate_scope.go
- modified: pose-mcp/internal/cli/report.go
- created: pose-mcp/internal/cli/validate_reads_index_test.go

### Delivery targets
- capability:validate-reads-leave-the-index-alone module:pose-mcp profile:composed-capability entrypoint:pose-mcp/internal/cli/validate.go

### Accepted limits
- L1: A Git command the engine runs through another constructor, or a tool a validation step runs, can still refresh the index. Accepted because every engine read goes through `gitCommandContext`, and what a step runs is the project's command, which the Harne8 bugfix of its own scripts already covers.
- L2: Outside Linux there is no anonymous memory file, so a diff reads a private copy of the index in a temporary directory, which a dry run there would create. Accepted because the copy keeps the checkout's index untouched, which is the defect, and the dry-run test that forbids any file runs on Linux only.

### API/contract changes
None: the reads answer as before; only where their stat refresh is written changes.

### Data/storage changes
None.

### Technical risks
If Git stopped skipping a failed refresh lock silently, the diff would fail instead of answering; the equivalence test would show it.

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
R1 as drafted asked for `diff.autoRefreshIndex=false`. Measured on a stale stat cache, `git diff --name-only HEAD` rewrites the index even with `GIT_OPTIONAL_LOCKS=0`, and with `diff.autoRefreshIndex=false` it lists a file whose stat alone changed, which breaks the constraint that reads keep their results. So every engine Git command carries `GIT_OPTIONAL_LOCKS=0` (enough for `git status`), and a diff against the work tree runs unchanged but reads the index through `GIT_INDEX_FILE=/proc/self/fd/N`, an anonymous memory file (`memfd`) holding the index's bytes. Git resolves that link to `/memfd:pose-index/none (deleted)`, cannot create its lock beside it (the directory does not exist, for any user), and skips the refresh write silently, so the answer is the plain diff's and nothing is written. The memory file keeps the index's modification time, because Git trusts an entry's stat only when its file is older than the index: a fresher copy would hide a change made in the index's own second. A repository without an index file reads plainly, as there is no stat cache to write. Two earlier versions were refused: a private copy of the index in a temporary file, by the full suite (a dry run creates no file, `TestUpdatePreviewHasNoPhysicalMutationEvents`); and answers rebuilt from `git status`, by review round 1, which found five states they got wrong (submodules ignored by configuration, conflicts, symlinks, a POSE root below the repository top). Outside Linux the copy is used (L2).

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
- 2026-11-26: `GIT_OPTIONAL_LOCKS=0` in `gitCommandContext`; the work-tree diffs of the ready-transition read, scope selection, `report --since` and the review history ledger read `git status` (`WorktreeChanges`, `WorktreeChangedNames`). `TestValidationReadsLeaveTheIndexAlone`, `TestConcurrentValidationReadsNeverChangeTheIndex` and `TestWorktreeReadsLeaveTheIndexAlone` fail with each guard mutated out (the optional-locks setting, each caller's `git diff` restored, the content comparison); the last also checks the answers equal `git diff` for a commit since the base, a modified, a deleted and a staged-new file. A first private-index version was refused by the full suite (D1).
- 2026-11-26: review round 1 (rrn-b249b5d958a5d8ff) requested changes with N1–N9: the status-based answers differed from `git diff` for a submodule ignored by `.gitmodules`, a submodule with uncommitted content, symlinks and a directory replaced by a file, unmerged entries, a POSE root below the repository top, intent-to-add, staged-then-reverted changes and a range base. They were withdrawn: the callers run their original `git diff` again, reading the index through a `memfd` (D1). `TestReadOnlyIndexDiffAnswersAsThePlainDiff` builds those states and requires each read to answer byte for byte as the plain diff, leave `.git/index` identical and leave no lock; it also runs the review history ledger's diff (N9). Each guard fails its test mutated out.
- 2026-11-26: review round 2 (rrn-233e9d0e27a7230a) requested changes. N10: the memfd carried a new time, so a racily clean entry was trusted on its stat and a real change was hidden; the memfd (and the copy outside Linux) now keeps the index's modification time, and `TestReadOnlyIndexDiffSeesARacilyCleanEntry` fails with the current time in its place. N11: without an index file the read runs plainly instead of failing, and `check` applies the read-only index to its diff only (`TestValidationReadsAnswerWithoutAnIndex` fails with the missing-index case removed; the diff-only restriction is behaviour-preserving, as `ls-files` never writes, and has no test of its own). N13: scope selection keeps the cause of a failed read, and the index read reports its own error. N12 is a follow-up.

### Results summary
The full `pose-mcp` suite passes; the canonical result is refreshed with `pose validate --carry`.

### Requirement trace
- R1 [satisfied] test:TestValidationReadsLeaveTheIndexAlone, test:TestReadOnlyIndexDiffAnswersAsThePlainDiff, test:TestReadOnlyIndexDiffSeesARacilyCleanEntry, test:TestValidationReadsAnswerWithoutAnIndex
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
- [open] Keep a split-index read from touching the modification time of `.git/sharedindex.<hash>`, which Git updates when it reads the shared index (review round 2, N12). (owner:@pose-maintainers crit:low review:2026-12-26)
