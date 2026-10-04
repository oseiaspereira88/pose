# POSE 7.0 legacy cleanup plan — 2026-10-04

Spec: [pose-v7-legacy-cleanup-plan](../specs/2026-10-04-pose-v7-legacy-cleanup-plan.md).
Decisions it builds on: [ADR 2026-10-04](../adr/2026-10-04-obligations-are-projected-action-requests-are-persisted.md).

This is a plan, not a migration. 6.x applies no breaking change. `pose migrate v7
--dry-run` measures the corpus and writes nothing; `--apply` is refused in 6.x. A
7.0 apply must ship with its own recovery checkpoint and revision guard, the way
`pose close --apply` and `pose spec-transfer apply` do, before any class below is
converted.

## Snapshot

| Field | Value |
|---|---|
| Repository head measured | `9c3a038` (pose-dist `main`) |
| Command | `pose migrate v7 --dry-run` (candidate engine 6.3.0-dev) |
| Wall time | 0.5 s |

| Class | Count | Dry-run action |
|---|---:|---|
| Flat specs | 318 | none (current layout) |
| Dated folder specs | 3 | keep read-only |
| Undated legacy folder specs | 0 | convert |
| `status: blocked` specs | 0 | convert |
| Flat spec amendment journals | 0 | none (current) |
| Review policy legacy adoption keys | 5 | convert |
| Review policy unknown keys | 0 | refuse until resolved |
| Sealed review bundles | 1307 | keep read-only |
| …of which without `governing_contracts` | 483 | keep read-only |
| Attestations with a verified envelope | 0 | keep read-only |
| Attestations with declared identity | 1301 | keep read-only |
| Attestations without attribution or supplement | 1285 | keep read-only |
| Incomplete transfers (this project's side) | 0 | refuse until resolved |

The first run reported 10 incomplete transfers. That was the inventory reading
each transfer's status from the source project's side: the ten ABM transfers are
`activated` here, the destination, and their `source-retired` receipt lives in
Harne8. The predicate now uses the current project and its role in the plan. A
consumer's own dry-run is the only measure of its corpus.

## What never changes

Sealed bundles, attestations, signed envelopes, attribution supplements, change
sets sealed by `pose index` and history records are never rewritten, re-sealed or
re-signed. A major may change what *new* records must carry; old ones keep the
meaning they had under the contracts their seal date implies, and stay verifiable.
Unknown stays unknown: a record without attribution renders as
legacy-undifferentiated, never as a reconstructed role.

## Removal criteria

Each class is retired only when its criterion holds on the consumers' dry-runs, not
on this repository alone.

- **`status: blocked`.** Converted to `in-progress` plus an operational wait (an
  action request or an external wait) carrying the reason. Removable when no
  consumer dry-run reports a blocked spec and adoption metrics v2 (which already
  exclude blocked from the terminal denominator) have shipped for one minor.
  Measured benefit: one fewer status in every lifecycle check, readiness branch and
  metric; today `blocked` is non-terminal and maps to an operational wait in the
  readiness `Terminal`/`Cause` fields, so the information survives the conversion.
- **Legacy review policy adoption keys** (`component_aware_adopted_at`,
  `review_bundles_adopted_at`, `evidence_vocabulary_reconciled_at`,
  `explicit_judgment_adopted_at`, `structural_causality_adopted_at`). Each moves
  into `contract_adoptions` with the same date, so every cutoff keeps its meaning.
  Removable after one minor in which `pose doctor` reports `review.adoption-source`
  for every consumer. Measured benefit: one declaration path per contract instead
  of two, and the shadowing case (`legacy_shadowed`) disappears.
- **Unknown review policy keys.** A major refuses to start while one exists, so a
  misspelled key cannot silently leave a gate off (requirement R3). Today the
  dry-run reports each one as a risk.
- **Undated folder specs.** Converted to dated flat files only when every
  reference resolves to the new path. None exist here.
- **Dated folder specs.** Kept readable; no removal proposed. Converting them buys
  nothing measurable and would move paths that change sets already cite.
- **Bundles without `governing_contracts`, declared-identity and
  undifferentiated attestations.** Kept. A major may require verified identity and
  attribution for new attestations only.

## Preconditions for a 7.0 apply

1. No incomplete transfer on the project's side (the dry-run refuses otherwise).
2. A clean working tree and a revision guard: the apply binds to the dry-run's
   revision and refuses if it moved.
3. A checkpoint per converted class, so an interrupted apply resumes instead of
   re-running.
4. A rehearsal of every gate (`check --strict`, `lint-spec --all`, review bundle
   verification) against the converted tree before the real apply, as learned in
   the ABM reconciliation.

## Out of scope

Reforming the engine because a major is coming, converting declared identity into
verified, and removing history.
