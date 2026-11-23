---
slug: pose-validate-carry-report-and-closeout-coverage
status: draft
created_at: 2026-11-23
completed_at:
supersedes:
depends_on: pose-validate-carries-unchanged-checks
remediates:
priority: 2
components: pose-mcp
task_type: feature
surface: standard
delivers:
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
- R3: An end-to-end fixture shall produce a carried canonical record, seal a review bundle over it, verify it and run `closeout-check`, asserting that carried evidence is judged at the head it ran at.
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
- [ ] Confirm the requirements against the current engine before starting.

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
