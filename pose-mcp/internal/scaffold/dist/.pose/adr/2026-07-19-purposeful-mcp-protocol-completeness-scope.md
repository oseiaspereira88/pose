# ADR: Purposeful MCP protocol completeness scope

## Status
Accepted (2026-07-19) — spec `pose-mcp-protocol-completeness`

## Context

The known architectural limits documented "resources, prompts and live
catalog refresh are not yet part of the public surface" — an open-ended
"not yet" with no criterion for closing it. The spec explicitly constrains
adoption "by use case, not checklist compliance," and its non-goals forbid
exposing repository files wholesale or turning prompts into hidden policy.
Meanwhile the four POSE `pose_list_*` tools return unbounded arrays —
governance use cases (hundreds of specs or reports in a mature repository)
need bounded, resumable reads, which is a real primitive gap the [MCP tools
specification](https://modelcontextprotocol.io/specification/2025-06-18/server/tools)
addresses with cursor pagination.

## Decision

Two decisions, one per remaining gap:

**Pagination (R1) — implement.** `pose_list_specs`, `pose_list_roadmaps`,
`pose_list_knowledge` and `pose_list_reports` accept optional `cursor`/
`limit`; a shared, versioned, base64-opaque position token
(`internal/mcpserver/pagination.go`) walks each list's existing
deterministic order. Omitting both arguments preserves the exact
pre-pagination response — compatibility for tools-only and pagination-naive
clients is by construction, not a fallback branch. `ListReports`' sort was
changed from `sort.Slice` to `sort.SliceStable` so timestamp ties cannot
produce a non-deterministic order a cursor could silently skip past.

**Resources and prompts (R3) — do not implement, by design.** Every
governed read already exists as a typed, schema-validated, project-scoped,
policy-gated tool. Adding MCP `resources` (URI-addressable arbitrary
content) would be exactly "expose repository files wholesale" — the
explicit non-goal. Adding MCP `prompts` (client-invokable templates) risks
encoding procedure outside the reviewable `.pose/workflows/*.md` files,
i.e. "turn prompts into hidden policy" — also the explicit non-goal. R3's
"bounded read contexts and explicit workflows" are satisfied by the
existing `pose_get_workflow`/`pose_get_skill` tools, which already return
exactly that: bounded, reviewable procedure text, gated the same as every
other tool. `capabilities` therefore continues advertising `tools` only.

**Catalog change / reconnect (R2) — define, don't build machinery for.**
The tool catalog is a pure function of the running binary: it cannot change
during a session, so `capabilities.tools.listChanged: false` is verified
true by `TestToolCatalogIsStableWithinAProcessLifetime`, not merely
asserted. A catalog change only happens across a release (new binary, new
`serverInfo.version` — spec `pose-version-contract`); the documented
reconnect signal for a client is observing that version change and
re-`initialize`-ing. Building `notifications/tools/list_changed` machinery
for a catalog that structurally cannot change mid-session would be the
checklist compliance the spec explicitly rejects.

## Consequences

- Positive: governance reads over large repositories are now boundable and
  resumable without any breaking change to existing callers.
- Positive: the open-ended "not yet" limitation becomes a closed, reasoned
  "by design, and here is exactly why" — reviewable and stable, not a
  standing backlog item.
- Trade-off: a future legitimate need for resources or prompts (e.g. an
  approved semantic-search adapter) requires a new ADR revisiting this
  boundary, not a quiet extension of the current tool set.
- Residual: pagination cursors are position-based over each list's existing
  sort; a list mutated between two calls to the same client (items
  added/removed mid-walk) can shift positions — acceptable for POSE's
  read-mostly governance use case and consistent with how MCP cursors are
  generally understood (a snapshot-consistency guarantee was never claimed).
