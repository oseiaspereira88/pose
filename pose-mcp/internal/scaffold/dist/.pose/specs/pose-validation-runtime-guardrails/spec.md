---
slug: pose-validation-runtime-guardrails
status: draft
created_at: 2026-07-18
completed_at:
supersedes:
depends_on: pose-structured-validation-results
priority: 17
---

# Spec: Validation runtime guardrails and Harness isolation

## 1. Intent

### Goal
bound local checks and delegate untrusted execution through an isolated Harness contract.
### Business value
Prevents hung or hostile checks while preserving POSE as governance layer.
### Constraints
- Portable local limits vary by OS; isolation belongs to Harness.
### Non-goals
- Turn the CLI into a remote code execution service.

## 2. Requirements

### Functional
- R1: Checks shall support timeout, output limit and cancellation with explicit states.
- R2: Policy shall classify checks requiring isolated execution.
- R3: Remote plans shall bind project, spec, check plan, digests and approval identity.

### Non-functional
- Terminate child process groups where the platform permits.

### Security
- Use least privilege, immutable inputs, network policy and expiring identity.

### Compatibility
- Existing checks receive documented safe defaults.

## 3. Technical Plan

### Affected areas
- Validation runner/matrix, results and Harness boundary.

### API/contract changes
- Add runtime limits and an authorized execution-plan envelope.

### Data/storage changes
- Persist limits, executor identity and result digests.

### Technical risks
- Platform process semantics can leave descendants after timeout.

### Primary references
- [Go context cancellation](https://pkg.go.dev/context)
- [SLSA 1.2](https://slsa.dev/spec/v1.2/)

## 4. Tasks

### Planning
- [ ] Confirm baseline and fixtures against [Go context cancellation](https://pkg.go.dev/context).

### Implementation
- [ ] Specify timeout, cancellation, output and isolation semantics. ([reference](https://pkg.go.dev/context))
- [ ] Implement process-group guardrails and structured failures. ([reference](https://slsa.dev/spec/v1.2/))
- [ ] Threat-test the Harness envelope, policy and result binding. ([reference](https://pkg.go.dev/context))

### Validation
- [ ] Run `go test ./pose-mcp/internal/cli/... -run 'Timeout|Cancel|Validate'` and retain the result artifact. ([reference](https://pkg.go.dev/context))
- [ ] Run `pose check --strict` and inspect readiness. ([reference](https://slsa.dev/spec/v1.2/))

## 5. Decisions

- Create an ADR before changing this contract; compare alternatives against [Go context cancellation](https://pkg.go.dev/context).

## 6. Validation

**Strategy:** validate unit behavior, negative/security cases, contract fixtures and an end-to-end consumer path.

### Planned deterministic checks
- Test: `go test ./pose-mcp/internal/cli/... -run 'Timeout|Cancel|Validate'`.
- Structure: `pose check --strict`.
- Readiness: `pose lint-spec pose-validation-runtime-guardrails --ready-check`.

### Execution status
- Not executed. This planning spec remains `draft`; delivery requires gate evidence.

## 7. Final Report

- Delivered scope: none; this spec defines future implementation.
- Residual risk: Platform process semantics can leave descendants after timeout.
- Follow-ups: none until implementation starts.

