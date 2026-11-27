---
slug: pose-validate-index-reads-residuals
status: done
created_at: 2026-11-27
completed_at: 2026-11-27
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
- R2: The versioned report shall carry no absolute path of the checkout when the change list cannot be read: it states that the list could not be read, and the warning on standard error carries Git's message (round 5, N1; D1).
- R3: The bound on Git's message shall cut at a character, never inside a UTF-8 sequence (round 5, N4).
- R4: A failing `ls-files` in worktree scope selection shall fail the selection with its cause instead of silently dropping untracked files (round 5, N5).
- R5: The guards listed by rounds 2–5 shall each have a failing test: restoring `--path-format`, the `X^-` cut, the report's stderr warning, `GitEnglish` on the check, lookup and hand-over, the memfd write fallback, the private copy keeping the index's time, a stand-in Git that echoes an unknown `--path-format`, and a locale test independent of installed translations; the ledger's context is L1 (rounds 1 N6–N7, 2 N16, 3 N21, 5 N2).

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
- modified: pose-mcp/internal/cli/validate_reads_index_test.go
- modified: pose-mcp/internal/pose/git_worktree_reads_linux_test.go

### Delivery targets
- capability:validate-index-reads-residuals module:pose-mcp profile:composed-capability entrypoint:pose-mcp/internal/cli/validate.go

### Accepted limits
- L1: The review history ledger passing its own deadline to the index lookup has no failing test. Accepted because observing it needs a stand-in Git that outlasts the ledger's 30 s deadline, and the lookup is bounded by its own 30 s otherwise, so a regression changes the bound, not the outcome.

### API/contract changes
`pose validate` scope selection fails with its cause when the untracked listing fails, where it used to narrow silently, and names changes relative to the root. The report states that its list could not be read, without Git's message.

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

### Decision D1
- Basis: R2
R2 as drafted asked the redaction to replace a whole absolute path in Git's message. A path in free text has no delimiter: spaces, quotes and brackets are legal in it, and replacing known directories as substrings left tails of other paths and cut relative names (review round 1, N2–N6). So the versioned report states only that the list could not be read, and Git's message goes to the warning on standard error, which is not versioned.

## 6. Validation

### Strategy
Each requirement gets a regression test shown failing before its code.

### Deterministic checks

#### Test
- Command: `go test ./...` (in `pose-mcp`), then `pose validate --carry`.

### Execution log
- 2026-11-27: R1 adds `--relative` to the ready-transition diff; R2 replaces the root, repository top, Git directories, temporary and home directories whole before the generic pattern, which now also starts after `:`, a backtick or `[`; R3 cuts at a character (`TruncateUTF8`); R4 fails scope selection on a failed untracked listing; R5 adds stand-in Git tests. Each guard fails its test mutated out: `TestReadyTransitionsSeeSpecsBelowTheRepositoryTop`, `TestTheReportStatesNoPath`, `TestBoundsCutAtACharacter`, `TestScopeSelectionReportsAFailedUntrackedListing`, `TestTheIndexLookupNeedsNoPathFormat`, `TestIndexReadsRunGitInEnglish`, `TestTheIndexFallbacksKeepTheAnswer`, `TestTheRemainingIndexReadGuards`.
- 2026-11-27: the first matrix failed `release-test-compile-darwin` and `-windows`: a test used the Linux-only memfd hooks; it moved to the Linux test file. Review round 1 (rrn-977f027e0e7c7ee5) requested changes. N1: the ready transition reads HEAD's text with `HEAD:./`, relative to the root, so a spec in progress at HEAD below the repository top is not a new transition. N2–N6: the redaction is withdrawn (D1). N7: scope selection runs its diff with `--relative`. `TestReadyTransitionsSeeSpecsBelowTheRepositoryTop` and `TestTheReportStatesNoPath` fail with `HEAD:./`, `--relative` or the note mutated.
- 2026-11-27: review round 2 (rrn-3dba382397f270c7) requested changes: `--relative` dropped changes outside the root, narrowing validation (N1). Withdrawn: the diff names changes from the repository top, and each is named from the root (`scopeRootRelative`): inside it without the prefix, outside it as `../<name>`, a change outside every module that selects the root. The test changes a tracked file at the top and fails with either branch mutated out. N3: R3 and R5 no longer name the removed note bound, and the bounded-note assertion is gone. N4: the report warning test requires Git's message. N2 is a follow-up.
- 2026-11-27: review round 3 (rrn-c3a4b1c0fd2bd2d6) requested changes: `diff.relative=true` in the configuration made the diff root-relative, so the new naming misplaced changes (N1); the diff now pins `diff.relative=false`. N2: the root prefix keeps a leading space and a failed lookup fails the selection. N3: a change outside the root is named by its relative path from the root, also for a root two levels down. N5: stale comments fixed. `TestReadyTransitionsSeeSpecsBelowTheRepositoryTop` (with `diff.relative=true`) and `TestScopeSelectionBelowADirectoryWithALeadingSpace` fail with the pin or the prefix rule mutated. N4 is a follow-up.

### Results summary
The full `pose-mcp` suite passes; the canonical result is refreshed with `pose validate --carry`.

### Requirement trace
- R1 [satisfied] test:TestReadyTransitionsSeeSpecsBelowTheRepositoryTop, test:TestScopeSelectionBelowADirectoryWithALeadingSpace
- R2 [satisfied] test:TestTheReportStatesNoPath
- R3 [satisfied] test:TestBoundsCutAtACharacter
- R4 [satisfied] test:TestScopeSelectionReportsAFailedUntrackedListing
- R5 [satisfied] test:TestTheIndexLookupNeedsNoPathFormat, test:TestIndexReadsRunGitInEnglish, test:TestTheIndexFallbacksKeepTheAnswer, test:TestTheRemainingIndexReadGuards

### Known gaps
None.

## 7. Final Report

### Delivered scope
Ready transitions and scope selection see a root below the repository top; the report states no path and bounds cut at characters; a failed untracked listing fails scope selection; every index-read guard but the ledger's deadline has a failing test.

### Files and modules changed
`pose-mcp` check, report, validate scope and the read-only index reads, with their tests.

### Validation executed
`go test ./...` in `pose-mcp`; `pose check --strict`; `pose validate --carry --json-out .pose/results/delivery-validation.json`.

### Residual risks
L1 above.

### Follow-ups
Origin: `pose-validate-read-only-index-followups` (done 2026-11-27).
- [open] Name carry exclusions and the `show <head>:<path>` reads of carry from the root, as scope selection now does, for when carry runs below the repository top (review round 2, N2). (owner:@pose-maintainers crit:low review:2026-12-27)
- [open] List untracked files outside the root when the root is below the repository top, as tracked changes outside it now are (review round 3, N4; pre-existing). (owner:@pose-maintainers crit:low review:2026-12-27)
