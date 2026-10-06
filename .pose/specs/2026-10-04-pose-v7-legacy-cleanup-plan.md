---
slug: pose-v7-legacy-cleanup-plan
status: done
created_at: 2026-10-04
completed_at: 2026-10-05
supersedes:
depends_on: pose-legacy-contract-cutoffs, pose-blocked-semantics-alignment, pose-flat-spec-amendments, pose-effective-governance-projection, pose-transfer-preserves-obligations, pose-agency-readiness-pilot
priority: 3
components: pose-mcp
delivers: surface:migrate-v7-dry-run
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
- created: pose-mcp/internal/pose/legacy_inventory.go
- created: pose-mcp/internal/cli/migrate_dryrun.go
- created: pose-mcp/internal/cli/migrate_dryrun_test.go
- modified: pose-mcp/internal/cli/cli.go
- modified: pose-mcp/internal/cli/help_catalog.go
- modified: pose-mcp/internal/pose/review_attribution.go
- modified: docs-site/docs/cli.md
- modified: POSE.md
- modified: locales/pt-BR/POSE.md
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md
- modified: .pose/indexes/validation-matrix.json
- renamed: .pose/changelogs/unreleased/pose-v7-legacy-cleanup-plan.md -> .pose/changelogs/v7.0.0/pose-v7-legacy-cleanup-plan.md

Reconciled against the tree at activation.

### Delivery targets

- surface:migrate-v7-dry-run module:pose-mcp profile:cli-surface entrypoint:pose-mcp/cmd/pose/main.go

### Technical risks

- Scope creep; benefit measured per item.

## 4. Tasks

### Planning
- [x] Activate: confirm intent, re-read the cited sources at HEAD, reconcile Artifacts
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

### Requirement trace

- R1 [satisfied] surface:migrate-v7-dry-run evidence:integration <interrupted transfers measured on this repository's side, see report> test:TestMigrateDryRunInventoriesLegacyWithoutWriting check:migrate-dryrun-integration check:migrate-dryrun-reachability
- R2 [satisfied] <dry-run writes nothing; apply is refused in 6.x and its recovery and revision guard are stated in the plan> test:TestMigrateDryRunInventoriesLegacyWithoutWriting check:migrate-dryrun-integration
- R3 [satisfied] <unknown review policy keys are reported as risks> test:TestMigrateDryRunInventoriesLegacyWithoutWriting check:migrate-dryrun-integration
- R4 [satisfied] <report .pose/reports/pose-v7-legacy-cleanup-plan.md: sealed records are kept read-only and never rewritten>
- R5 [satisfied] <report .pose/reports/pose-v7-legacy-cleanup-plan.md states each removal criterion with its measured benefit>

## 7. Final Report

Measured at 9c3a038 (`pose migrate v7 --dry-run`, 0.5 s): 318 flat and 3 dated folder specs, no undated folder and no blocked spec, 5 legacy review policy adoption keys, no unknown key, 1307 sealed bundles (483 without `governing_contracts`), 1301 declared-identity attestations (1285 without attribution or supplement), no incomplete transfer. A first run reported 10 incomplete transfers because it read each transfer's status as the source project; the ten ABM transfers are `activated` on this, the destination side. The predicate now uses the current project and its role in the plan. R2's apply half is deliberately not built in 6.x: `--apply` is refused and the plan lists what a 7.0 apply must carry (revision guard, per-class checkpoint, gate rehearsal). The help catalog entries for five earlier commands were updated in this commit because the catalog lagged their flags.

### Delivered scope

Delivers the dry-run inventory and the measured plan; no apply in 6.x. Each requirement is traced to its tests and matrix checks under Validation. Reviewed by agent:claude-opus-5-5 in the same session that implemented it; no separate execution or person reviewed it.

### Follow-ups

None recorded at planning time.
