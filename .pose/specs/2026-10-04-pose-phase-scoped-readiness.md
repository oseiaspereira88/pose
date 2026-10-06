---
slug: pose-phase-scoped-readiness
status: done
created_at: 2026-10-04
completed_at: 2026-10-05
supersedes:
depends_on: pose-blocked-semantics-alignment, pose-obligation-projection, pose-action-requests
priority: 1
components: pose-mcp
delivers: surface:phase-scoped-readiness
task_type: feature
---

# Spec: Project readiness per phase and scope with conservative continuation

## 1. Intent

### Goal

Add a readiness projection for start, execution, review, closeout and release over fine
targets, preserving the meaning of the legacy `Ready bool`.

### Business value

Global eligibility cannot express that a closeout-only judgment allows implementation, a
product decision restricts one requirement, or a publication acceptance restricts only
the release.

### Constraints

Absence of an explicit relation is never proof of independence. Partial eligibility
lives in a new field; `Ready` keeps its meaning.

Program source: backlog items POSE-17 (P1, wave 2) of the
[third consolidated analysis](../reports/2026-10-03-pose-consolidated-analysis.md) (findings F07, F08; sources
E06, E07, E08, E09). Cross-cutting decisions: [ADR](../adr/2026-10-04-obligations-are-projected-action-requests-are-persisted.md). Owner proposed:
@pose-maintainers. This is a planning spec: no requirement is declared satisfied.

### Non-goals

A requirement scheduler; global authorization from the absence of known blockers; new
persisted spec statuses.

### Anti-mechanization guardrail

Não introduzir dezenas de estados persistidos da spec.

## 2. Requirements

### Functional

- R1: Readiness output shall add a `phases` block with start, execution, review, closeout and release, each with state (clear, restricted, unknown), restricting obligations and coverage.
- R2: An action effective only at closeout shall not restrict start or execution by implicit effect.
- R3: A localized restriction (e.g. on R4) shall not be rendered as a full stop without a sufficient causal relation.
- R4: For unrestricted targets the output shall say "no direct restriction found in the consulted sources; independence not demonstrated" unless explicit relations and sufficient coverage support "executable under these conditions".
- R5: Legacy consumers of `Ready` shall receive the same value and meaning as before; a test asserts it across the fixture corpus.

### Non-functional

- Same projection feeds state, start preview and MCP readiness.

### Security

- None specific beyond the shared constraints.

### Compatibility

- Additive fields.

## 3. Technical Plan

### Affected areas

Readiness domain, start preview, state, MCP readiness/next steps.

### Artifacts

- created: .pose/specs/2026-10-04-pose-phase-scoped-readiness.md
- created: pose-mcp/internal/pose/readiness_phases.go
- created: pose-mcp/internal/pose/readiness_phases_test.go
- modified: pose-mcp/internal/cli/state_attention.go
- modified: pose-mcp/internal/cli/adversarial_corpus_test.go
- modified: pose-mcp/internal/cli/testdata/adversarial/README.md
- modified: pose-mcp/internal/mcpserver/server.go
- modified: pose-mcp/internal/mcpserver/obligations_test.go
- modified: pose-mcp/internal/mcpserver/testdata/tool-catalog.golden.json
- modified: docs-site/docs/mcp.md
- modified: POSE.md
- modified: locales/pt-BR/POSE.md
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md
- modified: .pose/indexes/validation-matrix.json
- renamed: .pose/changelogs/unreleased/pose-phase-scoped-readiness.md -> .pose/changelogs/v7.0.0/pose-phase-scoped-readiness.md

Reconciled against the tree at activation.

### Delivery targets

- surface:phase-scoped-readiness module:pose-mcp profile:cli-surface entrypoint:pose-mcp/cmd/pose/main.go

### Technical risks

- False precision; conservative wording is a tested contract.

## 4. Tasks

### Planning
- [x] Activate: confirm intent, re-read the cited sources at HEAD, reconcile Artifacts
- [ ] Run `pose assess discover --component pose-mcp` if the state is stale for the touched area

### Implementation
- [ ] Write the failing tests named in Validation first (the gate must fail before it passes)
- [ ] Implement incrementally, one requirement group per commit with `POSE-Spec: pose-phase-scoped-readiness`

### Validation
- [ ] Run the deterministic checks below and retain results

## 5. Decisions

No material decision beyond the transversal ADR at planning time.

## 6. Validation

### Strategy

Matrix fixture: closeout-only action, R4-only decision, release-only acceptance, unknown
coverage; legacy Ready invariance.

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
- Command: `pose lint-spec pose-phase-scoped-readiness --strict` and `pose check --strict`
- Scope: this spec and the instance
- Expected: pass

### Requirement trace

- R1 [satisfied] surface:phase-scoped-readiness evidence:integration test:TestToolsCall_SpecReadiness_PhasesBesideLegacyReady check:phase-readiness-integration test:TestAPhaseWithAnUnreadProducerIsUnknownNotClear
- R2 [satisfied] test:TestACloseoutOnlyRequestDoesNotRestrictStartOrExecution check:phase-readiness-integration
- R3 [satisfied] test:TestALocalizedRestrictionIsNotAFullStopNorProofOfIndependence check:phase-readiness-integration
- R4 [satisfied] test:TestALocalizedRestrictionIsNotAFullStopNorProofOfIndependence check:phase-readiness-integration
- R5 [satisfied] test:TestLegacyReadyKeepsItsMeaning check:phase-readiness-integration

## 7. Final Report

### Delivered scope

Delivers the phases block on readiness and MCP. Each requirement is traced to its tests and matrix checks under Validation. Reviewed by agent:claude-opus-5-5 in the same session that implemented it; no separate execution or person reviewed it.

### Follow-ups

None recorded at planning time.
