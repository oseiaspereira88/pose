---
slug: pose-validate-index-reads-residuals
status: draft        # draft | in-progress | done | superseded | abandoned; blocked = operational wait, not terminal
created_at: 2026-11-27
completed_at:        # stamped on the transition to status: done
supersedes:          # slug of the superseded spec (when applicable)
depends_on:          # prerequisites, inline list: other-spec, milestone:<roadmap>/<id>, roadmap:<slug>
remediates:          # optional explicit links: spec:slug@defect-fix, finding:rva-<16hex>/F1@revert (never inferred from depends_on)
priority: 2
components: pose-mcp
task_type: bugfix
surface: standard
delivers:            # optional typed refs: surface:id, contract:id, capability:id, infrastructure:id, governance:id
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
- R5: The guards listed by rounds 2–5 shall each have a failing test: restoring `--path-format`, the `X^-` cut, the report note bound and stderr warning, the ledger's context, `GitEnglish` on the check, lookup and hand-over, the memfd write fallback, the private copy keeping the index's time, a stand-in Git that echoes an unknown `--path-format`, and a locale test independent of installed translations (rounds 1 N6–N7, 2 N16, 3 N21, 5 N2).

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
Declared when the spec starts.

### Delivery targets
Declared when the spec starts.

### API/contract changes
To be detailed when the spec starts.

### Data/storage changes
To be detailed when the spec starts.

### Technical risks
To be detailed when the spec starts.

## 4. Tasks

### Planning
- [ ] Confirm each requirement against the current engine before starting.

### Implementation
- [ ] Implement each requirement with a test that fails before it.

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

### Results summary

### Requirement trace

### Known gaps

## 7. Final Report

### Delivered scope

### Files and modules changed

### Validation executed

### Residual risks

### Follow-ups
Origin: `pose-validate-read-only-index-followups` (done 2026-11-27).
