---
slug: pose-falsifier-reconsideration
status: done
created_at: 2026-10-04
completed_at: 2026-10-05
supersedes:
depends_on: pose-assumption-validity-scope
priority: 3
components: pose-mcp
delivers: capability:falsifier-reconsideration
task_type: feature
---

# Spec: Close the loop of falsifiers and expected effects with explicit reconsideration

## 1. Intent

### Goal

Optionally link a material decision to an expected effect and observable falsifier, and
raise a reconsideration obligation when a pertinent observation contradicts it.

### Business value

A decision justified before the change must be reconsiderable when observations
contradict its basis or expected effect.

### Constraints

Opt-in for material decisions; never judges architectural adequacy automatically; keeps
rationale and history.

Program source: backlog items POSE-32 (P3, wave 4) of the
[third consolidated analysis](../reports/2026-10-03-pose-consolidated-analysis.md) (findings F11; sources
E11, E19, E21). Cross-cutting decisions: [ADR](../adr/2026-10-04-obligations-are-projected-action-requests-are-persisted.md). Owner proposed:
@pose-maintainers. This is a planning spec: no requirement is declared satisfied.

### Non-goals

Automatic causal inference; automatic architecture invalidation.

### Anti-mechanization guardrail

Observar contradição e pedir julgamento, sem transformar hipótese em quality gate
universal.

## 2. Requirements

### Functional

- R1: A pertinent observation contrary to a selected falsifier shall produce a reconsideration candidate.
- R2: The previous decision shall keep rationale and history.
- R3: Architectural adequacy shall not be judged automatically.
- R4: The mechanism shall be opt-in for material decisions and not require causal modelling in small fixes.
- R5: Decision nodes may declare `Expected effect` and an observable `Falsifier check` reference.

### Non-functional

- None specific beyond the shared constraints.

### Security

- None specific beyond the shared constraints.

### Compatibility

- Existing decisions unaffected.

## 3. Technical Plan

### Affected areas

Design basis, evidence binding, obligation adapters.

### Artifacts

- created: .pose/specs/2026-10-04-pose-falsifier-reconsideration.md
- created: pose-mcp/internal/pose/falsifier_observation.go
- created: pose-mcp/internal/pose/falsifier_observation_test.go
- modified: pose-mcp/internal/pose/design_basis.go
- modified: pose-mcp/internal/pose/obligation_projection.go
- modified: pose-mcp/internal/pose/readiness_phases.go
- modified: .pose/templates/spec.md
- modified: locales/pt-BR/.pose/templates/spec.md
- modified: pose-mcp/internal/scaffold/dist/.pose/templates/spec.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/.pose/templates/spec.md
- modified: POSE.md
- modified: locales/pt-BR/POSE.md
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md
- modified: .pose/indexes/validation-matrix.json
- created: .pose/changelogs/unreleased/pose-falsifier-reconsideration.md

Reconciled against the tree at activation.

### Delivery targets

- capability:falsifier-reconsideration module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go

### Technical risks

- Impossible falsifiers; reviewer judgment, not syntax.

## 4. Tasks

### Planning
- [x] Activate: confirm intent, re-read the cited sources at HEAD, reconcile Artifacts
- [ ] Run `pose assess discover --component pose-mcp` if the state is stale for the touched area

### Implementation
- [ ] Write the failing tests named in Validation first (the gate must fail before it passes)
- [ ] Implement incrementally, one requirement group per commit with `POSE-Spec: pose-falsifier-reconsideration`

### Validation
- [ ] Run the deterministic checks below and retain results

## 5. Decisions

No material decision beyond the transversal ADR at planning time.

## 6. Validation

### Strategy

Fixture with a failing falsifier check raises candidate; no automatic status change.

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
- Command: `pose lint-spec pose-falsifier-reconsideration --strict` and `pose check --strict`
- Scope: this spec and the instance
- Expected: pass

### Requirement trace

- R1 [satisfied] test:TestAFailedFalsifierCheckRaisesAReconsiderationCandidate check:falsifier-integration
- R2 [satisfied] test:TestAPassingOrAbsentObservationRaisesNothingAndTheDecisionIsKept check:falsifier-integration
- R3 [satisfied] <the candidate is advisory and states the failure is not a verdict> test:TestAFailedFalsifierCheckRaisesAReconsiderationCandidate check:falsifier-integration
- R4 [satisfied] test:TestFalsifierFieldsAreOptInAndValidated check:falsifier-integration
- R5 [satisfied] test:TestFalsifierFieldsAreOptInAndValidated check:falsifier-integration

## 7. Final Report

### Delivered scope

Delivers optional Expected effect and Falsifier check fields and the falsifier-observed projection. Each requirement is traced to its tests and matrix checks under Validation. Reviewed by agent:claude-opus-5-5 in the same session that implemented it; no separate execution or person reviewed it.

### Follow-ups

None recorded at planning time.
