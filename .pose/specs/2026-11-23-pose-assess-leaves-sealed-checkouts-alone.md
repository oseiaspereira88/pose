---
slug: pose-assess-leaves-sealed-checkouts-alone
status: draft
created_at: 2026-11-23
completed_at:
supersedes:
depends_on: 
remediates:
priority: 2
components: pose-mcp
task_type: bugfix
surface: standard
delivers:
---

# Spec: pose-assess-leaves-sealed-checkouts-alone

## 1. Intent

### Goal
Make `pose assess` commands leave a sealed review checkout unchanged, so a reviewer running a planned assessment tool does not fail its own run.

### Business value
Reviewer runs failed with `the run changed files in the sealed worktree` after `pose assess integrate` or `pose assess tech-debt` persisted `.pose/assessments/*` and `.pose/state/*` inside the sealed checkout: two runs of `pose-validate-carries-unchanged-checks` and several Harne8 rounds. The workaround, copying the tree into scratch, fails in read-only sandboxes.

### Constraints
Assessments outside sealed checkouts keep persisting as today. The output a reviewer sees is the same JSON the persisted run would write.

### Non-goals
Changing assessment content or scoring.

## 2. Requirements

### Functional
- R1: `pose assess` subcommands shall accept `--no-persist`, printing their result without writing any file.
- R2: When the working root is a sealed review checkout (`POSE_SEALED_CHECKOUT` or the review runtime marker), `pose assess` shall not persist, whether or not `--no-persist` is given, and shall say so.
- R3: The review brief shall plan assessment tools with the non-persisting form.
- R4: A regression test shall run each assessment command inside a sealed checkout fixture and prove the tree unchanged.

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
