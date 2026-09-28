# ADR: Versioned governance outcomes cohorts and lineage

## Status
Accepted (2026-09-27; producer implementation and strict validation complete)

## Context

The schema 1 governance outcomes report aggregates report history, sealed
review bundles and attestations, but does not return the identity used to tell
a replayed history row from a distinct attempt, expose the review band's
cohort, or expose explicit finding-to-remediation references. Consumers cannot
offer those views without either inferring relationships or reading raw review
artifacts themselves. The existing report history already contains
`report_type`, `task_slug` and `sequence`; review plans seal a band; remediation
links are explicit in spec frontmatter and validated against immutable review
artifacts.

## Decision

Extend `pose_governance_stats` and `pose stats governance` as a schema 2
projection over existing local artifacts. Deduplicate report attempts by the
existing `(report_type, task_slug, sequence)` identity; preserve separate
sequence values and their negative outcomes. Count rows missing the identity as
identity-unknown and surface conflicting identity payloads as coverage loss.

Accept only the closed report type and review band vocabularies. Apply report
type only to report-history attempt dimensions and band only to review and
finding-linked remediation dimensions; return the applicability and the
denominator for each facet so a filter never implies a cohort that the source
cannot prove.

Return a bounded, deterministic list of explicit finding-to-remediation
references containing only attestation ID, finding ID, source/remediation
spec slugs, category and source band. Do not return reviewer, owner, rationale,
evidence text, transcripts, or paths. Keep all reads local and read-only; no
new event/outbox store or collector is introduced. New consumers reject
unsupported schema versions.

## Consequences

- Schema 1 consumers must be upgraded before they can consume version 2; the
  engine will not silently reshape or version-downgrade the report.
- Replayed identity-bearing records no longer change attempt rates, while
  distinct sequence values—including failures—remain separate attempts.
- A band filter leaves attempt metrics unchanged, and a report-type filter
  leaves review metrics unchanged; the report states this facet scope.
- Legacy rows without identity remain visible in identity-unknown coverage.
- Detailed lineage is capped and reports truncation; links authorize the user
  to open only the same project's existing spec/review surfaces.
- The projection remains an observation and does not claim causal effect or
  delivery quality.

Rejected alternatives: a new outbox/event journal duplicates the append-only
authority; passing through whole attestations exposes more review data than the
consumer needs; matching titles or commit order manufactures causality.
