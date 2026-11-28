---
slug: pose-validate-carry-report-and-closeout-coverage
status: done
created_at: 2026-11-23
completed_at: 2026-11-28
supersedes:
depends_on: pose-validate-carries-unchanged-checks
remediates:
priority: 2
components: pose-mcp
task_type: feature
surface: standard
delivers: capability:validate-carry-report-and-closeout-coverage
---

# Spec: pose-validate-carry-report-and-closeout-coverage

## 1. Intent

### Goal
Close the two follow-ups of `pose-validate-carries-unchanged-checks`: `--report` must present carried checks as carried, and an end-to-end test must seal, verify and close a scope over a carried record.

### Business value
F2: `validate --carry --report` lists carried checks among executed commands with their old outcome and duration, without `carried_from` or `carried_generated_at`, so a newly dated report presents reused observations as new ones. F1: the delivery reader is tested for the carried head and time, but no test proves bundle sealing, `review verify` and `closeout-check` judge a carried record correctly.

### Constraints
Executed checks are reported as today. A carried check is never shown as executed.

### Non-goals
Changing carry eligibility or selection.

## 2. Requirements

### Functional
- R1: The report shall list carried checks separately from executed commands, each with its outcome, `carried_from` and `carried_generated_at`; an all-carried run lists no executed command.
- R2: The structured summary feeding the report shall keep carried and executed checks distinct.
- R3: An end-to-end fixture shall seal a review bundle over a canonical record in the shape `validate --carry` writes (a check with `carried_from` and `carried_generated_at`, carried from a commit of the spec's change set), verify it, close the spec and run `closeout-check`, asserting that the carried evidence is judged at the head it ran at (D1).
- R4: This spec closes follow-ups F1 and F2 of `pose-validate-carries-unchanged-checks`.

### Non-functional
Existing behaviour outside the stated requirements is unchanged.

### Security
No new input crosses a trust boundary without the existing confinement.

### Compatibility
Records and indexes written by earlier engines keep their meaning.

## 3. Technical Plan

### Affected areas
`pose-mcp` (CLI and engine), the manual and its embedded copy.

### Artifacts
- created: .pose/specs/2026-11-23-pose-validate-carry-report-and-closeout-coverage.md
- modified: pose-mcp/internal/cli/report.go
- modified: pose-mcp/internal/cli/validate.go
- modified: pose-mcp/internal/cli/validate_carry_test.go
- modified: pose-mcp/internal/cli/review_closeout_test.go
- modified: POSE.md
- modified: locales/pt-BR/POSE.md
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md

### Delivery targets
- capability:validate-carry-report-and-closeout-coverage module:pose-mcp profile:composed-capability entrypoint:pose-mcp/internal/cli/validate.go

### API/contract changes
The report `pose validate --report` writes gains a `## Carried Checks` section; carried checks leave `## Validation Commands` and `## Results`.

### Data/storage changes
None.

### Technical risks
A carried check is sealed only when the commit it ran at belongs to the spec's change set (or its provenance matches), as validation results are today; the end-to-end test uses that realistic shape.

## 4. Tasks

### Planning
- [x] Confirm the requirements against the current engine before starting.

### Implementation
- [x] Implement each requirement with a test that fails before it.

### Validation
- [ ] `pose validate --carry` and `pose check --strict`.

## 5. Decisions

### Decision D1
- Basis: R3
R3 as drafted asked the fixture to produce the carried record with the engine. The review fixture has no validation matrix, and a carry run needs a matrix, modules and a prior canonical result; building that around the review machinery would test validate twice. The record is written in the exact shape `validate --carry` writes, carried from a commit of the spec's change set as a real carry is, while `TestCarryRunsOnlyTheChangedModule` and its neighbours prove the engine writes that shape. The record's provenance digest is a placeholder, so the currency path exercised is the commit-in-change-set one; the provenance path is covered by the delivery tests.

## 6. Validation

### Strategy
Each requirement gets a regression test shown failing before its code.

### Deterministic checks

#### Test
- Command: `go test ./...` (in `pose-mcp`), then `pose validate --carry`.

### Execution log
- 2026-11-28: R1/R2 `validationSummaryOf` keeps carried checks in `Carried` (outcome, `carried_from`, `carried_generated_at`), out of commands and results; `renderReportMarkdown` adds `## Carried Checks` and says no command ran when every check was carried. R3 `TestACarriedRecordIsSealedVerifiedAndClosed` seals a bundle over a record whose unit check was carried from the feature commit to a later record commit of the spec, checks it is sealed `carried-forward` at the commit it ran at, then auto-attests, verifies and closes. A first version carried from a commit outside the change set with a placeholder provenance and the check was not sealed, which is the delivery reader's rule, not a defect. `TestCarryReportListsCarriedChecksApart` and the R3 test fail with the carried split, the all-carried note or the reader's carried head mutated out.
- 2026-11-28: review round 1 (rrn-2d48a8ac0f67e6c6) requested changes. N1: the all-carried note appears only when no check ran or was skipped (`reportListsACheck`); a mixed run with a skipped check fails the test with the condition removed. N2: the R3 test now runs `closeout-check` and asserts `closeout.terminal=true`; R3's wording and decision D1 state why the record is hand-written in the carry shape. N4: the manual in both locales and the embedded copies describes the report. N5: the test checks the headings exist, the carried outcome, and that a carried check is absent from Results; imports are in order. N6: a carried line keeps its skip reason and exit code. N3 is a follow-up: the origin spec is done, so its F1 and F2 lines are not edited; this spec records that it closes them.

### Results summary
The full `pose-mcp` suite passes; the canonical result is refreshed with `pose validate --carry`.

### Requirement trace
- R1 [satisfied] test:TestCarryReportListsCarriedChecksApart
- R2 [satisfied] test:TestCarryReportListsCarriedChecksApart
- R3 [satisfied] test:TestACarriedRecordIsSealedVerifiedAndClosed
- R4 [satisfied] (this spec's Follow-ups record that it closes F1 and F2; the origin spec is done and not edited)

### Known gaps
None.

## 7. Final Report

### Delivered scope
`validate --carry --report` lists carried checks apart, with where and when they ran; a carried canonical record is sealed, verified and closed end to end.

### Files and modules changed
`pose-mcp` report rendering and validation summary, with their tests.

### Validation executed
`go test ./...` in `pose-mcp`; `pose check --strict`; `pose validate --carry --json-out .pose/results/delivery-validation.json`.

### Residual risks
None.

### Follow-ups
None. Closes F1 and F2 of `pose-validate-carries-unchanged-checks`.
