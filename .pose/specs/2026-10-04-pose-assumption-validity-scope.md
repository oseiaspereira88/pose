---
slug: pose-assumption-validity-scope
status: done
created_at: 2026-10-04
completed_at: 2026-10-05
supersedes:
depends_on: pose-obligation-contract, pose-obligation-projection, pose-abm-capability-adoption
priority: 3
components: pose-mcp
delivers: capability:premise-validity
task_type: feature
---

# Spec: Represent material staleness of assumptions and their scopes

## 1. Intent

### Goal

Let selected material assumptions declare a validity scope and material triggers, and
project `premise-stale` obligations when the bound contract or evidence changes.

### Business value

An assumption can keep old evidence after the contract, provider or context that
supported it changed.

### Constraints

Validity by context and materiality before calendar expiry; trivial assumptions get no
TTL.

Program source: backlog items POSE-31 (P3, wave 4) of the
[third consolidated analysis](../reports/2026-10-03-pose-consolidated-analysis.md) (findings F11; sources
E11, E19). Cross-cutting decisions: [ADR](../adr/2026-10-04-obligations-are-projected-action-requests-are-persisted.md). Owner proposed:
@pose-maintainers. This is a planning spec: no requirement is declared satisfied.

### Non-goals

Universal TTL on assumptions; monitoring every provider in the core.

### Anti-mechanization guardrail

Validade por contexto e materialidade antes de expiração por calendário.

## 2. Requirements

### Functional

- R1: A relevant contract version change shall signal review of the bound assumption.
- R2: Old evidence shall stay retained without being presented as valid for a new context.
- R3: Trivial assumptions shall not receive a mandatory arbitrary TTL.
- R4: The signal shall request judgment and shall not invalidate the whole design automatically.
- R5: Optional fields `Valid scope` and `Stale trigger` shall be parsed by the design basis reader.

### Non-functional

- None specific beyond the shared constraints.

### Security

- None specific beyond the shared constraints.

### Compatibility

- Existing assumptions unaffected.

## 3. Technical Plan

### Affected areas

Design basis parser, obligation adapters.

### Artifacts

- created: .pose/specs/2026-10-04-pose-assumption-validity-scope.md
- created: pose-mcp/internal/pose/premise_validity.go
- created: pose-mcp/internal/pose/assumption_validity_test.go
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
- renamed: .pose/changelogs/unreleased/pose-assumption-validity-scope.md -> .pose/changelogs/v7.0.0/pose-assumption-validity-scope.md

Reconciled against the tree at activation.

### Delivery targets

- capability:premise-validity module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go

### Technical risks

- Ritual fields; opt-in, material only.

## 4. Tasks

### Planning
- [x] Activate: confirm intent, re-read the cited sources at HEAD, reconcile Artifacts
- [ ] Run `pose assess discover --component pose-mcp` if the state is stale for the touched area

### Implementation
- [ ] Write the failing tests named in Validation first (the gate must fail before it passes)
- [ ] Implement incrementally, one requirement group per commit with `POSE-Spec: pose-assumption-validity-scope`

### Validation
- [ ] Run the deterministic checks below and retain results

## 5. Decisions

No material decision beyond the transversal ADR at planning time.

## 6. Validation

### Strategy

Contract version bump fixture raises premise-stale; trivial assumption no obligation.

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
- Command: `pose lint-spec pose-assumption-validity-scope --strict` and `pose check --strict`
- Scope: this spec and the instance
- Expected: pass

### Requirement trace

- R1 [satisfied] test:TestAContractChangeAsksForJudgmentOnTheBoundPremise check:premise-validity-integration
- R2 [satisfied] <the evidence stays recorded and is described as judged against the pinned content> test:TestAContractChangeAsksForJudgmentOnTheBoundPremise check:premise-validity-integration
- R3 [satisfied] test:TestTrivialAssumptionsGetNoTTLAndCalendarTriggersAreNotEvaluated check:premise-validity-integration
- R4 [satisfied] <advisory on review, scoped to the assumption and its decisions> test:TestAContractChangeAsksForJudgmentOnTheBoundPremise check:premise-validity-integration
- R5 [satisfied] test:TestValidityFieldsAreParsedAndDoNotChangeExistingDigests check:premise-validity-integration test:TestPortugueseValidityLabelsAreParsed

## 7. Final Report

### Delivered scope

Delivers optional Valid scope and pinned Stale trigger fields and the premise-stale projection; no calendar expiry. Each requirement is traced to its tests and matrix checks under Validation. Reviewed by agent:claude-opus-5-5 in the same session that implemented it; no separate execution or person reviewed it.

### Follow-ups

None recorded at planning time.
