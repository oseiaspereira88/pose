# ADR: DORA and adoption metrics — explicit events, an explicit "unavailable" state

## Status
Accepted (2026-07-19) — spec `pose-dora-adoption-metrics`

## Context

POSE had no way to answer "does governance adoption correlate with real
delivery outcomes" — the whole point of this spec, and the thing that
distinguishes measuring value from measuring artifact volume. Two
different data shapes are needed: DORA's four-plus-one metrics (which
describe production delivery behavior POSE cannot observe on its own —
deployments and incidents happen outside the repository) and adoption
metrics (which POSE already has every input for, in specs and workflow
history). The spec's Non-goal is explicit and easy to violate by
convenience: never infer deployments or incidents from commits alone,
and never produce a per-person ranking.

Alternatives considered:

1. **Infer deployment/incident signals from git history or CI webhooks
   automatically.** Attractive (zero manual step), but directly
   contradicts the Non-goal — a merge is not a deployment, and CI success
   is not "no incident." Any inference layer would need its own set of
   assumptions that could be silently wrong, exactly the "correlation
   misrepresented as causation" Technical risk the spec calls out.
2. **Default missing metrics to zero.** Simpler denominator handling, but
   directly contradicts the Compatibility requirement and is actively
   misleading — a team with zero *recorded* deployments is not the same
   as a team with zero *actual* deployment frequency.
3. **Explicit event ingestion (`pose record-deployment`/`record-incident`)
   with a minimal, identity-free schema, an honest three-state metric
   result (`value` / `unavailable` + reason), and adoption metrics derived
   entirely from data POSE already owns** (spec frontmatter, workflow
   history) rather than a second event-ingestion path.

## Decision

Option 3.

- **Events are the only input, never inferred.** `deploymentEvent`
  (`internal/cli/dora_events.go`) requires `application`, `environment`,
  `status` (success|failure) and `source` (manual|ci|webhook — the
  "quality metadata" R1 asks for: who/what is vouching for this record);
  `lead_time_seconds` is optional and explicit, never derived from commit
  timestamps. `incidentEvent` requires `application`, `started_at`,
  `severity`; `resolved_at` absent means still open. Storage is
  append-only monthly JSONL under `.pose/events/{deployments,incidents}/`,
  mirroring the existing `.pose/reports/history/` convention.
- **No identity field exists anywhere in the schema** — not `author`, not
  `user`, not `principal`. `TestNoDORAOrAdoptionTypeExposesIndividualIdentity`
  reflects over every DORA/adoption struct's JSON tags and fails the build
  if one is ever added — the "never individual scores" constraint is
  enforced by a test, not a comment.
- **Every metric is `{value, unit, available, reason, sample_size}`, never
  a bare number.** `unavailable` fires per-metric based on that metric's
  own real denominator (zero deployment events → deployment frequency and
  change failure rate both unavailable; zero deployments *with*
  `lead_time_seconds` → lead time unavailable even if deployments exist;
  zero resolved incidents → recovery time unavailable) — never a global
  "any data at all" gate, since a team might have deployments but no
  incidents (which should read as a real 100% reliability, not
  "unavailable"), or incidents but no deployments recorded yet.
- **Reliability (DORA's 2023-era fifth metric) is a documented proxy**:
  percentage of window-days with no ongoing major/critical incident —
  chosen because POSE has no SLO configuration to compute a "true" SLA-
  based reliability figure, and a documented proxy beats a fabricated
  precise one. Explicitly written into the metric's own doc comment and
  the CLI reference so nobody mistakes it for an official DORA formula.
- **Adoption metrics need zero new events** — `internal/cli/adoption_metrics.go`
  derives Activation (earliest of a `done` spec or a passing history
  record), Time-to-First-Gate (activation minus the earliest spec's
  `created_at`, the closest available proxy for "adoption start" since
  POSE persists no install timestamp), Retention (active weeks over weeks
  since activation) and Task Success (done / (done+abandoned+blocked),
  pending specs excluded from the denominator since they haven't resolved
  either way) entirely from `.pose/specs/*` frontmatter and the existing
  `.pose/reports/history/` JSONL — reusing `readFlatFrontmatter` and
  `readHistory`/`historyRecord`/`parseHistoryTime` rather than a second
  ingestion path.
- **Retention/deletion (Security)**: `pose events-housekeeping
  list-expired|purge [--apply]` mirrors the existing
  `knowledge-housekeeping`/`reports-housekeeping` dry-run-by-default
  pattern, operating on whole monthly files. Aggregation (the Security
  requirement's third clause) is structural: neither `dora-metrics` nor
  `adoption-metrics` ever prints a per-event row, only window aggregates.
- **OTel and Harne8 projections (Affected areas) are deliberately not
  wired into the one-shot CLI commands.** `pose record-deployment`/
  `dora-metrics` are short-lived processes; instrumenting them with
  OpenTelemetry spans provides little value over the structured stdout
  they already produce, unlike the long-running MCP server
  `pose-otel-observability` targets. Harne8 control-plane projection is
  explicitly out of scope until `pose-harne8-control-plane-integration`
  (roadmap 7, milestone 4) exists to receive it.

## Consequences

- Positive: every "no data" state reads as an honest `unavailable` with a
  reason, never a silently-wrong zero — verified for both DORA
  (`TestDORAMetricsUnavailableWithNoData`) and adoption
  (`TestAdoptionMetricsUnavailableBeforeActivation`).
- Positive: the individual-ranking constraint is a compile-time-adjacent
  guarantee (a reflection test over the JSON schema), not a code-review
  convention that could quietly erode.
- Negative: ingestion is manual/CI-driven (an operator or their CI must
  call `record-deployment`/`record-incident`) — no automatic collector
  exists yet. Acceptable: automatic collection is exactly the inference
  the Non-goal forbids without a defined, trusted event source.
- Neutral: Reliability's proxy definition may not match what a team
  expects from "DORA Reliability" if they've seen a different formula
  elsewhere — documented explicitly in three places (code comment, CLI
  reference, this ADR) to manage that expectation rather than hide it.
