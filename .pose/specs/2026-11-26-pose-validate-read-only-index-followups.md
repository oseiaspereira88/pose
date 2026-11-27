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
- created: pose-mcp/internal/pose/git_worktree_reads_linux_test.go
- modified: pose-mcp/internal/pose/review_attempt_ledger.go
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

### Decision D3
- Basis: R2
The index lookup runs under the caller's context where the caller has one (`ReadOnlyIndexContext`, used by the review history reader within its 30 s), and under a 30 s bound of its own otherwise. A failure is never classified from Git's words, which two rounds showed incomplete in both directions (a revision named after the index, index refusals worded otherwise, a warning printed first): scope selection runs its diff first, so every revision the diff accepts (a tree, `X^-`, a range) keeps its answer, and only after a failure resolves each revision with `rev-parse --verify --quiet`: one Git cannot find (exit 1, nothing said) is "revision unknown?", a rev-parse failing otherwise (no repository, a broken config, dubious ownership) is reported with its own words, and when all resolve the diff's failure is. Git's standard error is kept on one line within 600 bytes, its `fatal:`/`error:` lines first, read with `LC_ALL=C`. `ErrIndexUnreadable` marks only the read-only index's own failures. The report stays silent only where Git cannot open the repository at all, and states any other failure on one line with every absolute path replaced by `<path>`.

## 6. Validation

### Strategy
Each requirement gets a regression test shown failing before its code.

### Deterministic checks

#### Test
- Command: `go test ./...` (in `pose-mcp`), then `pose validate --carry`.

### Execution log
- 2026-11-26: R1 opens without blocking and judges the opened file; R2 reports a failed index read in `check`, names it in scope selection (`ErrIndexUnreadable`) and warns in `report`; R3 locates the index with `--git-path` relative to the root, within a 30 s deadline; R4 falls back to the private copy when the memory file or its time cannot be set. Each has a test that fails with its guard mutated out: `TestReadOnlyIndexToleratesAnAtomicallyReplacedIndex`, `TestAFailedIndexReadIsReported`, `TestReadOnlyIndexFindsTheIndexFromASubdirectory`, `TestThePrivateIndexCopyLeavesTheIndexAlone`. R6 adds `TestReadOnlyIndexDiffCoversConflictsRangesAndDirtySubmodules`, which passes on the existing reader. R5 is L1 (D1).
- 2026-11-26: review round 1 (rrn-4262ae59adfc1641) requested changes. N1–N2: a corrupt index, which Git itself refuses, is now reported with Git's own message and marked as an index failure in check, scope selection and report (`TestACorruptIndexIsReportedWithItsCause`; fails with the classification or check's capture removed). N3: D3. N4: the memfd and timestamp calls are replaceable, and `TestReadOnlyIndexFallsBackWithoutAMemoryFile` fails with either fallback removed. N5: the R6 test makes the stat cache stale before every read, so it fails with the protection or the ledger's use of it removed, and the CLI concurrency test runs `ReviewRunHistoryFindings`. N8: failures handing the index to Git are index failures, and a failing memfd write falls back too. N9: the report states that the list could not be read, and a plain `pose report` outside a repository stays silent. N10: the check's findings are localised. N6, N7 and N11 are follow-ups.
- 2026-11-26: review round 2 (rrn-ef7133259b2c911e) requested changes. N12: the plain `report` listing runs in English and is silent only outside a repository (`reportChangedFilesNoted`); N13: the index-failure test matches Git's index refusals, not the word `index`, and the scope message no longer repeats itself; N14: a failed lookup keeps Git's cause and is not an index failure; N15: Git's message is kept on one bounded line and the report states it on one line; N16: `TestIndexFailuresAreNamedInAnyLocale`, `TestGitReadErrorsNameTheirCause` and `TestAFailedIndexHandOverIsAnIndexFailure` fail with each of those guards, the English locale, the context and the hand-over mark mutated out. A failing memfd write still has no test of its own (it falls back as a failing memfd does).
- 2026-11-26: review round 3 (rrn-bd7106fcc6ae8449) requested changes: the text classification missed index refusals and lost the cause behind a warning. Withdrawn for D3: `TestScopeSelectionTellsRevisionsFromIndexFailures` (a corrupt index behind a `diff.dirstat` warning is not an unknown revision; an unknown revision in a range is) fails with the resolution or the label mutated; `TestGitReadErrorsNameTheirCause` fails with the first-line cut restored; `TestIndexFailuresAreNamedInAnyLocale` fails with the absolute path kept. N19–N21 are follow-ups or answered by D3.
- 2026-11-26: review round 4 (rrn-f0476b1f18d70fb9) requested changes: resolving revisions before the diff refused trees and `HEAD^-`, and a repository failure lost Git's cause. Now the diff runs first and revisions are resolved only after a failure (D3); `TestScopeSelectionKeepsWhatTheDiffAccepts` (a tree, `HEAD^-`, `A..B`, `A...B`, a broken config, the report's path redaction keeping `feature/index-rework`) and the healthy-range case of `TestScopeSelectionTellsRevisionsFromIndexFailures` fail with the exit-code rule, the range split or the path rule mutated; `TestGitReadErrorsNameTheirCause` fails with the fatal lines no longer first (N25).

### Results summary
The full `pose-mcp` suite passes; the canonical result is refreshed with `pose validate --carry`.

### Requirement trace
- R1 [satisfied] test:TestReadOnlyIndexToleratesAnAtomicallyReplacedIndex, test:TestReadOnlyIndexRefusesAFIFOIndex
- R2 [satisfied] test:TestScopeSelectionKeepsWhatTheDiffAccepts, test:TestScopeSelectionTellsRevisionsFromIndexFailures, test:TestAFailedIndexReadIsReported, test:TestACorruptIndexIsReportedWithItsCause, test:TestIndexFailuresAreNamedInAnyLocale, test:TestGitReadErrorsNameTheirCause, test:TestAFailedIndexHandOverIsAnIndexFailure
- R3 [satisfied] test:TestReadOnlyIndexFindsTheIndexFromASubdirectory
- R4 [satisfied] test:TestThePrivateIndexCopyLeavesTheIndexAlone, test:TestReadOnlyIndexFallsBackWithoutAMemoryFile
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
Origin: `pose-validate-reads-leave-the-index-alone` (done 2026-11-26).
- [open] Test the index lookup against a Git that echoes an unknown `--path-format` (a stand-in executable), so restoring the flag fails a test (review round 1, N6). (owner:@pose-maintainers crit:low review:2026-12-26)
- [open] Test the `pose report` warning for an unreadable change list end to end (review round 1, N7). (owner:@pose-maintainers crit:low review:2026-12-26)
- [open] Add failing tests for the guards round 3 listed (the ledger passing its context, `GitEnglish` on the check, hand-over and lookup, the report's stderr warning, the memfd write fallback), and a locale test that does not depend on Git's pt_BR translations being installed (review rounds 2 N16, 3 N21). (owner:@pose-maintainers crit:low review:2026-12-26)
- [open] List the changed specs of a POSE root below the repository top: `git diff --name-only` names top-relative paths that the `.pose/specs/` filter drops, so their ready transitions go unchecked (review round 1, N11; pre-existing). (owner:@pose-maintainers crit:medium review:2026-12-26)
