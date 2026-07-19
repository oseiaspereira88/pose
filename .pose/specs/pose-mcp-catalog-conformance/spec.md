---
slug: pose-mcp-catalog-conformance
status: draft
created_at: 2026-07-18
completed_at:
supersedes:
depends_on: pose-version-contract
priority: 1
---

# Spec: Exact MCP catalog conformance

## 1. Intent

### Goal
make source registry, runtime `tools/list`, schemas and documentation an exact tested contract.
### Business value
Eliminates known ADR/runtime/catalog drift before third-party clients depend on it.
### Constraints
- Keep the current read-oriented security boundary and distinguish optional Conductor tools.
### Non-goals
- Add validation execution or unrelated protocol primitives.

## 2. Requirements

### Functional
- R1: The test suite shall compare canonical tool IDs and input schemas with a golden contract.
- R2: Optional tools shall declare activation conditions and pass enabled and disabled tests.
- R3: Documentation and registry metadata shall be generated or checked against the same catalog.

### Non-functional
- Keep catalog tests deterministic and client-independent.

### Security
- Classify each tool by read, gate or external-side-effect risk.

### Compatibility
- Any removal or incompatible schema change requires an ADR and release note.

## 3. Technical Plan

### Affected areas
- `pose-mcp/internal/mcpserver/`, `server.json`, MCP docs and registry packaging.

### API/contract changes
- The runtime catalog becomes a release-gated public API.

### Data/storage changes
- Store a versioned golden catalog fixture.

### Technical risks
- Golden files can normalize real drift if updates are not review-gated.

### Primary references
- [MCP tools specification](https://modelcontextprotocol.io/specification/2025-06-18/server/tools)
- [Official MCP Registry](https://registry.modelcontextprotocol.io/)

## 4. Tasks

### Planning
- [ ] Confirm baseline and fixtures against [MCP tools specification](https://modelcontextprotocol.io/specification/2025-06-18/server/tools).

### Implementation
- [ ] Inventory runtime and optional catalogs with explicit risk classes. ([reference](https://modelcontextprotocol.io/specification/2025-06-18/server/tools))
- [ ] Generate or verify registry metadata and docs from the canonical catalog. ([reference](https://registry.modelcontextprotocol.io/))
- [ ] Exercise `tools/list` and invalid calls against exact JSON Schemas. ([reference](https://modelcontextprotocol.io/specification/2025-06-18/server/tools))

### Validation
- [ ] Run `go test ./pose-mcp/internal/mcpserver/... -run 'Catalog|ToolsList|Schema'` and retain the result artifact. ([reference](https://modelcontextprotocol.io/specification/2025-06-18/server/tools))
- [ ] Run `pose check --strict` and inspect readiness projections. ([reference](https://registry.modelcontextprotocol.io/))

## 5. Decisions

- Create an ADR before changing this public or structural contract; compare alternatives against [MCP tools specification](https://modelcontextprotocol.io/specification/2025-06-18/server/tools).

## 6. Validation

**Strategy:** validate unit behavior, negative/security cases, public contract fixtures and an end-to-end consumer path.

### Planned deterministic checks
- Test: `go test ./pose-mcp/internal/mcpserver/... -run 'Catalog|ToolsList|Schema'`.
- Structure: `pose check --strict`.
- Readiness: `pose lint-spec pose-mcp-catalog-conformance --ready-check`.

### Execution status
- Not executed. This planning spec remains `draft`; delivery requires recorded gate evidence.

## 7. Final Report

- Delivered scope: none; this spec defines future implementation.
- Residual risk: Golden files can normalize real drift if updates are not review-gated.
- Follow-ups: none until implementation starts.
