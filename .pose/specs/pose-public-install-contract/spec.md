---
slug: pose-public-install-contract
status: draft
created_at: 2026-07-18
completed_at:
supersedes:
depends_on: pose-version-contract
priority: 3
---

# Spec: Real public install contract

## 1. Intent

### Goal
publish a stable, copyable and verified download/install path with no owner or repository placeholders.
### Business value
Removes the highest-friction break in the freemium conversion path.
### Constraints
- Support current Linux, macOS and Windows artifacts and verify integrity before execution.
### Non-goals
- Deliver every package-manager channel.

## 2. Requirements

### Functional
- R1: Documentation shall resolve a released asset for every supported OS and architecture.
- R2: The install flow shall verify integrity before placing the binary on `PATH`.
- R3: A clean-host test shall install, initialize and run `pose doctor --json` plus `pose check --strict`.

### Non-functional
- Keep bootstrap small, auditable and account-free.

### Security
- Never recommend piping unverified network content to a privileged shell.

### Compatibility
- Document supported shells, operating systems and minimum dependencies.

## 3. Technical Plan

### Affected areas
- README, quickstart, release assets, installer E2E and `pose-action` examples.

### API/contract changes
- The README quickstart becomes a tested public entry contract.

### Data/storage changes
- No repository data changes beyond ordinary `pose install`.

### Technical risks
- Release asset naming drift can silently break copyable commands.

### Primary references
- [GitHub Actions hardening](https://docs.github.com/en/actions/security-for-github-actions/security-guides/security-hardening-for-github-actions)
- [Sigstore documentation](https://docs.sigstore.dev/)

## 4. Tasks

### Planning
- [ ] Confirm baseline and fixtures against [GitHub Actions hardening](https://docs.github.com/en/actions/security-for-github-actions/security-guides/security-hardening-for-github-actions).

### Implementation
- [ ] Specify asset resolution and verification per platform. ([reference](https://docs.github.com/en/actions/security-for-github-actions/security-guides/security-hardening-for-github-actions))
- [ ] Replace all action and repository placeholders with released coordinates. ([reference](https://docs.sigstore.dev/))
- [ ] Add clean-container and clean-VM smoke tests for the published quickstart. ([reference](https://docs.github.com/en/actions/security-for-github-actions/security-guides/security-hardening-for-github-actions))

### Validation
- [ ] Run `go test ./pose-mcp/internal/cli/... -run 'Install|Doctor'` and retain the result artifact. ([reference](https://docs.github.com/en/actions/security-for-github-actions/security-guides/security-hardening-for-github-actions))
- [ ] Run `pose check --strict` and inspect readiness projections. ([reference](https://docs.sigstore.dev/))

## 5. Decisions

- Create an ADR before changing this public or structural contract; compare alternatives against [GitHub Actions hardening](https://docs.github.com/en/actions/security-for-github-actions/security-guides/security-hardening-for-github-actions).

## 6. Validation

**Strategy:** validate unit behavior, negative/security cases, public contract fixtures and an end-to-end consumer path.

### Planned deterministic checks
- Test: `go test ./pose-mcp/internal/cli/... -run 'Install|Doctor'`.
- Structure: `pose check --strict`.
- Readiness: `pose lint-spec pose-public-install-contract --ready-check`.

### Execution status
- Not executed. This planning spec remains `draft`; delivery requires recorded gate evidence.

## 7. Final Report

- Delivered scope: none; this spec defines future implementation.
- Residual risk: Release asset naming drift can silently break copyable commands.
- Follow-ups: none until implementation starts.
