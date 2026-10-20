# ADR: Expose the closeout execution plan as an opt-in MCP projection

## Status
Accepted for the B07 implementation; independent review pending.

## Context

B07 requires CLI and MCP consumers to retain unavailable Git observations and
materialization conflicts. The CLI already reads `Store.PlanCloseout`; MCP's
`pose_closeout_state` reports review/lifecycle state, a different projection.
Replacing its response would change existing clients and introduce Git as a
new prerequisite for every historical lifecycle query.

## Decision

Add optional `include_plan: true` to `pose_closeout_state`. For a spec scope,
return the unchanged state fields plus `execution_plan`, computed by the same
domain method the CLI uses. Compute the plan first and propagate failures without
returning a successful lifecycle payload. Omission preserves the existing shape.
Non-spec execution plans continue to be refused by the domain.

Keep dispatch historical: a successful final observation records completion,
never current fulfillment or approval. Refuse adapter writes without a trusted
producer/output contract; the existing adapter schema supplies no such authority.

## Consequences

The additive MCP parameter and stricter dispatch refusal are minor-version work.
Existing read clients retain their contract; clients requesting an execution
plan explicitly acquire its Git prerequisites and bounded read failures.
The CLI/MCP share calculation and diagnostics, avoiding a second policy engine.

Rejected: a mandatory execution plan would break offline historical queries.
Rejected: a new MCP planning tool would duplicate discovery and authorization
surface for the same scope. Rejected: directory-prefix output permissions do
not identify a producer and can authorize tampering with sealed evidence.

No sandbox, transaction against external Git writers, policy adoption or
run-to-attestation protocol is introduced. Consumers of future governed effects
must revalidate current inputs; a recorded completion cannot authorize them.

Authority: [B07 spec](../specs/2026-10-13-pose-git-and-dispatch-snapshot-integrity.md), R1–R7.
