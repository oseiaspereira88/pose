---
slug: pose-stack-catalog-expansion
status: draft
created_at: 2026-07-18
completed_at:
supersedes:
depends_on: pose-structured-validation-results
priority: 18
---

# Spec: Polyglot stack catalog

## 1. Intent

### Goal
add maintained profiles for Python, .NET and modern build ecosystems.
### Business value
Expands addressable repositories without low-level setup.
### Constraints
- Delegate to native tools and never download dependencies implicitly.
### Non-goals
- Certify every framework or replace overrides.

## 2. Requirements

### Functional
- R1: Python and .NET profiles shall detect standard markers and propose checks.
- R2: Profiles shall declare prerequisites, confidence and override behavior.
- R3: Fixtures shall cover absent tools, multiple managers and conflicting markers.

### Non-functional
- Keep detection offline, bounded and deterministic.

### Security
- Never execute project files during detection.

### Compatibility
- Existing Node.js, Go, Rust and Java selection remains unchanged.

## 3. Technical Plan

### Affected areas
- Detection, wizard, matrix, docs and fixtures.

### API/contract changes
- Version profile IDs and default check semantics.

### Data/storage changes
- Add maintainer, status and compatibility metadata.

### Technical risks
- Defaults can be expensive in large repos; expose overrides.

### Primary references
- [Python Packaging User Guide](https://packaging.python.org/)
- [.NET CLI build](https://learn.microsoft.com/en-us/dotnet/core/tools/dotnet-build)

## 4. Tasks

### Planning
- [ ] Confirm baseline and fixtures against [Python Packaging User Guide](https://packaging.python.org/).

### Implementation
- [ ] Define profile lifecycle, support tiers and conflicts. ([reference](https://packaging.python.org/))
- [ ] Implement Python and .NET profiles without implicit installs. ([reference](https://learn.microsoft.com/en-us/dotnet/core/tools/dotnet-build))
- [ ] Add fixture compatibility tests and generated profile docs. ([reference](https://packaging.python.org/))

### Validation
- [ ] Run `go test ./pose-mcp/internal/cli/... -run 'Stack|Wizard|Matrix'` and retain the result artifact. ([reference](https://packaging.python.org/))
- [ ] Run `pose check --strict` and inspect readiness. ([reference](https://learn.microsoft.com/en-us/dotnet/core/tools/dotnet-build))

## 5. Decisions

- Create an ADR before changing this contract; compare alternatives against [Python Packaging User Guide](https://packaging.python.org/).

## 6. Validation

**Strategy:** validate unit behavior, negative/security cases, contract fixtures and an end-to-end consumer path.

### Planned deterministic checks
- Test: `go test ./pose-mcp/internal/cli/... -run 'Stack|Wizard|Matrix'`.
- Structure: `pose check --strict`.
- Readiness: `pose lint-spec pose-stack-catalog-expansion --ready-check`.

### Execution status
- Not executed. This planning spec remains `draft`; delivery requires gate evidence.

## 7. Final Report

- Delivered scope: none; this spec defines future implementation.
- Residual risk: Defaults can be expensive in large repos; expose overrides.
- Follow-ups: none until implementation starts.
