---
slug: pose-evidence-gate-means-validation
status: in-progress
created_at: 2026-11-01
supersedes:
depends_on: pose-closeout-names-the-missing-trailer
remediates:
priority: 1
components: pose-mcp
task_type: bugfix
delivers: capability:evidence-gate-means-validation
completed_at:
---

# Spec: The evidence gate means validation

## 1. Intent

### Goal

A scope that requires validation evidence seals only with a validation result, and evidence the engine derives from the subject does not stand in for one.

### Business value

The gate asks whether the evidence of a bundle is empty. Design evidence, which `reviewBundleEvidence` derives from the subject itself, fills the list, so the gate never fires for a spec with a change set. 23 bundles of this repository sealed that way: 19 were never attested, a seal spent on a bundle that could not be approved, and 4 were approved, one of them the final bundle of `pose-dist-adopt-published-v6`.

### Constraints

- Closed scopes keep their approvals: verification prepares a done spec again, so a done spec answers to the gate as it was sealed.
- A scope that requires no validation evidence seals as it did.

### Non-goals

- Deciding what a validation result must cover: that stays with plan, criteria and tools.
- Reviewing the five approved bundles again.

## 2. Requirements

### Functional

- R1: A bundle of a scope that requires validation evidence and is not a done spec shall report the evidence blocker when its only evidence is derived from the subject, and a result whose report has any path shall count as validation.
- R2: `pose review auto-attest` shall refuse the same bundle with the same reason.
- R3: A closed scope sealed with derived evidence alone shall keep its approval in `review verify`, `review-check`, closeout and the federated proof of a consumer.

### Non-functional

- No new read: the decision uses the evidence already gathered.

### Security

- The gate tightens: a seal that passed may now refuse, and nothing that refused now passes.

### Compatibility

- A spec in progress whose validation does not attribute to it now stops at the seal, where it stopped at the attestation.

## 3. Technical Plan

### Affected areas

`internal/pose/review_bundle.go`.

### Artifacts

- created: .pose/specs/2026-11-01-pose-evidence-gate-means-validation.md
- created: .pose/starts/pose-evidence-gate-means-validation.json
- created: .pose/changelogs/unreleased/pose-evidence-gate-means-validation.md
- modified: .pose/indexes/validation-matrix.json
- modified: pose-mcp/internal/pose/review_bundle.go
- created: pose-mcp/internal/pose/evidence_gate_test.go
- modified: pose-mcp/internal/pose/missing_change_set_test.go
- modified: pose-mcp/internal/cli/amendment_history_close_test.go
- modified: pose-mcp/internal/pose/delegated_review_adoption_test.go
- modified: pose-mcp/internal/pose/review_path_attribution_test.go
- modified: pose-mcp/internal/cli/closeout_names_missing_trailer_test.go
- modified: .pose/specs/2026-11-01-pose-closeout-names-the-missing-trailer.md
- modified: .pose/specs/2026-11-01-pose-milestone-closes-when-reached.md
- modified: .pose/specs/2026-10-10-pose-amendment-history-holds-before-close.md
- modified: .pose/specs/2026-10-30-pose-delegation-result-before-marker.md
- modified: .pose/specs/2026-10-30-pose-cli-test-wall-time.md

### Delivery targets

- capability:evidence-gate-means-validation module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go

### API/contract changes

`review bundle` and `review auto-attest` refuse a bundle whose evidence is only derived, when the scope requires validation evidence.

## 4. Tasks

- [ ] Count validation results, not evidence
- [ ] Measure closed scopes with the old and the new reader
- [ ] Give the stale follow-ups their disposition
- [ ] Independent review and closeout

## 5. Decisions

### Decision D1
- Basis: R1, R2
- Minimal option: drop design evidence from the bundle
- Selected option: keep it, and have the gate count only evidence that is not derived, and exempt a done spec, which verification prepares again
- Rationale: design evidence answers structural criteria, and only the gate misread it
- Consequences: the gate reads check, class, an empty module and the `derived:design-delta/` report together, because a report path alone is the operator's; a done spec can be sealed again over derived evidence, which only `pose review auto-attest` refuses
- Falsifier: a result of a run with no module, in class structure, under check `assess-design`

## 6. Validation

### Strategy

An integration test over a change set with no attributed result, seen failing first, and a measure of the closed scopes sealed with derived evidence alone.

### Requirement trace

- R1 [planned] capability:evidence-gate-means-validation evidence:integration check:evidence-gate-validation-integration test:TestDerivedEvidenceIsNotValidation test:TestAResultAtADerivedPathIsValidation
- R2 [planned] check:evidence-gate-validation-integration test:TestAutoAttestRefusesDerivedEvidence
- R3 [planned] check:evidence-gate-validation-integration test:TestClosedScopeKeepsItsDerivedSeal

## 7. Final Report

### Delivered scope

### Residual risks

### Follow-ups
