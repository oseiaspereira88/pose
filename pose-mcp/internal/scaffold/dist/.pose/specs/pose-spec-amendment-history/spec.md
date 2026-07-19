---
slug: pose-spec-amendment-history
status: draft
created_at: 2026-07-18
completed_at:
supersedes:
depends_on: pose-requirement-evidence-traceability
priority: 11
---

# Spec: Spec amendment history

## 1. Intent

### Goal
record material intent and acceptance-criteria changes as append-only reviewed amendments.
### Business value
Prevents a spec from being rewritten after evidence without auditable rationale.
### Constraints
- Keep editorial corrections lightweight while preserving material changes.
### Non-goals
- Record every spelling correction as an amendment.

## 2. Requirements

### Functional
- R1: Material requirement additions, withdrawals or semantic changes shall create an amendment event.
- R2: Each event shall identify affected IDs, rationale, author/reviewer and timestamp.
- R3: Closeout shall reject unacknowledged amendments made after referenced evidence.

### Non-functional
- Keep events merge-friendly and deterministic.

### Security
- Use repository identities or pseudonymous IDs; minimize personal data.

### Compatibility
- Published IDs are never renumbered and withdrawn criteria remain addressable.

## 3. Technical Plan

### Affected areas
- Spec lifecycle, linter, indexes, MCP and changelog guidance.

### API/contract changes
- Define material amendment and approval semantics.

### Data/storage changes
- Store append-only amendment entries with a schema version.

### Technical risks
- Over-sensitive detection can burden harmless editorial work.

### Primary references
- [OpenSpec](https://github.com/Fission-AI/OpenSpec)
- [GitHub Spec Kit](https://github.com/github/spec-kit/blob/main/docs/reference/overview.md)

## 4. Tasks

### Planning
- [ ] Confirm baseline and fixtures against [OpenSpec](https://github.com/Fission-AI/OpenSpec).

### Implementation
- [ ] Define material-change taxonomy and amendment schema. ([reference](https://github.com/Fission-AI/OpenSpec))
- [ ] Detect unrecorded semantic changes to published requirements. ([reference](https://github.com/github/spec-kit/blob/main/docs/reference/overview.md))
- [ ] Render history and test post-evidence mutation cases. ([reference](https://github.com/Fission-AI/OpenSpec))

### Validation
- [ ] Run `go test ./pose-mcp/... -run 'Amendment|Requirement'` and retain the result artifact. ([reference](https://github.com/Fission-AI/OpenSpec))
- [ ] Run `pose check --strict` and inspect readiness. ([reference](https://github.com/github/spec-kit/blob/main/docs/reference/overview.md))

## 5. Decisions

- Create an ADR before changing this contract; compare alternatives against [OpenSpec](https://github.com/Fission-AI/OpenSpec).

## 6. Validation

**Strategy:** validate unit behavior, negative/security cases, contract fixtures and an end-to-end consumer path.

### Planned deterministic checks
- Test: `go test ./pose-mcp/... -run 'Amendment|Requirement'`.
- Structure: `pose check --strict`.
- Readiness: `pose lint-spec pose-spec-amendment-history --ready-check`.

### Execution status
- Not executed. This planning spec remains `draft`; delivery requires gate evidence.

## 7. Final Report

- Delivered scope: none; this spec defines future implementation.
- Residual risk: Over-sensitive detection can burden harmless editorial work.
- Follow-ups: none until implementation starts.

