---
slug: pose-recurrence-effectiveness
status: draft
created_at: 2026-07-18
completed_at:
supersedes:
depends_on: pose-followup-ownership-sla
priority: 13
---

# Spec: Recurrence effectiveness measurement

## 1. Intent

### Goal
measure whether systemic rules, workflows or fixes reduce repeated failures.
### Business value
Closes the feedback edge instead of treating escalation creation as success.
### Constraints
- Use team/process signals and minimum sample sizes.
### Non-goals
- Suppress intermittent failures automatically or rank individuals.

## 2. Requirements

### Functional
- R1: Every escalation shall identify its intervention and observation window.
- R2: The engine shall compare recurrence rate and cost before and after intervention with data-quality warnings.
- R3: An ineffective intervention shall reopen or spawn governed follow-up.

### Non-functional
- Keep calculations reproducible from append-only local history.

### Security
- Aggregate by task/context and exclude personal performance ranking.

### Compatibility
- Missing duration or cost fields produce partial, not fabricated, metrics.

## 3. Technical Plan

### Affected areas
- History, recurrence check, stats/insights, reports and workflow.

### API/contract changes
- Add intervention and evaluation states to recurrence lifecycle.

### Data/storage changes
- Extend history with optional duration/cost and intervention references.

### Technical risks
- Small samples and task-mix changes can mislead.

### Primary references
- [DORA metrics](https://dora.dev/guides/dora-metrics/)
- [OpenTelemetry signals](https://opentelemetry.io/docs/concepts/signals/)

## 4. Tasks

### Planning
- [ ] Confirm baseline and fixtures against [DORA metrics](https://dora.dev/guides/dora-metrics/).

### Implementation
- [ ] Define intervention, window, rate and cost semantics. ([reference](https://dora.dev/guides/dora-metrics/))
- [ ] Add deterministic before/after history projections. ([reference](https://opentelemetry.io/docs/concepts/signals/))
- [ ] Test sparse data, regressions and ineffective interventions. ([reference](https://dora.dev/guides/dora-metrics/))

### Validation
- [ ] Run `go test ./pose-mcp/... -run 'Recurrence|Stats|Insight'` and retain the result artifact. ([reference](https://dora.dev/guides/dora-metrics/))
- [ ] Run `pose check --strict` and inspect readiness. ([reference](https://opentelemetry.io/docs/concepts/signals/))

## 5. Decisions

- Create an ADR before changing this contract; compare alternatives against [DORA metrics](https://dora.dev/guides/dora-metrics/).

## 6. Validation

**Strategy:** validate unit behavior, negative/security cases, contract fixtures and an end-to-end consumer path.

### Planned deterministic checks
- Test: `go test ./pose-mcp/... -run 'Recurrence|Stats|Insight'`.
- Structure: `pose check --strict`.
- Readiness: `pose lint-spec pose-recurrence-effectiveness --ready-check`.

### Execution status
- Not executed. This planning spec remains `draft`; delivery requires gate evidence.

## 7. Final Report

- Delivered scope: none; this spec defines future implementation.
- Residual risk: Small samples and task-mix changes can mislead.
- Follow-ups: none until implementation starts.
