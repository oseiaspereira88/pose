# ADR: Delivery integrity graph and Git-observed provenance

## Status
Accepted (2026-08-02) — specs `pose-artifact-provenance-ledger` and
`pose-delivery-surface-assurance`

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
checks carry a closed `evidenceClass`; profiles require classes while results
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
