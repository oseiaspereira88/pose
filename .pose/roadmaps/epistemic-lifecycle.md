---
slug: epistemic-lifecycle
status: draft
created_at: 2026-10-04
depends_on: agency-readiness
---

# Roadmap: Epistemic lifecycle and major cleanup plan

**Program:** POSE agency and readiness, wave 4 of 4.
**Outcome:** material premises and decisions can be reconsidered when their context or observable falsifiers change, and a measured, dry-run-proven plan exists for the legacy cleanup a major release would carry.

Source: [third consolidated analysis](../reports/2026-10-03-pose-consolidated-analysis.md),
items POSE-31 to POSE-33. Future work, after validation of the central waves.

## Milestone: premise-validity
- after: spec:pose-abm-capability-adoption
- target_start:
- target_due:
- specs: pose-assumption-validity-scope

**Exit gate:** a relevant contract change raises premise review; trivial assumptions get no TTL; the signal asks for judgment.

## Milestone: falsifier-loop
- after: premise-validity
- target_start:
- target_due:
- specs: pose-falsifier-reconsideration

**Exit gate:** a contrary observation raises a reconsideration candidate; history and rationale are kept; adequacy is never judged automatically.

## Milestone: major-cleanup-plan
- after: spec:pose-agency-readiness-pilot, spec:pose-transfer-preserves-obligations
- target_start:
- target_due:
- specs: pose-v7-legacy-cleanup-plan

**Exit gate:** dry-run over both brownfield corpora writes nothing and explains every effect; removal criteria are measured per field.

## Risk controls

- Validity by context and materiality before calendar expiry.
- A major is not a pretext to reform the engine without measured benefit.
