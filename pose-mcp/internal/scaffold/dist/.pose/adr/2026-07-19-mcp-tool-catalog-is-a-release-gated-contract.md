# ADR: MCP tool catalog is a release-gated contract

## Status
Accepted (2026-07-19) — spec `pose-mcp-catalog-conformance`

## Context

The MCP surface (21 tools: 18 POSE + 3 optional Conductor) had no frozen
contract. The capability assessment recorded real drift: an earlier ADR
referenced a `pose_validate` tool that the runtime never advertised, the MCP
documentation omitted the Conductor tools entirely, and nothing prevented a
code change from silently renaming a tool or narrowing an input schema that
third-party clients depend on. The
[MCP tools specification](https://modelcontextprotocol.io/specification/2025-06-18/server/tools)
treats the advertised catalog and its JSON Schemas as the discovery contract.

Alternatives considered:

1. **Documentation discipline only** — already failed; docs and ADRs drifted
   from the runtime with no failing signal.
2. **Generate docs and registry metadata from the runtime at build time** —
   removes drift but hides the contract from review; a code change would
   rewrite the "contract" it should be judged against.
3. **Golden fixture + conformance tests** — the runtime catalog is serialized
   into a reviewed fixture; any divergence between runtime, fixture, docs and
   registry metadata fails `go test`.

## Decision

Option 3. The catalog contract:

- `internal/mcpserver/testdata/tool-catalog.golden.json` freezes the exact
  `tools/list` payload (names, descriptions, input schemas) plus a governance
  record per tool. It is updated only via
  `go test ./internal/mcpserver -run Golden -update` followed by human review
  of the diff — golden updates are review-gated, never automatic.
- Every tool declares a **risk class**: `read` (repository-owned governance
  state), `gate` (deterministic local gates, no writes/network) or
  `external-side-effect` (emits events to an external system).
- **Optional tools** (`conductor_run_*`) are always advertised, declare an
  explicit activation condition (Conductor reporter configuration) and must
  pass tests in both the enabled and disabled paths. Disabled calls return
  `isError` with configuration guidance, not protocol errors.
- **Docs conformance:** `docs-site/docs/mcp.md` must list exactly the
  advertised tool names — no undocumented tools, no documented ghosts
  (`TestCatalogDocsConformance`). The historical `pose_validate` reference is
  resolved by this rule: validation execution stays CLI-side and is not an
  MCP tool.
- **Registry conformance:** `server.json` name, transport and version fields
  are asserted against the runtime identity and the authoritative version
  (with `internal/version/contract_test.go`).
- **Removals or incompatible schema changes** require an ADR and a release
  note before the golden may be updated. Additive, backward-compatible fields
  require only the reviewed golden diff plus docs.

## Consequences

- Positive: catalog drift (rename, schema change, undocumented tool) fails the
  test suite at development time; third-party clients get a stable, versioned
  discovery contract tied to the release version.
- Positive: risk classes give reviewers and policy authors (OPA) a stable
  vocabulary for authorization decisions.
- Trade-off: intentional catalog evolution now costs a golden regeneration and
  review; that friction is the point — the golden diff is the reviewable
  contract change.
- Residual: golden updates could still normalize drift if reviews rubber-stamp
  them; CONTRIBUTING requires the reviewer to treat golden diffs as public
  API changes.
