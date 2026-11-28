---
slug: pose-review-declared-limits-residuals
status: in-progress        # draft | in-progress | done | superseded | abandoned; blocked = operational wait, not terminal
created_at: 2026-11-27
completed_at:        # stamped on the transition to status: done
supersedes:          # slug of the superseded spec (when applicable)
depends_on:          # prerequisites, inline list: other-spec, milestone:<roadmap>/<id>, roadmap:<slug>
remediates:          # optional explicit links: spec:slug@defect-fix, finding:rva-<16hex>/F1@revert (never inferred from depends_on)
priority: 3
components: pose-mcp
task_type: bugfix
surface: standard
delivers: capability:review-declared-limits-residuals
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
- created: .pose/specs/2026-11-27-pose-review-declared-limits-residuals.md
- modified: pose-mcp/internal/pose/accepted_limits.go
- modified: pose-mcp/internal/pose/accepted_limits_test.go
- created: pose-mcp/internal/pose/accepted_limits_residuals_test.go
- modified: pose-mcp/internal/pose/review_bundle.go
- modified: pose-mcp/internal/pose/review_conclusions.go
- modified: pose-mcp/internal/cli/accepted_limits_lint_test.go
- modified: POSE.md
- modified: locales/pt-BR/POSE.md
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md

### Delivery targets
- capability:review-declared-limits-residuals module:pose-mcp profile:composed-capability entrypoint:pose-mcp/internal/cli/lintspec.go

### API/contract changes
`pose review attest` refuses, for any decision, a within-limit finding whose limit the bundle did not seal. A recorded run-backed attestation an earlier engine derived within an unsealed limit still matches its draft.

### Data/storage changes
None.

### Technical risks
Inline code spans are read per line; a span Markdown would continue on the next line is read as text, so a comment opener there can hide what follows, as the earlier engine did (round 1, N7, a follow-up).

## 4. Tasks

### Planning
- [x] Confirm each requirement against the current engine before starting.

### Implementation
- [x] Implement each requirement with a test that fails before it.

### Validation
- [ ] `pose validate --carry` and `pose check --strict`.

## 5. Decisions

### Decision D1
- Basis: R8
The legacy derivation is kept beside the current one only to compare records: when a run-backed attestation differs from its draft only in findings, and the earlier derivation (within-limit for any named limit) reproduces them, the record still matches. New attestations always use the current derivation.

## 6. Validation

### Strategy
Each requirement gets a regression test shown failing before its code.

### Deterministic checks

#### Test
- Command: `go test ./...` (in `pose-mcp`), then `pose validate --carry`.

### Execution log
- 2026-11-28: R1 reads the heading's words with a comment removed (`isAcceptedLimitsHeading`); R2 reads inline code spans by run length (`insideCodeSpan`); R3 opens fences only up to three spaces of indent; R4 lets the next heading close the section whatever comment it holds, words the indented-heading problem and bounds problems to 20 plus a count; R5 the manual in both locales and the embedded copies; R6 refuses an unsealed limit for any decision (`unsealedLimitBlockers`); R7 tests the checks themselves; R8 keeps the earlier derivation for comparison (D1) and removes `acceptedLimitsOf`. `TestAcceptedLimitsResiduals`, `TestNoAttestationNamesAnUnsealedLimit`, `TestAnEarlierWithinLimitDerivationStillMatches` and `TestTheChecksPrintWithinLimitFindings` fail with each guard mutated out.
- 2026-11-28: review round 1 (rrn-5b302e25c07a005e) requested changes. N1: only a line that is an unindented heading as written, its `#` first and outside a comment, closes the section; one that reads as a heading only once a comment is removed stays inside and refuses. N2: each problem is bounded to 300 bytes. N3: `PreflightReviewAttestation` refuses an unsealed limit for any decision, so preview and write agree; the error says attestation, not approval. N8: stale comments fixed. `TestAHeadingRevealedByACommentDoesNotCloseTheSection`, `TestAcceptedLimitProblemsAreBoundedInSize` and `TestThePreviewRefusesAnUnsealedLimitForAnyDecision` fail with each guard mutated out. N4–N7 are follow-ups.

### Results summary
The full `pose-mcp` suite passes; the canonical result is refreshed with `pose validate --carry`.

### Requirement trace
- R1 [satisfied] test:TestAcceptedLimitsResiduals
- R2 [satisfied] test:TestAcceptedLimitsResiduals
- R3 [satisfied] test:TestAcceptedLimitsResiduals
- R4 [satisfied] test:TestAcceptedLimitsResiduals, test:TestAHeadingRevealedByACommentDoesNotCloseTheSection, test:TestAcceptedLimitProblemsAreBoundedInSize
- R5 [satisfied] (manual, both locales)
- R6 [satisfied] test:TestNoAttestationNamesAnUnsealedLimit, test:TestThePreviewRefusesAnUnsealedLimitForAnyDecision
- R7 [satisfied] test:TestTheChecksPrintWithinLimitFindings
- R8 [satisfied] test:TestAnEarlierWithinLimitDerivationStillMatches

### Known gaps
None.

## 7. Final Report

### Delivered scope
The accepted-limits reader handles comments in the heading, code spans and indented code, closes on the next heading and bounds its problems; no attestation names an unsealed limit; the checks print within-limit findings; earlier records keep matching; the manual states the comment rules.

### Files and modules changed
`pose-mcp` accepted limits, review admission and conclusions, with their tests; the manual in both locales and its embedded copies.

### Validation executed
`go test ./...` in `pose-mcp`; `pose check --strict`; `pose validate --carry --json-out .pose/results/delivery-validation.json`.

### Residual risks
None beyond those of the origin spec.

### Follow-ups
Origin: `pose-review-declared-limits-followups` (done 2026-11-27).
- [open] Refuse a heading whose words hold a stray `-->` or a comment that closes on the next line (review round 1, N4). (owner:@pose-maintainers crit:low review:2026-12-28)
- [open] Count a tab as indentation to the next multiple of four when deciding whether a line opens a fence (review round 1, N5). (owner:@pose-maintainers crit:low review:2026-12-28)
- [open] Read a fence nested in a list item at four spaces as a fence, or record the changed answer as a decision (review round 1, N6). (owner:@pose-maintainers crit:low review:2026-12-28)
- [open] Read an inline code span continued over lines as Markdown does, so a comment opener inside it hides nothing (review round 1, N7). (owner:@pose-maintainers crit:low review:2026-12-28)
