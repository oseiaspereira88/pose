---
slug: pose-dora-adoption-metrics
status: done
created_at: 2026-07-18
completed_at: 2026-07-19
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

- ADR `.pose/adr/2026-07-19-dora-adoption-metrics-explicit-events-and-unavailable-state.md` (Accepted): explicit event ingestion only (never inferred from commits), an identity-free schema enforced by a reflection test, a three-state metric result (`value`/`unavailable`+reason) evaluated per-metric against its own real denominator, a documented proxy definition for the Reliability metric, and adoption metrics derived entirely from data POSE already owns rather than a second ingestion path. Rejected: inference from git/CI (violates the Non-goal); defaulting missing metrics to zero (violates Compatibility, actively misleading).

## 6. Validation

**Strategy:** validate event-ingestion quality gates (required fields, valid enums, timestamp ordering), append-only monthly storage, all five DORA metrics against a synthetic history with known expected values, the unavailable-not-zero state with no data, application-scoped isolation, the identity-free schema (structural, via reflection), all four adoption views against synthetic specs/history, and retention/deletion housekeeping.

### Planned deterministic checks
- Test: `go -C pose-mcp test ./internal/cli/... -run 'DORA|Adoption|Deployment|Incident|EventsHousekeeping|Identity' -v -count=1`.
- Structure: `pose check --strict`.
- Readiness: `pose lint-spec pose-dora-adoption-metrics --ready-check`.

### Requirement trace
- R1 [satisfied] `record-deployment`/`record-incident` require application/environment-or-timestamps/status-or-severity and a `source` (quality metadata), reject invalid enums, malformed timestamps and resolved-before-started orderings; check:test (TestRecordDeploymentValidation, TestRecordIncidentValidation, TestEventsAreAppendOnlyMonthlyJSONL)
- R2 [satisfied] all five DORA metrics computed only from valid per-metric denominators, `unavailable`+reason with no fabricated zero, application-scoped isolation; check:test (TestDORAMetricsUnavailableWithNoData, TestDORAMetricsComputesFromSyntheticHistory, TestDORAMetricsApplicationFilterIsolatesData)
- R3 [satisfied] activation, time-to-first-gate, retention and task success computed from existing spec/history data with an explicit unavailable-before-activation state; check:test (TestAdoptionMetricsUnavailableBeforeActivation, TestAdoptionMetricsComputesActivationTimeToGateRetentionTaskSuccess)

### Execution status
Executed on 2026-07-19 with a development build (`pose 0.9.0-dev`):

- `go -C pose-mcp test ./internal/cli/... -run 'DORA|Adoption|Deployment|Incident|EventsHousekeeping|Identity' -v -count=1` — SUCCESS (13 tests).
- `go -C pose-mcp test ./... -count=1` — SUCCESS after `go -C pose-mcp generate ./internal/scaffold`.
- `pose check --strict` — SUCCESS.
- `pose lint-spec pose-dora-adoption-metrics --strict` — SUCCESS.
- `pose validate --strict --module pose-mcp --report` — SUCCESS (report retained under `.pose/reports/`).
- Constraint (DORA metrics are team/application outcomes, never individual scores): `TestNoDORAOrAdoptionTypeExposesIndividualIdentity` reflects over every event/report struct's JSON tags and fails on any identity-shaped field name.
- Compatibility (without external data, report unavailable, never zero): `TestDORAMetricsUnavailableWithNoData` and `TestAdoptionMetricsUnavailableBeforeActivation` both assert the explicit unavailable state, not a default zero.
- Security (minimize identities; deletion/retention/aggregation): schema has no identity field beyond application/source; `pose events-housekeeping list-expired|purge [--apply]` (TestEventsHousekeepingListAndPurge) provides retention/deletion; `dora-metrics`/`adoption-metrics` never emit per-event rows (structural aggregation).

## 7. Final Report

- Delivered scope: `pose record-deployment`/`record-incident` (explicit, quality-gated event ingestion), `pose dora-metrics` (all five current DORA metrics with a three-state result and a documented Reliability proxy), `pose adoption-metrics` (activation/time-to-first-gate/retention/task-success derived from existing spec and history data), `pose events-housekeeping` (retention/deletion).
- Residual risk: correlation between adoption and delivery outcomes may still be misrepresented as causation by a reader of the two reports side by side — mitigated by keeping the two metric families in genuinely separate commands/reports (never a single blended "adoption caused this DORA number" output) and by this ADR documenting the Reliability proxy's limits explicitly; ingestion remains manual/CI-driven with no automatic collector, which is a deliberate scope boundary, not a gap.
- Follow-ups: none — both requirement families are satisfied with executed evidence and no sandbox-unavailable gap (all computation is local, no network or external infrastructure needed to test end to end).
