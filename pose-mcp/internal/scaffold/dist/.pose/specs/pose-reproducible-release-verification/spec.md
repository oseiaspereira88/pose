---
slug: pose-reproducible-release-verification
status: draft
created_at: 2026-07-18
completed_at:
supersedes:
depends_on: pose-slsa-provenance, pose-release-compatibility-matrix
priority: 9
---

# Spec: Independent release verification

## 1. Intent

### Goal
verify release artifacts from a clean environment and quantify reproducibility limits.
### Business value
Catches packaging, provenance and compatibility failures the producer can miss.
### Constraints
- Report nondeterministic inputs explicitly.
### Non-goals
- Guarantee identical binaries across unsupported environments.

## 2. Requirements

### Functional
- R1: A separate job shall download, authenticate and inspect artifacts without producer state.
- R2: The verifier shall compare files, versions, checksums, signatures, SBOM and provenance.
- R3: Rebuild comparisons shall document deterministic matches and explained differences.

### Non-functional
- Keep verification isolated from producer credentials and caches.

### Security
- Execute artifacts only after identity and digest verification.

### Compatibility
- Exercise every supported target natively or with documented emulation.

## 3. Technical Plan

### Affected areas
- Independent workflow, container/VM fixtures and release report.

### API/contract changes
- Publish a consumer-verification procedure and result schema.

### Data/storage changes
- Retain verification logs and reproducibility deltas.

### Technical risks
- Sharing producer workflow or credentials creates circular evidence.

### Primary references
- [SLSA 1.2](https://slsa.dev/spec/v1.2/)
- [The Update Framework](https://theupdateframework.io/)

## 4. Tasks

### Planning
- [ ] Confirm baseline and fixtures against [SLSA 1.2](https://slsa.dev/spec/v1.2/).

### Implementation
- [ ] Create an independent verifier environment and trust policy. ([reference](https://slsa.dev/spec/v1.2/))
- [ ] Verify archive contents and all linked metadata before execution. ([reference](https://theupdateframework.io/))
- [ ] Attempt controlled rebuilds and classify nondeterministic differences. ([reference](https://slsa.dev/spec/v1.2/))

### Validation
- [ ] Run `pose check --strict` and retain the result artifact. ([reference](https://slsa.dev/spec/v1.2/))
- [ ] Run `pose check --strict` and inspect readiness projections. ([reference](https://theupdateframework.io/))

## 5. Decisions

- Create an ADR before changing this public or structural contract; compare alternatives against [SLSA 1.2](https://slsa.dev/spec/v1.2/).

## 6. Validation

**Strategy:** validate unit behavior, negative/security cases, public contract fixtures and an end-to-end consumer path.

### Planned deterministic checks
- Test: `pose check --strict`.
- Structure: `pose check --strict`.
- Readiness: `pose lint-spec pose-reproducible-release-verification --ready-check`.

### Execution status
- Not executed. This planning spec remains `draft`; delivery requires recorded gate evidence.

## 7. Final Report

- Delivered scope: none; this spec defines future implementation.
- Residual risk: Sharing producer workflow or credentials creates circular evidence.
- Follow-ups: none until implementation starts.
