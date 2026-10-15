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

- created: pose-mcp/internal/cli/review_assurance_proof_cli_test.go
- created: .pose/reports/2026-10-15-pose-b01-disclosure-qualification.md
- modified: .pose/results/pose-implementation-plan.json
- modified: .pose/roadmaps/pose-consolidation-program.md
- modified: .pose/roadmaps/record-truthfulness.md
- modified: .pose/assessments/README.md
- modified: .pose/assessments/consolidated.md
- modified: .pose/assessments/integrations.md
- modified: .pose/assessments/technical-debt.md

### Delivery targets

- capability:review-assurance-proof-verification module:pose-mcp profile:composed-capability entrypoint:pose-mcp/internal/pose/review_bundle.go

## 4. Tasks

- [x] Reconcile merged B04, action-answer #179 and publication complement #181; preserve the original start and R1–R4/D1 baseline.
- [x] Exercise signed positive, damaged/missing proof and valid identity with invalid criteria through CLI JSON/text and MCP.
- [ ] Run current native strict validation and targeted race checks.
- [ ] Obtain the required different-actor review and verify the sealed subject.
- [ ] Apply native closeout and confirm published evidence before releasing B02.

## 5. Decisions

### Decision D1
- Basis: R1, R3, R4
- Decision: Reuse the stored-envelope verifier inside the shared authority checker; keep semantic approval checks outside that checker.
- Rationale: Every assurance consumer must require cryptographic authentication, but a signature does not prove that a criterion passed. Reverting this scoped change is the rollback; no stored records are rewritten.

## 6. Validation

### Strategy

Test damaged signatures, unsigned content tampering, missing proof, revoked pins, coherent claims with other bundle bindings, invalid human confirmation and valid identities with invalid criteria. Compare direct disclosure, verdict, review-check and closeout, plus JSON and human rendering. Run the complete Go suite and CI/Security on the published head.

### Requirement trace

Current qualification (2026-10-15), following B04 and #179/#181:

- R1 [satisfied] test:TestReviewAssuranceRejectsInvalidCryptographicProof and test:TestReviewAssuranceProofCLIAndMCPConsumers
- R2 [satisfied] test:TestReviewAssuranceBindsTheSuppliedBundle and test:TestABMReviewAuthority
- R3 [satisfied] test:TestReviewAssuranceKeepsIdentitySeparateFromApproval, test:TestReviewAssuranceProofCLIAndMCPConsumers and test:TestReviewSurfacesDiscloseDeclaredAssuranceOfAHumanLabel
- R4 [satisfied] test:TestReviewConsumersAgreeOnTamperedProof, test:TestReviewAssuranceInvalidProofCannotVerifyHumanConfirmation and test:TestReviewAssuranceProofCLIAndMCPConsumers

Domain and CLI/MCP qualification passed. Full validation, race, review and closeout are recorded as they run in [the qualification report](../reports/2026-10-15-pose-b01-disclosure-qualification.md). B01 remains partial until native terminal closeout. The renamed human-confirmation test now matches the existing ReviewAssurance integration selector; no matrix check is weakened.

Historical implementation evidence (2026-10-13, preserved):

Regression mapping: R1 uses TestReviewAssuranceRejectsInvalidCryptographicProof; R2 adds TestReviewAssuranceBindsTheSuppliedBundle; R3 uses TestReviewAssuranceKeepsIdentitySeparateFromApproval; R4 uses TestReviewConsumersAgreeOnTamperedProof and TestReviewAssuranceInvalidProofCannotVerifyHumanConfirmation. Execution evidence and exact tested commits are recorded in [PR #158](https://github.com/oseiaspereira88/pose/pull/158). Keep the spec in-progress until independent review and closeout.

### Known gaps

Historical implementation environment (2026-10-13): local gofmt, lint-spec and artifact-check --strict passed. The local Go toolchain is truncated; the native module-validation attempt ended FAILURE with 128 required checks failing because Go is absent from PATH. Retain its report and history, and use CI for functional evidence. Do not claim local Go or race-detector success. Local structural validation uses the published 7.4.1 binary; CI builds the implementation.

## 7. Final Report

### Delivered scope

Implementation and regression tests delivered. Consult the PR checks for current CI execution evidence. Independent review and spec closure remain pending.

### Residual risks

Existing review claims still use current pins/grants and current expiry evaluation. Historical trust and reuse-source verification remain in B04. These existing semantics are not broadened here.

### Follow-ups

- [done] B04 historical claim-time and reuse-source provenance were closed natively and published by #178. Current trust remains governed by live pins/grants; no historical authorization receipt is implied.
