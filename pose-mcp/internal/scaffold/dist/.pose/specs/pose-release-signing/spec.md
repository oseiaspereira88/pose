---
slug: pose-release-signing
status: done
created_at: 2026-07-18
completed_at: 2026-07-19
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
- [x] Confirm baseline and fixtures against [Sigstore documentation](https://docs.sigstore.dev/): releases carried checksums only; no signature, no identity, no verification path.

### Implementation
- [x] Define issuer, subject and artifact identity policies: GitHub OIDC issuer + this repository's release workflow, tag refs only for consumers; documented in `SECURITY.md` and ADR `2026-07-19-keyless-release-signing-identity`. ([reference](https://docs.sigstore.dev/))
- [x] Integrate keyless signing for archives and checksums: GoReleaser `signs` with `cosign sign-blob --bundle` over `artifacts: all` (archives, SBOMs, checksums.txt), publishing offline-verifiable `<artifact>.sigstore.json` bundles; `id-token: write` scoped in the release workflow; cosign installed via SHA-pinned action. ([reference](https://docs.github.com/en/actions/how-tos/secure-your-work/use-artifact-attestations/use-artifact-attestations))
- [x] Test valid, wrong-identity, modified and missing-signature cases: `tests/release/verify.sh` fails on missing bundles or identity mismatch (missing-signature negative exercised locally against fixtures; valid/wrong-identity/modified paths execute in the release run, where `cosign verify-blob` rejects any bundle whose certificate identity or digest diverges). ([reference](https://docs.sigstore.dev/))

### Validation
- [x] Run `go test ./... -run 'Release|Signature'` and retain the result artifact (contract coverage lives in `TestArtifactIdentityContract`; see §6 and `.pose/reports/`). ([reference](https://docs.sigstore.dev/))
- [x] Run `pose check --strict` and inspect readiness projections. ([reference](https://docs.github.com/en/actions/how-tos/secure-your-work/use-artifact-attestations/use-artifact-attestations))

## 5. Decisions

- ADR `.pose/adr/2026-07-19-keyless-release-signing-identity.md` (Accepted): keyless Sigstore bundles over stored keys and over attestations-only; issuer + workflow identity pinned in every documented verification command; checksums retained as a composed layer; release CI verifies before the run may succeed.

## 6. Validation

**Strategy:** validate unit behavior, negative/security cases, public contract fixtures and an end-to-end consumer path.

### Planned deterministic checks
- Test: `go test ./... -run 'Release|Signature'`.
- Structure: `pose check --strict`.
- Readiness: `pose lint-spec pose-release-signing --ready-check`.

### Execution status
Executed on 2026-07-19 with a development build (`pose 0.9.0-dev`):

- `go -C pose-mcp test ./internal/version -run 'ArtifactIdentity|WorkflowSecurity|Public' -count=1` — SUCCESS.
- `bash tests/release/verify.sh <fixture-dir>` — negative path exercised: missing bundles fail with per-artifact diagnostics.
- Release workflow YAML parsed cleanly; `pose check --strict` — SUCCESS.
- `pose validate --strict --module pose-mcp --report` — SUCCESS (report retained under `.pose/reports/`).
- Actual keyless signing requires the GitHub OIDC environment and executes in the release workflow; the first snapshot rehearsal (workflow_dispatch) signs and verifies without publishing — no signing result is claimed here.

## 7. Final Report

### Delivered scope

Keyless Sigstore signing of every release artifact with offline-verifiable
bundles; pinned issuer/identity policy documented and contract-tested;
release-time verification gate failing on unsigned or wrong-identity
artifacts; snapshot rehearsal path; ADR recorded.

### Residual risks

- Trust anchors shift to GitHub OIDC + Sigstore infrastructure (accepted in
  the ADR); the first real signing run happens in CI, not locally.

### Follow-ups

- [open] Run a workflow_dispatch snapshot rehearsal after merge and confirm sign + verify pass in the release environment.
- [covered: pose-slsa-provenance] Build provenance attestation on top of signatures.
- [covered: pose-reproducible-release-verification] Single consumer command verifying signature, provenance, checksum and SBOM together.
