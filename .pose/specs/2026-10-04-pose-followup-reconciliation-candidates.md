---
slug: pose-followup-reconciliation-candidates
status: in-progress
created_at: 2026-10-04
completed_at:
supersedes:
depends_on: pose-open-backlog-reconciliation, pose-obligation-contract, pose-state-attention
priority: 2
components: pose-mcp
delivers: surface:followup-candidates
task_type: feature
---

# Spec: Signal follow-up reconciliation candidates without auto-close

## 1. Intent

### Goal

Raise target-terminal, evidence-present, overdue and possible-duplicate candidates with
stable refs and evidence links, leaving disposition to the authoritative follow-up
mechanism.

### Business value

Residual backlog drifts; new evidence should be visible without the engine asserting
semantic equivalence.

### Constraints

Lexical similarity never produces duplicate without judgment; candidates are not new
artifacts or blocking questions.

Program source: backlog items POSE-25 (P2, wave 3) of the
[third consolidated analysis](../reports/2026-10-03-pose-consolidated-analysis.md) (findings F10; sources
E01, E22, E25, E26). Cross-cutting decisions: [ADR](../adr/2026-10-04-obligations-are-projected-action-requests-are-persisted.md). Owner proposed:
@pose-maintainers. This is a planning spec: no requirement is declared satisfied.

### Non-goals

Auto-closing by test existence; reclassifying debt as execution gate.

### Anti-mechanization guardrail

Não transformar todo candidato em novo artifact ou pergunta bloqueante.

## 2. Requirements

### Functional

- R1: A done target shall produce a review candidate, not an automatic disposition.
- R2: Lexical similarity shall not produce `duplicate` without judgment.
- R3: Each suggestion shall show why it was raised and the limits of the evidence.
- R4: An open follow-up shall not disappear from the projection without a disposition or an explicit coverage limitation.
- R5: `pose followups --candidates` and the Attention residual section shall list candidates.

### Non-functional

- None specific beyond the shared constraints.

### Security

- None specific beyond the shared constraints.

### Compatibility

- Existing follow-up formats unchanged.

## 3. Technical Plan

### Affected areas

Follow-up aggregation, Attention residual section.

### Artifacts

- created: .pose/specs/2026-10-04-pose-followup-reconciliation-candidates.md
- created: pose-mcp/internal/pose/followup_candidates.go
- created: pose-mcp/internal/pose/followup_candidates_test.go
- modified: pose-mcp/internal/pose/obligation_projection.go
- modified: pose-mcp/internal/cli/followups.go
- created: pose-mcp/internal/cli/followup_candidates_test.go
- modified: POSE.md
- modified: locales/pt-BR/POSE.md
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md
- modified: .pose/indexes/validation-matrix.json
- created: .pose/changelogs/unreleased/pose-followup-reconciliation-candidates.md

Reconciled against the tree at activation.

### Delivery targets

- surface:followup-candidates module:pose-mcp profile:cli-surface entrypoint:pose-mcp/cmd/pose/main.go

### Technical risks

- Noise; candidates ranked by evidence strength, never closed.

## 4. Tasks

### Planning
- [x] Activate: confirm intent, re-read the cited sources at HEAD, reconcile Artifacts
- [ ] Run `pose assess discover --component pose-mcp` if the state is stale for the touched area

### Implementation
- [ ] Write the failing tests named in Validation first (the gate must fail before it passes)
- [ ] Implement incrementally, one requirement group per commit with `POSE-Spec: pose-followup-reconciliation-candidates`

### Validation
- [ ] Run the deterministic checks below and retain results

## 5. Decisions

No material decision beyond the transversal ADR at planning time.

## 6. Validation

### Strategy

Fixtures for each candidate kind; no-auto-close invariant.

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
- Command: `pose lint-spec pose-followup-reconciliation-candidates --strict` and `pose check --strict`
- Scope: this spec and the instance
- Expected: pass

## 7. Final Report

### Delivered scope

Not started. Filled at closeout from the requirement trace and the change sets.

### Follow-ups

None recorded at planning time.
