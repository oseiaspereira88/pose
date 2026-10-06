---
slug: pose-validation-check-additions-are-not-material
status: done
created_at: 2026-10-05
completed_at: 2026-10-06
supersedes:          # slug of the superseded spec (when applicable)
depends_on: 
remediates: spec:pose-abm-structural-delta@defect-fix
priority: 1
components: pose-mcp
task_type: bugfix
surface: minimal
delivers: capability:validation-check-materiality
---

# Spec: Adding a validation check is not a material structural fact

## 1. Intent

### Goal

Stop reading `.pose/indexes/validation-matrix.json` as one opaque file: a change that only adds new checks is reported as non-material `validation-check` facts, while any other change to the matrix stays a material `delivery-metadata` fact.

### Business value

The second measurement for action request act-08a9fd2d9a0e50bb (causality closeout) found that 30 of 34 recent specs would select `structural-materiality@1`, and in 21 of them the only material fact was the validation matrix, because registering a spec's own check rewrites the file. A mapping owed for every registered check is a ceremony every instance pays, and it trains reviewers to paste "the matrix registers the check that proves Rn". The maintainer chose to fix this in the engine before adopting the contract.

### Constraints

Materiality stays one-directional: a reading that cannot prove the change is additive falls back to the material fact. An added check that shares a name with any check already in the matrix is not additive, because it collides with the evidence identity of the existing one. Nothing recorded is rewritten.

### Non-goals

Judging whether an added check is adequate to its requirement; that remains the requirement trace and the review criteria.

## 2. Requirements

### Functional

- R1: When the only difference in the validation matrix is one or more checks appended under existing `checks` arrays, each with a name no check in the previous matrix carries, the structural delta shall report one non-material `validation-check` fact per added check and no `delivery-metadata` fact for the matrix.
- R2: Any other change to the matrix shall stay a material `delivery-metadata` fact: a removed or edited check, a changed mode, stack, override or profile, an added check whose name already exists, an added module, a created or deleted matrix, or a side that does not parse.
- R3: The delta parser version shall change so a report produced under the old reading is distinguishable from one produced under the new one.
- R4: The manual shall state that registering a new check is reported and not charged.

### Non-functional

- No extra Git reads: both sides are already read for the metadata fact.

### Security

- An addition that reuses an existing check name stays material, so it cannot shadow existing evidence unreviewed.

### Compatibility

- Bundles keep their sealed structure; only new plans read the refined facts.

## 3. Technical Plan

### Affected areas

Structural delta detector, manual.

### Artifacts

- created: .pose/specs/2026-10-05-pose-validation-check-additions-are-not-material.md
- modified: pose-mcp/internal/pose/design_delta.go
- created: pose-mcp/internal/pose/design_delta_matrix_test.go
- modified: POSE.md
- modified: locales/pt-BR/POSE.md
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md
- modified: .pose/indexes/validation-matrix.json
- created: .pose/changelogs/unreleased/pose-validation-check-additions-are-not-material.md
- created: .pose/starts/pose-validation-check-additions-are-not-material.json

### Delivery targets

- capability:validation-check-materiality module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go

### Technical risks

- A new check with a trivial command can still be registered without a structural mapping; its adequacy is what the requirement trace and the correctness criterion judge, as for any check that already exists.

## 6. Validation

### Strategy

Git fixtures with a base and head matrix for each case of R1 and R2, asserting the kinds, actions and materiality of the reported deltas.

### Deterministic checks

#### Test
- Command: `cd pose-mcp && go test ./internal/pose -run ValidationCheckMateriality`
- Expected: pass

### Requirement trace

- R1 [satisfied] test:TestValidationCheckMaterialityReportsNewChecksWithoutCharging check:validation-check-materiality-integration
- R2 [satisfied] test:TestValidationCheckMaterialityKeepsEveryOtherChangeMaterial check:validation-check-materiality-integration
- R3 [satisfied] test:TestValidationCheckMaterialityChangesTheParserVersion check:validation-check-materiality-integration
- R4 [satisfied] <POSE.md and its pt-BR and distributed copies, "Observed structure and causal mapping"> test:TestValidationCheckMaterialityIsDocumented check:validation-check-materiality-integration

### Known gaps

None.

## 7. Final Report

### Delivered scope

The structural detector reports uniquely named additive validation checks as non-material validation-check facts; all other matrix changes remain material. Parser v2 and all four manual copies agree. Reviewed against TestValidationCheckMaterialityReportsNewChecksWithoutCharging and TestValidationCheckMaterialityKeepsEveryOtherChangeMaterial on 2026-10-06.

### Residual risks

The adequacy of an added check is judged by the trace, not by the structural contract.

### Follow-ups

None.
