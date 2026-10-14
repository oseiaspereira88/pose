---
type: decision-log
slug: adr-shared-worktree-runtime-review
owner: @pose-maintainers
sensitivity: public-internal
created_at: 2026-10-14
last_reviewed_at: 2026-10-14
expires_at: 2027-01-12
source_refs:
  spec: "pose-consolidation-batch-14-authoring"
  workflow: "documentation-update"
  commands: ["pose lint-spec pose-consolidation-batch-14-authoring --strict"]
  external_sources: []  # [{url: "", accessed_at: "YYYY-MM-DD"}]
---

# decision-log: adr-shared-worktree-runtime-review

## Context

Review the [domain/runtime proposal](../adr/2026-10-14-shared-domain-and-optional-worktree-runtime.md)
for B31/B33/B35 before functional start, preserving the accepted trust, disclosure
and scoped-reader boundaries.

## Current state

Last reviewed: 2026-10-14. Proposal is Proposed; specs draft, no IPC/cache/runtime
or adoption implemented. TTL of 90 days covers staged documentary authoring and
the contract review before functional execution; reassess when design/scope changes.
The durable rationale stays in the ADR, not in this expiring operational note.

## Next checks

Resolve interface/envelope/version compatibility, route/producer inventory, local
IPC identity/permissions/platforms, cache accounting/view ownership and measured
QoS thresholds. Execute planned mutation/race/parity/attach/fallback cases only
when the implementation scope is authorized. Lint/merge do not qualify runtime.

## Risks

Shared mutable fields can contaminate reads; entry counts do not bound RAM. MCP
stdio/UID/cache hits are not authority; watch hints are not current generation.
Ambiguous effects cannot be retried in fallback without receipt/reconciliation.

## Next owner

@pose-maintainers under the existing contract authority, without a new universal
human confirmation or default runtime adoption.

## References

- [Authoring spec](../specs/2026-10-14-pose-consolidation-batch-14-authoring.md)
- [B31](../specs/2026-10-14-pose-common-domain-and-cli-mcp-handoff.md)
- [B33](../specs/2026-10-14-pose-bounded-isolated-graph-caches.md)
- [B35](../specs/2026-10-14-pose-optional-worktree-runtime.md)
