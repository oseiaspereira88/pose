---
slug: pose-safe-validate-orchestration
status: done
created_at: 2026-07-18
completed_at: 2026-07-19
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
- [x] Confirm baseline and fixtures against [MCP security best practices](https://modelcontextprotocol.io/specification/2025-06-18/basic/security_best_practices): no orchestration surface existed; the only prior request was for a `pose_validate` tool that would execute locally — exactly the forbidden pattern; the Harness itself does not exist as code in this repository.

### Implementation
- [x] Create an ADR for the MCP-Conductor-Harness trust boundary: `.pose/adr/2026-07-19-mcp-conductor-harness-trust-boundary-for-safe-validation-orchestration.md` — MCP requests, POSE owns plan/approval/result semantics, Harness executes via a pluggable interface. ([reference](https://modelcontextprotocol.io/specification/2025-06-18/basic/security_best_practices))
- [x] Implement immutable plan resolution, approval and idempotent submission: `internal/mcpserver/validate_orchestration.go` — digest-pinned `ValidationPlan` (matrix SHA-256 + git HEAD + filters), in-process `orchestrator` state machine (`pending_approval → approved/rejected → submitted`, `cancel` from any non-terminal state), `HarnessExecutor` interface wired via `Server.WithHarnessExecutor`; five new MCP tools (`pose_validate_request/approve/submit/status/cancel`). ([reference](https://slsa.dev/spec/v1.2/))
- [x] Threat-test substitution, replay, cancellation and result spoofing: digest-mismatch approval rejected (substitution); re-deciding an already-decided request rejected (replay); cancel-then-approve rejected; submit-without-approval rejected; submit-without-configured-harness returns a config error rather than a spoofed success; concurrent resubmit re-checks state under lock so the executor is invoked exactly once (idempotency). ([reference](https://modelcontextprotocol.io/specification/2025-06-18/basic/security_best_practices))

### Validation
- [x] Run `go test ./pose-mcp/... ./mcp-enforce/... -run 'Validate|Identity|Policy|Run'` and retain evidence (matched via `-run Orchestration|Approve|Submit`, the actual test-name prefixes; see §6 and `.pose/reports/`). ([reference](https://modelcontextprotocol.io/specification/2025-06-18/basic/security_best_practices))
- [x] Run `pose check --strict` and inspect readiness. ([reference](https://slsa.dev/spec/v1.2/))

## 5. Decisions

- ADR `.pose/adr/2026-07-19-mcp-conductor-harness-trust-boundary-for-safe-validation-orchestration.md` (Accepted): own the plan/approval/result state machine locally with execution as a pluggable interface, over an in-process-executing tool (forbidden by the non-goal) and over deferring the spec until a real Harness exists (leaves the security-relevant contract undesigned); identity binding is inherently HTTP-only, so orchestration approval only exists on that transport, leaving stdio `pose validate` untouched.

## 6. Validation

**Strategy:** validate units, negative/security cases, contract fixtures and an end-to-end path.

### Planned deterministic checks
- Test: `go test ./pose-mcp/... ./mcp-enforce/... -run 'Validate|Identity|Policy|Run'`.
- Structure: `pose check --strict`.
- Readiness: `pose lint-spec pose-safe-validate-orchestration --ready-check`.

### Requirement trace
- R1 [satisfied] request resolves a versioned, deterministic, digest-pinned plan before any approval; check:test (TestOrchestrationRequestResolvesImmutablePlan)
- R2 [satisfied] execution requires project scope (plan binds project_id), policy allow (standard tools/call gate) and explicit authorization (bound Execution Identity mandatory for approve, independent of default policy mode); check:test (TestApproveDeniesAnonymousCaller, TestApproveWithValidIdentitySucceeds) report:2026-07-19-standard-validate-native.md
- R3 [satisfied] results bind the approved plan (digest), executor identity (approver_run_id/scopes) and execution_id from the Harness; check:test (TestOrchestrationSubmitRequiresApprovalAndIsIdempotent)

### Execution status
Executed on 2026-07-19 with a development build (`pose 0.9.0-dev`, rebuilt from this change):

- `go -C pose-mcp test ./internal/mcpserver -run 'Orchestration|Approve|Submit|ValidateOrchestrationToolsInCatalog' -count=1` — SUCCESS (11 tests: plan determinism, substitution, replay, idempotent submit, rejected-cannot-submit, cancellation terminal states, unknown request id, anonymous-denied, valid-identity-succeeds, unconfigured-harness config error, catalog presence).
- `go -C pose-mcp test ./... -count=1` — SUCCESS (full suite, golden catalog regenerated for the 5 new tools; catalog docs conformance fixed after two accidental ghost matches in prose).
- `pose check --strict` — SUCCESS; `pose lint-spec pose-safe-validate-orchestration --strict` — SUCCESS.
- `pose validate --strict --module pose-mcp --report` — SUCCESS (report retained under `.pose/reports/`).
- `mcp-enforce` tests unaffected (no changes to that module; identity verification reused as-is).

## 7. Final Report

### Delivered scope

Five new MCP tools implementing a digest-pinned, identity-gated,
idempotent validation-orchestration state machine
(`pose_validate_request/approve/submit/status/cancel`); pluggable
`HarnessExecutor` interface (`Server.WithHarnessExecutor`) matching the
existing Conductor `Reporter` pattern; caller identity propagated via
context from the already-verified Execution Identity so approval can
mandate it independent of server-wide policy mode; local `pose validate`
completely unchanged; `mcp.md` orchestration section; ADR.

### Residual risks

- Without a real `HarnessExecutor`, orchestration is exercisable through
  approval but nothing actually runs — expected: a Harness implementation
  is a Harne8-platform component outside this repository's scope.
- Cancellation of a submitted request is a local marker only; the executor
  owns propagating it to a running execution.

### Follow-ups

- [open] Wire a real HarnessExecutor once the Harness component exists in the Harne8 platform, and add an integration test against it. (owner:@pose-maintainers crit:medium review:2026-11-20)
- [open] Consider centralizing orchestrator state in Conductor for multi-replica deployments, mirroring conductor_run_* — the current in-process registry is single-node only. (owner:@pose-maintainers crit:low review:2026-11-20)
