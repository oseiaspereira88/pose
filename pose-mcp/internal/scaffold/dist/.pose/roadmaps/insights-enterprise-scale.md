---
slug: insights-enterprise-scale
status: active
created_at: 2026-07-18
depends_on:
---

# Roadmap: Insights and enterprise scale

**Portfolio order:** 7 of 7
**Outcome:** prove delivery value across repositories and compose the open core with Harne8 for durable, policy-governed operation.

This roadmap intentionally follows trustworthy evidence and stable project contracts. Apply OpenTelemetry and DORA without converting team-level outcomes into individual performance rankings.

## Milestone: observability-foundation
- after:
- target_start: 2026-11-02
- target_due: 2026-12-04
- specs: pose-otel-observability

**Exit gate:** server operation emits privacy-bounded standard signals.

## Milestone: delivery-outcomes
- after: observability-foundation
- target_start: 2026-12-07
- target_due: 2027-01-15
- specs: pose-dora-adoption-metrics

**Exit gate:** teams correlate governance adoption with delivery outcomes using explicit data-quality rules.

## Milestone: governance-intelligence
- after: delivery-outcomes
- target_start: 2027-01-18
- target_due: 2027-02-26
- specs: pose-semantic-governance-assist, pose-cross-repo-portfolio

**Exit gate:** human-reviewed semantic assistance and portfolio projections work across repositories.

## Milestone: control-plane-composition
- after: governance-intelligence
- target_start: 2027-03-01
- target_due: 2027-03-31
- specs: pose-harne8-control-plane-integration

**Exit gate:** multi-repository pilots prove tenant, policy, retention and offline-degradation boundaries.

## Risk controls

- Keep raw source content and secrets out of telemetry.
- Define tenancy, deletion and retention before central ingestion.
- Keep POSE usable offline when Harne8 or an observability backend is absent.
