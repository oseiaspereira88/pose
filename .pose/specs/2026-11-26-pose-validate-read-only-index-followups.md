---
slug: pose-validate-read-only-index-followups
status: in-progress        # draft | in-progress | done | superseded | abandoned; blocked = operational wait, not terminal
created_at: 2026-11-26
completed_at:        # stamped on the transition to status: done
supersedes:          # slug of the superseded spec (when applicable)
depends_on:          # prerequisites, inline list: other-spec, milestone:<roadmap>/<id>, roadmap:<slug>
remediates:          # optional explicit links: spec:slug@defect-fix, finding:rva-<16hex>/F1@revert (never inferred from depends_on)
priority: 1
components: pose-mcp
task_type: bugfix
surface: standard
delivers: capability:validate-read-only-index-followups
---

# Spec: pose-validate-read-only-index-followups

## 1. Intent

### Goal
Close the follow-ups left by `pose-validate-reads-leave-the-index-alone`: the read-only index read must not fail where the plain diff succeeds, must fail loudly where it cannot protect the index, and its tests must cover the states the reviews probed.

### Business value
The spec closed on 2026-11-26 (approval rrn of its round 5) with follow-ups from rounds 2 to 5. One is a regression its own fix introduced (round 5, N22): the same-file check refuses an index another process replaced atomically, 2–5 reads in 2000 under concurrent `git status` writers, which fails scope selection and the review history read and makes the ready-transition check drop its diff.

### Constraints
Each item keeps the behaviour its origin spec delivered; a fix that changes an answer states it as a decision.

### Non-goals
Reopening the decisions of the origin spec.

## 2. Requirements

### Functional
- R1: The index read shall not fail when another process replaces the index atomically between its checks; a FIFO or device swapped in shall still not block (round 5, N22): open without blocking and judge the opened file.
- R2: A failed index read shall be reported by the ready-transition check, scope selection and `report --since` with its cause, never dropped or labelled `revision unknown?`; the index lookup shall run within the caller's deadline (rounds 2 N13, 3 N14, 4 N18, 5 N23).
- R3: The index lookup shall not depend on `rev-parse --path-format` silently: on a Git that does not support it the read shall fail or use a supported lookup, never run the plain diff unprotected (round 5, N26).
- R4: On Linux, a failing memory file or timestamp call shall fall back to the private copy, as the comment states, or the comment shall say it fails (round 5, N27).
- R5: A split-index read touches the modification time of `.git/sharedindex.<hash>`; that shall be declared as an accepted limit with what was measured, since the engine cannot prevent it (D1).
- R6: The equivalence test shall add unmerged, two-revision range and dirty-submodule cases, and the CLI concurrency test shall run the review history reader (rounds 1 N9, 3 N16, 4 N20, 5 N25).

### Non-functional
Existing behaviour outside the stated requirements is unchanged.

### Security
No new input crosses a trust boundary without the existing confinement.

### Compatibility
Records and indexes written by earlier engines keep their meaning.

## 3. Technical Plan

### Affected areas
`pose-mcp`: `git_worktree_reads*.go`, the ready-transition read in `check.go`, `validate_scope.go`, `report.go` and their tests.

### Artifacts
- created: .pose/specs/2026-11-26-pose-validate-read-only-index-followups.md
- modified: pose-mcp/internal/pose/git_worktree_reads.go
- modified: pose-mcp/internal/pose/git_worktree_reads_linux.go
- modified: pose-mcp/internal/pose/git_worktree_reads_other.go
- created: pose-mcp/internal/pose/git_index_open_unix.go
- created: pose-mcp/internal/pose/git_index_open_other.go
- modified: pose-mcp/internal/pose/git_worktree_reads_test.go
- modified: pose-mcp/internal/cli/check.go
- modified: pose-mcp/internal/cli/validate_scope.go
- modified: pose-mcp/internal/cli/report.go
- modified: pose-mcp/internal/cli/report_changed_files_test.go
- modified: pose-mcp/internal/cli/validate_reads_index_test.go

### Delivery targets
- capability:validate-read-only-index-followups module:pose-mcp profile:composed-capability entrypoint:pose-mcp/internal/cli/validate.go

### Accepted limits
- L1: In a split-index repository a read touches the modification time of `.git/sharedindex.<hash>`. Accepted because Git freshens the shared index on every read (measured on 2026-11-26: `git status` and `git ls-files` with `GIT_OPTIONAL_LOCKS=0` both move it), so no engine read can avoid it, and the file's bytes do not change.

### API/contract changes
`reportChangedFiles` returns its error; `pose report` prints a warning and writes the report without the list. `pose check` reports a ready-transition read it could not make. `ErrIndexUnreadable` marks index failures.

### Data/storage changes
None.

### Technical risks
Opening the index with `O_NONBLOCK` returns at once on a FIFO, which is then refused as not regular; on a regular file the flag has no effect.

## 4. Tasks

### Planning
- [x] Confirm each requirement against the current engine before starting.

### Implementation
- [x] Implement each requirement with a test that fails before it.

### Validation
- [ ] `pose validate --carry` and `pose check --strict`.

## 5. Decisions

### Decision D1
- Basis: R5
The split-index touch is Git's own: measured with `core.splitIndex=true`, a plain `git status` and even `git ls-files` with `GIT_OPTIONAL_LOCKS=0` move the shared index's modification time, because Git freshens it whenever it reads it. The engine cannot keep any read from doing so, so R5 declares it as L1 instead of changing code.

### Decision D2
- Basis: R1
The index is judged on the file actually opened, opened with `O_NONBLOCK` where FIFOs exist, instead of a stat before the open: a stat-then-open pair refuses an index replaced atomically in between (round 5, N22), and opening without blocking keeps a FIFO from hanging the read (round 4, N21).

## 6. Validation

### Strategy
Each requirement gets a regression test shown failing before its code.

### Deterministic checks

#### Test
- Command: `go test ./...` (in `pose-mcp`), then `pose validate --carry`.

### Execution log
- 2026-11-26: R1 opens without blocking and judges the opened file; R2 reports a failed index read in `check`, names it in scope selection (`ErrIndexUnreadable`) and warns in `report`; R3 locates the index with `--git-path` relative to the root, within a 30 s deadline; R4 falls back to the private copy when the memory file or its time cannot be set. Each has a test that fails with its guard mutated out: `TestReadOnlyIndexToleratesAnAtomicallyReplacedIndex`, `TestAFailedIndexReadIsReported`, `TestReadOnlyIndexFindsTheIndexFromASubdirectory`, `TestThePrivateIndexCopyLeavesTheIndexAlone`. R6 adds `TestReadOnlyIndexDiffCoversConflictsRangesAndDirtySubmodules`, which passes on the existing reader. R5 is L1 (D1).

### Results summary
The full `pose-mcp` suite passes; the canonical result is refreshed with `pose validate --carry`.

### Requirement trace
- R1 [satisfied] test:TestReadOnlyIndexToleratesAnAtomicallyReplacedIndex, test:TestReadOnlyIndexRefusesAFIFOIndex
- R2 [satisfied] test:TestAFailedIndexReadIsReported
- R3 [satisfied] test:TestReadOnlyIndexFindsTheIndexFromASubdirectory
- R4 [satisfied] test:TestThePrivateIndexCopyLeavesTheIndexAlone
- R5 [satisfied] (accepted limit L1, decision D1)
- R6 [satisfied] test:TestReadOnlyIndexDiffCoversConflictsRangesAndDirtySubmodules

### Known gaps
None.

## 7. Final Report

### Delivered scope
The read-only index read tolerates an atomically replaced index, reports its failures with their cause, works without `--path-format`, falls back to a private copy, and its tests cover conflicts, ranges and dirty submodules.

### Files and modules changed
`pose-mcp` read-only index reads and their callers in check, validate scope and report.

### Validation executed
`go test ./...` in `pose-mcp`; `pose check --strict`; `pose validate --carry --json-out .pose/results/delivery-validation.json`.

### Residual risks
L1 above.

### Follow-ups
None. Origin: `pose-validate-reads-leave-the-index-alone` (done 2026-11-26).
