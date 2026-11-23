---
slug: pose-review-declared-threat-model
status: draft
created_at: 2026-11-23
completed_at:
supersedes:
depends_on: 
remediates:
priority: 4
components: pose-mcp
task_type: feature
surface: standard
delivers:
---

# Spec: pose-review-declared-threat-model

## 1. Intent

### Goal
Let a spec declare the accepted residual limits of a guarantee it makes, and let the review brief treat findings inside those limits as follow-ups to classify, not blockers to chase.

### Business value
Each of the 23 rounds of `pose-validate-carries-unchanged-checks` found a more remote adversarial variant: concurrent HEAD movement, case folding, Unicode normalization, symlinks inside submodules. Every one was real, but no round could converge, because the spec had no place to say which environment the guarantee assumes. A cap on rounds would approve something unreviewed; a declared threat model converges by explicit classification.

### Constraints
A declared limit never hides a defect inside the declared scope. The reviewer may dispute a limit; the dispute is a finding with its own classification. Declared limits are visible in the closeout and the attestation.

### Non-goals
Limiting the number of review rounds.

## 2. Requirements

### Functional
- R1: A spec shall be able to declare `### Accepted limits` entries with stable ids (`L<N>`), each naming the assumption and why it is accepted.
- R2: The review brief shall list the declared limits and instruct the reviewer to classify a finding that falls inside one as a follow-up naming the limit, or to dispute the limit explicitly.
- R3: The conclusion and the attestation shall record which findings were classified under which limit.
- R4: `lint-spec --ready-check` shall require accepted limits to be justified and the ADR, when present, to reference them.

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
