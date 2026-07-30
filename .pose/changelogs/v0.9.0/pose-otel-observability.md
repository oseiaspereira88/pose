---
spec: pose-otel-observability
category: added
breaking: false
refs:
---

`pose serve-mcp` can now emit OpenTelemetry traces, metrics and
correlated structured logs for every tool call, off by default and
requiring both `POSE_OTEL_ENABLED=1` and `OTEL_EXPORTER_OTLP_ENDPOINT`
to activate — otherwise fully inert, zero network. Every signal carries
only the tool name and its catalog risk class; latency, policy-denial
and in-flight-concurrency metrics never include arguments, paths, repo
names or user identifiers. Logs are trace-correlated JSON on stderr with
paths and secret-shaped content redacted. An unreachable or misconfigured
collector never blocks server startup, a tool call, or shutdown. See
`docs-site/docs/mcp.md#observability`.
