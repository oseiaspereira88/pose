---
slug: pose-release-signing
status: draft
created_at: 2026-07-18
completed_at:
supersedes:
depends_on: pose-version-contract
priority: 6
---

# Spec: Keyless release signing

## 1. Intent

### Goal
sign release artifacts and checksums with verifiable workload identity.
### Business value
Lets adopters authenticate POSE instead of trusting a same-channel checksum.
### Constraints
- Prefer keyless OIDC signing and publish offline-verifiable bundles.
### Non-goals
- Design a private enterprise PKI.

## 2. Requirements

### Functional
- R1: Every release archive and checksum manifest shall have a verifiable signature or bundle.
- R2: Instructions shall pin expected issuer and repository identity.
- R3: Release CI shall fail on unsigned artifacts or identity mismatch.

### Non-functional
- Keep verification cross-platform and scriptable.

### Security
- Avoid long-lived signing keys in repository secrets.

### Compatibility
- Retain checksums while adding signature verification.

## 3. Technical Plan

### Affected areas
- GoReleaser, release workflow, docs and verification smoke tests.

### API/contract changes
- Release identity and verification commands become public contracts.

### Data/storage changes
- Publish signature bundles beside release assets.

### Technical risks
- Cryptographic validity without issuer constraints accepts the wrong signer.

### Primary references
- [Sigstore documentation](https://docs.sigstore.dev/)
- [GitHub artifact attestations](https://docs.github.com/en/actions/how-tos/secure-your-work/use-artifact-attestations/use-artifact-attestations)

## 4. Tasks

### Planning
- [ ] Confirm baseline and fixtures against [Sigstore documentation](https://docs.sigstore.dev/).

### Implementation
- [ ] Define issuer, subject and artifact identity policies. ([reference](https://docs.sigstore.dev/))
- [ ] Integrate keyless signing for archives and checksums. ([reference](https://docs.github.com/en/actions/how-tos/secure-your-work/use-artifact-attestations/use-artifact-attestations))
- [ ] Test valid, wrong-identity, modified and missing-signature cases. ([reference](https://docs.sigstore.dev/))

### Validation
- [ ] Run `go test ./... -run 'Release|Signature'` and retain the result artifact. ([reference](https://docs.sigstore.dev/))
- [ ] Run `pose check --strict` and inspect readiness projections. ([reference](https://docs.github.com/en/actions/how-tos/secure-your-work/use-artifact-attestations/use-artifact-attestations))

## 5. Decisions

- Create an ADR before changing this public or structural contract; compare alternatives against [Sigstore documentation](https://docs.sigstore.dev/).

## 6. Validation

**Strategy:** validate unit behavior, negative/security cases, public contract fixtures and an end-to-end consumer path.

### Planned deterministic checks
- Test: `go test ./... -run 'Release|Signature'`.
- Structure: `pose check --strict`.
- Readiness: `pose lint-spec pose-release-signing --ready-check`.

### Execution status
- Not executed. This planning spec remains `draft`; delivery requires recorded gate evidence.

## 7. Final Report

- Delivered scope: none; this spec defines future implementation.
- Residual risk: Cryptographic validity without issuer constraints accepts the wrong signer.
- Follow-ups: none until implementation starts.
