# ADR: Release temporal evidence and attempt lineage

## Status

Proposed — 2026-10-14. Supplemental proposal for B39/B41/B40; not adopted or
implemented. Preserve ADR-018, release archival by ledger, independent verification
and sealed-history authority. Reconcile and accept the applicable structural
contract before functional start; this document selects no JSON schema version,
CLI syntax, migration default or universal locking mechanism.

## Context

The [release analysis](../reports/2026-10-12-pose-front-06-release-sufficiency.md)
requires a prospective temporal contract for F06-09. P05 exposes candidate
freshness distinct from historical integrity; P07–P09 expose incomplete asset and
replay bindings. The current journal allows one evidence value per state and
rejects a conflicting second value, while publication supersession must invalidate
proof bound to the earlier publication. The
[consistency analysis](../reports/2026-10-12-pose-front-09-consistency-recovery.md)
shows a prepare interrupted before manifest that cannot resume; its rollback
may remove material after compensation fails. That latter failure remains a
code-founded hypothesis until injected in a fixture.

Existing ADRs already establish immutable manifests/tags, append-only factual
events, proof before asset execution and no rewriting of sealed specs. This
proposal supplies temporal and lineage boundaries; it does not replace them or
create another engine of gates or mutable source of requirements.

## Proposed decision

- Distinguish candidate-current evaluation, integrity of the sealed cut and replay
  of observed publication. Name mode, source bindings, applicable contract/policy,
  coverage and assurance. Changes to current version/policy alone do not corrupt
  history; tampered payloads and inadequate proof are diagnosed in every mode.
- Recompute evidence hashes and verify asset/publication bindings when replayed.
  Publisher and verifier have separate roles; differing legitimate providers are
  allowed, while repository/tag/commit and applicable identity contracts must match.
  Partial asset/platform/channel coverage never becomes total verified assurance.
- Preserve append-only lineage for repeated/conflicting observations. An explicit
  successor/attempt relation invalidates current use of proof bound to a superseded
  publication without deleting the predecessor. This does not authorize changing
  provider assets or Git tags, or a second release assignment of the same spec.
  Exact attempt identity, correction semantics and schema compatibility must be
  resolved and reviewed before implementation; they are not supported today.
- Keep operation intention/phase/receipt separate from release lifecycle events.
  Recovery uses observed inventory, pre/postimages and completed effects; it may
  finish a demonstrated intent, but cannot invent a historical manifest via backfill.
  Failed compensation preserves the remaining copy and reports incomplete recovery.
- Bind explicit cut scope/refs/digests to the selected fragments and existing
  sufficiency evaluators. Exclusion outside the promised scope differs from a
  policy-authorized deferral within it. Assignment discovery is shared with the
  global checker, rejects duplicates before moving the queue and discloses losses.
- Apply new semantics prospectively with compatibility classification and migration
  preview. Recording a validated external publication is a factual audit operation;
  missing past authorization is disclosed, not converted into a denial that the
  observed publication occurred. Adequate proof remains necessary for assurance.

## Consequences

The reader reports more precise validity and coverage; existing historical bytes
and meanings remain intact. Before implementation, acceptance must settle the
schema/producer/consumer inventory, legacy behavior, support of policy knobs,
identity/lineage representation, coordination/durability guarantees and fixtures.
No gate, migration, release or authority exception is adopted by this proposal.
B39/B41 can be implemented as separate units after their recorded prerequisites;
B40 integrates them after both are qualified, without a B39/B41 dependency cycle.

## Alternatives rejected

- Revalidate every old release against today's version/policy: silently changes the
  meaning of historical cuts and reports normal version advancement as corruption.
- Trust stored evidence_digest or State=verified: permits altered assets and stale
  bindings to affect open-next and other consumers without proof revalidation.
- Rewrite the last event or move the tag: erases lineage and violates immutable
  snapshots and append-only evidence.
- Backfill a manifest to make recovery pass: invents a freeze without an observed
  intent/inventory and hides missing evidence.
- Copy all product commitments into a new ledger: duplicates sources and evaluators
  already responsible for sufficiency and federated scope acceptance.

## Review triggers

Review before B39/B41/B40 start when the exact schema/compatibility and correction
semantics are fixed. Revisit if another provider/channel needs different identity
or coverage, or a durability guarantee is proposed beyond the measured filesystem
model. Track in knowledge:adr-release-temporal-evidence-review.

## Related specs

- [B39](../specs/2026-10-14-pose-release-publication-evidence-replay.md)
- [B41](../specs/2026-10-14-pose-release-prepare-record-recovery.md)
- [B40](../specs/2026-10-14-pose-release-scope-selection-and-exclusivity.md)
