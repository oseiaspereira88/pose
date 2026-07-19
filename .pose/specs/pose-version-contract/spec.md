---
slug: pose-version-contract
status: draft
created_at: 2026-07-18
completed_at:
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
- [ ] Confirm baseline and fixtures against [Semantic Versioning](https://semver.org/).

### Implementation
- [ ] Define the authority and development-version policy in an ADR. ([reference](https://semver.org/))
- [ ] Replace hard-coded MCP and registry values with generated metadata. ([reference](https://modelcontextprotocol.io/specification/2025-06-18/basic/lifecycle))
- [ ] Add release and dirty-tree golden tests across every public version surface. ([reference](https://semver.org/))

### Validation
- [ ] Run `go test ./pose-mcp/... -run 'Version|Initialize'` and retain the result artifact. ([reference](https://semver.org/))
- [ ] Run `pose check --strict` and inspect readiness projections. ([reference](https://modelcontextprotocol.io/specification/2025-06-18/basic/lifecycle))

## 5. Decisions

- Create an ADR before changing this public or structural contract; compare alternatives against [Semantic Versioning](https://semver.org/).

## 6. Validation

**Strategy:** validate unit behavior, negative/security cases, public contract fixtures and an end-to-end consumer path.

### Planned deterministic checks
- Test: `go test ./pose-mcp/... -run 'Version|Initialize'`.
- Structure: `pose check --strict`.
- Readiness: `pose lint-spec pose-version-contract --ready-check`.

### Execution status
- Not executed. This planning spec remains `draft`; delivery requires recorded gate evidence.

## 7. Final Report

- Delivered scope: none; this spec defines future implementation.
- Residual risk: Build-time injection can make local and packaged binaries diverge unless tests cover both.
- Follow-ups: none until implementation starts.
