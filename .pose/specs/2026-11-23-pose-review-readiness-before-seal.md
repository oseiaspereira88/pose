---
slug: pose-review-readiness-before-seal
status: draft
created_at: 2026-11-23
completed_at:
supersedes:
depends_on: 
remediates:
priority: 1
components: pose-mcp
task_type: feature
surface: standard
delivers:
---

# Spec: pose-review-readiness-before-seal

## 1. Intent

### Goal
Refuse at seal, and let the reviewer check inside its run, every condition the conclusion preflight and `pose close` would otherwise refuse after a full review round.

### Business value
Closing `pose-validate-carries-unchanged-checks` took 23 review rounds. About seven were lost to conditions known before the reviewer started: a missing delivery declaration that `close` refused only after approval, a decision with no requirement basis, a finding id with a round prefix, `prior_findings` naming findings no concluded round holds open, criterion evidence outside the criterion's component, and a note file over 16 KiB. Each cost a seal, a delegated run and about 40 minutes.

### Constraints
The checks reuse the existing preflight and closeout predicates; no rule is duplicated. A seal refusal names the rule and the fix. Nothing weakens a gate: what is refused later is refused earlier, never accepted.

### Non-goals
Capping review rounds, changing review independence, or judging the subject itself.

## 2. Requirements

### Functional
- R1: `pose review bundle --seal` shall refuse a spec scope whose attributed changes touch a delivery root without a delivery declaration, with the same rule and message `pose close` uses (`undeclared-delivery`).
- R2: The seal shall refuse a spec scope whose decisions do not each name a requirement or constraint basis that the design check requires, with the same rule the conclusion preflight applies.
- R3: A new `pose review draft-check <run|bundle> --draft <file>` shall run inside the review sandbox without writing. It reports every reason the conclusion preflight would refuse that draft: unreadable block, prefixed new finding ids, `prior_findings` outside the open findings, criterion evidence outside its component, undeclared decisions in mappings. It exits non-zero when any applies.
- R4: The review brief shall instruct the reviewer to run `draft-check` on its draft before finishing, and list the open prior findings it must account for.
- R5: `pose delegate` shall refuse a note file over the adapter limit with a message naming the note, its size and the limit, instead of the generic adapter-file error.
- R6: When the instance has a conclusions issuer but the run is not concluded automatically, `pose delegate review` shall print the exact `pose review conclude <run>` command as the next step.

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
