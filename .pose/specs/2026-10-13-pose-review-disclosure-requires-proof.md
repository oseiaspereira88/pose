---
slug: pose-review-disclosure-requires-proof
status: in-progress
created_at: 2026-10-13
completed_at:
depends_on:
priority: 0
components: pose-mcp
task_type: bugfix
surface: minimal
delivers: capability:review-assurance-proof-verification
---

# Spec: Require verified review proof before disclosing verified assurance

## 1. Intent

### Goal

Make identity, separation and confirmation disclosure depend on the same cryptographic proof and claim bindings as review verification.

### Business value

Complete the review-disclosure portion of backlog B01/T2-03 after the action-answer correction in PR #157. The shared authority checker validates claim fields but not the signed envelope. A coherent claim with a damaged signature therefore produces a rejected verdict and a verified disclosure. Reuse knowledge:action-answer-proof-and-historical-time and knowledge:review-assurance-requires-cryptographic-proof.

### Constraints

Preserve declared legacy records, existing current issuer pins/grants and review expiry semantics. Keep identity assurance independent of criteria/decision validity. This delta does not change adoption policy, historical authorization, reuse provenance, approval-write preflight (B02) or cognitive-independence claims.

## 2. Requirements

- R1: A malformed, missing, invalid or untrusted signed authority proof shall never disclose verified reviewer identity, verified separation or a verified claim.
- R2: A verified disclosure shall bind the attestation to the supplied bundle and preserve all existing authority principal, audience, role, issuer and execution checks.
- R3: A trusted signature and valid authority claim may disclose verified identity while semantic criteria or the review decision prevent approval; declared legacy identity shall remain declared.
- R4: Review verify, review-check, closeout and attribution shall consume the shared proof result; invalid proof shall never produce verified human confirmation, including in JSON and human rendering.

## 3. Technical Plan

### Artifacts

- created: .pose/specs/2026-10-13-pose-review-disclosure-requires-proof.md
- created: .pose/starts/pose-review-disclosure-requires-proof.json
- created: .pose/changelogs/unreleased/pose-review-disclosure-requires-proof.md
- created: .pose/knowledge/2026-10-13-decision-log-review-assurance-requires-cryptographic-proof.md
- created: .pose/reports/2026-10-13-review-proof-validate-native.md
- modified: .pose/reports/history/standard-validate-native.jsonl
- modified: pose-mcp/internal/pose/review_bundle.go
- created: pose-mcp/internal/pose/review_assurance_proof_test.go

### Delivery targets

- capability:review-assurance-proof-verification module:pose-mcp profile:composed-capability entrypoint:pose-mcp/internal/pose/review_bundle.go

## 5. Decisions

### Decision D1
- Basis: R1, R3, R4
- Decision: Reuse the stored-envelope verifier inside the shared authority checker; keep semantic approval checks outside that checker.
- Rationale: Every assurance consumer must require cryptographic authentication, but a signature does not prove that a criterion passed. Reverting this scoped change is the rollback; no stored records are rewritten.

## 6. Validation

### Strategy

Test damaged signatures, unsigned content tampering, missing proof, revoked pins, coherent claims with other bundle bindings, invalid human confirmation and valid identities with invalid criteria. Compare direct disclosure, verdict, review-check and closeout, plus JSON and human rendering. Run the complete Go suite and CI/Security on the published head.

### Requirement trace

Regression mapping: R1 uses TestReviewAssuranceRejectsInvalidCryptographicProof; R2 adds TestReviewAssuranceBindsTheSuppliedBundle; R3 uses TestReviewAssuranceKeepsIdentitySeparateFromApproval; R4 uses TestReviewConsumersAgreeOnTamperedProof and TestReviewInvalidProofCannotVerifyHumanConfirmation. Execution evidence and exact tested commits are recorded in [PR #158](https://github.com/oseiaspereira88/pose/pull/158). Keep the spec in-progress until independent review and closeout.

### Known gaps

Local gofmt, lint-spec and artifact-check --strict passed. The local Go toolchain is truncated; the native module-validation attempt ended FAILURE with 128 required checks failing because Go is absent from PATH. Retain its report and history, and use CI for functional evidence. Do not claim local Go or race-detector success. Local structural validation uses the published 7.4.1 binary; CI builds the implementation.

## 7. Final Report

### Delivered scope

Implementation and regression tests delivered. Consult the PR checks for current CI execution evidence. Independent review and spec closure remain pending.

### Residual risks

Existing review claims still use current pins/grants and current expiry evaluation. Historical trust and reuse-source verification remain in B04. These existing semantics are not broadened here.

### Follow-ups

- [open] Deliver the B04 review-claim historical validity and reuse-source provenance contract in a separate scoped delta. (owner:@pose-maintainers crit:medium review:2026-11-13)
