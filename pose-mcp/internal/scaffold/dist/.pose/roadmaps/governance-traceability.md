---
slug: governance-traceability
status: done
created_at: 2026-07-18
depends_on:
---

# Roadmap: Governance traceability

**Portfolio order:** 3 of 7
**Outcome:** connect intent, evidence, change history, residual work and reusable knowledge into a reviewable delivery chain.

This roadmap deepens POSE's strongest differentiator: closing the loop after a spec is written. It avoids turning traceability into surveillance; evidence is scoped to delivery artifacts and never used as an individual productivity score.

## Milestone: trace-core
- after:
- target_start: 2026-08-24
- target_due: 2026-09-25
- specs: pose-requirement-evidence-traceability, pose-followup-ownership-sla

**Exit gate:** requirements and residual work have durable identifiers, owners and evidence links.

## Milestone: governed-change
- after: trace-core
- target_start: 2026-09-28
- target_due: 2026-10-23
- specs: pose-spec-amendment-history, pose-knowledge-consumption-traceability

**Exit gate:** intent changes and reused knowledge are append-only and reviewable.

## Milestone: measured-closure
- after: governed-change
- target_start: 2026-10-26
- target_due: 2026-11-06
- specs: pose-recurrence-effectiveness

**Exit gate:** an auditor can traverse requirement to evidence to follow-up or knowledge, and measure whether systemic fixes worked.

## Risk controls

- Preserve stable requirement IDs and append-only amendment events.
- Require human confirmation for semantic duplicate or disposition decisions.
- Minimize actor data and define retention before collecting approval identity.
