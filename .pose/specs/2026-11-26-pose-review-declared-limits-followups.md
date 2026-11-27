---
slug: pose-review-declared-limits-followups
status: in-progress        # draft | in-progress | done | superseded | abandoned; blocked = operational wait, not terminal
created_at: 2026-11-26
completed_at:        # stamped on the transition to status: done
supersedes:          # slug of the superseded spec (when applicable)
depends_on:          # prerequisites, inline list: other-spec, milestone:<roadmap>/<id>, roadmap:<slug>
remediates:          # optional explicit links: spec:slug@defect-fix, finding:rva-<16hex>/F1@revert (never inferred from depends_on)
priority: 2
components: pose-mcp
task_type: bugfix
surface: standard
delivers: capability:review-declared-limits-followups
---

# Spec: pose-review-declared-limits-followups

## 1. Intent

### Goal
Close the follow-ups left by `pose-review-declared-threat-model`: the remaining Markdown the accepted-limits reader misreads, a limit named by a non-approving draft, the manual `--finding` path, the published schemas, and the text output of the checks.

### Business value
The spec closed on 2026-11-26 after eleven review rounds; its Final Report lists two follow-ups and its approving attestation five more (round 11, N1–N5).

### Constraints
Each item keeps the behaviour its origin spec delivered; a fix that changes an answer states it as a decision.

### Non-goals
Reopening the decisions of the origin spec.

## 2. Requirements

### Functional
- R1: An indented continuation line (one to three spaces) that starts with `# ` shall not end the section and cut the limit: it is either read as the limit's text or reported, and the limit is not sealed in part (round 11, N1).
- R2: An unterminated `<!--` anywhere, also after a closed comment, and any comment marker inside the accepted limits section, inline code included, shall be reported, and no limit shall be sealed cut by them (rounds 9 N4, 10 N4, 11 N2; D1).
- R3: `pose review attest --finding` shall accept a limit for a `within-limit` finding, or the CLI reference shall stop listing it there (round 11, N3).
- R4: A non-approving draft whose follow-up names a limit the bundle did not seal shall be refused or recorded without `within-limit`, so no attestation names a limit that does not exist (round 11, N4).
- R5: The published review plan and bundle schemas shall declare `accepted_limits` (and `limitations`), with the schema drift test covering them (round 11, N5).
- R6: `pose review-check` and `pose closeout-check` shall show within-limit findings in their text output, as their JSON does (rounds 9 N5, 10 N5).

### Non-functional
Existing behaviour outside the stated requirements is unchanged.

### Security
No new input crosses a trust boundary without the existing confinement.

### Compatibility
Records and indexes written by earlier engines keep their meaning.

## 3. Technical Plan

### Affected areas
`pose-mcp`: `accepted_limits.go`, `review_conclusions.go`, `review_bundle.go`, the attest CLI, the published schemas, `review-check`/`closeout-check` output, `docs-site/docs/cli.md`.

### Artifacts
- created: .pose/specs/2026-11-26-pose-review-declared-limits-followups.md
- modified: pose-mcp/internal/pose/accepted_limits.go
- modified: pose-mcp/internal/pose/accepted_limits_test.go
- modified: pose-mcp/internal/pose/review_conclusions.go
- modified: pose-mcp/internal/pose/review_plan.go
- modified: pose-mcp/internal/cli/lintspec.go
- modified: pose-mcp/internal/pose/schema_test.go
- modified: pose-mcp/schemas/v1/review-plan.schema.json
- modified: pose-mcp/schemas/v1/review-bundle.schema.json
- modified: pose-mcp/internal/cli/review_closeout.go
- modified: pose-mcp/internal/cli/accepted_limits_lint_test.go
- modified: docs-site/docs/cli.md
- modified: POSE.md
- modified: locales/pt-BR/POSE.md
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md

### Delivery targets
- capability:review-declared-limits-followups module:pose-mcp profile:composed-capability entrypoint:pose-mcp/internal/cli/lintspec.go

### API/contract changes
`pose review attest --finding` takes a ninth field, `limit`. The review plan and bundle schemas declare `accepted_limits` and `limitations`. `pose review-check` and `pose closeout-check` print `review.within_limit` / `closeout.within_limit` lines.

### Data/storage changes
None.

### Technical risks
A spec whose accepted limits section held an HTML comment now has its limits refused; none in this repository does.

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
R2 as drafted asked that a comment written in inline code keep its text. Telling inline code from a real comment is the kind of Markdown reading that took the origin spec four rounds to make fail-closed; instead, the reader makes one pass over the raw text, tracking comments and fences on the same lines it reads the section from. Outside the section a comment hides what it holds, as Markdown does, a marker in inline code (an odd number of backticks before it on its line) opens none, and comments inside fences are text. Inside the section, and on its heading line, any comment marker refuses the whole section; an unterminated comment anywhere refuses every limit. The ready check reads the same raw text, and the review plan states the problems that kept a spec's limits from being sealed among its warnings.

## 6. Validation

### Strategy
Each requirement gets a regression test shown failing before its code.

### Deterministic checks

#### Test
- Command: `go test ./...` (in `pose-mcp`), then `pose validate --carry`.

### Execution log
- 2026-11-27: R1 reports a heading-like line indented under a limit; R2 reports any unterminated comment and any comment marker in the section (D1); R3 adds `limit` to `--finding`; R4 records `within-limit` only for a sealed limit; R5 declares the fields in both schemas; R6 prints within-limit findings. `TestAcceptedLimitsRefuseWhatWouldCutThem`, `TestOnlyASealedLimitIsRecordedWithinLimit`, `TestSchemaDrift`, `TestTheBundleSchemaDeclaresTheAcceptedLimits` and `TestWithinLimitFindingsAreTypedAndShown` fail with each guard mutated out.
- 2026-11-27: this spec's own ready check failed: its R2 names `<!--` in inline code, which the new check took for an unterminated comment. A marker in inline code now opens no comment (`htmlCommentStart`, `stripLimitComments`); `TestAcceptedLimitsIgnoreAMarkerNamedInCode` fails with the backtick rule mutated out. The five specs of this repository that declare accepted limits pass the ready check unchanged.
- 2026-11-27: review round 1's first run reached its time and capture limits with no draft (a 2.5 GB Go cache); the rerun (rrn-79b4d7a1126643d3) requested changes: the ready check read comment-stripped text (N1), and the separate comment pre-scan disagreed with the parser on comments by the heading, after an indented heading, in fences and after another comment (N2–N4). Withdrawn for a single raw pass (D1); `TestAcceptedLimitsReadCommentsWhereTheSectionIs` (the reviewer's cases, a fence holding an unterminated marker, an indented heading under an indented marker, N5), the ready-check case and the plan-warning case fail with the raw read, the heading rule, the indent rule, the fence rule, the comment tracking or the warning mutated out. `--finding` keeps a limit only for `within-limit` (part of N7).

### Results summary
The full `pose-mcp` suite passes; the canonical result is refreshed with `pose validate --carry`.

### Requirement trace
- R1 [satisfied] test:TestAcceptedLimitsRefuseWhatWouldCutThem
- R2 [satisfied] test:TestAcceptedLimitsReadCommentsWhereTheSectionIs, test:TestReadyCheckJudgesAcceptedLimits, test:TestTheReviewPlanSealsTheAcceptedLimits, test:TestAcceptedLimitsRefuseWhatWouldCutThem, test:TestAcceptedLimitsIgnoreAMarkerNamedInCode, test:TestAcceptedLimitsReadWrappedEntriesAndSkipExamples
- R3 [satisfied] test:TestWithinLimitFindingsAreTypedAndShown
- R4 [satisfied] test:TestOnlyASealedLimitIsRecordedWithinLimit
- R5 [satisfied] test:TestSchemaDrift, test:TestTheBundleSchemaDeclaresTheAcceptedLimits
- R6 [satisfied] test:TestWithinLimitFindingsAreTypedAndShown

### Known gaps
None.

## 7. Final Report

### Delivered scope
The accepted-limits reader refuses what would cut a limit; a within-limit finding can be typed, names only a sealed limit, is declared in the schemas and shown by the checks.

### Files and modules changed
`pose-mcp` accepted limits, conclusions, attest CLI, check output and schemas; the CLI reference and the manual in both locales with their embedded copies.

### Validation executed
`go test ./...` in `pose-mcp`; `pose check --strict`; `pose validate --carry --json-out .pose/results/delivery-validation.json`.

### Residual risks
None beyond those of the origin spec.

### Follow-ups
Origin: `pose-review-declared-threat-model` (done 2026-11-26).
- [open] Read inline code spans as Markdown does (a stray backtick, double-backtick spans) when deciding whether a marker opens a comment (review round 1, N6). (owner:@pose-maintainers crit:low review:2026-12-27)
- [open] Refuse a manual attestation of any decision whose within-limit finding names a limit the bundle did not seal (review round 1, N7). (owner:@pose-maintainers crit:low review:2026-12-27)
- [open] Test the within-limit lines through `pose review-check` and `pose closeout-check` themselves (review round 1, N8). (owner:@pose-maintainers crit:low review:2026-12-27)
- [open] Keep a run-backed attestation an earlier engine recorded within an unsealed limit from differing from its recomputed draft, or record the change of derivation (review round 1, N9). (owner:@pose-maintainers crit:low review:2026-12-27)
