# ADR: Recurrence intervention effectiveness measurement

## Status
Accepted (2026-07-19) — spec `pose-recurrence-effectiveness`

## Context

`recurrence-check` flags repeated failures and the escalation workflow turns
them into rules, workflows or specs — but nothing measured whether those
interventions worked. Creating the escalation was implicitly treated as
success, which is exactly the anti-pattern the closed loop exists to prevent.
The measurement must be reproducible from append-only local history, use
minimum sample sizes, and aggregate by task/context — never ranking
individuals ([DORA](https://dora.dev/guides/dora-metrics/) explicitly warns
against individual productivity metrics).

Alternatives considered:

1. **Manual pilot review by memory** — the status quo (the escalation
   workflow asked for a 45-day review with no data source); unmeasured and
   routinely skipped.
2. **External analytics backend** — violates the local-first boundary;
   outcome integrations belong to the insights-and-scale roadmap.
3. **Registered interventions + deterministic before/after projection over
   history JSONL.**

## Decision

Option 3:

- **Registration (R1):** `pose recurrence-effect --register` appends to
  `.pose/reports/history/interventions.jsonl` (schema 1): task slug,
  intervention ref (`rule:|workflow:|spec:<name>`, spec refs validated),
  observation window in days, rationale, pseudonymous author, RFC3339
  timestamp. Living in `history/` puts it under the existing
  `history-check` git-tracking gate.
- **Projection (R2):** `pose recurrence-effect` compares failures per task
  slug in the window before vs after each intervention, plus average
  duration/cost when records carry the new optional telemetry
  (`pose report --duration-seconds/--cost-usd`). Data-quality warnings are
  first-class: `insufficient sample` (below `--min-sample`, default 3) and
  `observation window incomplete` force an `INCONCLUSIVE` verdict — small
  samples and short windows mislead, so they never produce a verdict.
  Missing telemetry yields explicitly partial metrics, never fabricated
  numbers.
- **Feedback edge (R3):** an `INEFFECTIVE` verdict (complete window,
  sufficient sample, no failure reduction) prints the governed action —
  reopen or spawn an owned follow-up via the recurrence-escalation
  workflow, which now registers the intervention at ship time and reviews
  with this command. `--fail-ineffective` makes the verdict blocking where
  policy wants it (audit/CI); the default stays informative.
- **Privacy:** aggregation is by task slug and context; no author of
  failures is read, stored or reported.

## Consequences

- Positive: the loop's last edge is measured — systemic fixes are kept,
  adjusted or discarded on evidence, not on the satisfaction of having
  created them.
- Positive: fully offline and reproducible; re-running over the same
  history yields the same verdicts.
- Trade-off: task-mix changes between windows can still mislead; the
  warnings surface sample/window quality but human review owns the final
  keep/adjust/discard decision.
- Residual: telemetry adoption is optional — until reports carry
  duration/cost, cost comparisons stay partial by design.
