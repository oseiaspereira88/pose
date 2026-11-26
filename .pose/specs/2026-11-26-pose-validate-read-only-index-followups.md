---
slug: pose-validate-read-only-index-followups
status: draft        # draft | in-progress | done | superseded | abandoned; blocked = operational wait, not terminal
created_at: 2026-11-26
completed_at:        # stamped on the transition to status: done
supersedes:          # slug of the superseded spec (when applicable)
depends_on:          # prerequisites, inline list: other-spec, milestone:<roadmap>/<id>, roadmap:<slug>
remediates:          # optional explicit links: spec:slug@defect-fix, finding:rva-<16hex>/F1@revert (never inferred from depends_on)
priority: 1
components: pose-mcp
task_type: bugfix
surface: standard
delivers:            # optional typed refs: surface:id, contract:id, capability:id, infrastructure:id, governance:id
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
- R5: A split-index read shall leave `.git/sharedindex.<hash>` untouched, or the limit shall be declared with the governed read view's answer to it measured (rounds 2 N12, 3 N15, 4 N19, 5 N24).
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
Origin: `pose-validate-reads-leave-the-index-alone` (done 2026-11-26).
