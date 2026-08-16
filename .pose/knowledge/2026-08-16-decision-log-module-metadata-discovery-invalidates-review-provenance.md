---
type: decision-log
slug: module-metadata-discovery-invalidates-review-provenance
owner: @pose-maintainers
sensitivity: public-internal
created_at: 2026-08-16
last_reviewed_at: 2026-08-16
expires_at: 2026-11-14
source_refs:
  spec: "pose-upgrade-path-audit-fixes"
  workflow: "bugfix"
  commands: ["pose update --force", "pose install", "pose index", "pose review bundle --seal"]
  external_sources: []
---

# decision-log: module-metadata-discovery-invalidates-review-provenance

## Context

Auditing `pose update`/`pose install` across 7 real repositories ahead of
the v1.4.0 release found that running either command on `pose-dist` itself
(the product's own dogfooding instance) discovered
`examples/brownfield-kits/direct-adoption/fixture/service/go.mod` — an
illustrative fixture, not a real deliverable module — as a genuine module
and added it to `.pose/indexes/module-metadata.json`. That single addition
recomputed the global provenance digest (`graph.ProvenanceDigest`, computed
over the whole delivery-integrity graph, not per-spec — see the
`pose-gate-closeout-procedure` session note this decision-log
generalizes) and superseded the review attestation of every already-closed
spec whose scope touches `module-metadata.json`: between 3 and 22 specs
across the audit's trial runs, depending on which other specs had most
recently closed.

## Current state

Fixed the trigger only (spec `pose-upgrade-path-audit-fixes`, R7):
`testdata`, `fixture` and `fixtures` directories are now excluded from
module discovery in all three independent walkers
(`discoverValidationModules` in validate.go, `scanModules` in index.go,
and `internal/pose/discovery.go`'s capability scanner). A genuine,
intentional module-metadata change — a real module added, removed or
reclassified — still invalidates every closed spec's review attestation
that touches it, exactly as designed today. That broader behavior was
deliberately left unchanged (Decision 2 in the spec above): narrowing the
provenance digest so it only invalidates reviews whose scope actually
includes the changed module is a real design trade-off (what counts as
"unrelated," whether a narrower digest weakens the guarantee it exists to
provide) that deserves its own spec and explicit review, not a side effect
of fixing a discovery false-positive.

## Next checks

- None deterministic yet. If a future spec pursues narrowing the
  provenance digest, `pose-mcp/internal/pose/delivery_integrity.go`
  (`ProvenanceDigest`) and `pose review bundle --seal`'s scope-provenance
  matching are the two places to start reading.

## Risks

- Any *other* directory-discovery path added in the future (a fourth
  scanner, or a stack detector that walks the tree independently) needs
  the same `testdata`/`fixture`/`fixtures` exclusion applied by hand —
  there is no single shared walker all three call today (each has its own
  `ignored` map/condition; see the follow-up already on record about this
  in `pose-review-bundle-roadmap-path-portability`/session knowledge about
  "enumerate by hand" gaps recurring across this repository).
- Every maintainer who runs `pose update --force`/`pose install` on
  `pose-dist` itself after a *real* module-metadata change should expect
  to reseal/reattest every closed spec whose review scope includes
  `module-metadata.json`, per `pose-gate-closeout-procedure` — this is
  expected, not a bug, and this decision-log's fix does not change it.

## Next owner

@pose-maintainers — no active handoff; revisit only if the digest-scoping
follow-up (spec `pose-upgrade-path-audit-fixes`, Follow-ups) is picked up.

## References

- spec:pose-upgrade-path-audit-fixes (Decision 2, R7, Non-goals)
- knowledge: session note `pose-gate-closeout-procedure` (this repository's
  own local agent memory, not a `.pose/knowledge/` artifact) — first
  documented that the provenance digest is global-over-the-graph, not
  per-spec, and that reseal/reattest is required after any reindex that
  changes it.
