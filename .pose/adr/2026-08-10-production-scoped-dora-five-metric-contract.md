# ADR: Production-scoped DORA five-metric contract

## Status
Accepted (2026-08-10) — spec `pose-dora-five-metrics-v2`

## Context

The existing DORA report still emitted a former Reliability proxy as its fifth metric, aggregated incidents across environments and calculated recovery from every resolved incident. The current DORA model instead includes deployment rework rate and defines failed deployment recovery around failures caused by deployments. Production environment names also vary between projects, so silently treating every environment as production would contaminate denominators.

Historical JSONL events do not contain deployment rework classification or incident environment. Guessing those fields would turn missing evidence into apparently precise values.

## Decision

- Write schema-v2 delivery events while continuing to decode schema-v1 JSONL.
- Require `deployment_kind=planned|rework` for new deployment events and `environment` for new incident events.
- Select one explicit environment in `dora-metrics`; default the selector to the literal `production` and allow projects to pass their actual production environment name.
- Apply the environment selector to every deployment and incident denominator.
- Replace `reliability` with `deployment_rework_rate` and keep the other established metric identifiers for compatibility.
- Calculate recovery only from resolved incidents with `caused_by_deployment=true` in the selected scope.
- Mark rework rate unavailable if any scoped legacy deployment lacks a kind. Do not interpret missing kind as planned.
- Expose schema version and data-quality exclusions in the report so consumers can distinguish no events from incomplete legacy classification.

## Alternatives considered

1. Treat missing deployment kind as planned. Rejected because it creates an optimistically biased zero rework rate.
2. Hard-code `prod|production` aliases. Rejected because aliases cannot identify a project's actual production environment reliably.
3. Keep Reliability as an additional sixth metric. Rejected because the command promises the current five-metric DORA contract; product-specific reliability belongs in a separately named report.
4. Infer deployment-caused incidents by timestamp proximity. Rejected because temporal proximity is not causation and violates explicit-event ingestion.

## Consequences

- New ingestion automation must provide one additional required field.
- Existing event files remain readable and immutable.
- Four metrics can continue using valid legacy deployment data; rework remains explicitly unavailable until all scoped deployments in the selected window are classified.
- Projects using an environment name other than `production` must pass `--environment` when querying metrics.
