---
slug: pose-action-answers-reverify-on-read
status: in-progress
created_at: 2026-10-13
completed_at:
depends_on:
priority: 0
components: pose-mcp
task_type: bugfix
surface: minimal
changelog:
delivers: capability:action-answer-read-verification
---

# Spec: Reverify persisted action answers before consuming their assurance

## 1. Intent

### Goal

Prevent a persisted verified label or an altered signed answer from satisfying an action request. Share verification through the domain reader consumed by CLI, MCP, obligations and gates.

### Business value

Backlog B01 (T2-01/T2-03), with the action-claim temporal prerequisite from B04. Decision context: knowledge:action-answer-proof-and-historical-time. The front-2 probes reproduced a verified label without proof and an issuer claim signing another answer. Existing SSH verification only ran in the textual renderer.

### Constraints

Preserve declared legacy answers and SSH verification after key rotation, as required by pose-signed-action-answers R7. Do not reauthorize an old role against today's membership, auto-upgrade declared history or change adoption policy. This delta does not complete review-attestation disclosure, waiver authority or the entire B04 re-use contract.

## 2. Requirements

- R1: LoadActionRequest shall verify every answered event claiming verified assurance or carrying SSH/issuer proof before deriving a view; absent, malformed or mismatched proof shall return action-claim-not-verified and never satisfy a governed condition.
- R2: Verification shall bind the recorded request digest, actor, answer and reason; SSH proof shall additionally bind its existing namespace/idempotency statement and its recorded key/fingerprint/presence metadata.
- R3: Issuer validity shall use the operation's supplied time on write and the recorded answer time on read. Future issuance, reversed expiry and expiry before acceptance shall fail; expiry after a valid historical acceptance shall not erase that acceptance. Current issuer pins/grants remain the existing explicit trust boundary.
- R4: Declared answers without proof shall remain declared; verified assurance shall be derived from valid proof, not a journal label. SSH history shall remain verifiable after registered-key rotation without interpreting recorded presence flags as proof.
- R5: The shared reader shall make tampering visible as incomplete coverage to obligations and prevent satisfaction in adopted gates; CLI JSON and MCP shall propagate the same verification failure.

## 3. Technical Plan

### Artifacts

- created: .pose/specs/2026-10-13-pose-action-answers-reverify-on-read.md
- created: .pose/starts/pose-action-answers-reverify-on-read.json
- created: .pose/changelogs/unreleased/pose-action-answers-reverify-on-read.md
- created: .pose/knowledge/2026-10-13-decision-log-action-answer-proof-and-historical-time.md
- created: .pose/reports/2026-10-13-standard-validate-native.md
- modified: .pose/reports/history/standard-validate-native.jsonl
- modified: pose-mcp/internal/pose/action_request.go
- modified: pose-mcp/internal/pose/action_resolution.go
- modified: pose-mcp/internal/pose/principal_keys.go
- created: pose-mcp/internal/pose/action_answer_read_test.go
- created: pose-mcp/internal/cli/action_read_proof_test.go
- created: pose-mcp/internal/mcpserver/action_read_proof_test.go
- created: .pose/reports/2026-10-15-pose-b01-action-qualification.md
- modified: .pose/results/pose-implementation-plan.json
- modified: .pose/roadmaps/pose-consolidation-program.md
- modified: .pose/roadmaps/record-truthfulness.md
- modified: .pose/assessments/README.md
- modified: .pose/assessments/consolidated.md
- modified: .pose/assessments/integrations.md
- modified: .pose/assessments/technical-debt.md

### Delivery targets

- capability:action-answer-read-verification module:pose-mcp profile:composed-capability entrypoint:pose-mcp/internal/pose/action_request.go

## 4. Tasks

- [x] Reconcile B04 terminal closeout and preserve the existing start baseline.
- [x] Exercise persisted-proof failure through CLI JSON and MCP transport.
- [x] Run domain/surface regression pairs and targeted race detection.
- [x] Generate current global native validation, seal review, and obtain different-actor judgment.
- [ ] Apply native closeout only after review verification and acceptance.

## 5. Decisions

### Decision D1
- Basis: R3
- Decision: Reuse the existing claim verifier with an explicit evaluation time; distinguish historical acceptance from new-operation admissibility. Keep live issuer pins/grants, and preserve the existing journal-contained SSH contract after rotation.
- Rationale: Rechecking historical expiry against wall clock would invalidate an answer that was accepted legitimately; trusting its label would miss byte tampering. This bugfix does not introduce a historical authorization receipt or universal trust policy.

## 6. Validation

### Strategy

Regression pairs for proofless verified labels, issuer/SSH tampering, historical expiry, future issuance, reversed expiry, declared history, key rotation, mixed proofs and obligations coverage. Run domain tests, broader module tests and race tests where dependencies are available; retain explicit limits otherwise.

### Requirement trace

Current qualification after B04/#178 (prior execution history retained below):

- R1 [satisfied] test:TestActionReadRefusesProoflessVerifiedAndDegradesConsumers
- R2 [satisfied] test:TestActionReadReverifiesSSHBindingsAndMetadata and test:TestActionReadReverifiesIssuerBindings
- R3 [satisfied] test:TestActionReadKeepsHistoricalExpiryAndSecondPrecision and test:TestActionClaimRefusesFutureOrReversedExpiryAtWrite
- R4 [satisfied] test:TestActionReadDerivesProofAndPreservesDeclaredAndRotatedSSH
- R5 [satisfied] test:TestActionReadCLIRejectsPersistedProoflessVerifiedAnswer, test:TestActionReadMCPRejectsPersistedProoflessVerifiedAnswer and test:TestActionReadRefusesProoflessVerifiedAndDegradesConsumers

All above tests passed on the reconciled main plus these surface tests, with a
second run under -race (pose 1.165s, CLI 1.076s, MCP 1.050s). Global native validation passed 132/132 on the published qualification subject;
current results are in .pose/results/delivery-validation.json. Independent review was approved by agent:independent-codex-pr179 and the
maintainer explicitly accepted its conclusions on 2026-10-15. The acceptance
journal is answered/satisfied; current evidence/review and final closeout state
are recorded by the native b01-action-* results. Prior failed native attempts
remain historical evidence.

The complete Go suite passed on implementation commit 30c2776a5de2f2e9b8d923974c08938242a66be8 in [CI run 38331161368](https://github.com/oseiaspereira88/pose/actions/runs/38331161368). Executed mapping: R1/R5 use TestActionReadRefusesProoflessVerifiedAndDegradesConsumers; R2 uses TestActionReadReverifiesSSHBindingsAndMetadata and TestActionReadReverifiesIssuerBindings; R3 uses TestActionReadKeepsHistoricalExpiryAndSecondPrecision and TestActionClaimRefusesFutureOrReversedExpiryAtWrite; R4 uses TestActionReadDerivesProofAndPreservesDeclaredAndRotatedSSH. The same test job passed the installer, migration, quickstart, install/upgrade, launch-demo, delivery-image and action-runtime checks.

Local gofmt, lint-spec, artifact-check --strict and check --strict passed; the latter retained 30 existing warnings. These structural checks used the published 7.4.1 binary. The available Go archive is truncated: compile -V segfaults, so no local Go test success or race-detector execution is claimed. The module validate attempt is retained as FAILURE (Go unavailable in PATH; 128 required checks failed). Functional evidence comes from the CI Go suite. Remaining governance and security jobs must finish before review readiness.

## 7. Final Report

### Delivered scope

Implementation and regression tests delivered in #157. Current qualification after
the merge of B04/#178 adds CLI JSON/text/list and MCP request/list/presentation
regressions, including incomplete Attention/obligations coverage. Consult
[B01 action qualification](../reports/2026-10-15-pose-b01-action-qualification.md)
for current execution evidence and review/closeout state. Preserve R1–R5, D1 and
the existing start baseline; no second start or retrospective proof is created.

### Residual risks

A journal-contained SSH signature proves its signed bytes, not an immutable historical authorization receipt. Issuer pin revocation remains live. General historical trust/re-use, waiver authority and review-attestation disclosure remain separate backlog deltas.

### Follow-ups

- [done] B04's scoped historical trust and review-source reuse contract was closed natively and published by #178; it explicitly does not establish immutable historical authorization. Consume its spec/closeout evidence rather than infer delivery from the merge.
