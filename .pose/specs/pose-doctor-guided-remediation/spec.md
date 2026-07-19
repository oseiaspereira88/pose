---
slug: pose-doctor-guided-remediation
status: draft
created_at: 2026-07-18
completed_at:
supersedes:
depends_on: pose-public-install-contract
priority: 27
---

# Spec: Doctor-guided remediation

## 1. Intent

### Goal
turn diagnosable failures into safe actionable remediation.
### Business value
Reduces first-run abandonment and support load without hiding failures.
### Constraints
- Default to advice or dry-run and require explicit apply for mutation.
### Non-goals
- Edit arbitrary files or install system dependencies silently.

## 2. Requirements

### Functional
- R1: Every finding shall have stable code, severity, evidence and remediation.
- R2: Machine output shall distinguish detectable, fixable and externally blocked.
- R3: Safe fixes shall support preview, confirmation, idempotency and recheck.

### Non-functional
- Keep diagnosis fast, offline and platform-aware.

### Security
- Never print secrets, elevate privileges or bypass TLS.

### Compatibility
- Preserve JSON fields or version the doctor schema.

## 3. Technical Plan

### Affected areas
- Doctor, CLI UX, docs anchors, installer and fixtures.

### API/contract changes
- Define diagnostic codes and opt-in fix action schema.

### Data/storage changes
- Local remediation logs exclude sensitive values.

### Technical risks
- Overconfident fixes can damage custom setups.

### Primary references
- [Diátaxis](https://diataxis.fr/)
- [JSON Schema](https://json-schema.org/specification)

## 4. Tasks

### Planning
- [ ] Confirm baseline and fixtures against [Diátaxis](https://diataxis.fr/).

### Implementation
- [ ] Inventory failure modes and define stable remediation codes. ([reference](https://diataxis.fr/))
- [ ] Implement dry-run fixes for confined reversible conditions. ([reference](https://json-schema.org/specification))
- [ ] Test clean, degraded, blocked and secret-redaction scenarios. ([reference](https://diataxis.fr/))

### Validation
- [ ] Run `go test ./pose-mcp/internal/cli/... -run 'Doctor|Remediation|Redact'` and retain evidence. ([reference](https://diataxis.fr/))
- [ ] Run `pose check --strict` and inspect readiness. ([reference](https://json-schema.org/specification))

## 5. Decisions

- Create an ADR before changing this contract; compare [Diátaxis](https://diataxis.fr/).

## 6. Validation

**Strategy:** validate units, negative/security cases, contract fixtures and an end-to-end path.

### Planned deterministic checks
- Test: `go test ./pose-mcp/internal/cli/... -run 'Doctor|Remediation|Redact'`.
- Structure: `pose check --strict`.
- Readiness: `pose lint-spec pose-doctor-guided-remediation --ready-check`.

### Execution status
- Not executed. This planning spec remains `draft`; delivery requires gate evidence.

## 7. Final Report

- Delivered scope: none; this spec defines future implementation.
- Residual risk: Overconfident fixes can damage custom setups.
- Follow-ups: none until implementation starts.
