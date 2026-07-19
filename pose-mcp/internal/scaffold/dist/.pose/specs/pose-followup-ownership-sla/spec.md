---
slug: pose-followup-ownership-sla
status: draft
created_at: 2026-07-18
completed_at:
supersedes:
depends_on: pose-standalone-dogfood
priority: 12
---

# Spec: Follow-up ownership and service levels

## 1. Intent

### Goal
give every open follow-up an owner, urgency, review date and escalation policy.
### Business value
Stops residual work from becoming a permanent unowned text backlog.
### Constraints
- Use SLAs as triage promises, not unconditional implementation deadlines.
### Non-goals
- Build a full issue tracker or workforce scheduler.

## 2. Requirements

### Functional
- R1: Every open follow-up shall declare owner, criticality and next-review date.
- R2: Overdue follow-ups shall be queryable and optionally blocking by policy.
- R3: Disposition changes shall preserve actor, rationale and target validation.

### Non-functional
- Keep ownership portable across local aliases and external mappings.

### Security
- Avoid personal contact data and restrict sensitive content from broad MCP reads.

### Compatibility
- Legacy follow-ups migrate to explicit `unowned` with a remediation window.

## 3. Technical Plan

### Affected areas
- Follow-up syntax, parser/linter, indexes, CLI/MCP and knowledge policy.

### API/contract changes
- Extend follow-ups without weakening closeout dispositions.

### Data/storage changes
- Persist ownership and review metadata in structured records.

### Technical risks
- Blocking every overdue item can freeze delivery; policy must be risk-based.

### Primary references
- [Backstage ownership model](https://backstage.io/docs/features/software-catalog/descriptor-format/#specowner-required)
- [DORA metrics](https://dora.dev/guides/dora-metrics/)

## 4. Tasks

### Planning
- [ ] Confirm baseline and fixtures against [Backstage ownership model](https://backstage.io/docs/features/software-catalog/descriptor-format/#specowner-required).

### Implementation
- [ ] Define owner, criticality, review and escalation fields. ([reference](https://backstage.io/docs/features/software-catalog/descriptor-format/#specowner-required))
- [ ] Add overdue and ownership projections to follow-up aggregation. ([reference](https://dora.dev/guides/dora-metrics/))
- [ ] Test migrations, missing owners, expired reviews and restricted visibility. ([reference](https://backstage.io/docs/features/software-catalog/descriptor-format/#specowner-required))

### Validation
- [ ] Run `go test ./pose-mcp/... -run 'Followup|Owner|Overdue'` and retain the result artifact. ([reference](https://backstage.io/docs/features/software-catalog/descriptor-format/#specowner-required))
- [ ] Run `pose check --strict` and inspect readiness. ([reference](https://dora.dev/guides/dora-metrics/))

## 5. Decisions

- Create an ADR before changing this contract; compare alternatives against [Backstage ownership model](https://backstage.io/docs/features/software-catalog/descriptor-format/#specowner-required).

## 6. Validation

**Strategy:** validate unit behavior, negative/security cases, contract fixtures and an end-to-end consumer path.

### Planned deterministic checks
- Test: `go test ./pose-mcp/... -run 'Followup|Owner|Overdue'`.
- Structure: `pose check --strict`.
- Readiness: `pose lint-spec pose-followup-ownership-sla --ready-check`.

### Execution status
- Not executed. This planning spec remains `draft`; delivery requires gate evidence.

## 7. Final Report

- Delivered scope: none; this spec defines future implementation.
- Residual risk: Blocking every overdue item can freeze delivery; policy must be risk-based.
- Follow-ups: none until implementation starts.
