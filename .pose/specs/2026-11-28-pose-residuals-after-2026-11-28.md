---
slug: pose-residuals-after-2026-11-28
status: draft        # draft | in-progress | done | superseded | abandoned; blocked = operational wait, not terminal
created_at: 2026-11-28
completed_at:        # stamped on the transition to status: done
supersedes:          # slug of the superseded spec (when applicable)
depends_on:          # prerequisites, inline list: other-spec, milestone:<roadmap>/<id>, roadmap:<slug>
remediates:          # optional explicit links: spec:slug@defect-fix, finding:rva-<16hex>/F1@revert (never inferred from depends_on)
priority: 3
components: pose-mcp
task_type: bugfix
surface: standard
delivers:            # optional typed refs: surface:id, contract:id, capability:id, infrastructure:id, governance:id
---

# Spec: pose-residuals-after-2026-11-28

## 1. Intent

### Goal
Close what the three residual specs closed on 2026-11-27 and 2026-11-28 left open: the index reads below the repository top, the resume races inside the plans, and the remaining Markdown edges of the accepted-limits reader.

### Business value
`pose-validate-index-reads-residuals`, `pose-review-resume-residuals` and `pose-review-declared-limits-residuals` were approved with follow-ups in their Final Reports and attestations. All are low; each is a gap in a reader or a test, not a wrong answer on the paths this repository uses.

### Constraints
Each item keeps the behaviour its origin spec delivered; a fix that changes an answer states it as a decision.

### Non-goals
Reopening the decisions of the origin spec.

## 2. Requirements

### Functional
- R1: Index reads below the repository top: carry's exclusions and its `show <head>:<path>` reads are named from the root; an untracked file outside the root is listed; a changed spec whose file name has a space or a non-ASCII letter is a ready-transition candidate; a test names the root-prefix lookup; `report.go` no longer describes the removed note; the R1 trace of the origin cites a ready-transition test (origin `pose-validate-index-reads-residuals`, follow-ups and round 4 N1–N4).
- R2: Resume races: the claim-publication conflict is tested through `beforeContinuationClaim`; the plans' waits take the dispatch's context; a run with a random recorded runtime name is resumed end to end in a test; the handoff plan's pending branches are tested (origin `pose-review-resume-residuals`, follow-ups and round 3 N10–N11).
- R3: Accepted-limits Markdown edges: a heading with a stray `-->` or a comment closing on the next line refuses the section; a tab counts as indentation to the next multiple of four; a fence nested in a list item, opened at four spaces and closed at two, hides nothing; a fenced block indented under a limit is reported, not flattened into its text; an inline code span continued over lines is read as Markdown does (origin `pose-review-declared-limits-residuals`, follow-ups and round 3 N12–N15).
- R4: A spec's Known gaps and Residual risks name the follow-ups it leaves open (round 3, N16).

### Non-functional
Existing behaviour outside the stated requirements is unchanged.

### Security
No new input crosses a trust boundary without the existing confinement.

### Compatibility
Records and indexes written by earlier engines keep their meaning.

## 3. Technical Plan

### Affected areas
`pose-mcp`: `validate_carry.go`, `validate_scope.go`, `check.go`, `report.go`, `review_dispatch.go`, `review_run_continuity.go`, `delegation_handoff.go`, `accepted_limits.go`, and their tests.

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
Origins: `pose-validate-index-reads-residuals`, `pose-review-resume-residuals`, `pose-review-declared-limits-residuals`.
