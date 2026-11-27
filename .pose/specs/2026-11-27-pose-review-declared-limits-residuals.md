---
slug: pose-review-declared-limits-residuals
status: draft        # draft | in-progress | done | superseded | abandoned; blocked = operational wait, not terminal
created_at: 2026-11-27
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

# Spec: pose-review-declared-limits-residuals

## 1. Intent

### Goal
Close what `pose-review-declared-limits-followups` left open: comments inside the heading words, inline code spans read as Markdown does, an indented code block taken for a fence, bounded and accurate refusal messages, a false plan warning, the manual text, the manual attestation check, end-to-end check tests, the legacy derivation and dead code.

### Business value
The spec closed on 2026-11-27 with follow-ups in its Final Report (round 1, N6–N9) and its approving attestation (round 2, N10–N15).

### Constraints
Each item keeps the behaviour its origin spec delivered; a fix that changes an answer states it as a decision.

### Non-goals
Reopening the decisions of the origin spec.

## 2. Requirements

### Functional
- R1: A comment marker inside the heading words of the accepted limits section shall refuse the section, not skip it silently (round 2, N10).
- R2: Inline code spans shall be read as Markdown reads them (a stray backtick, double-backtick spans) when deciding whether a marker opens a comment, so no spec without limits gets a false plan warning that changes its plan digest (rounds 1 N6, 2 N12).
- R3: Backticks inside a four-space indented code block shall not open a fence that hides a later accepted limits section (round 2, N15).
- R4: Refusal messages shall say what they refuse (an indented next heading, a comment on the next section's heading) and be bounded in number and size (round 2, N13).
- R5: The manual, in both locales and the embedded copies, shall state that any comment marker in the section or on its heading refuses the section and that an unterminated comment refuses every limit (round 2, N11).
- R6: A manual attestation of any decision whose within-limit finding names a limit the bundle did not seal shall be refused (round 1, N7).
- R7: The within-limit lines shall be tested through `pose review-check` and `pose closeout-check` themselves (round 1, N8).
- R8: A run-backed attestation an earlier engine recorded within an unsealed limit shall not read as differing from its recomputed draft, or the change of derivation shall be recorded (round 1, N9); `acceptedLimitsOf`, left without a caller, shall be removed (round 2, N14).

### Non-functional
Existing behaviour outside the stated requirements is unchanged.

### Security
No new input crosses a trust boundary without the existing confinement.

### Compatibility
Records and indexes written by earlier engines keep their meaning.

## 3. Technical Plan

### Affected areas
`pose-mcp`: `accepted_limits.go`, `review_closeout.go`, `review_conclusions.go`, the manual in both locales, and their tests.

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
Origin: `pose-review-declared-limits-followups` (done 2026-11-27).
