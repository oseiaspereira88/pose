---
slug: pose-delegated-review-run-provenance
status: draft
created_at: 2026-10-09
completed_at:
supersedes:
depends_on: pose-delegated-review-dispatch, pose-native-attestation-issuer
priority: 2
components: pose-mcp
task_type: feature
changelog:
delivers:
---

# Spec: Independence read from a signed run record

## 1. Intent

### Goal

Satisfy `different-actor` for a delegated review from evidence — the run record — instead of the `agent:independent-` reviewer prefix.

### Business value

Under declared assurance the engine accepts `different-actor` when the reviewer is named `agent:independent-…`; on 2026-10-09 the reviewing agent had to adopt that literal prefix to be accepted. Anyone can type it, and the engine's own comment calls it a declaration.

Origin: the 2026-10-09 review of `pose-native-attestation-issuer` by Codex through a personal launcher; it found three real defects in three rounds and exposed what the engine lacks (ADR `2026-10-09-delegated-review-is-an-adapter`, roadmap `delegated-review`).

### Constraints

Honest labelling: a machine-attested run is disclosed as such, never as a person. Requirements assume the ADR's provisional answers until `pose-delegated-review-contract` closes.

## 2. Requirements

### Functional

- R1: A dispatch run record shall be signable by a pinned native issuer, binding brief digest, transcript digest, adapter, vendor, model and bundle digest.
- R2: An attestation produced from a run shall reference the run; the verifier shall read the reviewer's vendor and model from the referenced run record.
- R3: `different-actor` shall be satisfied by a run whose vendor or model differs from the implementation principal's declared vendor and model, per the contract spec's answer; the `agent:independent-` prefix shall no longer satisfy it in instances that adopt `delegated-review`.
- R4: Review assurance shall disclose a run-backed review as `machine-attested`, distinct from `declared` and from a person's signed confirmation.

## 3. Technical Plan

### Affected areas

Run record signing through the native issuer, an attestation field referencing the run, and the verifier change behind the capability.

### Artifacts

- created: .pose/specs/2026-10-09-pose-delegated-review-run-provenance.md

Implementation artifacts are declared when the spec starts.

## 4. Tasks

- [ ] Signed run record
- [ ] Attestation-to-run reference
- [ ] Verifier reads vendor/model from the run
- [ ] Assurance disclosure

## 5. Decisions

No decision recorded yet; the contract is ADR `2026-10-09-delegated-review-is-an-adapter`, pending acceptance.

## 6. Validation

### Strategy

Adversarial tests: a prefix-only reviewer is refused under the capability; a run by the implementer's own model is refused; a tampered run record fails its signature.

### Requirement trace

## 7. Final Report

### Delivered scope

Not started: part of roadmap `delegated-review`, opened on 2026-10-09.

### Residual risks

None yet.

### Follow-ups
