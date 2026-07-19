---
slug: pose-slsa-provenance
status: done
created_at: 2026-07-18
completed_at: 2026-07-19
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
- [x] Confirm baseline and fixtures against [SLSA 1.2](https://slsa.dev/spec/v1.2/): no provenance existed; builder is GitHub-hosted, ephemeral-credentialed, not isolated from the workflow definition — the L2/L3 boundary drove the claim.

### Implementation
- [x] Threat-model the builder and select the supportable SLSA claim: **Build L2** (hosted platform, signed provenance), explicitly not L3; limitation published in `SECURITY.md` (ADR `2026-07-19-slsa-build-l2-provenance-claim`). ([reference](https://slsa.dev/spec/v1.2/))
- [x] Generate provenance for every archive: `actions/attest-build-provenance@v4` in the tag pipeline attests every archive plus `checksums.txt` by digest (SLSA v1 predicate, in-toto format); `attestations: write` + `id-token: write` scoped; no manual release mutation. ([reference](https://docs.github.com/en/actions/how-tos/secure-your-work/use-artifact-attestations/use-artifact-attestations))
- [x] Add independent subject, source and builder verification fixtures: the independent verifier runs `gh attestation verify --repo --signer-workflow` for the platform archive and checksum manifest — digest mismatch, wrong repository or untrusted builder fail; `TestAttestedReleaseContract` pins the attestation step, subjects and scopes. ([reference](https://slsa.dev/spec/v1.2/))

### Validation
- [x] Run `go test ./... -run 'Provenance|Release'` and retain the result artifact (contract coverage in `TestAttestedReleaseContract`; see §6 and `.pose/reports/`). ([reference](https://slsa.dev/spec/v1.2/))
- [x] Run `pose check --strict` and inspect readiness projections. ([reference](https://docs.github.com/en/actions/how-tos/secure-your-work/use-artifact-attestations/use-artifact-attestations))

## 5. Decisions

- ADR `.pose/adr/2026-07-19-slsa-build-l2-provenance-claim.md` (Accepted): GitHub artifact attestations over the slsa-github-generator reusable builder (deferred L3); bounded L2 claim with published limitation; provenance composes with — never replaces — Sigstore bundles and checksums.

## 6. Validation

**Strategy:** validate unit behavior, negative/security cases, public contract fixtures and an end-to-end consumer path.

### Planned deterministic checks
- Test: `go test ./... -run 'Provenance|Release'`.
- Structure: `pose check --strict`.
- Readiness: `pose lint-spec pose-slsa-provenance --ready-check`.

### Execution status
Executed on 2026-07-19 with a development build (`pose 0.9.0-dev`):

- `go -C pose-mcp test ./internal/version -run 'AttestedRelease|ArtifactIdentity' -count=1` — SUCCESS.
- `go -C pose-mcp test ./... -count=1` — SUCCESS (full suite); all workflow YAML parsed cleanly.
- `pose check --strict` — SUCCESS; `pose validate --strict --module pose-mcp --report` — SUCCESS (report retained under `.pose/reports/`).
- Attestation generation requires the tag pipeline's OIDC environment; the first tagged release produces and the Verify release workflow consumes the real provenance — no attestation result is claimed here.

## 7. Final Report

### Delivered scope

SLSA v1 provenance attestation for every archive and the checksum manifest in
the tag pipeline; bounded, published Build L2 claim with the L3 limitation
stated; consumer verification command pinned to repo + signer workflow in
`SECURITY.md`; independent-verifier integration; contract test; ADR.

### Residual risks

- The build is not isolated from the workflow definition (L2, not L3) —
  published as a limitation, with the reusable-builder upgrade as future
  work if the trust story demands it.

### Follow-ups

- [open] After the first tagged release, confirm `gh attestation verify` passes for all six archives and checksums.txt, and record the evidence.
- [covered: pose-reproducible-release-verification] Consumer-side verification of provenance together with all other layers.
