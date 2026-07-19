# ADR: Knowledge consumption refs and advisory retrieval

## Status
Accepted (2026-07-19) — spec `pose-knowledge-consumption-traceability`

## Context

Knowledge artifacts have TTL, owners and sensitivity, but nothing showed
whether they actually influenced work: handoffs could expire unread or be
silently load-bearing. The spec requires consumption visibility and optional
semantic retrieval — without making POSE an opaque vector store, without
auto-applying restricted content and while operating fully offline
([NIST AI RMF](https://www.nist.gov/itl/ai-risk-management-framework),
[MCP security best practices](https://modelcontextprotocol.io/specification/2025-06-18/basic/security_best_practices)).

Alternatives considered for usage signals:

1. **Manual usage-event log** — an event stream nobody writes is worse than
   none; it decays into fiction.
2. **Derived citations** — specs cite consumed knowledge with a stable
   `knowledge:<slug>` token; usage is deterministically recomputed from the
   repository on every run.

For retrieval: embedding backends were rejected as the shipped default (the
non-functional requirement demands full operation without a semantic
backend); a deterministic lexical baseline ships instead.

## Decision

- **Stable consumption refs (R1):** `knowledge:<slug>` tokens in spec bodies
  are the citation contract. `pose knowledge-check` fails on dangling refs
  (`knowledge.ref_failures` metric) — citations must resolve to a governed
  artifact.
- **Usage signals (R2):** `pose knowledge-usage` projects per-artifact
  citations (citing specs, count, owner, expiry) recomputed from the
  repository — no stored event stream, no actor tracking. The output states
  the invariant explicitly: signals inform the owner's review; **TTL is
  never extended automatically**. Popularity does not preserve knowledge —
  owners do.
- **Advisory retrieval (R3):** `pose knowledge-suggest <query>` ranks
  non-restricted artifacts with the same deterministic lexical engine used
  for follow-up clustering. Rationale is exposed (shared terms + score),
  `restricted` artifacts are filtered **before** retrieval with the filtered
  count visible, and the output requires human confirmation before citing —
  suggestions never gate and never auto-apply.
- **Semantic adapters stay external and opt-in:** no provider is approved by
  default; adding one requires an ADR naming the provider, its data
  exposure and the sensitivity policy. Model output never becomes a gate.
- **Compatibility:** existing knowledge needs no migration — refs and
  projections are additive.

## Consequences

- Positive: expiring memory becomes measurably useful; the quarterly audit
  can see load-bearing artifacts approaching expiry and unread ones wasting
  review time.
- Positive: retrieval is explainable and reproducible; sensitivity
  enforcement precedes ranking rather than being filtered from results.
- Trade-off: lexical ranking is weaker than embeddings; that is the honest
  offline baseline, and adapters remain possible behind explicit approval.
- Residual: citation requires discipline (an uncited-but-used artifact is
  invisible); workflows and skills instruct citing, and review enforces it.
