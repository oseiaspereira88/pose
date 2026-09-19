# ADR: Governance outcomes remain append-only projections

## Status
Proposed

## Context

POSE already owns append-only report history, sealed review bundles and
attestations. A governance report must distinguish what was observed from what
is missing; adding a second event journal or deriving a quality score would
create both privacy risk and an incentive to optimize a proxy. The first ABM
observability slice also has no adopted remediation event contract, so it must
not infer lineage from similar text or from best-effort usage telemetry.

## Decision

Expose governance outcomes as a bounded, read-only projection over the existing
local artifacts. Keep CLI and MCP on the same `GovernanceOutcomesReport` schema,
version the projection, preserve invalid/unknown coverage, and keep first
approval, attempts, findings, stale state, duration and cost as separate
dimensions. Do not persist a new event, call a model, send data over the
network, or aggregate identities. Add remediation lineage only in a later
contract that provides explicit refs and cycle/orphan validation.

## Consequences

- Existing history remains the authority and can be replayed without mutation.
- A project with old or incomplete history receives an honest partial report,
  not a fabricated zero or a false causal conclusion.
- Harne8 can consume the projection without granting it closeout authority;
  adoption and policy gates remain separate.
- A later lineage spec must define its own producer and migration before R4/R5
  can move from deferred to satisfied.
