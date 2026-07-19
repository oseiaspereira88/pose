---
slug: pose-agent-skills-conformance
status: draft
created_at: 2026-07-18
completed_at:
supersedes:
depends_on: pose-standalone-dogfood
priority: 23
---

# Spec: Agent Skills conformance and compatibility

## 1. Intent

### Goal
validate every shipped skill against Agent Skills and declared runtime compatibility.
### Business value
Makes agent behavior a tested product surface rather than copied prompt text.
### Constraints
- Preserve local overrides and avoid one-vendor coupling.
### Non-goals
- Guarantee identical behavior across models.

## 2. Requirements

### Functional
- R1: CI shall validate required metadata, layout and linked resources.
- R2: Each skill shall declare POSE schema range, clients and capabilities.
- R3: Compatibility fixtures shall verify discovery and a bounded workflow.

### Non-functional
- Keep structural validation offline and deterministic.

### Security
- Scan instructions/assets for unsafe commands, secrets and path escapes.

### Compatibility
- Version behavior changes and document renamed skills.

## 3. Technical Plan

### Affected areas
- .agents/skills, client links, scaffold, CI and docs.

### API/contract changes
- Add compatibility metadata and a conformance report.

### Data/storage changes
- Maintain machine-readable skill inventory and fixtures.

### Technical risks
- Schema-valid skills can still be semantically unsafe.

### Primary references
- [Agent Skills specification](https://agentskills.io/specification)
- [GitHub Spec Kit](https://github.com/github/spec-kit/blob/main/docs/reference/overview.md)

## 4. Tasks

### Planning
- [ ] Confirm baseline and fixtures against [Agent Skills specification](https://agentskills.io/specification).

### Implementation
- [ ] Define metadata and supported-client policy. ([reference](https://agentskills.io/specification))
- [ ] Add spec validation, link checking and security lint. ([reference](https://github.com/github/spec-kit/blob/main/docs/reference/overview.md))
- [ ] Execute representative client discovery/workflow fixtures. ([reference](https://agentskills.io/specification))

### Validation
- [ ] Run `pose check --strict && go test ./pose-mcp/... -run 'Skill|Scaffold'` and retain evidence. ([reference](https://agentskills.io/specification))
- [ ] Run `pose check --strict` and inspect readiness. ([reference](https://github.com/github/spec-kit/blob/main/docs/reference/overview.md))

## 5. Decisions

- Create an ADR before changing this contract; compare [Agent Skills specification](https://agentskills.io/specification).

## 6. Validation

**Strategy:** validate units, negative/security cases, contract fixtures and an end-to-end path.

### Planned deterministic checks
- Test: `pose check --strict && go test ./pose-mcp/... -run 'Skill|Scaffold'`.
- Structure: `pose check --strict`.
- Readiness: `pose lint-spec pose-agent-skills-conformance --ready-check`.

### Execution status
- Not executed. This planning spec remains `draft`; delivery requires gate evidence.

## 7. Final Report

- Delivered scope: none; this spec defines future implementation.
- Residual risk: Schema-valid skills can still be semantically unsafe.
- Follow-ups: none until implementation starts.

