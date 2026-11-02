---
slug: pose-evidence-gate-means-validation
status: done
created_at: 2026-11-01
supersedes:
depends_on: pose-closeout-names-the-missing-trailer
remediates:
priority: 1
components: pose-mcp
task_type: bugfix
delivers: capability:evidence-gate-means-validation
completed_at: 2026-11-02
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
- Reviewing the four approved bundles again.

## 2. Requirements

### Functional

- R1: A new seal of a scope that requires validation evidence shall be refused when its only evidence is derived from the subject, whatever the lifecycle of the scope, and a result shall count as validation whatever check, class, module and report path it carries.
- R2: `pose review auto-attest` shall refuse the same bundle with the same reason.
- R3: A closed scope sealed with derived evidence alone shall keep its approval in `review verify`, `review-check`, closeout and the federated proof of a consumer.

### Non-functional

- One more read of the spec when a bundle holds derived evidence alone, and of the index when it is about to be sealed.

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

- [x] Count validation results, not evidence
- [x] Measure closed scopes with the old and the new reader
- [x] Give the stale follow-ups their disposition
- [x] Independent review and closeout

## 5. Decisions

### Decision D1
- Basis: R1, R2
- Minimal option: drop design evidence from the bundle
- Selected option: keep it, have the gate count only evidence that is not derived, refuse every new seal without a result, and let verification of a done spec read the gate as absence alone
- Rationale: design evidence answers structural criteria, and only the gate misread it
- Consequences: derived evidence is recognized by an id and a report computed from its own provenance digest, since every other field is the operator's to give a result; the preparation of a done spec reports less than its seal refuses
- Falsifier: a result of a run read as derived without an id and a report built to match its digest

## 6. Validation

### Strategy

An integration test over a change set with no attributed result, seen failing first, and a measure of the closed scopes sealed with derived evidence alone.

### Requirement trace

- R1 [satisfied] capability:evidence-gate-means-validation evidence:integration check:evidence-gate-validation-integration test:TestDerivedEvidenceIsNotValidation test:TestAResultAtADerivedPathIsValidation test:TestNoNewSealOverDerivedEvidence test:TestDesignEvidenceIsRecognizedByItsOwnDigest report:.pose/results/delivery-validation.json commit:e2aaf40b192372a0ef5c03cd2b4881e652a82484
- R2 [satisfied] check:evidence-gate-validation-integration test:TestAutoAttestRefusesDerivedEvidence report:.pose/results/delivery-validation.json commit:e2aaf40b192372a0ef5c03cd2b4881e652a82484
- R3 [satisfied] check:evidence-gate-validation-integration test:TestClosedScopeKeepsItsDerivedSeal report:.pose/results/delivery-validation.json commit:e2aaf40b192372a0ef5c03cd2b4881e652a82484

## 7. Final Report

### Delivered scope

A new seal of a scope that requires validation evidence holds a validation result, whatever the lifecycle of the scope, and `pose review auto-attest` refuses a bundle without one. Derived evidence is recognized by an id and a report computed from its own provenance digest, which all 882 such entries of this repository carry. Closed scopes keep their approvals in verify, review-check, closeout and federated proof. Stale follow-ups received their disposition, and the reversed R3 of `pose-amendment-history-holds-before-close` reads as historical, with its test named for what it measures. Independently reviewed by Codex in three rounds: the first found that verification lost closed derived seals and that a report path could make a real result read as derived, the second that every field but the digest was the operator's and that the lifecycle exception let a seal depend on a status read.

### Residual risks

The gate establishes that a validation result exists, not what it covers: plan, criteria and tools answer that. The four approved bundles sealed over derived evidence alone stand, the last of `pose-dist-adopt-published-v6`. The preparation of a done spec reports less than its seal refuses.

### Follow-ups

None.
