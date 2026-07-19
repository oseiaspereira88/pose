---
slug: pose-dora-adoption-metrics
status: draft
created_at: 2026-07-18
completed_at:
supersedes:
depends_on: pose-requirement-evidence-traceability, pose-otel-observability
priority: 31
---

# Spec: DORA and adoption-value metrics

## 1. Intent

### Goal
correlate governance adoption with team delivery outcomes and product success.
### Business value
Tests whether POSE creates value instead of optimizing artifact volume.
### Constraints
- DORA metrics are team/application outcomes and never individual scores.
### Non-goals
- Infer deployments/incidents from commits alone or create rankings.

## 2. Requirements

### Functional
- R1: Adapters shall ingest explicit deployment and incident events with quality metadata.
- R2: The five current DORA metrics shall calculate only with valid definitions and denominators.
- R3: Adoption views shall include activation, time-to-first-gate, retention and task success.

### Non-functional
- Make windows, filters and missing data transparent.

### Security
- Minimize identities and support deletion, retention and aggregation.

### Compatibility
- Without external data, report `unavailable`, never zero.

## 3. Technical Plan

### Affected areas
- Event adapters, insights, OTel, docs and Harne8 projections.

### API/contract changes
- Define team/application identity, metrics and data-quality states.

### Data/storage changes
- Store normalized delivery/adoption events in the control plane.

### Technical risks
- Correlation may be misrepresented as causation.

### Primary references
- [DORA metrics guide](https://dora.dev/guides/dora-metrics/)
- [OpenTelemetry signals](https://opentelemetry.io/docs/concepts/signals/)

## 4. Tasks

### Planning
- [ ] Confirm baseline and fixtures against [DORA metrics guide](https://dora.dev/guides/dora-metrics/).

### Implementation
- [ ] Define deployment, incident, application and adoption events. ([reference](https://dora.dev/guides/dora-metrics/))
- [ ] Implement quality-aware DORA and adoption calculations. ([reference](https://opentelemetry.io/docs/concepts/signals/))
- [ ] Validate synthetic histories and prohibit individual ranking. ([reference](https://dora.dev/guides/dora-metrics/))

### Validation
- [ ] Run `go test ./pose-mcp/... -run 'Insight|Metric|Event'` and retain evidence. ([reference](https://dora.dev/guides/dora-metrics/))
- [ ] Run `pose check --strict` and inspect readiness. ([reference](https://opentelemetry.io/docs/concepts/signals/))

## 5. Decisions

- Create an ADR before changing this contract; compare [DORA metrics guide](https://dora.dev/guides/dora-metrics/).

## 6. Validation

**Strategy:** validate units, negative/security cases, contract fixtures and an end-to-end path.

### Planned deterministic checks
- Test: `go test ./pose-mcp/... -run 'Insight|Metric|Event'`.
- Structure: `pose check --strict`.
- Readiness: `pose lint-spec pose-dora-adoption-metrics --ready-check`.

### Execution status
- Not executed. This planning spec remains `draft`; delivery requires gate evidence.

## 7. Final Report

- Delivered scope: none; this spec defines future implementation.
- Residual risk: Correlation may be misrepresented as causation.
- Follow-ups: none until implementation starts.

