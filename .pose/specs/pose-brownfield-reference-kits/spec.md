---
slug: pose-brownfield-reference-kits
status: draft
created_at: 2026-07-18
completed_at:
supersedes:
depends_on: pose-standalone-dogfood, pose-monorepo-validation-recipes, pose-agent-skills-conformance
priority: 28
---

# Spec: Brownfield reference kits

## 1. Intent

### Goal
publish executable adoption kits for existing repos using POSE alone and with Spec Kit/OpenSpec.
### Business value
Demonstrates incremental value without demanding a governance rewrite.
### Constraints
- Represent imperfect repositories and keep lifecycle authority explicit.
### Non-goals
- Promise automatic semantic migration without curation.

## 2. Requirements

### Functional
- R1: Kits shall cover direct adoption, Spec Kit import and OpenSpec import/reconciliation.
- R2: Each kit shall progress from visibility to blocking gates with rollback.
- R3: CI shall execute commands and assert preservation, warnings and readiness.

### Non-functional
- Keep kits small, reproducible and release-versioned.

### Security
- Use sanitized fixtures and test symlink, overwrite and boundary rejection.

### Compatibility
- Document mapping loss and retain source provenance.

## 3. Technical Plan

### Affected areas
- Examples, import adapters, docs, CI and extension fixtures.

### API/contract changes
- Publish pathways and authority-transfer rules.

### Data/storage changes
- Version source fixtures, mapping reports and post-adoption snapshots.

### Technical risks
- Idealized examples conceal real migration costs.

### Primary references
- [GitHub Spec Kit](https://github.com/github/spec-kit/blob/main/docs/reference/overview.md)
- [OpenSpec](https://github.com/Fission-AI/OpenSpec)

## 4. Tasks

### Planning
- [ ] Confirm baseline and fixtures against [GitHub Spec Kit](https://github.com/github/spec-kit/blob/main/docs/reference/overview.md).

### Implementation
- [ ] Design representative greenfield, brownfield and mixed-SDD fixtures. ([reference](https://github.com/github/spec-kit/blob/main/docs/reference/overview.md))
- [ ] Implement staged guides with preservation assertions. ([reference](https://github.com/Fission-AI/OpenSpec))
- [ ] Measure time-to-first-gate and document mapping loss. ([reference](https://github.com/github/spec-kit/blob/main/docs/reference/overview.md))

### Validation
- [ ] Run `go test ./pose-mcp/internal/cli/... -run 'Import|Install|Preserve'` and retain evidence. ([reference](https://github.com/github/spec-kit/blob/main/docs/reference/overview.md))
- [ ] Run `pose check --strict` and inspect readiness. ([reference](https://github.com/Fission-AI/OpenSpec))

## 5. Decisions

- Create an ADR before changing this contract; compare [GitHub Spec Kit](https://github.com/github/spec-kit/blob/main/docs/reference/overview.md).

## 6. Validation

**Strategy:** validate units, negative/security cases, contract fixtures and an end-to-end path.

### Planned deterministic checks
- Test: `go test ./pose-mcp/internal/cli/... -run 'Import|Install|Preserve'`.
- Structure: `pose check --strict`.
- Readiness: `pose lint-spec pose-brownfield-reference-kits --ready-check`.

### Execution status
- Not executed. This planning spec remains `draft`; delivery requires gate evidence.

## 7. Final Report

- Delivered scope: none; this spec defines future implementation.
- Residual risk: Idealized examples conceal real migration costs.
- Follow-ups: none until implementation starts.
