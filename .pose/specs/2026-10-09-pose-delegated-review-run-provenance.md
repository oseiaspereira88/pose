---
slug: pose-delegated-review-run-provenance
status: in-progress
created_at: 2026-10-09
completed_at:
supersedes:
depends_on: pose-delegated-review-dispatch, pose-native-attestation-issuer, pose-action-answers-reverify-on-read, pose-review-disclosure-requires-proof, pose-historical-trust-and-review-reuse, pose-delegated-review-attempt-ledger
priority: 2
components: pose-mcp
task_type: feature
changelog:
delivers: capability:delegated-run-provenance
---

# Spec: Independence read from a signed run record

## 1. Intent

### Goal

Satisfy `different-actor` for a delegated review from evidence — the run record — instead of the `agent:independent-` reviewer prefix.

### Business value

Under declared assurance the engine accepts `different-actor` when the reviewer is named `agent:independent-…`; on 2026-10-09 the reviewing agent had to adopt that literal prefix to be accepted. Anyone can type it, and the engine's own comment calls it a declaration.

Origin: the 2026-10-09 review of `pose-native-attestation-issuer` by Codex through a personal launcher; it found three real defects in three rounds and exposed what the engine lacks (ADR `2026-10-09-delegated-review-is-an-adapter`, roadmap `delegated-review`).

### Constraints

Honest labelling: a machine-attested run is disclosed as such, never as a person. The contract and delegated adapter ADR are accepted. Implementation authorized by continuation on 2026-10-25, baseline `3e4c26d`. Preserve existing R1–R4 and add the prospective P1–P6 deltas as R5–R10 before start; do not adopt the capability, authenticate a person or apply a draft automatically here.

## 2. Requirements

### Functional

- R1: A dispatch run record shall be signable by a pinned native issuer, binding brief digest, transcript digest, adapter, vendor, model and bundle digest.
- R2: An attestation produced from a run shall reference the run; the verifier shall read the reviewer's vendor and model from the referenced run record.
- R3: `different-actor` shall be satisfied by a run whose vendor or model differs from the implementation principal's declared vendor and model, per the contract spec's answer; the `agent:independent-` prefix shall no longer satisfy it in instances that adopt `delegated-review`.
- R4: Review assurance shall disclose a run-backed review as `machine-attested`, distinct from `declared` and from a person's signed confirmation.

- R5: A signed run proof shall bind project/worktree/scope/kind/bundle/commit, policy and adapter configuration, executor and implementation identities, argv/template/notes/input digests, timestamps and outcome; conflicting or unauthorized context shall refuse.
- R6: A verifier shall require explicit executor-attested trust and known normalized implementation/reviewer vendor/model, disclose their source, and never present a signed executor declaration as a vendor receipt or cognitive independence.
- R7: A proof shall distinguish the brief body and exact transmitted bytes, raw and redacted transcript/output digests, and output completeness; incomplete or truncated output shall not establish a completed review.
- R8: Current reads shall reverify issuer pins, authority/audience/project/time, exact sources and integrity through the existing B01/B04/native issuer chain; invalid or altered proof and legacy declared records shall not become run-backed assurance.
- R9: Proof shall retain B16 attempt/continuation lineage and rejection history; a signature shall not restore quota, erase a rejection or authorize effects from superseded inputs.
- R10: Domain, CLI, MCP, verify and Attention shall share proof/disclosure for same vendor/model, differing model or vendor, prefix-only, unknown baseline and tampered issuer/config/output cases; machine-attested review shall never satisfy a human requirement.

## 3. Technical Plan

### Affected areas

Run record signing through the native issuer, an attestation field referencing the run, and the verifier change behind the capability.

### Artifacts

- created: .pose/adr/2026-10-25-bind-delegated-review-authority-to-native-signed-run-proofs.md
- modified: .pose/assessments/README.md
- modified: .pose/assessments/consolidated.md
- modified: .pose/assessments/integrations.md
- modified: .pose/assessments/technical-debt.md
- created: .pose/changelogs/unreleased/pose-delegated-review-run-provenance.md
- modified: .pose/indexes/validation-matrix.json
- created: .pose/reports/2026-10-25-b17-run-provenance.md
- created: .pose/specs/2026-10-09-pose-delegated-review-run-provenance.md
- created: .pose/starts/pose-delegated-review-run-provenance.json
- modified: .pose/state/integrations.json
- modified: .pose/state/technical-debt.json
- modified: POSE.md
- modified: locales/pt-BR/POSE.md
- modified: pose-mcp/internal/cli/check.go
- modified: pose-mcp/internal/cli/help_catalog.go
- modified: pose-mcp/internal/cli/review_closeout.go
- created: pose-mcp/internal/cli/review_run_provenance.go
- created: pose-mcp/internal/cli/review_run_provenance_test.go
- modified: pose-mcp/internal/cli/state_attention.go
- created: pose-mcp/internal/mcpserver/review_run_provenance_test.go
- modified: pose-mcp/internal/pose/issuer_key.go
- modified: pose-mcp/internal/pose/live_policy_reads_test.go
- modified: pose-mcp/internal/pose/obligation_projection.go
- modified: pose-mcp/internal/pose/review_assurance.go
- modified: pose-mcp/internal/pose/review_attempt_ledger.go
- modified: pose-mcp/internal/pose/review_bundle.go
- modified: pose-mcp/internal/pose/review_closeout.go
- modified: pose-mcp/internal/pose/review_dispatch.go
- modified: pose-mcp/internal/pose/review_run_continuity.go
- created: pose-mcp/internal/pose/review_run_provenance.go
- created: pose-mcp/internal/pose/review_run_provenance_test.go
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md

### Delivery targets

- capability:delegated-run-provenance module:pose-mcp profile:composed-capability entrypoint:pose-mcp/internal/pose/review_dispatch.go

## 4. Tasks

- [x] Signed run record
- [x] Attestation-to-run reference
- [x] Verifier reads vendor/model from the run
- [x] Assurance disclosure

## 5. Decisions

### Decision D1
- Basis: R1, R2, R3, R4, R5, R6, R7, R8, R9, R10
- ADR: [Native signed run proofs](../adr/2026-10-25-bind-delegated-review-authority-to-native-signed-run-proofs.md)
- Knowledge: knowledge:review-assurance-requires-cryptographic-proof, knowledge:adr-sealed-review-bundles-review
- Decision: Capture immutable execution provenance at dispatch, sign a separate content-bound envelope with the existing pinned issuer, and use one current verifier for admission and disclosure.
- Rationale: Signing post-hoc configuration cannot prove which inputs the executor used; a reviewer prefix cannot authenticate separation.
- Consequences: Legacy runs remain declared. Explicit opt-in requires proof for different-actor agent reviews; catalog adoption and governed draft application remain later capability scope.

## 6. Validation

### Strategy

Adversarial tests: a prefix-only reviewer is refused under the capability; a run by the implementer's own model is refused; a tampered run record fails its signature.

### Required execution plan

| Scenario | Command | Expected evidence |
| --- | --- | --- |
| Signed binding and adversarial reads (required) | `cd pose-mcp && go test ./internal/pose ./internal/cli ./internal/mcpserver -run RunProvenance -count=1` | Vendor/model OR rule; prefix/unknown/same identity and tampering refuse; shared machine-attested disclosure never grants human authority |
| Publication and race safety (required) | `cd pose-mcp && go test -race ./internal/pose ./internal/cli ./internal/mcpserver -run RunProvenance -count=1` | Immutable proof publication without races or duplicate writes |
| Canonical composition/regression (required) | `pose validate --strict --json-out .pose/results/delivery-validation.json` with TMPDIR=/var/tmp | All registered checks pass with current provenance |
| Security (required) | `cd pose-mcp && govulncheck ./...` and pinned CI gitleaks | No vulnerabilities or secrets; private key/session excluded from evidence |

### Requirement trace

- R1 [satisfied] test:TestRunProvenanceSignedBindingAndDisclosure test:TestRunProvenanceProjectConfigAndImmutablePublication
- R2 [satisfied] test:TestRunProvenanceCLIProofAndAttestation test:TestRunProvenanceRejectingOrAmbiguousDraftCannotApprove
- R3 [satisfied] test:TestRunProvenanceSameModelAndUnknownBaselineRefused test:TestRunProvenanceLiveOptInDoesNotGrandfatherPrefix
- R4 [satisfied] test:TestRunProvenanceMCPReadsCurrentMachineProof test:TestRunProvenancePrefixAndHumanNeverUpgrade
- R5 [satisfied] test:TestRunProvenanceProjectConfigAndImmutablePublication test:TestRunProvenancePreservesArgvZeroAndResolvesWorktreeCommand
- R6 [satisfied] test:TestRunProvenanceSameModelAndUnknownBaselineRefused test:TestRunProvenancePrefixAndHumanNeverUpgrade
- R7 [satisfied] test:TestRunProvenancePartialStdinAndUnstartedAudit test:TestRunProvenanceTruncatedOutputAndPartialAudit
- R8 [satisfied] test:TestRunProvenanceTamperingRevocationAndExpiry test:TestRunProvenanceAttestationCannotPredateExecutionOrProof
- R9 [satisfied] test:TestRunProvenanceMissingAttemptReceiptInvalidatesAuthority test:TestRunProvenanceRejectingOrAmbiguousDraftCannotApprove
- R10 [satisfied] test:TestRunProvenanceCLIProofAndAttestation test:TestRunProvenanceMCPReadsCurrentMachineProof test:TestRunProvenancePrefixAndHumanNeverUpgrade

## 7. Final Report

### Delivered scope

Implemented immutable native-signed run proofs, exact attestation references, current vendor/model independence verification, live opt-in proof bar and shared machine-attested assurance across domain/CLI/MCP/Attention. Canonical validation, independent native review and closeout remain the delivery authority; see report `2026-10-25-b17-run-provenance.md`.

### Residual risks

Signed executor declarations do not prove vendor identity independently, human authority or cognitive independence. Raw bytes are hashed by the executor while redacted artifacts remain verifiable. Origin-worktree binding is local. Legacy runs stay declared; catalog adoption, mixed-version qualification, full conclusion validation/application and containment remain their canonical scopes.

### Follow-ups

No new residual task. Capability/adoption, containment and remote runner qualification remain in their existing canonical scopes; this delivery does not change their disposition.
