---
slug: pose-recurrence-effectiveness
status: done
created_at: 2026-07-18
completed_at: 2026-07-19
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
- [x] Confirm baseline and fixtures against [DORA metrics](https://dora.dev/guides/dora-metrics/): the escalation workflow asked for a 45-day review with no data source — escalation creation was implicitly success; history JSONL carried no duration/cost; context resumed from knowledge:contract-baseline-handoff.

### Implementation
- [x] Define intervention, window, rate and cost semantics: append-only `interventions.jsonl` (schema 1) with task slug, validated `rule:|workflow:|spec:` ref, window days, rationale, pseudonymous author and RFC3339 timestamp; optional `--duration-seconds`/`--cost-usd` telemetry on `pose report` (ADR `2026-07-19-recurrence-intervention-effectiveness-measurement`). ([reference](https://dora.dev/guides/dora-metrics/))
- [x] Add deterministic before/after history projections: `pose recurrence-effect` compares failures per task slug in the window before vs after each intervention plus average duration/cost when recorded; verdicts EFFECTIVE/INEFFECTIVE/INCONCLUSIVE with first-class data-quality warnings (insufficient sample below `--min-sample`, incomplete observation window); missing telemetry yields explicitly partial metrics. ([reference](https://opentelemetry.io/docs/concepts/signals/))
- [x] Test sparse data, regressions and ineffective interventions: `recurrence_effect_test.go` covers effective drop, ineffective regression with governed-action output and `--fail-ineffective` blocking, combined sparse+incomplete warnings and registration validation; recurrence-escalation workflow updated to register at ship time and review with the command. ([reference](https://dora.dev/guides/dora-metrics/))

### Validation
- [x] Run `go test ./pose-mcp/... -run 'Recurrence|Stats|Insight'` and retain the result artifact (see §6 and `.pose/reports/`). ([reference](https://dora.dev/guides/dora-metrics/))
- [x] Run `pose check --strict` and inspect readiness. ([reference](https://opentelemetry.io/docs/concepts/signals/))

## 5. Decisions

- ADR `.pose/adr/2026-07-19-recurrence-intervention-effectiveness-measurement.md` (Accepted): registered interventions + deterministic history projection over manual memory-based review and over external analytics; warnings force INCONCLUSIVE instead of misleading verdicts; blocking is opt-in; aggregation by task/context only, never individuals.

## 6. Validation

**Strategy:** validate unit behavior, negative/security cases, contract fixtures and an end-to-end consumer path.

### Planned deterministic checks
- Test: `go test ./pose-mcp/... -run 'Recurrence|Stats|Insight'`.
- Structure: `pose check --strict`.
- Readiness: `pose lint-spec pose-recurrence-effectiveness --ready-check`.

### Requirement trace
- R1 [satisfied] every intervention carries ref + observation window, validated at registration; check:test (TestRecurrenceEffectRegisterValidation)
- R2 [satisfied] before/after rate and telemetry comparison with data-quality warnings; check:test (TestRecurrenceEffectEffective, TestRecurrenceEffectWarnings) report:2026-07-19-standard-validate-native.md
- R3 [satisfied] INEFFECTIVE verdict demands governed follow-up, opt-in blocking via --fail-ineffective; check:test (TestRecurrenceEffectIneffectiveBlocksOptIn)

### Execution status
Executed on 2026-07-19 with a development build (`pose 0.9.0-dev`, rebuilt from this change):

- `go -C pose-mcp test ./internal/cli -run 'RecurrenceEffect' -count=1` — SUCCESS (all four behavior tests).
- `go -C pose-mcp test ./... -count=1` — SUCCESS (full suite).
- `pose check --strict` — SUCCESS; `pose lint-spec pose-recurrence-effectiveness --strict` — SUCCESS.
- `pose validate --strict --module pose-mcp --report` — SUCCESS (report retained under `.pose/reports/`).

## 7. Final Report

### Delivered scope

Intervention registry (`interventions.jsonl`, under the history-check gate),
deterministic before/after effectiveness projection with verdicts and
data-quality warnings, optional duration/cost telemetry on reports, opt-in
blocking policy, updated recurrence-escalation workflow (register at ship,
review by measurement), operating-manual documentation and ADR. The loop's
final edge — "did the systemic fix work?" — is now measured, not assumed.

### Residual risks

- Task-mix changes between windows can mislead; warnings surface data
  quality but the keep/adjust/discard decision stays human.
- Cost comparisons stay partial until telemetry adoption spreads.

### Follow-ups

- [open] Register the first real intervention when recurrence-check next flags a task, and review its verdict after the window. (owner:@pose-maintainers crit:medium review:2026-11-06)
- [covered: pose-otel-observability] Export effectiveness signals via OpenTelemetry in the insights roadmap.
