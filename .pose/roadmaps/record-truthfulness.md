---
slug: record-truthfulness
status: draft
created_at: 2026-10-04
depends_on:
---

# Roadmap: Record truthfulness (6.3.x)

**Program:** POSE agency and readiness, wave 0 of 4.
**Outcome:** POSE's records no longer imply more than the engine observed — adoption cutoffs, publication, review assurance, attribution and `blocked` mean what they say — and the existing backlog is reconciled before new features repeat delivered work.

Source: [third consolidated analysis](../reports/2026-10-03-pose-consolidated-analysis.md)
(backlog [JSON](../reports/2026-10-03-pose-consolidated-backlog.json), items POSE-01 to POSE-07)
and the transversal [ADR](../adr/2026-10-04-obligations-are-projected-action-requests-are-persisted.md).
This roadmap plans work; it declares no capability delivered.

Compatible fixes and diagnostics fit the 6.3.x maintenance line. Outputs whose
meaning changes (public-claims fields, attribution contract, metric v2) are
versioned and may ship in the next minor. Publication of the prepared 6.3.0
candidate stays under the existing release lifecycle and is not part of this roadmap.

## Milestone: compatibility-truth
- after:
- target_start:
- target_due:
- specs: pose-legacy-contract-cutoffs, pose-public-claims-publication-provenance, pose-flat-spec-amendments

**Exit gate:** every registered contract's legacy cutoff is consumed or reported; candidate, prepared and published versions are distinct; flat specs amend like folder specs.

## Milestone: review-truth
- after:
- target_start:
- target_due:
- specs: pose-review-assurance-disclosure, pose-review-attribution-roles

**Exit gate:** the 6.3.0 `human:oseias` attestations no longer read as human review; declared and verified separation render differently; attribution roles are prospective and legacy records keep their digests.

## Milestone: lifecycle-truth
- after: review-truth
- target_start:
- target_due:
- specs: pose-blocked-semantics-alignment, pose-open-backlog-reconciliation

**Exit gate:** `blocked` has one non-terminal meaning everywhere with transfer protection intact and metric v1 unchanged; every remaining requirement of the non-terminal specs is classified and only evidence-backed dispositions are applied.

## Risk controls

- Correct the effective data; never disable a gate to make history pass.
- Sealed bundles keep their stamped contracts; no retroactive governance.
- No auto-close from code existence; external operations are not performed by the program.
