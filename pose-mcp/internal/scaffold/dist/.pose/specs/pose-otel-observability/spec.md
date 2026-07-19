---
slug: pose-otel-observability
status: draft
created_at: 2026-07-18
completed_at:
supersedes:
depends_on: pose-structured-validation-results, pose-mcp-protocol-completeness
priority: 30
---

# Spec: OpenTelemetry operational signals

## 1. Intent

### Goal
emit privacy-bounded traces, metrics and logs for server and validation-plan operation.
### Business value
Makes reliability observable without coupling users to one backend.
### Constraints
- Telemetry remains opt-in and POSE continues offline.
### Non-goals
- Export source content, repo names or command output by default.

## 2. Requirements

### Functional
- R1: MCP and orchestration shall emit correlated spans with low-cardinality attributes.
- R2: Metrics shall cover latency, outcome, policy denial and saturation without user IDs.
- R3: Logs shall share trace context and redact paths, tokens and payloads.

### Non-functional
- Use OTLP exporters and bound buffering/backpressure.

### Security
- Document classification, redaction, sampling and endpoint trust.

### Compatibility
- No exporter configuration means no network transmission.

## 3. Technical Plan

### Affected areas
- MCP middleware, bootstrap, orchestration, telemetry config and docs.

### API/contract changes
- Define semantic conventions, opt-in config and failure behavior.

### Data/storage changes
- Emit externally; retain no new local personal telemetry.

### Technical risks
- High-cardinality labels or payload capture can leak structure and raise cost.

### Primary references
- [OpenTelemetry signals](https://opentelemetry.io/docs/concepts/signals/)
- [OpenTelemetry semantic conventions](https://opentelemetry.io/docs/specs/semconv/)

## 4. Tasks

### Planning
- [ ] Confirm baseline and fixtures against [OpenTelemetry signals](https://opentelemetry.io/docs/concepts/signals/).

### Implementation
- [ ] Define POSE span, metric and log conventions with privacy review. ([reference](https://opentelemetry.io/docs/concepts/signals/))
- [ ] Instrument MCP, policy and validation-plan lifecycles. ([reference](https://opentelemetry.io/docs/specs/semconv/))
- [ ] Test disabled mode, redaction, exporter failure, sampling and correlation. ([reference](https://opentelemetry.io/docs/concepts/signals/))

### Validation
- [ ] Run `go test ./pose-mcp/... -run 'Telemetry|Trace|Metric|Redact'` and retain evidence. ([reference](https://opentelemetry.io/docs/concepts/signals/))
- [ ] Run `pose check --strict` and inspect readiness. ([reference](https://opentelemetry.io/docs/specs/semconv/))

## 5. Decisions

- Create an ADR before changing this contract; compare [OpenTelemetry signals](https://opentelemetry.io/docs/concepts/signals/).

## 6. Validation

**Strategy:** validate units, negative/security cases, contract fixtures and an end-to-end path.

### Planned deterministic checks
- Test: `go test ./pose-mcp/... -run 'Telemetry|Trace|Metric|Redact'`.
- Structure: `pose check --strict`.
- Readiness: `pose lint-spec pose-otel-observability --ready-check`.

### Execution status
- Not executed. This planning spec remains `draft`; delivery requires gate evidence.

## 7. Final Report

- Delivered scope: none; this spec defines future implementation.
- Residual risk: High-cardinality labels or payload capture can leak structure and raise cost.
- Follow-ups: none until implementation starts.

