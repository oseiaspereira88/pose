---
slug: pose-extension-catalog-lifecycle
status: draft
created_at: 2026-07-18
completed_at:
supersedes:
depends_on: pose-agent-skills-conformance, pose-release-signing
priority: 24
---

# Spec: Signed extension catalog and lifecycle

## 1. Intent

### Goal
define install, update, conflict, removal and provenance for skills, workflows, rules and import adapters.
### Business value
Lets teams extend POSE without forks and creates an ecosystem path.
### Constraints
- Owners approve mutations; extensions cannot silently override security.
### Non-goals
- Create an unmoderated marketplace or execute installer scripts.

## 2. Requirements

### Functional
- R1: A manifest shall declare contents, compatibility, permissions, conflicts and provenance.
- R2: Lifecycle operations shall be dry-runnable, transactional and preserve user modifications.
- R3: A signed catalog shall support discovery, revocation and custom import adapters.

### Non-functional
- Keep installation deterministic from a digest.

### Security
- Verify signatures, confine paths, reject unsafe archives and require consent.

### Compatibility
- Core file contracts work without catalog or network.

## 3. Technical Plan

### Affected areas
- Extension domain, installer, import, catalog, signatures and docs.

### API/contract changes
- Create a versioned manifest and lifecycle command family.

### Data/storage changes
- Store lock/provenance separately from editable content.

### Technical risks
- Updates can overwrite policy or create dependency confusion.

### Primary references
- [OCI Distribution Specification](https://github.com/opencontainers/distribution-spec/blob/main/spec.md)
- [Sigstore documentation](https://docs.sigstore.dev/)

## 4. Tasks

### Planning
- [ ] Confirm baseline and fixtures against [OCI Distribution Specification](https://github.com/opencontainers/distribution-spec/blob/main/spec.md).

### Implementation
- [ ] Specify manifest, permission, conflict and provenance semantics. ([reference](https://github.com/opencontainers/distribution-spec/blob/main/spec.md))
- [ ] Implement dry-run transactional lifecycle with rollback. ([reference](https://docs.sigstore.dev/))
- [ ] Publish signed fixtures including skill and import adapters. ([reference](https://github.com/opencontainers/distribution-spec/blob/main/spec.md))

### Validation
- [ ] Run `go test ./pose-mcp/... -run 'Extension|Import|Install|Signature'` and retain evidence. ([reference](https://github.com/opencontainers/distribution-spec/blob/main/spec.md))
- [ ] Run `pose check --strict` and inspect readiness. ([reference](https://docs.sigstore.dev/))

## 5. Decisions

- Create an ADR before changing this contract; compare [OCI Distribution Specification](https://github.com/opencontainers/distribution-spec/blob/main/spec.md).

## 6. Validation

**Strategy:** validate units, negative/security cases, contract fixtures and an end-to-end path.

### Planned deterministic checks
- Test: `go test ./pose-mcp/... -run 'Extension|Import|Install|Signature'`.
- Structure: `pose check --strict`.
- Readiness: `pose lint-spec pose-extension-catalog-lifecycle --ready-check`.

### Execution status
- Not executed. This planning spec remains `draft`; delivery requires gate evidence.

## 7. Final Report

- Delivered scope: none; this spec defines future implementation.
- Residual risk: Updates can overwrite policy or create dependency confusion.
- Follow-ups: none until implementation starts.

