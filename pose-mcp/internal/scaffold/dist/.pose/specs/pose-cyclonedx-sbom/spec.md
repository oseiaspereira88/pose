---
slug: pose-cyclonedx-sbom
status: draft
created_at: 2026-07-18
completed_at:
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
- [ ] Confirm baseline and fixtures against [CycloneDX specification](https://cyclonedx.org/specification/overview/).

### Implementation
- [ ] Select format version, generator and artifact mapping. ([reference](https://cyclonedx.org/specification/overview/))
- [ ] Generate and schema-validate source and binary inventories. ([reference](https://openchainproject.org/license-compliance))
- [ ] Add direct-dependency and license-completeness policy checks. ([reference](https://cyclonedx.org/specification/overview/))

### Validation
- [ ] Run `go test ./... && pose check --strict` and retain the result artifact. ([reference](https://cyclonedx.org/specification/overview/))
- [ ] Run `pose check --strict` and inspect readiness projections. ([reference](https://openchainproject.org/license-compliance))

## 5. Decisions

- Create an ADR before changing this public or structural contract; compare alternatives against [CycloneDX specification](https://cyclonedx.org/specification/overview/).

## 6. Validation

**Strategy:** validate unit behavior, negative/security cases, public contract fixtures and an end-to-end consumer path.

### Planned deterministic checks
- Test: `go test ./... && pose check --strict`.
- Structure: `pose check --strict`.
- Readiness: `pose lint-spec pose-cyclonedx-sbom --ready-check`.

### Execution status
- Not executed. This planning spec remains `draft`; delivery requires recorded gate evidence.

## 7. Final Report

- Delivered scope: none; this spec defines future implementation.
- Residual risk: Generated license fields may be incomplete and need review policy.
- Follow-ups: none until implementation starts.
