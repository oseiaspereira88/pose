---
slug: pose-upgrade-compatibility-lab
status: draft
created_at: 2026-07-18
completed_at:
supersedes:
depends_on: pose-release-compatibility-matrix
priority: 26
---

# Spec: Upgrade compatibility lab

## 1. Intent

### Goal
continuously test real repository upgrades across the supported version window.
### Business value
Protects adopter-owned specs and evidence as the product evolves.
### Constraints
- Never mutate original fixtures; prove idempotency and preserve unknown content.
### Non-goals
- Support downgrades or arbitrary invalid instances.

## 2. Requirements

### Functional
- R1: The lab shall test every supported N-minus engine/schema pair.
- R2: Fixtures shall cover locales, user-modified managed files and populated artifacts.
- R3: Each path shall prove dry-run accuracy, idempotency and preservation.

### Non-functional
- Run in isolated copies with deterministic snapshots.

### Security
- Authenticate prior binaries and block path/symlink escapes.

### Compatibility
- Unsupported versions receive explicit remediation, not partial upgrade.

## 3. Technical Plan

### Affected areas
- Upgrade engine, migrations, fixtures, release CI and docs.

### API/contract changes
- Make the support matrix executable and release-blocking.

### Data/storage changes
- Version sanitized fixtures and expected migration plans.

### Technical risks
- Synthetic fixtures can miss real customization patterns.

### Primary references
- [The Update Framework](https://theupdateframework.io/)
- [SLSA 1.2](https://slsa.dev/spec/v1.2/)

## 4. Tasks

### Planning
- [ ] Confirm baseline and fixtures against [The Update Framework](https://theupdateframework.io/).

### Implementation
- [ ] Build populated-instance fixtures for each supported release. ([reference](https://theupdateframework.io/))
- [ ] Exercise dry-run, apply, reapply and preservation assertions. ([reference](https://slsa.dev/spec/v1.2/))
- [ ] Publish candidate results and unsupported-path diagnostics. ([reference](https://theupdateframework.io/))

### Validation
- [ ] Run `go test ./pose-mcp/internal/cli/... -run 'Upgrade|Migration|Preserve'` and retain evidence. ([reference](https://theupdateframework.io/))
- [ ] Run `pose check --strict` and inspect readiness. ([reference](https://slsa.dev/spec/v1.2/))

## 5. Decisions

- Create an ADR before changing this contract; compare [The Update Framework](https://theupdateframework.io/).

## 6. Validation

**Strategy:** validate units, negative/security cases, contract fixtures and an end-to-end path.

### Planned deterministic checks
- Test: `go test ./pose-mcp/internal/cli/... -run 'Upgrade|Migration|Preserve'`.
- Structure: `pose check --strict`.
- Readiness: `pose lint-spec pose-upgrade-compatibility-lab --ready-check`.

### Execution status
- Not executed. This planning spec remains `draft`; delivery requires gate evidence.

## 7. Final Report

- Delivered scope: none; this spec defines future implementation.
- Residual risk: Synthetic fixtures can miss real customization patterns.
- Follow-ups: none until implementation starts.

