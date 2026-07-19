# ADR: Requirement trace contract

## Status
Accepted (2026-07-19) — spec `pose-requirement-evidence-traceability`

## Context

The closeout gate proved a spec was *finished* (completed_at + dispositioned
follow-ups) but not *why each promised behavior was accepted*: requirement
IDs (`R<N>`) existed since the DoR gate, yet nothing linked them to the
checks, reports or commits that verified them. The capability assessment
scored this gap (evidence 3/5) and the roadmap requires an auditable
intent-to-closure chain. Constraint: links must be explicit and reviewable —
compliance is never inferred from file proximity.

Alternatives considered:

1. **Separate trace file** (`trace.json` per spec) — machine-friendly but
   splits the living contract in two; drifts from the spec on every edit and
   is invisible in review diffs of the spec itself.
2. **Infer links from validation reports** (match check names to R-IDs) —
   violates the explicitness constraint; mechanical proximity is not
   evidence.
3. **In-spec `### Requirement trace` subsection** under `## 6. Validation` —
   one reviewable document, diff-friendly, offline, parsed by the same
   engine that lints the spec.

## Decision

Option 3. The trace grammar and gates:

- **Grammar (additive):** `- R<N> [satisfied] <evidence>`,
  `- R<N> [waived: <reason>]`, `- R<N> [withdrawn: <reason>]`. Evidence is
  free text plus structured refs — `check:<name>`, `test:<id>`,
  `report:<file>`, `commit:<sha>` — which feed the reverse
  evidence→requirements index.
- **Gates** (`pose lint-spec`): malformed entries, invalid dispositions,
  duplicates and orphans (traced but undeclared IDs) always fail. On `done`
  specs **with** the section, every declared R-ID must be traced; `done`
  **without** the section is a visible warning — the staged migration path
  for the 10 pre-contract specs. The spec templates (en + pt-BR) scaffold
  the section, so every new spec is fully enforced at closeout.
- **Projections (R3):** metrics in lint output (`spec.trace.*`) and the new
  read-class MCP tool `pose_requirement_trace` returning requirements with
  entries, the `by_evidence` reverse index, missing and orphaned IDs. The
  tool is an additive catalog change (golden regenerated and reviewed per
  the catalog ADR).
- **Security:** the trace carries refs and rationale, not confidential test
  output; actor identity is not collected.

## Consequences

- Positive: an auditor can traverse requirement → disposition → evidence and
  back; the closeout gate now explains acceptance, not just completion.
- Positive: waived/withdrawn make scope decisions explicit instead of silent.
- Trade-off: closeout costs one trace bullet per requirement; that is the
  point — the reviewable link is the artifact.
- Residual: mechanical link coverage can be mistaken for evidence quality;
  review owns quality, the gate owns existence and consistency. The legacy
  warning should be flipped to an error once pre-contract specs are archived
  (tracked by the spec's follow-up).
