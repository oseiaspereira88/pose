# ADR: Seed the first amendment from the recorded start

## Status
Accepted (2026-10-24), spec `pose-amendment-history-holds-before-close`.

## Context
The first non-baseline amendment currently acknowledges only changed IDs. Existing start nodes then look newly added. Closing can publish a state that the next lint refuses. The append-only history ADR (2026-07-19) remains authoritative: no journal means no history gate; schema 1 is requirement-only, schema 2 requires explicit contract-node adoption.

## Decision
Seed a first non-baseline amendment from the existing valid completed start record, never from current edited text. Preserve original node hashes/state/relations and filter to requirements for schema 1. Record the seed now, with the current declared author and rationale identifying the recorded start; do not invent past approval or an authenticated timestamp. Validate the requested change before appending either event. Explicit --baseline keeps its existing current-state meaning. An existing nonempty journal is never rewritten or silently repaired. Invalid/incomplete/wrong-spec/tampered start records refuse seeding.

Serialize amendment writes with the existing per-spec lifecycle lock used by start/close; append baseline then change and Sync. A failed or torn append remains visible and requires explicit repair; there is no retry loop, truncation or rollback of published history. A durable baseline alone can be followed by a later explicit change retry.

Expose a readonly shared history assessment: missing/empty history remains opt-out, malformed history refuses, existing history uses the adopted schema gate. Lint reports pending findings as warnings on in-progress specs and errors on done specs; close checks before transition and again under the lifecycle lock. No automatic baseline, waiver, review or policy adoption at close.

## Consequences
Fix the existing promised close/lint consistency without changing schemas. Closing invalid opted-in histories is stricter and must be described in the changelog; no retrospective gate is imposed on specs without journals. Manual filesystem mutation remains outside cooperative locking guarantees. Rejected alternatives: current-text seed would erase evidence of the change; repairing old logs would violate append-only history; a second amendment taxonomy or journal would duplicate the contract. B11 terminal plan verification remains a separate spec and consumes this gate.
