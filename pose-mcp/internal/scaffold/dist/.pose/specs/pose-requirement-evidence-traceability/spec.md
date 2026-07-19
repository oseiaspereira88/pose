---
slug: pose-requirement-evidence-traceability
status: draft
created_at: 2026-07-18
completed_at:
supersedes:
depends_on: pose-standalone-dogfood
priority: 10
---

# Spec: Requirement-to-evidence traceability

## 1. Intent

### Goal
link stable requirement IDs to checks, results, commits and approval evidence.
### Business value
Makes the closeout gate explain why each promised behavior was accepted.
### Constraints
- Keep links explicit and reviewable; never infer compliance from file proximity.
### Non-goals
- Replace test frameworks or require one issue tracker.

## 2. Requirements

### Functional
- R1: Each active requirement shall map to declared verification cases with stable IDs.
- R2: Closeout shall identify satisfied, withdrawn or explicitly waived requirements.
- R3: Reports and MCP shall expose bidirectional requirement-to-result traversal.

### Non-functional
- Keep the trace schema diff-friendly and valid offline.

### Security
- Minimize actor identity and avoid confidential test output.

### Compatibility
- Existing specs remain readable through an additive migration.

## 3. Technical Plan

### Affected areas
- Spec contract, linting, reports/history, indexes and MCP.

### API/contract changes
- Add stable verification-link fields and closeout rules.

### Data/storage changes
- Add append-only trace records or versioned spec fields.

### Technical risks
- Mechanical link coverage can be mistaken for evidence quality.

### Primary references
- [OpenTelemetry signals](https://opentelemetry.io/docs/concepts/signals/)
- [SLSA 1.2](https://slsa.dev/spec/v1.2/)

## 4. Tasks

### Planning
- [ ] Confirm baseline and fixtures against [OpenTelemetry signals](https://opentelemetry.io/docs/concepts/signals/).

### Implementation
- [ ] Design stable requirement, verification-case and evidence identifiers. ([reference](https://opentelemetry.io/docs/concepts/signals/))
- [ ] Extend lint, report and index paths with bidirectional validation. ([reference](https://slsa.dev/spec/v1.2/))
- [ ] Add fixtures for satisfied, waived, stale and orphaned evidence. ([reference](https://opentelemetry.io/docs/concepts/signals/))

### Validation
- [ ] Run `go test ./pose-mcp/... -run 'Requirement|Evidence|Trace'` and retain the result artifact. ([reference](https://opentelemetry.io/docs/concepts/signals/))
- [ ] Run `pose check --strict` and inspect readiness. ([reference](https://slsa.dev/spec/v1.2/))

## 5. Decisions

- Create an ADR before changing this contract; compare alternatives against [OpenTelemetry signals](https://opentelemetry.io/docs/concepts/signals/).

## 6. Validation

**Strategy:** validate unit behavior, negative/security cases, contract fixtures and an end-to-end consumer path.

### Planned deterministic checks
- Test: `go test ./pose-mcp/... -run 'Requirement|Evidence|Trace'`.
- Structure: `pose check --strict`.
- Readiness: `pose lint-spec pose-requirement-evidence-traceability --ready-check`.

### Execution status
- Not executed. This planning spec remains `draft`; delivery requires gate evidence.

## 7. Final Report

- Delivered scope: none; this spec defines future implementation.
- Residual risk: Mechanical link coverage can be mistaken for evidence quality.
- Follow-ups: none until implementation starts.

