---
slug: pose-assess-leaves-sealed-checkouts-alone
status: in-progress
created_at: 2026-11-23
completed_at:
supersedes:
depends_on: 
remediates:
priority: 2
components: pose-mcp
task_type: bugfix
surface: standard
delivers: capability:assess-leaves-sealed-checkouts-alone
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
- created: .pose/specs/2026-11-23-pose-assess-leaves-sealed-checkouts-alone.md
- created: pose-mcp/internal/pose/assessment_sealed.go
- created: pose-mcp/internal/pose/assessment_sealed_test.go
- modified: pose-mcp/internal/pose/discovery.go
- modified: pose-mcp/internal/pose/integration.go
- modified: pose-mcp/internal/pose/techdebt.go
- modified: pose-mcp/internal/pose/review_plan.go
- modified: pose-mcp/internal/cli/assess.go
- created: pose-mcp/internal/cli/assess_sealed_test.go
- modified: pose-mcp/internal/cli/help_catalog.go
- modified: POSE.md
- modified: locales/pt-BR/POSE.md
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md


### Delivery targets

- capability:assess-leaves-sealed-checkouts-alone module:pose-mcp profile:composed-capability entrypoint:pose-mcp/internal/cli/assess.go


### API/contract changes
`pose assess discover|integrate|tech-debt` accept `--no-persist`. In a sealed review checkout they do not persist, and `init`, `snapshot` and `request` refuse. The review plan's assessment tools carry `--no-persist`, so bundles sealed from now on plan that form.


### Data/storage changes
None. The assessment writers of the Store refuse in a sealed checkout with `ErrSealedAssessmentWrite`.


### Technical risks
The sealed checkout is recognized by `POSE_SEALED_CHECKOUT`, which the dispatch sets with the checkout's path. A process that clears the variable is outside this guarantee (L1 below); the dispatch's own post-run comparison of the checkout still fails such a run.


## 4. Tasks

### Planning
- [ ] Confirm the requirements against the current engine before starting.

### Implementation
- [ ] Implement each requirement with a test that fails before it.

### Validation
- [ ] `pose validate --carry` and `pose check --strict`.

## 5. Decisions

### Decision D1
- Basis: R1, R2
The CLI decides not to persist, and the Store's assessment writers refuse in a sealed checkout as the last line, so no other caller can publish an assessment there either. A refusal from the writer would fail the command; the CLI therefore skips the writes and says why on standard error, keeping standard output the same result a persisted run prints.

### Decision D2
- Basis: R3
The review plan carries `--no-persist` explicitly rather than relying on the environment, so the planned command is correct in any checkout a reviewer copies it into.

## 6. Validation

### Strategy
Each requirement gets a regression test shown failing before its code.

### Deterministic checks

#### Test
- Command: `go test ./...` (in `pose-mcp`), then `pose validate --carry`.

### Execution log
- 2026-11-23: implemented R1–R4. Each guard has a test that fails with it mutated out: the CLI's sealed-checkout branch, the Store writer guard, `--no-persist`, and the refusal of `init`, `snapshot` and `request`.


### Results summary
The full `pose-mcp` suite passes; the canonical result is refreshed with `pose validate --carry`.


### Requirement trace
- R1 [satisfied] test:TestAssessNoPersistWritesNothing
- R2 [satisfied] test:TestAssessLeavesTheSealedCheckoutUnchanged
- R3 [satisfied] test:TestReviewPlanRunsAssessmentsWithoutPersisting
- R4 [satisfied] test:TestAssessLeavesTheSealedCheckoutUnchanged


### Known gaps
None.


## 7. Final Report

### Delivered scope
`pose assess discover|integrate|tech-debt --no-persist`; no persistence and refusal of the writing subcommands in a sealed review checkout; the Store's assessment writers refuse there; the review plan runs assessments without persisting.


### Files and modules changed
`pose-mcp` (CLI and engine), the manual in both locales and its embedded copies.


### Validation executed
`go test ./...` in `pose-mcp`; `pose check --strict`; `pose validate --carry --json-out .pose/results/delivery-validation.json`.


### Residual risks
L1: the guarantee rests on `POSE_SEALED_CHECKOUT`, which the dispatch sets; a reviewer process that clears it can still write, and the dispatch's comparison of the checkout after the run still fails that run.


### Follow-ups
None.


