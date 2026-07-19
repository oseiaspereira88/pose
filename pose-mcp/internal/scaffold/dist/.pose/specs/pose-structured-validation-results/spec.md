---
slug: pose-structured-validation-results
status: draft
created_at: 2026-07-18
completed_at:
supersedes:
depends_on: pose-standalone-dogfood
priority: 15
---

# Spec: Structured validation result contract

## 1. Intent

### Goal
emit stable JSON plus interoperable JUnit and SARIF projections from one result model.
### Business value
Unlocks CI annotations, MCP, traceability, analytics and Harness.
### Constraints
- Preserve human-readable logs and deterministic outcomes.
### Non-goals
- Translate arbitrary tool output perfectly or replace native reporters.

## 2. Requirements

### Functional
- R1: Every check result shall include stable ID, command metadata, timing, severity, outcome and skip reason.
- R2: The CLI shall emit versioned JSON and optional JUnit/SARIF projections.
- R3: Partial, tolerated and infrastructure failures shall remain distinguishable.

### Non-functional
- Keep output ordering stable and bound captured output.

### Security
- Redact configured secrets and omit inherited environment values.

### Compatibility
- Text output remains usable while machine formats are additive.

## 3. Technical Plan

### Affected areas
- Validation domain, CLI, reports/history, MCP and CI action.

### API/contract changes
- Define a versioned result schema and output conventions.

### Data/storage changes
- Persist schema version and evidence references.

### Technical risks
- Lossy projections may collapse POSE outcomes unless extensions are documented.

### Primary references
- [JSON Schema](https://json-schema.org/specification)
- [SARIF 2.1.0](https://docs.oasis-open.org/sarif/sarif/v2.1.0/sarif-v2.1.0.html)

## 4. Tasks

### Planning
- [ ] Confirm baseline and fixtures against [JSON Schema](https://json-schema.org/specification).

### Implementation
- [ ] Model canonical check, run and aggregate schemas. ([reference](https://json-schema.org/specification))
- [ ] Implement deterministic JSON and documented JUnit/SARIF mappings. ([reference](https://docs.oasis-open.org/sarif/sarif/v2.1.0/sarif-v2.1.0.html))
- [ ] Add golden cases for pass, fail, partial, skip, timeout and redaction. ([reference](https://json-schema.org/specification))

### Validation
- [ ] Run `go test ./pose-mcp/internal/cli/... -run 'Validate|Report|SARIF|JUnit'` and retain the result artifact. ([reference](https://json-schema.org/specification))
- [ ] Run `pose check --strict` and inspect readiness. ([reference](https://docs.oasis-open.org/sarif/sarif/v2.1.0/sarif-v2.1.0.html))

## 5. Decisions

- Create an ADR before changing this contract; compare alternatives against [JSON Schema](https://json-schema.org/specification).

## 6. Validation

**Strategy:** validate unit behavior, negative/security cases, contract fixtures and an end-to-end consumer path.

### Planned deterministic checks
- Test: `go test ./pose-mcp/internal/cli/... -run 'Validate|Report|SARIF|JUnit'`.
- Structure: `pose check --strict`.
- Readiness: `pose lint-spec pose-structured-validation-results --ready-check`.

### Execution status
- Not executed. This planning spec remains `draft`; delivery requires gate evidence.

## 7. Final Report

- Delivered scope: none; this spec defines future implementation.
- Residual risk: Lossy projections may collapse POSE outcomes unless extensions are documented.
- Follow-ups: none until implementation starts.

