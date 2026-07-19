---
slug: pose-mcp-catalog-conformance
status: done
created_at: 2026-07-18
completed_at: 2026-07-19
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
- [x] Confirm baseline and fixtures against [MCP tools specification](https://modelcontextprotocol.io/specification/2025-06-18/server/tools): 21 tools (18 POSE + 3 Conductor); Conductor tools undocumented in `mcp.md`; no golden contract; historical `pose_validate` ADR/runtime drift confirmed.

### Implementation
- [x] Inventory runtime and optional catalogs with explicit risk classes: `internal/mcpserver/catalog.go` classifies every tool as `read`, `gate` or `external-side-effect`; Conductor tools are `optional` with a declared activation condition. ([reference](https://modelcontextprotocol.io/specification/2025-06-18/server/tools))
- [x] Generate or verify registry metadata and docs from the canonical catalog: `TestCatalogDocsConformance` requires `docs-site/docs/mcp.md` to list exactly the advertised tool names (Conductor tools now documented); `TestCatalogRegistryConformance` pins `server.json` name and stdio transport. ([reference](https://registry.modelcontextprotocol.io/))
- [x] Exercise `tools/list` and invalid calls against exact JSON Schemas: `TestCatalogMatchesGolden` (byte-exact golden of names, descriptions and input schemas, review-gated `-update`), `TestCatalogGovernanceBijection`, `TestCatalogRequiredArgumentsEnforced` (negative path for every tool with required args); enabled/disabled Conductor paths covered by existing reporter tests plus `TestConductorRunOpen_ReporterNotConfigured`. ([reference](https://modelcontextprotocol.io/specification/2025-06-18/server/tools))

### Validation
- [x] Run `go test ./pose-mcp/internal/mcpserver/... -run 'Catalog|ToolsList|Schema'` and retain the result artifact (see §6 and `.pose/reports/`). ([reference](https://modelcontextprotocol.io/specification/2025-06-18/server/tools))
- [x] Run `pose check --strict` and inspect readiness projections. ([reference](https://registry.modelcontextprotocol.io/))

## 5. Decisions

- ADR `.pose/adr/2026-07-19-mcp-tool-catalog-is-a-release-gated-contract.md` (Accepted): golden fixture + conformance tests over docs-generation or documentation discipline; risk-class vocabulary; optional-tool activation semantics; removals/incompatible changes require ADR + release note; `pose_validate` drift resolved by declaring validation CLI-side, not an MCP tool.

## 6. Validation

**Strategy:** validate unit behavior, negative/security cases, public contract fixtures and an end-to-end consumer path.

### Planned deterministic checks
- Test: `go test ./pose-mcp/internal/mcpserver/... -run 'Catalog|ToolsList|Schema'`.
- Structure: `pose check --strict`.
- Readiness: `pose lint-spec pose-mcp-catalog-conformance --ready-check`.

### Execution status
Executed on 2026-07-19 with a development build (`pose 0.9.0-dev`):

- `go -C pose-mcp test ./internal/mcpserver -run 'Catalog|ToolsList|Schema|Initialize' -count=1` — SUCCESS.
- `go -C pose-mcp test ./internal/mcpserver -count=1` — SUCCESS (full package).
- `pose check --strict` — SUCCESS.
- `pose lint-spec pose-mcp-catalog-conformance --ready-check` — SUCCESS.
- `pose validate --strict --module pose-mcp --report` — SUCCESS (report retained under `.pose/reports/`).

## 7. Final Report

### Delivered scope

Versioned golden catalog fixture (`testdata/tool-catalog.golden.json`)
freezing the exact `tools/list` payload plus per-tool governance (risk class,
optional flag, activation); bijection and negative-path tests; docs and
registry conformance tests; Conductor tools documented in `mcp.md` with
activation conditions; catalog governance ADR; CONTRIBUTING review rule for
golden diffs.

### Residual risks

- Golden updates can still normalize real drift if reviewers rubber-stamp
  them; mitigated by the CONTRIBUTING rule treating golden diffs as public
  API changes and by the ADR requirement for removals.

### Follow-ups

- [covered: pose-mcp-protocol-completeness] Pagination, resources/prompts and refresh/reconnect contract for the MCP surface.
- [covered: pose-release-compatibility-matrix] Verify the published registry entry against each released tag.
