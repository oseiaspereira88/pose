---
slug: pose-slsa-provenance
status: draft
created_at: 2026-07-18
completed_at:
supersedes:
depends_on: pose-release-signing, pose-cyclonedx-sbom
priority: 8
---

# Spec: SLSA build provenance

## 1. Intent

### Goal
publish verifiable provenance linking artifacts to source, builder and invocation.
### Business value
Raises release trust from integrity metadata to verifiable build origin.
### Constraints
- Use a build model that meets only the properties the project claims.
### Non-goals
- Claim a SLSA level beyond independently verifiable evidence.

## 2. Requirements

### Functional
- R1: Every archive shall be a provenance subject identified by digest.
- R2: Provenance shall identify source revision, builder and invocation without secrets.
- R3: A clean verifier shall reject modified artifacts, wrong repositories and untrusted builders.

### Non-functional
- Generate provenance without manual release mutation.

### Security
- Use ephemeral credentials and isolate untrusted code from signing identity.

### Compatibility
- Publish standard in-toto/SLSA predicates beside releases.

## 3. Technical Plan

### Affected areas
- Release workflow, builder, attestations and verification docs.

### API/contract changes
- Publish an explicit SLSA claim with evidence and limitations.

### Data/storage changes
- Retain provenance bundles and verification results.

### Technical risks
- A signed but weakly isolated build can overstate the guarantee.

### Primary references
- [SLSA 1.2](https://slsa.dev/spec/v1.2/)
- [GitHub artifact attestations](https://docs.github.com/en/actions/how-tos/secure-your-work/use-artifact-attestations/use-artifact-attestations)

## 4. Tasks

### Planning
- [ ] Confirm baseline and fixtures against [SLSA 1.2](https://slsa.dev/spec/v1.2/).

### Implementation
- [ ] Threat-model the builder and select the supportable SLSA claim. ([reference](https://slsa.dev/spec/v1.2/))
- [ ] Generate provenance for every archive. ([reference](https://docs.github.com/en/actions/how-tos/secure-your-work/use-artifact-attestations/use-artifact-attestations))
- [ ] Add independent subject, source and builder verification fixtures. ([reference](https://slsa.dev/spec/v1.2/))

### Validation
- [ ] Run `go test ./... -run 'Provenance|Release'` and retain the result artifact. ([reference](https://slsa.dev/spec/v1.2/))
- [ ] Run `pose check --strict` and inspect readiness projections. ([reference](https://docs.github.com/en/actions/how-tos/secure-your-work/use-artifact-attestations/use-artifact-attestations))

## 5. Decisions

- Create an ADR before changing this public or structural contract; compare alternatives against [SLSA 1.2](https://slsa.dev/spec/v1.2/).

## 6. Validation

**Strategy:** validate unit behavior, negative/security cases, public contract fixtures and an end-to-end consumer path.

### Planned deterministic checks
- Test: `go test ./... -run 'Provenance|Release'`.
- Structure: `pose check --strict`.
- Readiness: `pose lint-spec pose-slsa-provenance --ready-check`.

### Execution status
- Not executed. This planning spec remains `draft`; delivery requires recorded gate evidence.

## 7. Final Report

- Delivered scope: none; this spec defines future implementation.
- Residual risk: A signed but weakly isolated build can overstate the guarantee.
- Follow-ups: none until implementation starts.
