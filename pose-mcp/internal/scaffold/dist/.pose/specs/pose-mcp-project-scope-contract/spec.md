---
slug: pose-mcp-project-scope-contract
status: draft
created_at: 2026-07-18
completed_at:
supersedes:
depends_on: pose-mcp-catalog-conformance
priority: 20
---

# Spec: MCP project-scope contract

## 1. Intent

### Goal
make project selection explicit and consistent across every multi-root MCP tool.
### Business value
Prevents ambiguous reads and policy decisions as POSE moves beyond one repository.
### Constraints
- A default project is convenience only; ambiguity must fail closed.
### Non-goals
- Introduce tenant storage into the local CLI.

## 2. Requirements

### Functional
- R1: Every project-capable tool shall expose the same `project_id` schema and resolution rules.
- R2: Unknown, unauthorized or ambiguous projects shall return distinct structured errors.
- R3: Tool results and audit events shall identify the resolved project without leaking host paths.

### Non-functional
- Keep single-project stdio ergonomics unchanged.

### Security
- Authorize the resolved project and tool together before repository access.

### Compatibility
- Allow an announced deprecation window for legacy implicit selection.

## 3. Technical Plan

### Affected areas
- MCP schemas, root resolver, bootstrap, policy input, audit and docs.

### API/contract changes
- Standardize `project_id`, error codes and resolution precedence.

### Data/storage changes
- No repository migration; update catalog and policy fixtures.

### Technical risks
- Path-derived roots can bypass logical identity if authorization occurs too late.

### Primary references
- [MCP tools specification](https://modelcontextprotocol.io/specification/2025-06-18/server/tools)
- [MCP authorization specification](https://modelcontextprotocol.io/specification/2025-06-18/basic/authorization)

## 4. Tasks

### Planning
- [ ] Confirm baseline and fixtures against [MCP tools specification](https://modelcontextprotocol.io/specification/2025-06-18/server/tools).

### Implementation
- [ ] Specify resolution precedence and structured project errors. ([reference](https://modelcontextprotocol.io/specification/2025-06-18/server/tools))
- [ ] Apply one project schema and authorization hook to every relevant tool. ([reference](https://modelcontextprotocol.io/specification/2025-06-18/basic/authorization))
- [ ] Test unknown, duplicate, unauthorized, default and discovered roots. ([reference](https://modelcontextprotocol.io/specification/2025-06-18/server/tools))

### Validation
- [ ] Run `go test ./pose-mcp/... -run 'Project|Root|Authorization'` and retain evidence. ([reference](https://modelcontextprotocol.io/specification/2025-06-18/server/tools))
- [ ] Run `pose check --strict` and inspect readiness. ([reference](https://modelcontextprotocol.io/specification/2025-06-18/basic/authorization))

## 5. Decisions

- Create an ADR before changing this contract; compare [MCP tools specification](https://modelcontextprotocol.io/specification/2025-06-18/server/tools).

## 6. Validation

**Strategy:** validate units, negative/security cases, contract fixtures and an end-to-end path.

### Planned deterministic checks
- Test: `go test ./pose-mcp/... -run 'Project|Root|Authorization'`.
- Structure: `pose check --strict`.
- Readiness: `pose lint-spec pose-mcp-project-scope-contract --ready-check`.

### Execution status
- Not executed. This planning spec remains `draft`; delivery requires gate evidence.

## 7. Final Report

- Delivered scope: none; this spec defines future implementation.
- Residual risk: Path-derived roots can bypass logical identity if authorization occurs too late.
- Follow-ups: none until implementation starts.

