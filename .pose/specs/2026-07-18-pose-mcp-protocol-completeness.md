---
slug: pose-mcp-protocol-completeness
status: done
created_at: 2026-07-18
completed_at: 2026-07-19
supersedes:
depends_on: pose-mcp-project-scope-contract
priority: 21
---

# Spec: Purposeful MCP protocol completeness

## 1. Intent

### Goal
complete lifecycle, pagination, refresh, resources and prompts where they improve governance.
### Business value
Makes POSE reliable across independent MCP clients without bloating tools.
### Constraints
- Adopt protocol primitives by use case, not checklist compliance.
### Non-goals
- Expose repository files wholesale or turn prompts into hidden policy.

## 2. Requirements

### Functional
- R1: Paginated list operations shall use opaque cursors with stable ordering.
- R2: Catalog changes shall define reconnect or list-change behavior.
- R3: Resources and prompts shall serve only bounded read contexts and explicit workflows.

### Non-functional
- Pass conformance under stdio and Streamable HTTP.

### Security
- Apply project policy and sensitivity filtering to every primitive.

### Compatibility
- Negotiate capabilities and preserve tools-only clients.

## 3. Technical Plan

### Affected areas
- MCP lifecycle, transports, capabilities, catalogs, resources/prompts and docs.

### API/contract changes
- Advertise only implemented capabilities and version behavior.

### Data/storage changes
- Add opaque cursor state or stateless encoding with expiry rules.

### Technical risks
- Long-lived sessions can observe stale catalogs.

### Primary references
- [MCP lifecycle](https://modelcontextprotocol.io/specification/2025-06-18/basic/lifecycle)
- [MCP resources specification](https://modelcontextprotocol.io/specification/2025-06-18/server/resources)

## 4. Tasks

### Planning
- [x] Confirm baseline and fixtures against [MCP lifecycle](https://modelcontextprotocol.io/specification/2025-06-18/basic/lifecycle): four `pose_list_*` tools returned unbounded arrays with no pagination; catalog change/reconnect behavior was undocumented; resources/prompts were an open-ended "not yet."

### Implementation
- [x] Map POSE use cases to lifecycle, pagination, resources and prompts: pagination maps to the real large-repository governance-read use case; resources/prompts map to nothing the existing 20 typed tools don't already cover better and more safely (ADR `2026-07-19-purposeful-mcp-protocol-completeness-scope`). ([reference](https://modelcontextprotocol.io/specification/2025-06-18/basic/lifecycle))
- [x] Implement stable cursors plus reconnect/list-change behavior: shared opaque, versioned cursor (`internal/mcpserver/pagination.go`) over each list's existing deterministic order (`ListReports` moved to `sort.SliceStable` to remove a tie-order gap); `next_cursor` additive to all four tools, `cursor`/`limit` omitted preserves the exact pre-pagination shape; catalog stability (no runtime tool add/remove) verified by test, with version-change-triggers-reconnect documented as the practical signal instead of building `list_changed` event machinery for a catalog that cannot change mid-session. ([reference](https://modelcontextprotocol.io/specification/2025-06-18/server/resources))
- [x] Run multi-client conformance over stdio and HTTP: `TestPaginationConsistentAcrossStdioAndHTTP` round-trips a real `dispatchRPC` response through JSON (as `ServeStdio` does on the wire) and asserts identical paging behavior to the HTTP path; `TestPaginationWalksEveryItemExactlyOnce`, omitted-args compatibility and invalid-cursor rejection cover the HTTP transport directly. ([reference](https://modelcontextprotocol.io/specification/2025-06-18/basic/lifecycle))

### Validation
- [x] Run `go test ./pose-mcp/internal/mcpserver/... -run 'Lifecycle|Pagination|Resource|Prompt|HTTP'` and retain evidence (matched via `-run Pagination|ToolCatalogIsStable|ListToolsShare`, the actual test-name prefixes; see §6 and `.pose/reports/`). ([reference](https://modelcontextprotocol.io/specification/2025-06-18/basic/lifecycle))
- [x] Run `pose check --strict` and inspect readiness. ([reference](https://modelcontextprotocol.io/specification/2025-06-18/server/resources))

## 5. Decisions

- ADR `.pose/adr/2026-07-19-purposeful-mcp-protocol-completeness-scope.md` (Accepted): implement pagination (real gap); deliberately do not implement resources/prompts (both would violate the spec's explicit non-goals — wholesale file exposure and hidden-policy prompts); define catalog stability by test and document version-change-as-reconnect-signal instead of building list-changed event machinery for a structurally static catalog.

## 6. Validation

**Strategy:** validate units, negative/security cases, contract fixtures and an end-to-end path.

### Planned deterministic checks
- Test: `go test ./pose-mcp/internal/mcpserver/... -run 'Lifecycle|Pagination|Resource|Prompt|HTTP'`.
- Structure: `pose check --strict`.
- Readiness: `pose lint-spec pose-mcp-protocol-completeness --ready-check`.

### Requirement trace
- R1 [satisfied] opaque, versioned cursors with stable underlying ordering across all four list tools; check:test (TestPaginationWalksEveryItemExactlyOnce, TestPaginationOmittedIsUnpaginatedAndBackwardCompatible, TestPaginationInvalidCursorIsRejected)
- R2 [satisfied] catalog immutability verified by test; reconnect-on-version-change documented as the defined behavior; check:test (TestToolCatalogIsStableWithinAProcessLifetime) report:2026-07-19-standard-validate-native.md
- R3 [waived: resources/prompts deliberately not implemented — see ADR] existing typed tools already serve bounded read contexts and explicit workflows; adding generic resources/prompts primitives would violate this spec's own non-goals.

### Execution status
Executed on 2026-07-19 with a development build (`pose 0.9.0-dev`, rebuilt from this change):

- `go -C pose-mcp test ./internal/mcpserver -run 'Pagination|ToolCatalogIsStable|ListToolsShare' -count=1` — SUCCESS (seven tests, both transports).
- `go -C pose-mcp test ./... -count=1` — SUCCESS (full suite, golden catalog regenerated for the pagination schema additions).
- `pose check --strict` — SUCCESS; `pose lint-spec pose-mcp-protocol-completeness --strict` — SUCCESS.
- `pose validate --strict --module pose-mcp --report` — SUCCESS (report retained under `.pose/reports/`).

## 7. Final Report

### Delivered scope

Opaque cursor pagination (`cursor`/`limit`/`next_cursor`) on all four
`pose_list_*` tools, fully additive and backward compatible; deterministic
tie-order fix in `ListReports`; verified (not merely declared) catalog
stability within a process lifetime with a documented version-change
reconnect signal; a deliberate, ADR-recorded decision not to implement
resources or prompts, closing the open-ended "not yet" in the architectural
limits doc; `mcp.md` and `architecture.md` documentation updates.

### Residual risks

- Position-based cursors can shift if a list is mutated mid-walk by another
  actor — acceptable for POSE's read-mostly governance use case; no
  snapshot-consistency guarantee is claimed or needed.

### Follow-ups

- [open] Revisit the resources/prompts boundary only if a concrete, ADR-worthy use case emerges (e.g. an approved semantic-search adapter) — do not extend it incrementally. (owner:@pose-maintainers crit:low review:2026-11-20)
