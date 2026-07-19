---
slug: pose-otel-observability
status: done
created_at: 2026-07-18
completed_at: 2026-07-19
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

- ADR `.pose/adr/2026-07-19-otel-observability-safe-by-construction-signals.md` (Accepted): safe-by-construction signals — a closed attribute set (tool name + catalog risk class only, ever) instead of capture-then-redact; stable OTel SDK (v1.44.0) for traces/metrics, but a small local trace-correlated structured logger instead of the still-alpha (v0.x) OTel Logs SDK/`otlploghttp`; one instrumentation point (`Server.callToolCtx`) covering both plain MCP tools and validate-orchestration tools uniformly; double opt-in gate (`POSE_OTEL_ENABLED` + `OTEL_EXPORTER_OTLP_ENDPOINT`) mirroring the existing `pose telemetry` trust pattern. Rejected: capture-then-redact (open-ended leak surface); adopting the alpha Logs SDK (stability risk for a governance tool).

## 6. Validation

**Strategy:** validate the opt-in gate (both env vars required, disabled = zero network), redaction (secrets and paths), trace-context correlation in logs, real OTLP export against a local HTTP receiver, graceful handling of an unreachable collector, and the actual tools/call wiring (successful call, policy-denied call, and a bare-struct-literal Server that never explicitly configured observability).

### Planned deterministic checks
- Test: `go -C pose-mcp test ./internal/observability/... ./internal/mcpserver/... -run 'Observability|FromEnv|Init|Secrets|Paths|Logger|Emits|Denied|Noop' -v -count=1`.
- Structure: `pose check --strict`.
- Readiness: `pose lint-spec pose-otel-observability --ready-check`.

### Requirement trace
- R1 [satisfied] every `tools/call` (MCP tools and the five `pose_validate_*` orchestration tools alike) gets one span via the single `callToolCtx` instrumentation point, tagged only with tool name + risk class (both fixed, low-cardinality); check:test (TestSuccessfulToolCallEmitsSpanAndDuration, TestPolicyDeniedCallIncrementsDenialMetric)
- R2 [satisfied] `pose.mcp.tool.call.duration` (latency histogram), `pose.mcp.policy.denial.count` (policy-denial counter), `pose.mcp.tool.call.inflight` (saturation, current concurrency) — none carry a user/principal/run id; check:test (TestInitEnabledExportsSpansAndMetricsToConfiguredEndpoint, TestPolicyDeniedCallIncrementsDenialMetric)
- R3 [satisfied] every log record carries `trace_id`/`span_id` from the active span context and passes the free-text message through path+secret redaction before being written; check:test (TestLoggerCorrelatesTraceContextAndRedacts, TestLoggerWithoutSpanOmitsCorrelationFields, TestSecretsRedaction, TestPathsRedaction)

### Execution status
Executed on 2026-07-19 with a development build (`pose 0.9.0-dev`):

- `go -C pose-mcp test ./internal/observability/... -v -count=1` — SUCCESS (11 tests: opt-in double-gate, config parsing, disabled-mode inertness, nil-Provider safety, secret/path redaction, trace-correlated logging, real OTLP export to a local receiver, graceful handling of an unreachable endpoint).
- `go -C pose-mcp test ./internal/mcpserver/... -v -count=1` — SUCCESS, including 3 new observability-wiring tests and every pre-existing test unmodified (proving the default no-op path is behaviorally identical to before this spec).
- `go -C pose-mcp test ./... -count=1` — SUCCESS after `go -C pose-mcp generate ./internal/scaffold`.
- `go -C pose-mcp vet ./...` — SUCCESS.
- `go -C pose-mcp mod tidy` — clean; new OTel dependencies (`go.opentelemetry.io/otel{,/sdk,/sdk/metric,/metric,/trace}` and the two OTLP/HTTP exporters, all `v1.44.0` stable) promoted from indirect to direct.
- `pose check --strict` — SUCCESS.
- `pose lint-spec pose-otel-observability --strict` — SUCCESS.
- `pose validate --strict --module pose-mcp --report` — SUCCESS (report retained under `.pose/reports/`).
- Non-functional (OTLP exporters, bounded buffering/backpressure): OTLP/HTTP batch span processor + periodic metric reader (both from the stable SDK); `Shutdown` wrapped in a 5s timeout in both `bootstrap.Run` serve paths so a dead collector cannot hang process shutdown; `TestInitEnabledDoesNotBlockOnUnreachableEndpoint` proves this bound holds.
- Security (classification, redaction, sampling, endpoint trust documented): documented in `docs-site/docs/mcp.md#observability` — attribute set is closed (tool + risk class only); `Secrets`/`Paths`/`Message` redaction proven by test; `OTEL_TRACES_SAMPLER_ARG` drives a standard `ParentBased(TraceIDRatioBased(...))` sampler; TLS on by default (`OTEL_EXPORTER_OTLP_INSECURE` must be explicit to disable), `OTEL_EXPORTER_OTLP_HEADERS` for collector auth.
- Compatibility (no exporter configuration means no network transmission): `TestFromEnvRequiresBothGates` and `TestInitDisabledIsInertNoop` prove both that either env var alone leaves `Enabled=false` and that a disabled `Provider` never touches the network.

## 7. Final Report

- Delivered scope: opt-in OpenTelemetry traces + metrics (stable SDK, OTLP/HTTP) and a trace-correlated, redacted structured logger for every MCP `tools/call` (including validate-orchestration), wired through one instrumentation point in `Server.callToolCtx`; a new `internal/observability` package (config, provider lifecycle, redaction, logger); graceful startup/shutdown in `internal/bootstrap.Run` including SIGINT/SIGTERM handling that did not exist before this spec.
- Residual risk: high-cardinality labels or payload capture is structurally prevented (closed attribute set), not just policy — but a future contributor adding a new attribute to the instrumentation point could still reintroduce that risk if they don't follow the same pattern; the ADR documents the constraint explicitly for that reason. Logs are not OTLP-exported (local structured writer only) until the OTel Logs SDK reaches a stable release.
- Follow-ups: see below.

### Follow-ups

- [open] Revisit OTLP log export (`go.opentelemetry.io/otel/sdk/log` + `otlploghttp`) once both reach a stable `v1.x` release — currently alpha (`v0.x`), deliberately not adopted. (owner:@pose-maintainers crit:low review:2026-10-19)

