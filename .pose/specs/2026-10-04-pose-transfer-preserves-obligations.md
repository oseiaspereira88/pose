---
slug: pose-transfer-preserves-obligations
status: in-progress
created_at: 2026-10-04
completed_at:
supersedes:
depends_on: pose-obligation-contract, pose-action-request-resolution, pose-phase-scoped-readiness
priority: 0
components: pose-mcp
delivers: capability:transfer-preserves-obligations
task_type: feature
---

# Spec: Preserve requests and obligations across spec transfer and authority reconciliation

## 1. Intent

### Goal

Map ActionRequest targets and obligation source refs during spec transfer so an action
tied to the old project or spec never authorizes the destination by accident.

### Business value

Transfer already guards staging with `blocked` and revision checks; requests introduce
bindings it must carry or invalidate.

### Constraints

Preserve the existing context and revision guard; moving a file never delegates
authority.

Program source: backlog items POSE-19 (P0, wave 2) of the
[third consolidated analysis](../reports/2026-10-03-pose-consolidated-analysis.md) (findings F05, F06, F07; sources
E15, E29). Cross-cutting decisions: [ADR](../adr/2026-10-04-obligations-are-projected-action-requests-are-persisted.md). Owner proposed:
@pose-maintainers. This is a planning spec: no requirement is declared satisfied.

### Non-goals

Cross-project scheduling; implicit authority delegation.

### Anti-mechanization guardrail

Preservar o contrato de contexto e revision guard existente.

## 2. Requirements

### Functional

- R1: Same slug and R-ID in different projects shall keep distinct identities across transfer.
- R2: An interrupted transfer shall not leave origin and destination both suggesting authorization.
- R3: Confirmations whose content or authority changes with the transfer shall be invalidated or explicitly re-confirmed.
- R4: Reads shall follow the canonical authority without erasing the origin of transported obligations.
- R5: Transfer preview shall list requests that are transferable, local-only or invalidated by the move.

### Non-functional

- None specific beyond the shared constraints.

### Security

- Revision guard re-checked at apply/resume.

### Compatibility

- Transfers without requests unchanged.

## 3. Technical Plan

### Affected areas

Spec transfer preview/apply/resume, action request journal remapping.

### Artifacts

- created: .pose/specs/2026-10-04-pose-transfer-preserves-obligations.md
- modified: pose-mcp/internal/pose/spec_transfer.go
- created: pose-mcp/internal/pose/spec_transfer_actions_test.go
- modified: POSE.md
- modified: locales/pt-BR/POSE.md
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md
- modified: .pose/indexes/validation-matrix.json
- created: .pose/changelogs/unreleased/pose-transfer-preserves-obligations.md

Reconciled against the tree at activation.

### Delivery targets

- capability:transfer-preserves-obligations module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go

### Technical risks

- Remapping errors; preview lists every effect before apply.

## 4. Tasks

### Planning
- [x] Activate: confirm intent, re-read the cited sources at HEAD, reconcile Artifacts
- [ ] Run `pose assess discover --component pose-mcp` if the state is stale for the touched area

### Implementation
- [ ] Write the failing tests named in Validation first (the gate must fail before it passes)
- [ ] Implement incrementally, one requirement group per commit with `POSE-Spec: pose-transfer-preserves-obligations`

### Validation
- [ ] Run the deterministic checks below and retain results

## 5. Decisions

No material decision beyond the transversal ADR at planning time.

## 6. Validation

### Strategy

Interrupted transfer with an open and a resolved request; same slug collision; authority
change invalidation.

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
- Command: `pose lint-spec pose-transfer-preserves-obligations --strict` and `pose check --strict`
- Scope: this spec and the instance
- Expected: pass

## 7. Final Report

### Delivered scope

Not started. Filled at closeout from the requirement trace and the change sets.

### Follow-ups

None recorded at planning time.
