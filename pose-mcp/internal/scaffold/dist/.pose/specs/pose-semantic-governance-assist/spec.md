---
slug: pose-semantic-governance-assist
status: draft
created_at: 2026-07-18
completed_at:
supersedes:
depends_on: pose-recurrence-effectiveness, pose-knowledge-consumption-traceability
priority: 32
---

# Spec: Human-reviewed semantic governance assist

## 1. Intent

### Goal
suggest related follow-ups, recurrence patterns and knowledge with explainable evidence.
### Business value
Adds semantic leverage while preserving deterministic authority.
### Constraints
- Suggestions are advisory and never mutate lifecycle automatically.
### Non-goals
- Make an LLM verdict a blocking check.

## 2. Requirements

### Functional
- R1: Each suggestion shall cite artifacts, score/rationale and provider metadata.
- R2: Sensitivity and project boundaries shall be enforced before retrieval.
- R3: Accepted/rejected suggestions shall feed evaluation without training on restricted content.

### Non-functional
- Provide lexical fallback and bounded latency/cost.

### Security
- Require approved providers, prompt-injection defenses and data policy.

### Compatibility
- Core closeout and recurrence work with semantic assist disabled.

## 3. Technical Plan

### Affected areas
- Follow-up/knowledge adapters, MCP, policy, evaluation and Harne8 UI.

### API/contract changes
- Define suggestion, confirmation and provenance schemas.

### Data/storage changes
- Store minimized decision feedback with retention labels.

### Technical risks
- Similarity can conflate related but non-equivalent obligations.

### Primary references
- [NIST AI RMF](https://www.nist.gov/itl/ai-risk-management-framework)
- [MCP security best practices](https://modelcontextprotocol.io/specification/2025-06-18/basic/security_best_practices)

## 4. Tasks

### Planning
- [ ] Confirm baseline and fixtures against [NIST AI RMF](https://www.nist.gov/itl/ai-risk-management-framework).

### Implementation
- [ ] Threat-model retrieval, injection and confirmation paths. ([reference](https://www.nist.gov/itl/ai-risk-management-framework))
- [ ] Implement provider-neutral cited suggestions with fallback. ([reference](https://modelcontextprotocol.io/specification/2025-06-18/basic/security_best_practices))
- [ ] Measure precision, rejection and unsafe-leakage on labeled fixtures. ([reference](https://www.nist.gov/itl/ai-risk-management-framework))

### Validation
- [ ] Run `go test ./pose-mcp/... -run 'Semantic|Followup|Knowledge|Policy'` and retain evidence. ([reference](https://www.nist.gov/itl/ai-risk-management-framework))
- [ ] Run `pose check --strict` and inspect readiness. ([reference](https://modelcontextprotocol.io/specification/2025-06-18/basic/security_best_practices))

## 5. Decisions

- Create an ADR before changing this contract; compare [NIST AI RMF](https://www.nist.gov/itl/ai-risk-management-framework).

## 6. Validation

**Strategy:** validate units, negative/security cases, contract fixtures and an end-to-end path.

### Planned deterministic checks
- Test: `go test ./pose-mcp/... -run 'Semantic|Followup|Knowledge|Policy'`.
- Structure: `pose check --strict`.
- Readiness: `pose lint-spec pose-semantic-governance-assist --ready-check`.

### Execution status
- Not executed. This planning spec remains `draft`; delivery requires gate evidence.

## 7. Final Report

- Delivered scope: none; this spec defines future implementation.
- Residual risk: Similarity can conflate related but non-equivalent obligations.
- Follow-ups: none until implementation starts.

