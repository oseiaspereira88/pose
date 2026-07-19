---
slug: pose-validation-runtime-guardrails
status: done
created_at: 2026-07-18
completed_at: 2026-07-19
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
- [x] Confirm baseline and fixtures against [Go context cancellation](https://pkg.go.dev/context): checks ran unbounded (no timeout, no output ceiling); no isolation classification; no Harness delegation contract existed.

### Implementation
- [x] Specify timeout, cancellation, output and isolation semantics: per-check `timeoutSeconds` with matrix defaults (600s safe default), `defaults.maxOutputBytes` (1 MiB safe default), `isolation: "required"` classification (ADR `2026-07-19-validation-runtime-guardrails-and-harness-delegation`). ([reference](https://pkg.go.dev/context))
- [x] Implement process-group guardrails and structured failures: `context.WithTimeout` + `Cmd.Cancel` killing the whole process group on Unix (documented fallback on other platforms); `outputLimiter` cancels on ceiling breach; both produce explicit `outcome: error` with `limit_state: timeout|output-limit`, never conflated with check failures. ([reference](https://slsa.dev/spec/v1.2/))
- [x] Threat-test the Harness envelope, policy and result binding: `--emit-plan` writes `executionPlan` binding project id, spec, git HEAD, matrix SHA-256 digest and an approval slot (`required: true`, identity/expiry unstamped); isolation-required checks always skip locally with a machine-readable reason and never execute in the CLI boundary. ([reference](https://pkg.go.dev/context))

### Validation
- [x] Run `go test ./pose-mcp/internal/cli/... -run 'Timeout|Cancel|Validate'` and retain the result artifact (see §6 and `.pose/reports/`). ([reference](https://pkg.go.dev/context))
- [x] Run `pose check --strict` and inspect readiness. ([reference](https://slsa.dev/spec/v1.2/))

## 5. Decisions

- ADR `.pose/adr/2026-07-19-validation-runtime-guardrails-and-harness-delegation.md` (Accepted): portable local guardrails + explicit Harness delegation over no local limits (coarse CI-only timeouts) and over local sandboxing (weakens the CLI boundary the roadmap forbids); guardrail states are always explicit, never silent.

## 6. Validation

**Strategy:** validate unit behavior, negative/security cases, contract fixtures and an end-to-end consumer path.

### Planned deterministic checks
- Test: `go test ./pose-mcp/internal/cli/... -run 'Timeout|Cancel|Validate'`.
- Structure: `pose check --strict`.
- Readiness: `pose lint-spec pose-validation-runtime-guardrails --ready-check`.

### Requirement trace
- R1 [satisfied] timeout and output-limit produce explicit cancellation states with duration; check:test (TestGuardrailTimeoutState, TestGuardrailOutputLimitState)
- R2 [satisfied] isolation: required checks are classified and never executed locally; check:test (TestGuardrailIsolationDelegation)
- R3 [satisfied] execution plan binds project, spec, matrix digest and an approval slot; check:test (TestGuardrailIsolationDelegation) report:2026-07-19-standard-validate-native.md

### Execution status
Executed on 2026-07-19 with a development build (`pose 0.9.0-dev`, rebuilt from this change):

- `go -C pose-mcp test ./internal/cli -run 'Guardrail' -count=1` — SUCCESS (timeout test observes ~1s cancellation, confirming the process group is actually killed).
- `go -C pose-mcp test ./... -count=1` — SUCCESS (full suite).
- `pose check --strict` — SUCCESS; `pose lint-spec pose-validation-runtime-guardrails --strict` — SUCCESS.
- `pose validate --strict --module pose-mcp --report` — SUCCESS (report retained under `.pose/reports/`).

## 7. Final Report

### Delivered scope

Per-check timeout and output-ceiling guardrails with explicit,
non-conflated failure states; process-group cancellation on Unix with a
documented Windows fallback; `isolation: "required"` classification that
never runs locally; `--emit-plan` execution-plan envelope binding project,
spec, git HEAD, matrix digest and an unstamped approval slot for Harness
delegation; operating-manual documentation and ADR.

### Residual risks

- On non-Unix platforms, descendants of a killed child may survive the
  timeout — documented limitation; untrusted workloads belong in the
  Harness path, not the local CLI.

### Follow-ups

- [open] Wire the approval-identity stamping (ADR-007 execution identity) once Conductor exposes the endpoint, so emitted plans become executable by the Harness. (owner:@pose-maintainers crit:medium review:2026-10-16)
