---
slug: pose-run-verification-pins-brief-inputs
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

# Spec: pose-run-verification-pins-brief-inputs

## 1. Intent

### Goal
Verify a delegated run's provenance against the brief inputs it was dispatched with, so records retained after the seal do not change the re-rendered brief and break verification of an approved run.

### Business value
Retaining the exhausted attempts of an earlier bundle after a later bundle was approved made `review verify` fail with `run provenance: brief/template/notes/transmitted bytes differ`, because the previous-round section of the re-rendered brief changed. The records had to be kept outside the repository.

### Constraints
A brief must still be bound to what the reviewer actually received; only inputs recorded at dispatch may stand in for a re-render.

### Non-goals
Changing round accounting.

## 2. Requirements

### Functional
- R1: The run record shall store the digest of every brief input, including the round ledger snapshot, at dispatch.
- R2: Verification shall re-render the brief from the recorded inputs, not from the current ledger.
- R3: Retaining earlier bundles' runs after an approval shall leave `review verify` and `pose close` of the approved scope unchanged.
- R4: A regression test shall approve a scope, retain an older bundle's runs and verify the approval still holds.

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
