---
slug: pose-package-manager-distribution
status: draft
created_at: 2026-07-18
completed_at:
supersedes:
depends_on: pose-public-install-contract, pose-release-signing, pose-release-compatibility-matrix
priority: 25
---

# Spec: Supported package-manager distribution

## 1. Intent

### Goal
publish authenticated releases through maintained macOS, Windows and Linux-friendly channels.
### Business value
Removes manual binary placement from mainstream freemium onboarding.
### Constraints
- Every channel consumes the same signed artifacts and documents rollback.
### Non-goals
- Maintain unofficial channels without a service level.

## 2. Requirements

### Functional
- R1: Homebrew and at least one Windows channel shall install the authenticated release.
- R2: Metadata shall update only after release verification passes.
- R3: A clean-host matrix shall install, run doctor/check and uninstall each package.

### Non-functional
- Measure publication lag and expose support status.

### Security
- Pin digests and use least-privilege publisher credentials.

### Compatibility
- Channel versions shall follow release and schema policy.

## 3. Technical Plan

### Affected areas
- Release automation, package manifests, docs and clean-host tests.

### API/contract changes
- Publish support tiers, update latency and deprecation per channel.

### Data/storage changes
- Retain generated manifests and publication results.

### Technical risks
- Compromised channel credentials can redirect trusted package names.

### Primary references
- [Homebrew Formula Cookbook](https://docs.brew.sh/Formula-Cookbook)
- [WinGet package manifests](https://learn.microsoft.com/en-us/windows/package-manager/package/manifest)

## 4. Tasks

### Planning
- [ ] Confirm baseline and fixtures against [Homebrew Formula Cookbook](https://docs.brew.sh/Formula-Cookbook).

### Implementation
- [ ] Select channels and ownership/service-level policy. ([reference](https://docs.brew.sh/Formula-Cookbook))
- [ ] Generate manifests from verified release metadata. ([reference](https://learn.microsoft.com/en-us/windows/package-manager/package/manifest))
- [ ] Run clean-host install, doctor, check, upgrade and uninstall tests. ([reference](https://docs.brew.sh/Formula-Cookbook))

### Validation
- [ ] Run `pose check --strict` and retain evidence. ([reference](https://docs.brew.sh/Formula-Cookbook))
- [ ] Run `pose check --strict` and inspect readiness. ([reference](https://learn.microsoft.com/en-us/windows/package-manager/package/manifest))

## 5. Decisions

- Create an ADR before changing this contract; compare [Homebrew Formula Cookbook](https://docs.brew.sh/Formula-Cookbook).

## 6. Validation

**Strategy:** validate units, negative/security cases, contract fixtures and an end-to-end path.

### Planned deterministic checks
- Test: `pose check --strict`.
- Structure: `pose check --strict`.
- Readiness: `pose lint-spec pose-package-manager-distribution --ready-check`.

### Execution status
- Not executed. This planning spec remains `draft`; delivery requires gate evidence.

## 7. Final Report

- Delivered scope: none; this spec defines future implementation.
- Residual risk: Compromised channel credentials can redirect trusted package names.
- Follow-ups: none until implementation starts.
