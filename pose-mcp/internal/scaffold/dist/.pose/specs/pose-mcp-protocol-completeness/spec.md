---
slug: pose-mcp-protocol-completeness
status: draft
created_at: 2026-07-18
completed_at:
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
- [ ] Confirm baseline and fixtures against [MCP lifecycle](https://modelcontextprotocol.io/specification/2025-06-18/basic/lifecycle).

### Implementation
- [ ] Map POSE use cases to lifecycle, pagination, resources and prompts. ([reference](https://modelcontextprotocol.io/specification/2025-06-18/basic/lifecycle))
- [ ] Implement stable cursors plus reconnect/list-change behavior. ([reference](https://modelcontextprotocol.io/specification/2025-06-18/server/resources))
- [ ] Run multi-client conformance over stdio and HTTP. ([reference](https://modelcontextprotocol.io/specification/2025-06-18/basic/lifecycle))

### Validation
- [ ] Run `go test ./pose-mcp/internal/mcpserver/... -run 'Lifecycle|Pagination|Resource|Prompt|HTTP'` and retain evidence. ([reference](https://modelcontextprotocol.io/specification/2025-06-18/basic/lifecycle))
- [ ] Run `pose check --strict` and inspect readiness. ([reference](https://modelcontextprotocol.io/specification/2025-06-18/server/resources))

## 5. Decisions

- Create an ADR before changing this contract; compare [MCP lifecycle](https://modelcontextprotocol.io/specification/2025-06-18/basic/lifecycle).

## 6. Validation

**Strategy:** validate units, negative/security cases, contract fixtures and an end-to-end path.

### Planned deterministic checks
- Test: `go test ./pose-mcp/internal/mcpserver/... -run 'Lifecycle|Pagination|Resource|Prompt|HTTP'`.
- Structure: `pose check --strict`.
- Readiness: `pose lint-spec pose-mcp-protocol-completeness --ready-check`.

### Execution status
- Not executed. This planning spec remains `draft`; delivery requires gate evidence.

## 7. Final Report

- Delivered scope: none; this spec defines future implementation.
- Residual risk: Long-lived sessions can observe stale catalogs.
- Follow-ups: none until implementation starts.
