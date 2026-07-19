---
slug: pose-cyclonedx-sbom
status: done
created_at: 2026-07-18
completed_at: 2026-07-19
supersedes:
depends_on: pose-version-contract
priority: 7
---

# Spec: CycloneDX release SBOM

## 1. Intent

### Goal
publish a component and license inventory tied to every release artifact.
### Business value
Improves vulnerability response, dependency transparency and enterprise adoption.
### Constraints
- Generate from exact build inputs and distinguish source from binary analysis.
### Non-goals
- Claim an SBOM proves absence of vulnerabilities or license risk.

## 2. Requirements

### Functional
- R1: Each release shall publish a CycloneDX SBOM with versions, hashes and known licenses.
- R2: The SBOM shall identify the release artifact or provenance subject it describes.
- R3: CI shall validate schema and fail on missing direct production dependencies.

### Non-functional
- Keep generation reproducible and diff-reviewable.

### Security
- Exclude secrets, private paths and credentials from metadata.

### Compatibility
- Publish a standard media type consumable by common scanners.

## 3. Technical Plan

### Affected areas
- GoReleaser, dependency metadata, NOTICE review and release assets.

### API/contract changes
- SBOM filename, format version and subject mapping become stable metadata.

### Data/storage changes
- Retain SBOMs with releases and optionally attest them.

### Technical risks
- Generated license fields may be incomplete and need review policy.

### Primary references
- [CycloneDX specification](https://cyclonedx.org/specification/overview/)
- [OpenChain ISO/IEC 5230](https://openchainproject.org/license-compliance)

## 4. Tasks

### Planning
- [x] Confirm baseline and fixtures against [CycloneDX specification](https://cyclonedx.org/specification/overview/): no inventory of any kind shipped with releases; direct production dependency surface is small (`mcp-enforce`) and parseable from `go.mod`.

### Implementation
- [x] Select format version, generator and artifact mapping: CycloneDX JSON via syft (SHA-pinned installer), one SBOM per archive named `<archive>.cdx.json` — filename mapping is stable public metadata (ADR `2026-07-19-cyclonedx-sbom-publication`). ([reference](https://cyclonedx.org/specification/overview/))
- [x] Generate and schema-validate source and binary inventories: binary analysis of the exact packaged artifact in GoReleaser (`syft scan ${artifact}`); `verify.sh` validates `bomFormat`, `specVersion` and non-empty `components`; source-level regeneration stays possible from the tagged tree and the analysis type is recorded by syft in SBOM metadata. ([reference](https://openchainproject.org/license-compliance))
- [x] Add direct-dependency and license-completeness policy checks: `verify.sh` fails when a direct production dependency parsed from `pose-mcp/go.mod` is missing from the SBOM; license completeness is reviewed against `NOTICE` per the ADR policy (no absence-of-risk claim). ([reference](https://cyclonedx.org/specification/overview/))

### Validation
- [x] Run `go test ./... && pose check --strict` and retain the result artifact (see §6 and `.pose/reports/`). ([reference](https://cyclonedx.org/specification/overview/))
- [x] Run `pose check --strict` and inspect readiness projections. ([reference](https://openchainproject.org/license-compliance))

## 5. Decisions

- ADR `.pose/adr/2026-07-19-cyclonedx-sbom-publication.md` (Accepted): CycloneDX over SPDX (spec-named, scanner-consumable, syft-native); per-archive binary analysis over one source-tree SBOM (ties inventory to the artifact the consumer downloaded); SBOMs are themselves Sigstore-signed; secrets and private paths never enter metadata.

## 6. Validation

**Strategy:** validate unit behavior, negative/security cases, public contract fixtures and an end-to-end consumer path.

### Planned deterministic checks
- Test: `go test ./... && pose check --strict`.
- Structure: `pose check --strict`.
- Readiness: `pose lint-spec pose-cyclonedx-sbom --ready-check`.

### Execution status
Executed on 2026-07-19 with a development build (`pose 0.9.0-dev`):

- `go -C pose-mcp test ./... -count=1` — SUCCESS (includes `TestArtifactIdentityContract` pinning the SBOM config).
- `bash tests/release/verify.sh <fixture-dir>` — SBOM schema check passes on a valid CycloneDX fixture and fails when a direct production dependency is absent (negative path exercised locally).
- `pose check --strict` — SUCCESS; `pose validate --strict --module pose-mcp --report` — SUCCESS (report retained under `.pose/reports/`).
- Real SBOM generation requires syft and executes in the release workflow; the first snapshot rehearsal produces and validates the inventories — no generation result is claimed here.

## 7. Final Report

### Delivered scope

Per-archive CycloneDX SBOM generation in GoReleaser (binary analysis of the
exact artifact, stable `<archive>.cdx.json` mapping); schema and
direct-dependency validation gates in the release run; SBOMs published and
Sigstore-signed as release assets; format/generator/mapping ADR; SECURITY.md
consumer guidance.

### Residual risks

- Generated license fields may be incomplete; the NOTICE review policy owns
  corrections and no license-risk-absence claim is made.
- The first real generation run happens in CI (syft unavailable offline).

### Follow-ups

- [open] Inspect the first snapshot rehearsal's SBOMs: confirm syft resolves the replaced `mcp-enforce` module path and review detected licenses against NOTICE. (owner:@pose-maintainers crit:medium review:2026-08-14)
- [covered: pose-slsa-provenance] Attest SBOM subjects in build provenance.
