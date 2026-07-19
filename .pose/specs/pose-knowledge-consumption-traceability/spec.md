---
slug: pose-knowledge-consumption-traceability
status: draft
created_at: 2026-07-18
completed_at:
supersedes:
depends_on: pose-requirement-evidence-traceability
priority: 14
---

# Spec: Knowledge consumption and semantic assist

## 1. Intent

### Goal
show when governed knowledge influences work and offer optional human-reviewed semantic retrieval.
### Business value
Makes expiring memory useful without turning POSE into an opaque vector store.
### Constraints
- Keep deterministic metadata authoritative and semantic ranking advisory.
### Non-goals
- Automatically apply restricted knowledge or make model output a gate.

## 2. Requirements

### Functional
- R1: Specs and reports shall cite consumed knowledge by stable reference.
- R2: Usage signals shall inform review without extending TTL automatically.
- R3: Semantic suggestions shall expose rationale, filter sensitivity and require confirmation.

### Non-functional
- Operate fully without a semantic backend.

### Security
- Enforce sensitivity before retrieval and require approved providers.

### Compatibility
- Existing knowledge remains valid with additive usage metadata.

## 3. Technical Plan

### Affected areas
- Knowledge schema/checks, spec links, MCP and optional adapter.

### API/contract changes
- Define deterministic consumption references and advisory responses.

### Data/storage changes
- Add minimized usage events with artifact ID, purpose and timestamp.

### Technical risks
- Popularity can preserve stale knowledge unless owners still review it.

### Primary references
- [NIST AI RMF](https://www.nist.gov/itl/ai-risk-management-framework)
- [MCP security best practices](https://modelcontextprotocol.io/specification/2025-06-18/basic/security_best_practices)

## 4. Tasks

### Planning
- [ ] Confirm baseline and fixtures against [NIST AI RMF](https://www.nist.gov/itl/ai-risk-management-framework).

### Implementation
- [ ] Define consumption events and sensitivity-safe retrieval policy. ([reference](https://www.nist.gov/itl/ai-risk-management-framework))
- [ ] Add reference validation and usage projections. ([reference](https://modelcontextprotocol.io/specification/2025-06-18/basic/security_best_practices))
- [ ] Prototype explainable semantic ranking with confirmation tests. ([reference](https://www.nist.gov/itl/ai-risk-management-framework))

### Validation
- [ ] Run `go test ./pose-mcp/... -run 'Knowledge|Sensitivity|Usage'` and retain the result artifact. ([reference](https://www.nist.gov/itl/ai-risk-management-framework))
- [ ] Run `pose check --strict` and inspect readiness. ([reference](https://modelcontextprotocol.io/specification/2025-06-18/basic/security_best_practices))

## 5. Decisions

- Create an ADR before changing this contract; compare alternatives against [NIST AI RMF](https://www.nist.gov/itl/ai-risk-management-framework).

## 6. Validation

**Strategy:** validate unit behavior, negative/security cases, contract fixtures and an end-to-end consumer path.

### Planned deterministic checks
- Test: `go test ./pose-mcp/... -run 'Knowledge|Sensitivity|Usage'`.
- Structure: `pose check --strict`.
- Readiness: `pose lint-spec pose-knowledge-consumption-traceability --ready-check`.

### Execution status
- Not executed. This planning spec remains `draft`; delivery requires gate evidence.

## 7. Final Report

- Delivered scope: none; this spec defines future implementation.
- Residual risk: Popularity can preserve stale knowledge unless owners still review it.
- Follow-ups: none until implementation starts.
