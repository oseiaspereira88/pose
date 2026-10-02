# ADR: Delivery integrity graph and Git-observed provenance

## Status
Accepted (2026-08-02) — specs `pose-artifact-provenance-ledger` and
`pose-delivery-surface-assurance`
Amended (2026-09-09) by spec `pose-emittable-analysis-evidence-classes` — see Amendments

## Context

POSE needs to answer two related but independent questions: which reviewed
spec claims a repository artifact, and whether that artifact participates in a
real delivery surface. Specs are declarations; Git is the factual witness for
repository changes; registered validation/composition evidence is the witness
for delivery. Collapsing these facts would make a declaration self-proving or
make framework-specific inference part of the governance engine.

Alternatives considered:

1. Separate artifact and surface indexes — rejected because cross-query and
   migration semantics would drift.
2. Infer provenance from timestamps, authors or path proximity — rejected
   because ambiguous history would be presented as fact.
3. Use one typed graph with distinct declared, observed and verified edges —
   selected because mismatches remain visible and each witness stays explicit.

## Decision

Generate one schema-versioned `.pose/indexes/delivery-integrity.json` graph.
Represent specs, change sets, artifacts, delivery targets, capabilities,
surfaces, entrypoints and evidence as typed nodes. Keep declaration edges
separate from Git-observed change edges and verification edges. Store stable
IDs and digests, never source content.

Resolve provenance only from explicit base/head revisions recorded by a
spec-linked report or commits with `POSE-Spec` trailers. Reject unsafe revision
syntax and never fall back to authorship or time heuristics. Keep reverse
indexes as deterministic projections of the same edge set.

Adopt the schema additively. Legacy narratives produce migration findings;
policy activation dates and governed roots control enforcement. Surface
assurance extends the graph without changing provenance node or edge semantics.

Delivery refs use the closed kinds `surface`, `contract`, `capability`,
`infrastructure` and `governance`. Their declaration creates `delivers`,
`implemented-by`, `composes` or `reaches`, and `validated-by` edges. Validation
checks carry a closed `evidenceClass`, which includes the analysis kinds
`lint`, `typecheck`, `security-scan` and `contract` (amended 2026-09-09);
profiles require classes while results
remain a distinct witness bound to the provenance digest. Surface profiles
always require `reachability` and one of `integration` or `e2e`; composed
capabilities require `integration` from the production composition root.

Roadmap cut criteria may reference only typed delivery refs, registered
`check:` names, `evidence:` levels or confined `manual-review:` reports. Raw
command text is rejected and never reaches the executor. A criterion is a
`gates` edge, not executable content, and roadmap success requires current
member closeout plus passing referenced evidence and no required graph finding.

## Consequences

- Positive: spec→artifact and artifact→spec traversal share one reproducible
  source and ambiguous history remains an explicit finding.
- Positive: the follow-on surface gate can prove composition without treating
  a green artifact check as delivery.
- Trade-off: the graph schema becomes a public compatibility contract and all
  producers/consumers need golden parity.
- Trade-off: explicit selectors and attribution cost more than heuristic
  inference, but they are falsifiable.
- Neutral: package/SBOM/SLSA provenance remains an independent release layer.

## Amendments

### 2026-09-09 — analysis results have classes of their own

Spec `pose-emittable-analysis-evidence-classes`.

**What changed.** `lint`, `typecheck`, `security-scan` and `contract` join the
closed `evidenceClass` vocabulary. They were previously reported as `build`, or
declared nothing at all.

**Why.** The closed vocabulary exists so a profile cannot demand what no check
may emit. It did not follow that every kind of result a check produces has a
class: static analysis, type checking, dependency scanning and contract
verification all collapsed into `build`, and a criterion asking for security
assurance was therefore satisfied by a successful compilation. That is the same
indistinction the single vocabulary closed between profiles and checks, one
level down — a gate that reads as met by evidence that says nothing about it.

**Options considered.**

1. Leave them as `build`. Rejected: the collapse is the defect, and
   `security-scan` is the case where the consequence leaves the repository.
2. Add only `security-scan`. Rejected: `lint` and `typecheck` have shipped
   default checks today and would keep declaring nothing, so the classless-check
   warning would stay for the two the vocabulary could have covered.
3. Add all four. Selected.

**Consequences.**

- `go vet` now emits `lint` rather than `build`. It is a linter, and calling it
  a build was the collapse in its clearest form. Because it was the only Go
  check emitting `build`, a `go build ./...` check was added so the class keeps
  a producer in that stack — its absence is why vet carried the label.
- `security-scan` and `contract` have no shipped default check. They are
  emittable — `pose validate` accepts a check declaring them — which is what
  separates them from the ten classes that were demandable and impossible before.
  A project registers its own; a shipped default that emits `security-scan` is a
  follow-up.
- An instance that declares one of the four cannot be validated by an engine
  that predates them. Observed, not argued: a binary built before this change
  fails `pose check --strict` with `stacks.node check lint has unknown
  evidenceClass "lint"`. The boundary is declaring the class on a check, not
  only demanding it in a profile — the matrix is validated too. This is the
  adoption cost of any change to a closed set, and no encoding avoids it; the
  set being closed is what makes it worth having.


### 2026-10-02 — the index stores validation evidence once

Spec `pose-delivery-integrity-index-compaction`.

**What changed.** The index file moves to schema 2. The graph a caller holds in
memory is unchanged and keeps schema 1; only what is written differs. Results that
share one validation run reference it, and the run's `scope_provenance` map is
stored once. The results that validate a delivery are one shared `validation-set`
node per distinct set, linked from each delivery with a single `validated-by` edge
and to its members with `contains` edges. A delivery path names its set instead of
listing every result. `changes` edges, which are each change set's observed paths,
are left out and named in `implied_edges`. Every edge is written once.

**Why.** The index went from 3.2 MB to 9.1 MB between v5.0.8 and v6.2.0 while specs
grew by a third. Three structures grew as deliveries times results: one
`validated-by` edge per delivery and passing result, 53 identical copies of a
247-entry `scope_provenance` map, and the same results repeated in every path. The
edge list also held 9,638 exact duplicates, because an entrypoint shared by many
targets was linked once per target. On this repository the same inputs now write
4.6 MB instead of 10.9 MB, with the same provenance digest, claims, change sets and
findings.

**Options considered.**

1. Remove only the duplicate edges. Exact and needs no schema change, but one edge
   per delivery and result remains, so growth stays quadratic.
2. Link a delivery to a module and evidence class instead of to results. Linear,
   but the edge would assert less: currency is checked per target and spec.
3. Share one node per distinct result set. Exact, and bounded by what differs
   between deliveries. Selected, together with edge deduplication.

**Consequences.**

- Reading expands the index back to the exact pairs, so every gate and MCP tool sees
  the same graph. The provenance digest and the scoped digests hash claims and
  change sets only and do not move; sealed bundles stay current.
- The graph's `input_digest` is recomputed over the expanded graph and changes once.
  No bundle consumes it.
- Worst case, every delivery has a distinct set, and the index costs what the
  expanded pairs cost; it is no larger. The saving depends on deliveries repeating
  their evidence, which gates on a module do.
- The index is now written by `IndexJSON`, not by marshalling the graph, because the
  MCP tools return the graph and their output must stay the expanded form.
- An engine that does not know schema 2 refuses the file as an invalid index and
  `pose index` from that engine writes schema 1 over it. Neither reads one as the
  other. A reader that walks the raw file for `validation-result` nodes follows one
  more hop, and finds no `changes` edges unless it derives them.
- A delivery path now lists its validation results in sorted order; they are a set,
  and the order they were found in cannot be given back from a stored set.
