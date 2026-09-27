---
slug: pose-abm-review-soundness-residuals
status: draft
created_at: 2026-09-26
supersedes:
depends_on: pose-abm-review-soundness
priority: 1
components: pose-mcp
task_type: bugfix
delivers:
---

# Spec: Review soundness obligations the executor did not deliver

## 1. Intent

### Goal
Carry the three requirements of Harne8's review-soundness coordinator that the
executor `pose-abm-review-soundness` neither delivered nor withdrew, so they
survive the coordinator's retirement.

### Business value
The reconciliation map of Harne8 (`.pose/reports/abm-authority-reconciliation.md`
in proj.harne8) disposes them as pending. One is a live gap: a `critical`
finding disposed as `wont-fix` passes attestation validation without owner,
rationale or review date, so it bypasses the accepted-risk gate
(`validateBundleAttestationWith`).

### Constraints
Owner: @pose-maintainers. Do not reopen `pose-abm-review-soundness`.
Planning only; no requirement is satisfied by this document.

### Non-goals
Do not change accepted-risk severities or the finding schema beyond what the
requirements need.

## 2. Requirements

- R1: A finding disposed as `wont-fix` whose severity is not an accepted-risk severity of the sealed gates is refused, or carries owner, rationale and review date like accepted risk; the rule applies to the Store, CLI, signed import and reuse alike.
- R2: A structurally valid attestation that does not approve is persisted for audit and never counted as approval; a structurally invalid one is refused with a stable diagnostic. A named test proves both.
- R3: Distributed profiles, skills, manual, schemas and scaffold never instruct an auto-pass the runtime refuses; a check fails when they do.

## 3. Technical Plan

### Affected areas
`pose-mcp/internal/pose/review_bundle.go` attestation validation, the
distributed skills and manual. To be detailed before activation.

### Artifacts
- created: .pose/specs/2026-09-26-pose-abm-review-soundness-residuals.md

### Delivery targets
Keep `delivers` empty while this is a draft.

## 4. Tasks

- [ ] Detail the plan and register targets before activation.

## 5. Decisions

Created as the open destination of the pending requirements when the
coordinator is retired with `spec-transfer --mode reconcile-terminal`.

## 6. Validation

To be defined before activation.

### Requirement trace
- R1 [deferred-integration: planning only] spec:pose-abm-review-soundness
- R2 [deferred-integration: planning only] spec:pose-abm-review-soundness
- R3 [deferred-integration: planning only] spec:pose-abm-review-soundness

## 7. Final Report

### Delivered scope
Planning artifact only.

### Residual risks
The `wont-fix` gap stays open until R1 is delivered.

### Follow-ups
