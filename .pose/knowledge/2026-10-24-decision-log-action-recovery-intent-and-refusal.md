---
type: decision-log
slug: action-recovery-intent-and-refusal
owner: @pose-maintainers
sensitivity: public-internal
created_at: 2026-10-24
last_reviewed_at: 2026-10-24
expires_at: 2027-01-22
source_refs:
  spec: "spec:pose-action-request-recovery-and-preflight"
  external_sources: []
---

# Decision log: action recovery intent and refusal

## Context

B10 recovers a B09-authorized predecessor retirement after partial successor creation.

## Current state

Use one immutable intent per predecessor and the existing nonblocking journal lock. A command refuses without recording an action event; previews never sign. Keep incomplete operations visible and restrictive. Reference the prospective ADR for protocol and compatibility.

## Next checks

Review within 90 days, or before introducing transactional storage or persisted refusal auditing. The 90-day TTL covers the consolidation rollout rather than changing durable rules.

## Risks

External filesystem replacement and authenticated historical role membership remain outside the protocol. Corruption requires explicit repair; do not discard reservations or truncate journals automatically.

## Next owner

@pose-maintainers reviews changes to persistence or audit contracts.
