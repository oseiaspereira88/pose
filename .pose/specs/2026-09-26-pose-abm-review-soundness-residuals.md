---
slug: pose-abm-review-soundness-residuals
status: in-progress
created_at: 2026-09-26
supersedes:
depends_on: pose-abm-review-soundness
priority: 1
components: pose-mcp
task_type: bugfix
delivers: governance:review-soundness-residuals
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
Implement the minimum shared gate and exercise the distributed review contract.

### Non-goals
Do not change accepted-risk severities or the finding schema beyond what the
requirements need.

## 2. Requirements

- R1: A finding disposed as `wont-fix` whose severity is not an accepted-risk severity of the sealed gates is refused, or carries owner, rationale and review date like accepted risk; the rule applies to the Store, CLI, signed import and reuse alike.
- R2: A structurally valid attestation that does not approve is persisted for audit and never counted as approval; a structurally invalid one is refused with a stable diagnostic. A named test proves both.
- R3: Distributed profiles, skills, manual, schemas and scaffold never instruct an auto-pass the runtime refuses; a check fails when they do.

## 3. Technical Plan

### Affected areas
Treat unresolved `wont-fix` findings as accepted risk, using the severities
frozen into the bundle and requiring owner, rationale and review date. Reuse
the shared verifier for Store, CLI, signed import and criterion reuse. Keep
non-approving decisions as immutable audit records; malformed envelope identity
and decision values remain write errors. Reconcile English and Portuguese
review instructions and embedded scaffold. Use
knowledge:module-metadata-discovery-invalidates-review-provenance.

### Artifacts
- created: .pose/specs/2026-09-26-pose-abm-review-soundness-residuals.md
- modified: pose-mcp/internal/pose/review_bundle.go
- modified: pose-mcp/internal/pose/bundle_finding_contract_test.go
- modified: pose-mcp/internal/pose/abm_review_soundness_test.go
- modified: pose-mcp/internal/cli/review_closeout_test.go
- created: pose-mcp/internal/scaffold/review_soundness_test.go
- modified: .agents/skills/pose-review/SKILL.md
- modified: POSE.md
- modified: locales/pt-BR/.agents/skills/pose-review/SKILL.md
- modified: locales/pt-BR/POSE.md
- modified: pose-mcp/internal/scaffold/dist/.agents/skills/pose-review/SKILL.md
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/.agents/skills/pose-review/SKILL.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md
- modified: .pose/indexes/validation-matrix.json
- created: .pose/changelogs/unreleased/pose-abm-review-soundness-residuals.md

### Delivery targets
- governance:review-soundness-residuals module:pose-mcp profile:release-governance entrypoint:pose-mcp/cmd/pose/main.go

## 4. Tasks

- [x] Detail the plan and register targets before activation.
- [ ] Reproduce the critical wont-fix acceptance and apply the shared gate.
- [ ] Prove rejection audit persistence and malformed decision refusal.
- [ ] Reconcile distributed instructions and add the contract check.
- [ ] Validate, review and close.

## 5. Decisions

Created as the open destination of the pending requirements when the
coordinator is retired with `spec-transfer --mode reconcile-terminal`.

## 6. Validation

Before the fix, `TestBundlePathWontFixUsesSealedAcceptedRiskGate` must fail on
critical and incomplete risks. After the fix it must accept only complete low
risks under the sealed policy. `TestABMReviewSoundnessNegativeDecisionIsAudit`
exercises persistence and non-approval; an invalid decision creates no record.
`TestDistributedReviewSoundnessContract` verifies the installed profiles and
instructions against runtime criterion preparation. Run the registered
`review-soundness-residuals-integration` check and the full pose-mcp matrix.

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
