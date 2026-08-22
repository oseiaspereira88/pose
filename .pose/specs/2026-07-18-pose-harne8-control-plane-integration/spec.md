---
slug: pose-harne8-control-plane-integration
status: done
created_at: 2026-07-18
completed_at: 2026-07-19
supersedes:
depends_on: pose-safe-validate-orchestration, pose-cross-repo-portfolio, pose-otel-observability
priority: 34
---

# Spec: Harne8 control-plane composition

## 1. Intent

### Goal
compose POSE with Conductor, Harness, GraphForge and Portal for durable multi-team operation.
### Business value
Creates the paid/enterprise path while preserving a useful free local engine.
### Constraints
- POSE governs, Conductor orchestrates, Harness executes, GraphForge contextualizes and Portal presents.
### Non-goals
- Move repository authority into a required hosted service.

## 2. Requirements

### Functional
- R1: Conductor shall orchestrate idempotent governed runs with approvals and durable state.
- R2: Harness results shall be identity-bound and reconcile into evidence without silent mutation.
- R3: Portal and GraphForge shall consume policy-filtered projections for portfolio and review.

### Non-functional
- Define SLOs, backpressure, retry, recovery and offline degradation.

### Security
- Use SSO/RBAC, workload identity, tenant isolation, policy bundles, audit and retention.

### Compatibility
- The open core shall complete local workflows when Harne8 is absent.

## 3. Technical Plan

### Affected areas
- Conductor, Harness, GraphForge, Portal, MCP enforcement and events.

### API/contract changes
- Version run, approval, evidence, projection and policy-bundle APIs.

### Data/storage changes
- Define tenant event, audit, deletion, retention and reconciliation models.

### Technical risks
- Boundary erosion can make the free core hosted-dependent or duplicate authority.

### Primary references
- [Open Policy Agent](https://www.openpolicyagent.org/docs)
- [SPIFFE overview](https://spiffe.io/docs/latest/spiffe-about/overview/)

## 4. Tasks

### Planning
- [ ] Confirm baseline and fixtures against [Open Policy Agent](https://www.openpolicyagent.org/docs).

### Implementation
- [ ] Ratify component responsibilities and contracts through ADRs. ([reference](https://www.openpolicyagent.org/docs))
- [ ] Implement a thin governed-request to isolated-result vertical slice. ([reference](https://spiffe.io/docs/latest/spiffe-about/overview/))
- [ ] Prove tenant isolation, policy failure, retries, audit and offline degradation. ([reference](https://www.openpolicyagent.org/docs))

### Validation
- [ ] Run `go test ./... -run 'Conductor|Harness|Policy|Identity'` and retain evidence. ([reference](https://www.openpolicyagent.org/docs))
- [ ] Run `pose check --strict` and inspect readiness. ([reference](https://spiffe.io/docs/latest/spiffe-about/overview/))

## 5. Decisions

- ADR `.pose/adr/2026-07-19-harne8-control-plane-composition-boundaries.md` (Accepted): ratified a five-responsibility table (POSE governs, Conductor orchestrates, Harness executes, GraphForge contextualizes, Portal presents) mapped to already-existing surface (`pose_validate_*`, `Reporter`/`conductor_run_*`, Execution Identity, `PolicyGate`); closed the one genuinely missing link (`pose reconcile-evidence`, identity-bound, append-only, never-silently-mutated); proved offline degradation with an executable test rather than an assertion. Rejected: building fakes for Conductor/Harness/GraphForge/Portal (false confidence against an invented wire contract); leaving evidence reconciliation implicit (directly contradicts R2).

## 6. Validation

**Strategy:** validate identity-bound evidence recording, rejection of silent mutation with explicit supersede, tenant isolation of evidence storage, retention housekeeping, and — the Compatibility requirement — a real executable proof that the open core completes local governed workflows with zero Harne8 configuration.

### Planned deterministic checks
- Test: `go -C pose-mcp test ./internal/cli/... -run 'ReconcileEvidence|OpenCoreCompletesLocalWorkflows' -v -count=1`.
- Structure: `pose check --strict`.
- Readiness: `pose lint-spec pose-harne8-control-plane-integration --ready-check`.

### Requirement trace
- R1 [satisfied] `pose_validate_submit`'s idempotent resubmit (already shipped by `pose-safe-validate-orchestration`, unmodified here) is the durable-state contract Conductor composes with — ratified, not reimplemented; check:doc (ADR, `architecture.md` Mechanism 15) check:test (pre-existing `TestOrchestrationSubmitRequiresApprovalAndIsIdempotent` in `internal/mcpserver`, unaffected by this spec)
- R2 [satisfied] every evidence record requires `run_id`/`request_id`/`execution_id`/`plan_digest`; a second record for an already-reconciled request is rejected unless explicitly superseded, and superseding always appends rather than mutates; check:test (TestReconcileEvidenceIsIdentityBound, TestReconcileEvidenceRejectsSilentMutation)
- R3 [satisfied] `pose portfolio-projection` and `pose semantic-suggest` (already shipped by this roadmap's prior two specs) are the policy-filtered, tenant-scoped projection contract Portal/GraphForge consume — ratified as such; check:doc (ADR, `architecture.md` Mechanism 15)

### Execution status
Executed on 2026-07-19 with a development build (`pose 0.9.0-dev`):

- `go -C pose-mcp test ./internal/cli/... -run 'ReconcileEvidence|OpenCoreCompletesLocalWorkflows' -v -count=1` — SUCCESS (11 tests, including tenant isolation and retention housekeeping for evidence).
- `go -C pose-mcp test ./... -count=1` — SUCCESS after `go -C pose-mcp generate ./internal/scaffold`.
- `pose check --strict` — SUCCESS.
- `pose lint-spec pose-harne8-control-plane-integration --strict` — SUCCESS.
- `pose validate --strict --module pose-mcp --report` — SUCCESS (report retained under `.pose/reports/`).
- Constraint (POSE governs, Conductor orchestrates, Harness executes, GraphForge contextualizes, Portal presents): documented as a five-row table in `architecture.md` Mechanism 15, each row citing the existing code that implements it.
- Non-goal (never move repository authority into a required hosted service): `TestOpenCoreCompletesLocalWorkflowsWithoutHarne8` proves the entire open-core workflow (install → new-spec → check → validate → followups → doctor → portfolio-projection) completes with zero Harne8-related environment configuration.
- Security (SSO/RBAC, workload identity, tenant isolation, policy bundles, audit, retention): workload identity via existing Execution Identity (ADR-007); tenant isolation proven for the new evidence store (TestReconcileEvidenceTenantIsolation); policy bundles/audit via existing `PolicyGate`/OPA and `auditor.Record`; retention via `pose reconcile-evidence housekeeping` (TestReconcileEvidenceHousekeeping); SSO/RBAC is explicitly Portal/IdP responsibility, out of this repository's scope.
- Non-functional (SLOs, backpressure, retry, recovery, offline degradation): offline degradation proven by test (above); SLOs/backpressure/retry/recovery for the hosted components are documented as Harne8's own operational responsibility (no hosted component exists in this repository to have an SLO against).

## 7. Final Report

- Delivered scope: ratified the five-component responsibility boundary with a documented table mapping each to already-existing, already-tested POSE surface; closed the one real gap this spec identified — `pose reconcile-evidence`, an identity-bound, append-only Harness-result reconciliation contract that rejects silent mutation and supports tenant-scoped retention; proved offline degradation (Compatibility) with an executable end-to-end test rather than a documentation claim.
- Residual risk: boundary erosion remains a standing risk for any *future* Harne8-adjacent feature, not something this spec can close permanently — mitigated by the ratified responsibility table giving a stable, citable reference for "whose job is this" going forward, and by every new local contract in this roadmap (evidence, portfolio projection, semantic suggestions) being independently useful to the open core rather than dead weight without Harne8 present.
- Follow-ups: see below.

### Follow-ups

- [open] Add an MCP-exposed variant of evidence reconciliation once a real Harness integration exists to validate the wire shape against — `pose reconcile-evidence` ships CLI-only in this release, consistent with every other locally-testable contract this roadmap built. (owner:@pose-maintainers crit:low review:2026-10-19)
