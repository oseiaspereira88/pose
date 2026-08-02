# ADR: Immutable hierarchical review and closeout evidence

## Status
Accepted (2026-08-02) — spec `pose-hierarchical-review-closeout`

## Context

POSE historically treated deterministic validation and lifecycle metadata as
sufficient closeout evidence. That allowed a spec to become `done`, a
milestone to roll up from member statuses and a roadmap to be marked `done`
without a durable review of the exact scope being closed. Review prose existed,
but was neither immutable nor bound to the reviewed inputs. The new contract
must work offline, remain reviewable in Git, support proportional independence
policy and avoid giving autonomous execution authority it did not already have.

Alternatives considered:

1. Infer approval from existing Markdown reports — rejected because narrative
   reports do not have complete criteria, stable decisions or freshness.
2. Store a mutable `review_status` on each spec/roadmap — rejected because edits
   erase history and cannot prove which content was approved.
3. Store immutable attempts bound to canonical scope digests, then derive
   closeout hierarchically — selected because evidence and staleness are
   deterministic without collapsing review into a build check.

## Decision

Store versioned review profiles under `.pose/review-profiles/` and immutable
attempts under `.pose/reviews/`. Address scopes with `spec:<slug>`,
`milestone:<roadmap>/<id>` and `roadmap:<slug>`. Bind each attempt to a SHA-256
digest of canonical closeout inputs; exclude lifecycle-only fields and the
review artifact itself, but include child closure digests at macro levels.

Derive milestone and roadmap closure from member closure, exit/cut criteria and
their own current approved review. Never infer review approval from narrative
text. Model autonomous continuation as a read-only terminal-state/next-action
projection; execution remains subject to the caller's existing authorization.

Bootstrap this mechanism through an explicit policy adoption date: scopes
started before adoption remain warnings unless configured otherwise. The spec
implementing the mechanism is reviewed with the newly shipped profile before
it transitions to `done`.

## Consequences

- Positive: every closeout decision is auditable and becomes stale when its
  governed inputs change.
- Positive: roadmap blockers resolve to the smallest child scope, criterion or
  finding that needs action.
- Positive: agents can continue objectively until terminal success without
  gaining credentials, destructive authority or permission for external writes.
- Trade-off: remediation creates a new review attempt rather than editing the
  old one, increasing artifact count while preserving history.
- Trade-off: canonicalization is a public compatibility contract and requires
  golden tests whenever its inputs change.
- Neutral: deterministic checks remain necessary evidence; review remains a
  separate judgment layer.
