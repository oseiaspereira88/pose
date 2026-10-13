---
type: decision-log
slug: action-answer-proof-and-historical-time
sensitivity: public-internal
created_at: 2026-10-13
owner: "@pose-maintainers"
last_reviewed_at: 2026-10-13
expires_at: 2026-11-13
source_refs:
  spec: "pose-action-answers-reverify-on-read"
  external_sources: []
---

# Decision log: Persisted proof and historical action acceptance

## Context

Backlog B01 found that the shared action reader propagated assurance labels while signature verification ran only in a textual renderer. Revalidating issuer expiry against wall clock on every read would instead discard valid historical answers.

## Decision

Verify proof and request/answer bindings in the domain reader and reject invalid evidence before deriving satisfaction. Reuse the issuer verifier with explicit write time and recorded acceptance precision on reads. Keep current issuer pins/grants as the existing revocation boundary. Preserve SSH journal-only verification after registered-key rotation.

## Rationale

A stored label is not a cryptographic result. A signature authenticates signed bytes; it does not make unsigned presence metadata authoritative. Acceptance time and present-day trust are separate inputs. Legacy second-resolution timestamps describe an interval, so historical validation uses its bounded precision rather than pretending a nanosecond clock was retained.

## Consequences

Invalid proof remains on disk but its consumer receives an explicit error/incomplete coverage. Declared legacy answers remain declared. This scoped fix does not introduce an immutable historical authorization receipt, reauthorize old role membership, or complete the broader review-source reuse contract. Regression tests cover both emitter paths and consumer degradation.

## References

- spec:pose-action-answers-reverify-on-read
- .pose/reports/2026-10-11-pose-front-02-trust-chain.md
- .pose/reports/2026-10-12-pose-front-10-consolidated-backlog.md
