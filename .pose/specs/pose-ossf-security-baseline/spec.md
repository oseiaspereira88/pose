---
slug: pose-ossf-security-baseline
status: draft
created_at: 2026-07-18
completed_at:
supersedes:
depends_on: pose-version-contract
priority: 5
---

# Spec: OpenSSF security baseline

## 1. Intent

### Goal
make static analysis, dependency review, secret scanning, permissions and Scorecard visible release gates.
### Business value
Closes preventable supply-chain gaps before signing raises perceived trust.
### Constraints
- Use least-privilege permissions and document justified exceptions.
### Non-goals
- Treat a Scorecard number as a security guarantee.

## 2. Requirements

### Functional
- R1: Pull requests shall run static analysis, dependency review and secret detection.
- R2: Workflow permissions and third-party action pinning shall be checked automatically.
- R3: Unresolved critical findings shall be explicit release-decision inputs.

### Non-functional
- Avoid duplicate scanners with identical coverage.

### Security
- Fail releases on unwaived critical findings; expire every exception.

### Compatibility
- Support fork contributions without exposing write tokens.

## 3. Technical Plan

### Affected areas
- .github workflows, dependency policy, security docs and release gates.

### API/contract changes
- Security status becomes an evidence-backed release input.

### Data/storage changes
- Store owned exception metadata and scanner outputs by policy.

### Technical risks
- Noisy scanners encourage broad waivers unless ownership is enforced.

### Primary references
- [OpenSSF Scorecard](https://scorecard.dev/)
- [GitHub Actions hardening](https://docs.github.com/en/actions/security-for-github-actions/security-guides/security-hardening-for-github-actions)

## 4. Tasks

### Planning
- [ ] Confirm baseline and fixtures against [OpenSSF Scorecard](https://scorecard.dev/).

### Implementation
- [ ] Establish the baseline Scorecard and threat-ranked backlog. ([reference](https://scorecard.dev/))
- [ ] Add SAST, dependency review and secret scanning gates. ([reference](https://docs.github.com/en/actions/security-for-github-actions/security-guides/security-hardening-for-github-actions))
- [ ] Pin permissions/actions and implement owned expiring exceptions. ([reference](https://scorecard.dev/))

### Validation
- [ ] Run `go test ./... && pose check --strict` and retain the result artifact. ([reference](https://scorecard.dev/))
- [ ] Run `pose check --strict` and inspect readiness projections. ([reference](https://docs.github.com/en/actions/security-for-github-actions/security-guides/security-hardening-for-github-actions))

## 5. Decisions

- Create an ADR before changing this public or structural contract; compare alternatives against [OpenSSF Scorecard](https://scorecard.dev/).

## 6. Validation

**Strategy:** validate unit behavior, negative/security cases, public contract fixtures and an end-to-end consumer path.

### Planned deterministic checks
- Test: `go test ./... && pose check --strict`.
- Structure: `pose check --strict`.
- Readiness: `pose lint-spec pose-ossf-security-baseline --ready-check`.

### Execution status
- Not executed. This planning spec remains `draft`; delivery requires recorded gate evidence.

## 7. Final Report

- Delivered scope: none; this spec defines future implementation.
- Residual risk: Noisy scanners encourage broad waivers unless ownership is enforced.
- Follow-ups: none until implementation starts.

