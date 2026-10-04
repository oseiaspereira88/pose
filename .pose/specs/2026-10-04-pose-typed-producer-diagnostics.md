---
slug: pose-typed-producer-diagnostics
status: in-progress
created_at: 2026-10-04
completed_at:
supersedes:
depends_on: pose-obligation-contract
priority: 1
components: pose-mcp
delivers: surface:typed-producer-diagnostics
task_type: feature
---

# Spec: Add reason codes and structured refs to the pending-state producers

## 1. Intent

### Goal

Make readiness, review, closeout and start emit typed diagnostics (code, refs,
satisfaction condition) and render their human messages from them.

### Business value

`Blockers []string` and `NextAction string` are not a safe contract for agents;
normalising them by regex would add another textual dependency.

### Constraints

Legacy strings stay populated during the transition; an opaque legacy blocker keeps
origin and a limitation flag and gains no invented actor or target.

Program source: backlog items POSE-10 (P1, wave 1) of the
[third consolidated analysis](../reports/2026-10-03-pose-consolidated-analysis.md) (findings F05, F09; sources
E06, E07, E08, E09, E15). Cross-cutting decisions: [ADR](../adr/2026-10-04-obligations-are-projected-action-requests-are-persisted.md). Owner proposed:
@pose-maintainers. This is a planning spec: no requirement is declared satisfied.

### Non-goals

Converting every diagnostic in one delivery; removing legacy strings now.

### Anti-mechanization guardrail

Normalizar fatos nos produtores; evitar classificação por regex de mensagens.

## 2. Requirements

### Functional

- R1: A pending review judgment shall identify criterion, source bundle/plan and satisfaction condition as structured fields.
- R2: An unresolved dependency shall identify the qualified ref, a causal code and the responsible domain.
- R3: Closeout blockers and next actions shall be emitted as typed items alongside the legacy strings, which are rendered from them.
- R4: Changing locale or wording shall not change identity or effect of a diagnostic; a test renders two locales and compares codes.
- R5: An unconverted legacy blocker shall surface as `legacy-opaque` with source and limited coverage.

### Non-functional

- Codes registered in one catalog with docs.

### Security

- None specific beyond the shared constraints.

### Compatibility

- Legacy fields kept; new fields additive.

## 3. Technical Plan

### Affected areas

Readiness, review bundle pendencies, closeout state, start obligations, rendering.

### Artifacts

- created: .pose/specs/2026-10-04-pose-typed-producer-diagnostics.md
- created: pose-mcp/internal/pose/diagnostic_codes.go
- created: pose-mcp/internal/pose/diagnostic_codes_test.go
- modified: pose-mcp/internal/pose/readiness.go
- modified: pose-mcp/internal/pose/review_bundle.go
- modified: pose-mcp/internal/pose/review_closeout.go
- modified: pose-mcp/internal/pose/start.go
- created: pose-mcp/internal/cli/typed_diagnostics_test.go
- modified: POSE.md
- modified: locales/pt-BR/POSE.md
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md
- modified: .pose/indexes/validation-matrix.json
- created: .pose/changelogs/unreleased/pose-typed-producer-diagnostics.md

Reconciled against the tree at activation.

### Delivery targets

- surface:typed-producer-diagnostics module:pose-mcp profile:cli-surface entrypoint:pose-mcp/cmd/pose/main.go

### Technical risks

- Two sources of truth during transition; strings are rendered from typed items, never the reverse.

## 4. Tasks

### Planning
- [x] Activate: confirm intent, re-read the cited sources at HEAD, reconcile Artifacts
- [ ] Run `pose assess discover --component pose-mcp` if the state is stale for the touched area

### Implementation
- [ ] Write the failing tests named in Validation first (the gate must fail before it passes)
- [ ] Implement incrementally, one requirement group per commit with `POSE-Spec: pose-typed-producer-diagnostics`

### Validation
- [ ] Run the deterministic checks below and retain results

## 5. Decisions

No material decision beyond the transversal ADR at planning time.

## 6. Validation

### Strategy

Per producer: a test asserts the typed item exists for each legacy string and that the
string is its rendering; locale invariance test.

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
- Command: `pose lint-spec pose-typed-producer-diagnostics --strict` and `pose check --strict`
- Scope: this spec and the instance
- Expected: pass

## 7. Final Report

### Delivered scope

Not started. Filled at closeout from the requirement trace and the change sets.

### Follow-ups

None recorded at planning time.
