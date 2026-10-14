---
title: Review temporal context and source provenance
status: Accepted
date: 2026-10-14
---

# Context

B04 requires historical claim validity distinct from current authorization. The
sealed bundle ADR and authority ADR already keep current issuer pins/grants live.
A local timestamp cannot establish an immutable receipt or historical policy.

# Decision

Evaluate authenticated historical claims at their signed attested_at and disclose
that base as issuer-asserted, never observed acceptance. Verify current pins and
grants separately; their removal prevents reliance and does not fabricate past
revocation. New operations evaluate time at their own explicit now. Equality is
valid at expiry; no expiry remains valid when allowed. Second precision denotes
that second's interval; nanosecond timestamps use their exact instant. No hidden
clock skew or universal TTL is introduced.

Expose explicit verified-source and current-reaffirmation reuse. An omitted mode
preserves legacy reference semantics and is disclosed as legacy reaffirmation,
never a verified historical pass. Verified-source validates the source signature,
bundle, bindings, sealed criteria, findings, decision and recursive provenance.
Use one bounded operation-local traversal; cycle, depth, node, edge or byte limit
is an inconclusive blocker. Reaffirmation requires the current review's judgment.

# Consequences

Additive disclosure and opt-in reuse semantics require minor-release classification.
There is no policy adoption, historical receipt backfill or rewriting of records.
Old declared records retain their sealed rules and never gain verified assurance.
