---
slug: pose-effective-governance-projection
status: in-progress
created_at: 2026-10-04
completed_at:
supersedes:
depends_on: pose-legacy-contract-cutoffs, pose-review-assurance-disclosure
priority: 1
components: pose-mcp
delivers: surface:effective-governance
task_type: feature
---

# Spec: Project supported, configured, applicable and effective governance

## 1. Intent

### Goal

Show, per capability and contract, whether the engine supports it, the policy configures
it, it applies to a scope or bundle, and it is effective — with the reason when it is
not.

### Business value

Engine version does not say which gates apply to an instance, scope and bundle. Atomic
start, contract nodes and causality closeout are implemented and not adopted; DoR has an
empty `adopted_at`; identity assurance resolves to declared. Users do feature
archaeology across schema, registry, adoptions, legacy dates and capability versions.

### Constraints

Pure introspection over the registry, capability fields and the effective plan; no
parallel policy layer; read-only.

Program source: backlog items POSE-08 (P1, wave 1) of the
[third consolidated analysis](../reports/2026-10-03-pose-consolidated-analysis.md) (findings F03, F05; sources
E04, E05, E08, E09, E19, E30). Cross-cutting decisions: [ADR](../adr/2026-10-04-obligations-are-projected-action-requests-are-persisted.md). Owner proposed:
@pose-maintainers. This is a planning spec: no requirement is declared satisfied.

### Non-goals

Auto-adopting capabilities; unifying all schemas in a breaking migration.

### Anti-mechanization guardrail

Introspecção sobre fontes existentes; nenhuma camada paralela de policy.

## 2. Requirements

### Functional

- R1: `pose state` and `pose_project_state` shall include a governance section listing each registered contract and optional capability with `supported`, `configured`, `applicable` and `effective` and a reason code for each false.
- R2: Atomic start without its capability field shall render supported and not adopted.
- R3: DoR with an empty `adopted_at` shall not be reported as an active readiness cutoff.
- R4: Contracts stamped in a bundle shall be distinguishable from the current policy and from legacy-read cutoffs when the projection is scoped to a spec or bundle.
- R5: The projection shall not write policy nor widen review criteria; a test asserts no file changes.

### Non-functional

- Computed from existing parsed state; no extra repository scan.

### Security

- None specific beyond the shared constraints.

### Compatibility

- Additive state section.

## 3. Technical Plan

### Affected areas

State builder, review policy and capability readers, MCP state tool, manual.

### Artifacts

- created: .pose/specs/2026-10-04-pose-effective-governance-projection.md
- created: pose-mcp/internal/pose/effective_governance.go
- created: pose-mcp/internal/pose/effective_governance_test.go
- modified: pose-mcp/internal/pose/state.go
- created: pose-mcp/internal/cli/state_governance.go
- created: pose-mcp/internal/cli/state_governance_test.go
- modified: pose-mcp/internal/cli/state.go
- modified: pose-mcp/internal/mcpserver/server.go
- modified: pose-mcp/internal/mcpserver/state_tool_test.go
- modified: pose-mcp/internal/mcpserver/testdata/tool-catalog.golden.json
- modified: docs-site/docs/mcp.md
- modified: POSE.md
- modified: locales/pt-BR/POSE.md
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md
- modified: .pose/indexes/validation-matrix.json
- created: .pose/changelogs/unreleased/pose-effective-governance-projection.md

Reconciled against the tree at activation.

### Delivery targets

- surface:effective-governance module:pose-mcp profile:cli-surface entrypoint:pose-mcp/cmd/pose/main.go

### Technical risks

- Drift between projection and actual gate logic; mitigated by calling the same adoption functions the gates call.

## 4. Tasks

### Planning
- [x] Activate: confirm intent, re-read the cited sources at HEAD, reconcile Artifacts
- [ ] Run `pose assess discover --component pose-mcp` if the state is stale for the touched area

### Implementation
- [ ] Write the failing tests named in Validation first (the gate must fail before it passes)
- [ ] Implement incrementally, one requirement group per commit with `POSE-Spec: pose-effective-governance-projection`

### Validation
- [ ] Run the deterministic checks below and retain results

## 5. Decisions

No material decision beyond the transversal ADR at planning time.

## 6. Validation

### Strategy

Fixtures for each adoption combination; the projection must call
`ContractAdoptedAt`/capability readers, verified by a test that edits a policy and
observes both gate and projection change together.

### Deterministic checks

#### Test
- Command: `cd pose-mcp && go test ./...`
- Scope: engine packages touched by this spec
- Expected: pass, including the new negative tests

#### Lint
- Command: `cd pose-mcp && go vet ./...`
- Scope: pose-mcp
- Expected: no findings

#### Security / Contract
- Command: `pose lint-spec pose-effective-governance-projection --strict` and `pose check --strict`
- Scope: this spec and the instance
- Expected: pass

### Requirement trace

- R1 [satisfied] surface:effective-governance evidence:integration test:TestStateGovernanceIsReachableAndReadOnly check:effective-governance-integration test:TestToolsCall_ProjectState_CarriesEffectiveGovernance
- R2 [satisfied] test:TestUnadoptedCapabilitiesAreSupportedAndNotInForce check:effective-governance-integration
- R3 [satisfied] test:TestTheProjectionReadsWhatTheGatesRead check:effective-governance-integration
- R4 [satisfied] test:TestScopedProjectionReportsTheBundlesStampedContracts check:effective-governance-integration
- R5 [satisfied] test:TestTheProjectionWritesNothing check:effective-governance-integration

## 7. Final Report

### Delivered scope

Not started. Filled at closeout from the requirement trace and the change sets.

### Follow-ups

None recorded at planning time.
