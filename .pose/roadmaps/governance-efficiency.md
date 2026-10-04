---
slug: governance-efficiency
status: draft
created_at: 2026-10-04
depends_on:
---

# Roadmap: Proportional and measured governance

**Program:** POSE agency and readiness, wave 3 of 4.
**Outcome:** governing a delivery costs less operator work without losing integrity — closeout is a recoverable plan, valid observations are reused by material equivalence, assessments and spec surface follow materiality, residual backlog drift is visible, waiting and rework are measured — and every adoption is decided by a pilot.

Source: [third consolidated analysis](../reports/2026-10-03-pose-consolidated-analysis.md),
items POSE-20 to POSE-27, POSE-29 and POSE-30.

## Milestone: recoverable-closeout
- after: spec:pose-phase-scoped-readiness, spec:pose-governed-effect-enforcement
- target_start:
- target_due:
- specs: pose-recoverable-closeout-plan, pose-material-equivalence-reuse

**Exit gate:** an interrupted closeout resumes without duplicate bundles or attestations; stale `results_path` is regenerated before sealing; judgment is never filled; reuse cites origin and input bindings.

## Milestone: proportional-surface
- after: recoverable-closeout
- target_start:
- target_due:
- specs: pose-adaptive-assessment-freshness, pose-progressive-spec-surface, pose-followup-reconciliation-candidates, pose-action-request-presentation

**Exit gate:** a trivial change keeps the minimal path with adopted gates intact; factual sections are generated, never intent or rationale; candidates are raised without auto-close; grouped requests keep independent answers.

## Milestone: measured-governance
- after: recoverable-closeout
- target_start:
- target_due:
- specs: pose-governance-wait-rework-observability

**Exit gate:** pending age, attributed waiting and known blocking are separate; unknown stays unknown; rework causes are classified only with evidence; benchmarks report time and bytes on fixed corpora.

## Milestone: agency-pilot
- after: measured-governance
- target_start:
- target_due:
- specs: pose-agency-readiness-pilot

**Exit gate:** stop/go recorded with baseline comparison, rollback conditions and delimited adoption scope.

## Milestone: abm-adoption
- after: spec:pose-open-backlog-reconciliation, spec:pose-flat-spec-amendments, spec:pose-effective-governance-projection
- target_start:
- target_due:
- specs: pose-abm-capability-adoption

**Exit gate:** contract nodes, atomic start and causality closeout each have an explicit adopt-or-defer decision backed by pilot and shadow evidence; no second implementation.

## Risk controls

- No aggregate efficiency or quality score; dimensions only.
- Automation is preview-first with explicit apply.
- Dogfooding is necessary but does not prove ergonomics for external adopters; record it as a limitation.
