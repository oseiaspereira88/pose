# ADR: OTel observability — safe-by-construction signals, no alpha dependencies

## Status
Accepted (2026-07-19) — spec `pose-otel-observability`

## Context

`pose serve-mcp` had no operational observability: an operator running it
in production could not see tool-call latency, error rates, policy
denials or concurrency without adding their own instrumentation. The spec
requires OpenTelemetry traces/metrics/logs, but with a hard constraint —
"telemetry remains opt-in and POSE continues offline" — and a hard
non-goal — never export source content, repo names or command output by
default. POSE also already has a completely separate, much simpler
opt-in telemetry mechanism (`pose telemetry enable`, `internal/cli/telemetry.go`
— anonymous adoption-metrics ping, no OTel) that this spec must not be
confused with or collide against.

Alternatives considered:

1. **Capture full request/response payloads and redact after the fact.**
   Simpler to instrument (wrap once, log everything), but redaction is
   then a blocklist racing an open-ended set of fields — exactly the
   Technical risk the spec calls out ("high-cardinality labels or payload
   capture can leak structure"). One missed field is a real leak.
2. **Adopt the OTel Logs SDK/exporter** (`go.opentelemetry.io/otel/sdk/log`,
   `otlploghttp`) for full OTLP log export. More "complete" on paper, but
   both are still pre-1.0 (`v0.x`, alpha API) as of this writing, while
   traces and metrics are stable `v1.44.0` — pinning a governance tool's
   dependency tree to an alpha API for one of three signals is a stability
   risk not worth taking yet.
3. **Safe-by-construction signals**: never capture a payload/argument/path
   in the first place — only ever attach the tool name (30 known values)
   and its catalog risk class (`read`/`gate`/`external-side-effect`, 3
   values) as span/metric/log attributes, with a defense-in-depth
   redaction pass on the one genuinely free-text field (the error
   message). Traces and metrics use the stable OTel SDK; logs are a
   small, local, trace-correlated structured JSON writer instead of the
   alpha Logs SDK.

## Decision

Option 3.

- **One instrumentation point**: `Server.callToolCtx` in
  `internal/mcpserver/server.go` — every `tools/call` (plain MCP tools and
  the five `pose_validate_*` orchestration tools alike, since they all
  flow through the same dispatch) gets one span, one duration
  measurement, one structured log line. "MCP and orchestration shall
  emit correlated spans" (R1) is satisfied by this single, uniform point
  rather than separate instrumentation scattered through
  `validate_orchestration.go` — the span context still propagates into
  `dispatch()` and beyond, so anything downstream that wants a child span
  can create one under the same trace.
- **Attributes are a fixed, closed set**: `tool` (catalog name) and
  `risk_class` (from `catalogGovernance`) on spans/metrics; `outcome`
  (`ok`/`policy_denied`/`error`) added at metric-record time. No argument,
  no project_id, no principal, no run id, no user id ever becomes an
  attribute — R2's "without user IDs" and the payload non-goal are
  structural, not policy.
- **Three metrics** (`internal/observability/provider.go`,
  `Instruments`): `pose.mcp.tool.call.duration` (histogram, ms — latency),
  `pose.mcp.policy.denial.count` (counter — policy denial), `pose.mcp.tool.call.inflight`
  (up-down counter — current concurrency, the saturation signal R2 asks
  for). All three created once per `Provider` and reused, never
  recreated per call.
- **Logs are a local structured writer, not OTLP-exported**: `Logger.Emit`
  (`internal/observability/log.go`) writes one JSON line to stderr per
  call, carrying `trace_id`/`span_id` pulled from the active span context
  (`trace.SpanContextFromContext`) — satisfying "logs shall share trace
  context" without the alpha Logs SDK. `Message()` (redact.go) is applied
  to the one free-text field (the dispatch error, when present) before
  it's ever stored — collapses secret-shaped substrings (reusing the same
  pattern shapes `pose-agent-skills-conformance` already uses, kept as an
  independent copy rather than a cross-package import for a handful of
  regexes) and absolute-filesystem-path substrings to `[PATH]`.
- **Opt-in is a double gate**, mirroring the existing `pose telemetry`
  trust pattern: `POSE_OTEL_ENABLED=1` (POSE's own flag) AND
  `OTEL_EXPORTER_OTLP_ENDPOINT` (the standard OTel env var, no default
  baked in) must both be set, or `Init` returns a fully-inert no-op
  `Provider` — zero allocation beyond the config read, zero network,
  every `Tracer`/`Meter`/`Log` call is a true no-op. `defaultObservability()`
  wires this into `Server.New`/`NewWithRootsAndPolicy` automatically; a
  bare `&Server{}` struct literal (used by several existing tests) falls
  back to a shared no-op instance via `Server.observability()` rather than
  panicking on a nil field.
- **Endpoint trust**: TLS is on by default (`OTEL_EXPORTER_OTLP_INSECURE`
  must be explicitly set to disable it) and `OTEL_EXPORTER_OTLP_HEADERS`
  lets an operator attach a collector auth token — the same trust model
  as any other OTLP/HTTP producer.
- **Sampling**: `OTEL_TRACES_SAMPLER_ARG` (0.0–1.0, default 1.0) drives a
  `ParentBased(TraceIDRatioBased(...))` sampler — standard OTel behavior.
- **Exporter failure never blocks the server**: `Init` fails fast only for
  a genuine misconfiguration (e.g. an unparseable endpoint URL) while
  actually enabled; `internal/bootstrap.Run` logs and falls back to the
  no-op provider rather than aborting startup. An unreachable collector at
  runtime fails individual export attempts silently (proven by
  `TestInitEnabledDoesNotBlockOnUnreachableEndpoint`); `Shutdown` is
  wrapped in a 5s timeout in both the stdio and HTTP serve paths so a dead
  collector can never hang process shutdown either.

## Consequences

- Positive: nothing added by this spec can regress the payload/repo/path
  non-goal by accident — the closed attribute set makes "what could leak"
  a fixed, reviewable list instead of an open-ended redaction surface.
- Positive: zero new dependency risk for the common case — a deployment
  that never sets the two env vars pays literally nothing (no goroutines,
  no exporters constructed, no imports of note beyond the no-op API
  surface already linked into the binary).
- Negative: logs are not OTLP-exported alongside traces/metrics — an
  operator wanting logs in their observability backend must ship stderr
  through their own log pipeline (already the norm for most CLI-shaped
  services) rather than getting it for free via OTLP. Tracked as a
  follow-up to revisit once `otel/sdk/log`/`otlploghttp` reach a stable
  `v1.x` release.
- Neutral: `internal/observability`'s `secretLikePatterns` intentionally
  duplicates (rather than imports) `internal/cli`'s equivalent list —
  avoids creating a dependency between two otherwise-independent leaf
  packages for a handful of literal regexes; if the pattern set needs to
  grow meaningfully, that's the trigger to extract a shared `internal/redact`
  package instead.
