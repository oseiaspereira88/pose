---
slug: pose-release-compatibility-matrix
status: draft
created_at: 2026-07-18
completed_at:
supersedes:
depends_on: pose-version-contract, pose-public-install-contract
priority: 4
---

# Spec: Release compatibility matrix

## 1. Intent

### Goal
prove engine, instance schema, scaffold, MCP metadata, docs and upgrades for each release candidate.
### Business value
Prevents a nominal release from distributing mutually incompatible parts.
### Constraints
- Separate SemVer compatibility from repository schema compatibility and test both.
### Non-goals
- Promise downgrade support.

## 2. Requirements

### Functional
- R1: A machine-readable matrix shall declare supported engine, schema and upgrade pairs.
- R2: Release CI shall test fresh install and every supported prior-version upgrade.
- R3: Documentation commands and MCP metadata shall be validated against the same candidate artifact.

### Non-functional
- Run fixtures with pinned, authenticated prior artifacts.

### Security
- Verify prior artifacts before executing compatibility tests.

### Compatibility
- Unsupported pairs shall fail with actionable diagnostics.

## 3. Technical Plan

### Affected areas
- Release workflow, migrations, scaffold fixtures, docs checks and MCP metadata.

### API/contract changes
- Publish a compatibility artifact and support policy.

### Data/storage changes
- Version the matrix and retain candidate results.

### Technical risks
- An unbounded version matrix can make release latency unacceptable.

### Primary references
- [Semantic Versioning](https://semver.org/)
- [The Update Framework](https://theupdateframework.io/)

## 4. Tasks

### Planning
- [ ] Confirm baseline and fixtures against [Semantic Versioning](https://semver.org/).

### Implementation
- [ ] Define the support window and compatibility schema. ([reference](https://semver.org/))
- [ ] Build fresh-install and N-minus upgrade fixtures from verified releases. ([reference](https://theupdateframework.io/))
- [ ] Gate release notes and docs on the candidate compatibility report. ([reference](https://semver.org/))

### Validation
- [ ] Run `go test ./pose-mcp/internal/cli/... -run 'Upgrade|Install|Schema'` and retain the result artifact. ([reference](https://semver.org/))
- [ ] Run `pose check --strict` and inspect readiness projections. ([reference](https://theupdateframework.io/))

## 5. Decisions

- Create an ADR before changing this public or structural contract; compare alternatives against [Semantic Versioning](https://semver.org/).

## 6. Validation

**Strategy:** validate unit behavior, negative/security cases, public contract fixtures and an end-to-end consumer path.

### Planned deterministic checks
- Test: `go test ./pose-mcp/internal/cli/... -run 'Upgrade|Install|Schema'`.
- Structure: `pose check --strict`.
- Readiness: `pose lint-spec pose-release-compatibility-matrix --ready-check`.

### Execution status
- Not executed. This planning spec remains `draft`; delivery requires recorded gate evidence.

## 7. Final Report

- Delivered scope: none; this spec defines future implementation.
- Residual risk: An unbounded version matrix can make release latency unacceptable.
- Follow-ups: none until implementation starts.
