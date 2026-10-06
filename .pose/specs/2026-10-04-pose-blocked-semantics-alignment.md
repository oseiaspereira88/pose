---
slug: pose-blocked-semantics-alignment
status: done
created_at: 2026-10-04
completed_at: 2026-10-05
supersedes:
depends_on: 
priority: 0
components: pose-mcp
delivers: surface:blocked-semantics
task_type: feature
---

# Spec: Align the meaning of blocked across lifecycle, transfer, readiness and metrics

## 1. Intent

### Goal

Give `blocked` one documented meaning — a non-terminal operational condition — in the
skill, the manual, readiness, state and metrics, while keeping its technical use in spec
transfer and every legacy reader working.

### Business value

The closeout skill calls `blocked` terminal; readiness treats it as a special non-ready
state and keeps done/superseded/abandoned terminal; adoption metrics put it in the
denominator of resolved specs (it lowers the success ratio without adding to the
numerator); spec transfer uses it during staging and when fresh evidence is required.
Waiting is being read as an outcome.

### Constraints

Do not convert legacy `blocked` specs to `in-progress`: some were never started. When
history is insufficient, keep the raw status and report the previous phase as unknown.
The old metric series keeps its meaning; a new versioned dimension carries the corrected
one.

Program source: backlog items POSE-05 (P0, wave 0) of the
[third consolidated analysis](../reports/2026-10-03-pose-consolidated-analysis.md) (findings F07; sources
E06, E13, E14, E15). Cross-cutting decisions: [ADR](../adr/2026-10-04-obligations-are-projected-action-requests-are-persisted.md). Owner proposed:
@pose-maintainers. This is a planning spec: no requirement is declared satisfied.

### Non-goals

Removing `blocked` in a patch release; unblocking records without a known cause; a
`waiting-*` status family.

### Anti-mechanization guardrail

Não eliminar um estado antes de migrar todos os seus usos técnicos.

## 2. Requirements

### Functional

- R1: The skill, manual, template comment and help text shall describe `blocked` as a non-terminal operational condition distinct from done, superseded and abandoned.
- R2: `pose state` and readiness shall report a blocked spec with its raw status, `terminal: false` and, when no cause is recorded, `cause: unknown`.
- R3: Adoption metrics shall keep the v1 resolved-ratio series unchanged and add a versioned v2 dimension that excludes operational waiting from resolved outcomes; both are labelled.
- R4: Spec transfer staging and fresh-evidence protection shall keep refusing the protected transitions during the alignment, verified by the existing interrupted-transfer tests.
- R5: A legacy blocked spec without a known start shall never be rewritten to `in-progress` by any command.
- R6: `pose lint-spec` shall warn (not fail) on `status: blocked` without a recorded cause, pointing to the ActionRequest/obligation model once adopted.

### Non-functional

- Inventory of every reader and writer of the `blocked` literal recorded in the spec before code changes.

### Security

- No change weakens the transfer guard.

### Compatibility

- Metric v1 unchanged; v2 additive; status values unchanged on disk.

## 3. Technical Plan

### Affected areas

Readiness, state, adoption metrics, spec transfer, closeout skill, manual and templates.

### Artifacts

- created: .pose/specs/2026-10-04-pose-blocked-semantics-alignment.md
- modified: pose-mcp/internal/pose/readiness.go
- created: pose-mcp/internal/pose/readiness_blocked_test.go
- modified: pose-mcp/internal/pose/spec_transfer.go
- modified: pose-mcp/internal/cli/adoption_metrics.go
- modified: pose-mcp/internal/cli/lintspec.go
- modified: pose-mcp/internal/cli/state_providers.go
- created: pose-mcp/internal/cli/blocked_semantics_test.go
- modified: .agents/skills/pose-spec-closeout/SKILL.md
- modified: .pose/templates/spec.md
- modified: docs-site/docs/concepts.md
- modified: docs-site/docs/frontmatter.md
- modified: POSE.md
- modified: locales/pt-BR/POSE.md
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md
- modified: pose-mcp/internal/scaffold/dist/.agents/skills/pose-spec-closeout/SKILL.md
- modified: pose-mcp/internal/scaffold/dist/.pose/templates/spec.md
- modified: .pose/indexes/validation-matrix.json
- renamed: .pose/changelogs/unreleased/pose-blocked-semantics-alignment.md -> .pose/changelogs/v7.0.0/pose-blocked-semantics-alignment.md

Reconciled against the tree at activation.

### Delivery targets

- surface:blocked-semantics module:pose-mcp profile:cli-surface entrypoint:pose-mcp/cmd/pose/main.go

### Reader and writer inventory

Recorded at activation from the tree. Writers: spec transfer (`spec_transfer.go`,
staging and fresh-evidence guard). Readers: readiness (`readiness.go`), lint and
check status validation (`lintspec.go`, `check.go`), adoption metrics
(`adoption_metrics.go`), state totals (`state_providers.go`), spec listing order
(`spec.go`), the closeout skill, the manuals, the spec template and the docs
pages `concepts.md`, `frontmatter.md` and `architecture.md` (whose state diagram
already drew `blocked ⇄ in_progress`). Measured on pose-dist at activation, v1
of the adoption metric read 3 folder-layout specs of the 321 the store lists.

### Technical risks

- Hidden readers of the literal; mitigated by the recorded inventory and grep-based test.

## 4. Tasks

### Planning
- [x] Activate: confirm intent, re-read the cited sources at HEAD, reconcile Artifacts
- [ ] Run `pose assess discover --component pose-mcp` if the state is stale for the touched area

### Implementation
- [ ] Write the failing tests named in Validation first (the gate must fail before it passes)
- [ ] Implement incrementally, one requirement group per commit with `POSE-Spec: pose-blocked-semantics-alignment`

### Validation
- [ ] Run the deterministic checks below and retain results

## 5. Decisions

No material decision beyond the transversal ADR at planning time.

## 6. Validation

### Strategy

Inventory first (grep of the literal across engine, scaffold and skills). Tests: metric
v1 golden unchanged, v2 excludes blocked; legacy blocked spec untouched by start/close;
transfer interrupted tests still pass.

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
- Command: `pose lint-spec pose-blocked-semantics-alignment --strict` and `pose check --strict`
- Scope: this spec and the instance
- Expected: pass

### Requirement trace

- R1 [satisfied] surface:blocked-semantics evidence:integration <skill, manual, template comment and help describe blocked as non-terminal> test:TestLintWarnsOnBlockedWithoutACauseAndNeverFails check:blocked-semantics-integration
- R2 [satisfied] test:TestBlockedReadinessIsNonTerminalAndExplainsItsCause check:blocked-semantics-integration test:TestStateListsBlockedSpecsWithTheirCause
- R3 [satisfied] test:TestAdoptionMetricsKeepsV1AndAddsAV2WithoutBlockedAsResolved check:blocked-semantics-integration
- R4 [satisfied] <existing interrupted-transfer tests kept passing> test:TestSpecTransferInterruptedOperationsResumeWithoutDuplicateAuthority check:blocked-semantics-integration test:TestSpecTransferReconcileTerminalResumesAfterInterruption
- R5 [satisfied] test:TestNoCommandRewritesALegacyBlockedSpec check:blocked-semantics-integration
- R6 [satisfied] test:TestLintWarnsOnBlockedWithoutACauseAndNeverFails check:blocked-semantics-integration

## 7. Final Report

### Delivered scope

Delivers non-terminal blocked semantics across readiness, state, metrics v2 and lint. Each requirement is traced to its tests and matrix checks under Validation. Reviewed by agent:claude-opus-5-5 in the same session that implemented it; no separate execution or person reviewed it.

### Follow-ups

None recorded at planning time.
