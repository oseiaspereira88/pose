---
slug: pose-version-contract
status: done
created_at: 2026-07-18
completed_at: 2026-07-19
supersedes:
depends_on:
priority: 0
---

# Spec: Authoritative version contract

## 1. Intent

### Goal
derive CLI, MCP, registry, scaffold and release metadata from one authoritative release version.
### Business value
Removes a P0 credibility defect and unlocks trustworthy release, package and compatibility automation.
### Constraints
- Preserve development builds and schema-version independence; do not infer compatibility from SemVer alone.
### Non-goals
- Redesign the release cadence or repository schema migration model.

## 2. Requirements

### Functional
- R1: When a release build is produced, every public version surface shall report the same normalized version.
- R2: When a development build runs, the system shall expose an explicit development identifier without impersonating a release.
- R3: A contract test shall enumerate CLI, MCP, registry and release metadata surfaces and fail on divergence.

### Non-functional
- Keep version injection reproducible and free of network access.

### Security
- Do not inject credentials or workflow identity into public version strings.

### Compatibility
- Preserve `pose version` output fields or version them through an ADR.

## 3. Technical Plan

### Affected areas
- `pose-mcp/cmd/`, MCP initialization, `server.json`, GoReleaser and release workflows.

### API/contract changes
- Public version output and MCP `serverInfo.version` become generated from one source.

### Data/storage changes
- No persistent migration; add compatibility metadata only if approved.

### Technical risks
- Build-time injection can make local and packaged binaries diverge unless tests cover both.

### Primary references
- [Semantic Versioning](https://semver.org/)
- [MCP lifecycle](https://modelcontextprotocol.io/specification/2025-06-18/basic/lifecycle)

## 4. Tasks

### Planning
- [x] Confirm baseline and fixtures against [Semantic Versioning](https://semver.org/): CLI `0.9.0-dev`, MCP `0.1.0` (hard-coded), registry `0.9.0` — drift confirmed.

### Implementation
- [x] Define the authority and development-version policy in an ADR: `.pose/adr/2026-07-19-authoritative-release-version-source.md`. ([reference](https://semver.org/))
- [x] Replace hard-coded MCP and registry values with generated metadata: `internal/version.Version` is the single authority; `internal/cli.Version` and `mcpserver` `serverInfo.version` derive from it; GoReleaser stamps the authoritative symbol; `server.json` is pinned to `version.ReleaseBase()` by contract test. ([reference](https://modelcontextprotocol.io/specification/2025-06-18/basic/lifecycle))
- [x] Add release and dirty-tree golden tests across every public version surface: `internal/version/contract_test.go` (dev policy, CLI, registry, GoReleaser ldflags) and initialize assertions for the HTTP and stdio transports in `internal/mcpserver/server_test.go`. ([reference](https://semver.org/))

### Validation
- [x] Run `go test ./pose-mcp/... -run 'Version|Initialize'` and retain the result artifact (see §6 and `.pose/reports/`). ([reference](https://semver.org/))
- [x] Run `pose check --strict` and inspect readiness projections. ([reference](https://modelcontextprotocol.io/specification/2025-06-18/basic/lifecycle))

## 5. Decisions

- ADR `.pose/adr/2026-07-19-authoritative-release-version-source.md` (Accepted): single authoritative Go symbol (`internal/version.Version`) stamped from the git tag via GoReleaser ldflags; development builds carry an explicit `-dev` suffix; checked-in registry metadata must equal `version.ReleaseBase()`; `.pose/schema-version` stays independent of SemVer. Generated-artifact and per-surface-constant alternatives were compared and rejected in the ADR.

## 6. Validation

**Strategy:** validate unit behavior, negative/security cases, public contract fixtures and an end-to-end consumer path.

### Planned deterministic checks
- Test: `go test ./pose-mcp/... -run 'Version|Initialize'`.
- Structure: `pose check --strict`.
- Readiness: `pose lint-spec pose-version-contract --ready-check`.

### Execution status
Executed on 2026-07-19 with a development build (`pose 0.9.0-dev`):

- `go -C pose-mcp test ./internal/version/... ./internal/mcpserver/ -run 'Version|Initialize' -count=1` — SUCCESS.
- `go -C pose-mcp test ./... -count=1` — SUCCESS (full suite, includes the embedded-scaffold drift guard).
- Release path exercised end-to-end: `go build -ldflags "-X github.com/harne8/pose-mcp/internal/version.Version=0.9.0"` produced a binary reporting `pose 0.9.0` on every surface.
- `pose check --strict` — SUCCESS.
- `pose lint-spec pose-version-contract --ready-check` — SUCCESS.
- `pose validate --strict --module pose-mcp --report` — SUCCESS (report retained under `.pose/reports/`).

## 7. Final Report

### Delivered scope

Single authoritative version source (`pose-mcp/internal/version`) consumed by
CLI, MCP (stdio + Streamable HTTP) and telemetry; GoReleaser stamps the
authoritative symbol; contract tests enumerate CLI, MCP, registry
(`server.json`) and release-pipeline surfaces and fail on divergence; explicit
`-dev` development identity; ADR recorded.

### Residual risks

- `server.json` publication to an external MCP registry is not yet automated —
  the contract test pins the checked-in file, but registry submission remains
  manual until the public-accuracy milestone.

### Follow-ups

- [covered: pose-release-compatibility-matrix] Automate registry metadata verification against the released tag in the release workflow.
- [covered: pose-mcp-catalog-conformance] Full MCP catalog/schema conformance (tool drift, protocol completeness).
