---
slug: pose-review-subject-includes-its-index
status: draft
created_at: 2026-11-23
completed_at:
supersedes:
depends_on: 
remediates:
priority: 3
components: pose-mcp
task_type: bugfix
surface: standard
delivers:
---

# Spec: pose-review-subject-includes-its-index

## 1. Intent

### Goal
Let a review bundle's subject include the index commit that follows the last attributed commit, so the sealed checkout indexes the validation record it is reviewed against.

### Business value
Committing a validation record with a spec trailer and then the index leaves the subject at the record commit, whose tree still indexes the previous record. The sealed checkout then reports `no passed structured validation evidence is attributed to the review scope`. Every round of the last batch needed an extra binding commit after the index to work around it.

### Constraints
The subject still names exactly the attributed material. Only derived administrative records (the material-binding closed list) may follow the subject head into the sealed checkout.

### Non-goals
Changing change-set attribution or the material digest.

## 2. Requirements

### Functional
- R1: When the commits after the last attributed commit up to the sealing head change only administrative records, the seal shall materialize the sealed checkout at the sealing head and record both heads.
- R2: Evidence attribution in the sealed checkout shall read the index of that checkout, so a record committed with the trailer and indexed afterwards is attributed.
- R3: A non-administrative change after the last attributed commit shall keep today's behaviour.
- R4: A regression test shall seal after `record (trailer) → index (no trailer)` and find the record's evidence attributed.

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
