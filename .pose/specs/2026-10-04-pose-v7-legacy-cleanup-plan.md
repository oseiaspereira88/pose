---
slug: pose-v7-legacy-cleanup-plan
status: draft
created_at: 2026-10-04
completed_at:
supersedes:
depends_on: pose-legacy-contract-cutoffs, pose-blocked-semantics-alignment, pose-flat-spec-amendments, pose-effective-governance-projection, pose-transfer-preserves-obligations, pose-agency-readiness-pilot
priority: 3
components: pose-mcp
task_type: refactor
---

# Spec: Plan schema, legacy field and blocked cleanup for an appropriate major

## 1. Intent

### Goal

Inventory readers, writers and the supported schema floor, and produce a deprecation
plan with a dry-run over the historical corpus, without applying breaking changes in
6.x.

### Business value

Representations accumulate; historical compatibility is part of POSE's value, so cleanup
needs measured benefit and proven migration.

### Constraints

A major is not a pretext to reform the whole engine; old records keep meaning, signature
and auditability.

Program source: backlog items POSE-33 (P3, wave 4) of the
[third consolidated analysis](../reports/2026-10-03-pose-consolidated-analysis.md) (findings F03, F07; sources
E08, E13, E14, E15, E19, E29). Cross-cutting decisions: [ADR](../adr/2026-10-04-obligations-are-projected-action-requests-are-persisted.md). Owner proposed:
@pose-maintainers. This is a planning spec: no requirement is declared satisfied.

### Non-goals

Breaking cleanup in 6.x; removing history or converting unknown into certainty.

### Anti-mechanization guardrail

Uma major não é pretexto para reformar o motor inteiro sem benefício medido.

## 2. Requirements

### Functional

- R1: The migration dry-run shall cover flat and folder specs, sealed bundles, declared/verified modes and interrupted transfers.
- R2: Dry-run shall explain effects without writing; apply shall have recovery and a revision guard.
- R3: Unknown fields shall not silence expected enforcement in supported combinations.
- R4: Old records shall keep meaning, signature and auditability.
- R5: The plan shall state the criterion for removing `blocked` and each legacy field, with the measured benefit.

### Non-functional

- None specific beyond the shared constraints.

### Security

- None specific beyond the shared constraints.

### Compatibility

- Planning only in 6.x.

## 3. Technical Plan

### Affected areas

Planning report and dry-run tooling.

### Artifacts

- created: .pose/specs/2026-10-04-pose-v7-legacy-cleanup-plan.md
- created: .pose/reports/pose-v7-legacy-cleanup-plan.md
- created: pose-mcp/internal/cli/migrate_dryrun.go
- created: pose-mcp/internal/cli/migrate_dryrun_test.go

The paths above are the planned surface at 2026-10-04; reconcile them at activation
against the tree, as the ABM specs did, before the first implementation commit.

### Technical risks

- Scope creep; benefit measured per item.

## 4. Tasks

### Planning
- [ ] Activate: confirm intent, re-read the cited sources at HEAD, reconcile Artifacts
- [ ] Run `pose assess discover --component pose-mcp` if the state is stale for the touched area

### Implementation
- [ ] Write the failing tests named in Validation first (the gate must fail before it passes)
- [ ] Implement incrementally, one requirement group per commit with `POSE-Spec: pose-v7-legacy-cleanup-plan`

### Validation
- [ ] Run the deterministic checks below and retain results

## 5. Decisions

No material decision beyond the transversal ADR at planning time.

## 6. Validation

### Strategy

Dry-run over both brownfield corpora with zero writes verified by tree hash.

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
- Command: `pose lint-spec pose-v7-legacy-cleanup-plan --strict` and `pose check --strict`
- Scope: this spec and the instance
- Expected: pass

## 7. Final Report

### Delivered scope

Not started. Filled at closeout from the requirement trace and the change sets.

### Follow-ups

None recorded at planning time.
