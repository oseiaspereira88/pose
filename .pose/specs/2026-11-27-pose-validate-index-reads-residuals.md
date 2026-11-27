---
slug: pose-validate-index-reads-residuals
status: in-progress        # draft | in-progress | done | superseded | abandoned; blocked = operational wait, not terminal
created_at: 2026-11-27
completed_at:        # stamped on the transition to status: done
supersedes:          # slug of the superseded spec (when applicable)
depends_on:          # prerequisites, inline list: other-spec, milestone:<roadmap>/<id>, roadmap:<slug>
remediates:          # optional explicit links: spec:slug@defect-fix, finding:rva-<16hex>/F1@revert (never inferred from depends_on)
priority: 2
components: pose-mcp
task_type: bugfix
surface: standard
delivers: capability:validate-index-reads-residuals
---

# Spec: pose-validate-index-reads-residuals

## 1. Intent

### Goal
Close what `pose-validate-read-only-index-followups` left open: changed specs of a POSE root below the repository top, the report's path redaction and UTF-8 cuts, a silent ls-files failure, and the guards still without a failing test.

### Business value
The spec closed on 2026-11-27 with follow-ups in its Final Report (review rounds 1–3) and its approving attestation (round 5, N1–N5). One is medium: with the POSE root below the repository top, a modified spec's ready transition goes unchecked without a finding.

### Constraints
Each item keeps the behaviour its origin spec delivered; a fix that changes an answer states it as a decision.

### Non-goals
Reopening the decisions of the origin spec.

## 2. Requirements

### Functional
- R1: With the POSE root below the repository top, the ready-transition read shall list changed specs (Git names them top-relative), so their ready transitions are checked (round 1 N11, round 5 N3).
- R2: The report's redaction shall replace a whole absolute path, also one with spaces, quotes, parentheses or a leading `:`, backtick or `[` (round 5, N1).
- R3: The bounds on Git's message and the report note shall cut at a character, never inside a UTF-8 sequence (round 5, N4).
- R4: A failing `ls-files` in worktree scope selection shall fail the selection with its cause instead of silently dropping untracked files (round 5, N5).
- R5: The guards listed by rounds 2–5 shall each have a failing test: restoring `--path-format`, the `X^-` cut, the report note bound and stderr warning, `GitEnglish` on the check, lookup and hand-over, the memfd write fallback, the private copy keeping the index's time, a stand-in Git that echoes an unknown `--path-format`, and a locale test independent of installed translations; the ledger's context is L1 (rounds 1 N6–N7, 2 N16, 3 N21, 5 N2).

### Non-functional
Existing behaviour outside the stated requirements is unchanged.

### Security
No new input crosses a trust boundary without the existing confinement.

### Compatibility
Records and indexes written by earlier engines keep their meaning.

## 3. Technical Plan

### Affected areas
`pose-mcp`: `check.go` (ready transitions), `report.go`, `validate_scope.go`, `git_worktree_reads*.go` and their tests.

### Artifacts
- created: .pose/specs/2026-11-27-pose-validate-index-reads-residuals.md
- modified: pose-mcp/internal/cli/check.go
- modified: pose-mcp/internal/cli/report.go
- modified: pose-mcp/internal/cli/validate_scope.go
- modified: pose-mcp/internal/pose/git_worktree_reads.go
- modified: pose-mcp/internal/pose/git_worktree_reads_linux.go
- created: pose-mcp/internal/pose/git_reads_residuals_test.go
- created: pose-mcp/internal/cli/validate_index_residuals_test.go
- modified: pose-mcp/internal/pose/git_worktree_reads_linux_test.go

### Delivery targets
- capability:validate-index-reads-residuals module:pose-mcp profile:composed-capability entrypoint:pose-mcp/internal/cli/validate.go

### Accepted limits
- L1: The review history ledger passing its own deadline to the index lookup has no failing test. Accepted because observing it needs a stand-in Git that outlasts the ledger's 30 s deadline, and the lookup is bounded by its own 30 s otherwise, so a regression changes the bound, not the outcome.

### API/contract changes
`pose validate` scope selection fails with its cause when the untracked listing fails, where it used to narrow silently. The report note replaces whole known directories before other absolute paths.

### Data/storage changes
None.

### Technical risks
A scope selection that used to pass with a failing `ls-files` now fails; that failure was silent narrowing before.

## 4. Tasks

### Planning
- [x] Confirm each requirement against the current engine before starting.

### Implementation
- [x] Implement each requirement with a test that fails before it.

### Validation
- [ ] `pose validate --carry` and `pose check --strict`.

## 5. Decisions

## 6. Validation

### Strategy
Each requirement gets a regression test shown failing before its code.

### Deterministic checks

#### Test
- Command: `go test ./...` (in `pose-mcp`), then `pose validate --carry`.

### Execution log
- 2026-11-27: R1 adds `--relative` to the ready-transition diff; R2 replaces the root, repository top, Git directories, temporary and home directories whole before the generic pattern, which now also starts after `:`, a backtick or `[`; R3 cuts at a character (`TruncateUTF8`); R4 fails scope selection on a failed untracked listing; R5 adds stand-in Git tests. Each guard fails its test mutated out: `TestReadyTransitionsSeeSpecsBelowTheRepositoryTop`, `TestTheReportRemovesWholePaths`, `TestBoundsCutAtACharacter`, `TestScopeSelectionReportsAFailedUntrackedListing`, `TestTheIndexLookupNeedsNoPathFormat`, `TestIndexReadsRunGitInEnglish`, `TestTheIndexFallbacksKeepTheAnswer`, `TestTheRemainingIndexReadGuards`.

### Results summary
The full `pose-mcp` suite passes; the canonical result is refreshed with `pose validate --carry`.

### Requirement trace
- R1 [satisfied] test:TestReadyTransitionsSeeSpecsBelowTheRepositoryTop
- R2 [satisfied] test:TestTheReportRemovesWholePaths
- R3 [satisfied] test:TestBoundsCutAtACharacter
- R4 [satisfied] test:TestScopeSelectionReportsAFailedUntrackedListing
- R5 [satisfied] test:TestTheIndexLookupNeedsNoPathFormat, test:TestIndexReadsRunGitInEnglish, test:TestTheIndexFallbacksKeepTheAnswer, test:TestTheRemainingIndexReadGuards

### Known gaps
None.

## 7. Final Report

### Delivered scope
Ready transitions see specs below the repository top; the report removes whole paths and cuts at characters; a failed untracked listing fails scope selection; every index-read guard but the ledger's deadline has a failing test.

### Files and modules changed
`pose-mcp` check, report, validate scope and the read-only index reads, with their tests.

### Validation executed
`go test ./...` in `pose-mcp`; `pose check --strict`; `pose validate --carry --json-out .pose/results/delivery-validation.json`.

### Residual risks
L1 above.

### Follow-ups
None. Origin: `pose-validate-read-only-index-followups` (done 2026-11-27).
