---
slug: pose-adaptive-assessment-freshness
status: in-progress
created_at: 2026-10-04
completed_at:
supersedes:
depends_on: pose-effective-governance-projection, pose-obligation-projection, pose-material-equivalence-reuse
priority: 2
components: pose-mcp
delivers: capability:assessment-freshness
task_type: feature
---

# Spec: Apply freshness and materiality to assessments

## 1. Intent

### Goal

Decide refresh or reuse of assessments per component and observed contract, explain
stale triggers, and align feature/closeout workflows with the engine rule.

### Business value

Mandatory discovery before and after every change becomes ritual when the relevant
observations are still valid.

### Constraints

Freshness needs binding (component, commit/content, tool version, contracts), not only
age.

Program source: backlog items POSE-23 (P2, wave 3) of the
[third consolidated analysis](../reports/2026-10-03-pose-consolidated-analysis.md) (findings F11; sources
E12, E24, E31). Cross-cutting decisions: [ADR](../adr/2026-10-04-obligations-are-projected-action-requests-are-persisted.md). Owner proposed:
@pose-maintainers. This is a planning spec: no requirement is declared satisfied.

### Non-goals

Skipping material discovery to reduce numbers; covering every language in the first
delivery.

### Anti-mechanization guardrail

Freshness precisa de binding e não só de idade.

## 2. Requirements

### Functional

- R1: A trivial change in an unrelated area shall not require global discovery by default.
- R2: A relevant contract change shall invalidate the corresponding assessment.
- R3: A missing or stale assessment shall produce an explicit obligation in the relevant scope.
- R4: Workflows, skills and AGENTS assessment timing shall follow the same rule as the engine; no remaining always-run contradicts it.
- R5: `pose assess discover` shall report reused vs refreshed components with the binding compared.

### Non-functional

- None specific beyond the shared constraints.

### Security

- None specific beyond the shared constraints.

### Compatibility

- `--update-state` keeps working.

## 3. Technical Plan

### Affected areas

Assessment scan, state freshness, feature and closeout skills, AGENTS template.

### Artifacts

- created: .pose/specs/2026-10-04-pose-adaptive-assessment-freshness.md
- created: pose-mcp/internal/pose/assessment_freshness.go
- created: pose-mcp/internal/pose/assessment_freshness_test.go
- modified: pose-mcp/internal/pose/discovery.go
- modified: pose-mcp/internal/pose/obligation_projection.go
- modified: pose-mcp/internal/pose/readiness_phases.go
- modified: pose-mcp/internal/cli/assess.go
- created: pose-mcp/internal/cli/assessment_freshness_workflow_test.go
- modified: AGENTS.md
- modified: locales/pt-BR/AGENTS.md
- modified: .agents/skills/pose-feature/SKILL.md
- modified: .agents/skills/pose-spec-closeout/SKILL.md
- modified: locales/pt-BR/.agents/skills/pose-feature/SKILL.md
- modified: locales/pt-BR/.agents/skills/pose-spec-closeout/SKILL.md
- modified: pose-mcp/internal/scaffold/dist/AGENTS.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/AGENTS.md
- modified: pose-mcp/internal/scaffold/dist/.agents/skills/pose-feature/SKILL.md
- modified: pose-mcp/internal/scaffold/dist/.agents/skills/pose-spec-closeout/SKILL.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/.agents/skills/pose-feature/SKILL.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/.agents/skills/pose-spec-closeout/SKILL.md
- modified: POSE.md
- modified: locales/pt-BR/POSE.md
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md
- modified: .pose/indexes/validation-matrix.json
- created: .pose/changelogs/unreleased/pose-adaptive-assessment-freshness.md

Reconciled against the tree at activation.

### Delivery targets

- capability:assessment-freshness module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go

### Technical risks

- Missed invalidation; contract-change test per assessment kind.

## 4. Tasks

### Planning
- [x] Activate: confirm intent, re-read the cited sources at HEAD, reconcile Artifacts
- [ ] Run `pose assess discover --component pose-mcp` if the state is stale for the touched area

### Implementation
- [ ] Write the failing tests named in Validation first (the gate must fail before it passes)
- [ ] Implement incrementally, one requirement group per commit with `POSE-Spec: pose-adaptive-assessment-freshness`

### Validation
- [ ] Run the deterministic checks below and retain results

## 5. Decisions

No material decision beyond the transversal ADR at planning time.

## 6. Validation

### Strategy

Binding mutation tests; workflow text test that no always-run instruction remains.

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
- Command: `pose lint-spec pose-adaptive-assessment-freshness --strict` and `pose check --strict`
- Scope: this spec and the instance
- Expected: pass

### Requirement trace

- R1 [satisfied] test:TestAssessmentFreshnessFollowsTheBindingNotTheClock check:assessment-freshness-integration
- R2 [satisfied] <a committed content change stales the binding> test:TestAssessmentFreshnessFollowsTheBindingNotTheClock check:assessment-freshness-integration
- R3 [satisfied] test:TestAStaleAssessmentIsAnAdvisoryObligation check:obligation-contract-integration
- R4 [satisfied] test:TestNoWorkflowAlwaysRunsDiscovery check:assessment-freshness-integration
- R5 [satisfied] <assess discover --if-stale prints reused or refreshed with the binding compared> test:TestAssessmentFreshnessFollowsTheBindingNotTheClock check:assessment-freshness-integration

## 7. Final Report

### Delivered scope

Not started. Filled at closeout from the requirement trace and the change sets.

### Follow-ups

None recorded at planning time.
