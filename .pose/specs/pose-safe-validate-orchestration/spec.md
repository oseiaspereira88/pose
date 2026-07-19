---
slug: pose-safe-validate-orchestration
status: draft
created_at: 2026-07-18
completed_at:
supersedes:
depends_on: pose-mcp-project-scope-contract, pose-structured-validation-results, pose-validation-runtime-guardrails
priority: 22
---

# Spec: Safe validation orchestration

## 1. Intent

### Goal
let agents request validation through approval, policy and isolated Harness execution.
### Business value
Resolves demand for `pose_validate` without unsafe subprocesses in MCP.
### Constraints
- MCP requests work; Harness executes it; POSE owns plan/result semantics.
### Non-goals
- Allow arbitrary commands or silent execution on `tools/call`.

## 2. Requirements

### Functional
- R1: A request shall resolve a versioned check plan before approval.
- R2: Execution shall require project/run scope, policy allow and explicit authorization.
- R3: Results shall bind the approved plan, executor identity and artifact digests.

### Non-functional
- Support cancellation, retries and idempotency.

### Security
- Use expiring identity, sandbox isolation, egress policy and decision audit.

### Compatibility
- Keep local `pose validate` unchanged and expose orchestration separately.

## 3. Technical Plan

### Affected areas
- MCP, Conductor reporter, Harness, validation planner, identity and audit.

### API/contract changes
- Define request, approval, execution and result states plus idempotency.

### Data/storage changes
- Persist minimal run state centrally; keep repository evidence portable.

### Technical risks
- Mutable plan material can turn a constrained tool into remote execution.

### Primary references
- [MCP security best practices](https://modelcontextprotocol.io/specification/2025-06-18/basic/security_best_practices)
- [SLSA 1.2](https://slsa.dev/spec/v1.2/)

## 4. Tasks

### Planning
- [ ] Confirm baseline and fixtures against [MCP security best practices](https://modelcontextprotocol.io/specification/2025-06-18/basic/security_best_practices).

### Implementation
- [ ] Create an ADR for the MCP-Conductor-Harness trust boundary. ([reference](https://modelcontextprotocol.io/specification/2025-06-18/basic/security_best_practices))
- [ ] Implement immutable plan resolution, approval and idempotent submission. ([reference](https://slsa.dev/spec/v1.2/))
- [ ] Threat-test substitution, replay, cancellation and result spoofing. ([reference](https://modelcontextprotocol.io/specification/2025-06-18/basic/security_best_practices))

### Validation
- [ ] Run `go test ./pose-mcp/... ./mcp-enforce/... -run 'Validate|Identity|Policy|Run'` and retain evidence. ([reference](https://modelcontextprotocol.io/specification/2025-06-18/basic/security_best_practices))
- [ ] Run `pose check --strict` and inspect readiness. ([reference](https://slsa.dev/spec/v1.2/))

## 5. Decisions

- Create an ADR before changing this contract; compare [MCP security best practices](https://modelcontextprotocol.io/specification/2025-06-18/basic/security_best_practices).

## 6. Validation

**Strategy:** validate units, negative/security cases, contract fixtures and an end-to-end path.

### Planned deterministic checks
- Test: `go test ./pose-mcp/... ./mcp-enforce/... -run 'Validate|Identity|Policy|Run'`.
- Structure: `pose check --strict`.
- Readiness: `pose lint-spec pose-safe-validate-orchestration --ready-check`.

### Execution status
- Not executed. This planning spec remains `draft`; delivery requires gate evidence.

## 7. Final Report

- Delivered scope: none; this spec defines future implementation.
- Residual risk: Mutable plan material can turn a constrained tool into remote execution.
- Follow-ups: none until implementation starts.
