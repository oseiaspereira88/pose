---
slug: pose-progressive-spec-surface
status: done
created_at: 2026-10-04
completed_at: 2026-10-05
supersedes:
depends_on: pose-effective-governance-projection, pose-state-attention, pose-adaptive-assessment-freshness
priority: 2
components: pose-mcp
delivers: surface:progressive-spec-surface
task_type: feature
---

# Spec: Offer a progressive spec surface and generate derivable factual content

## 1. Intent

### Goal

Show the sections a phase and materiality require, and generate factual content (files
changed, checks run, evidence refs) with source and scope instead of manual
transcription.

### Business value

The template repeats intent, plan, tasks, validation, summary and final report; filling
it competes with solving the problem and invites mechanical final reports.

### Constraints

Authoritative content (intent, requirements, constraints, non-goals, material
assumptions/decisions, delivery intent, dispositions) stays human/agent written. Tasks
are planner-local, not a governance contract.

Program source: backlog items POSE-24 (P2, wave 3) of the
[third consolidated analysis](../reports/2026-10-03-pose-consolidated-analysis.md) (findings F11; sources
E11, E12, E24). Cross-cutting decisions: [ADR](../adr/2026-10-04-obligations-are-projected-action-requests-are-persisted.md). Owner proposed:
@pose-maintainers. This is a planning spec: no requirement is declared satisfied.

### Non-goals

A mini Jira; removing template sections by global rule.

### Anti-mechanization guardrail

Reduzir autoria mecânica sem reduzir accountability de decisões materiais.

## 2. Requirements

### Functional

- R1: A trivial change shall use the minimal pertinent surface while preserving adopted gates.
- R2: Files changed and checks run shall be derivable (`pose spec facts <slug>`) from Git change sets and results without manual transcription.
- R3: The generator shall never invent intent, rationale or accepted risk.
- R4: The reduced view shall keep unknown, coverage and material pending items visible.
- R5: `pose new-spec --surface minimal|standard|full` shall scaffold the corresponding sections, recorded in frontmatter for lint.

### Non-functional

- None specific beyond the shared constraints.

### Security

- None specific beyond the shared constraints.

### Compatibility

- Full template remains valid.

## 3. Technical Plan

### Affected areas

Spec scaffolding, lint, factual summary generator, templates.

### Artifacts

- created: .pose/specs/2026-10-04-pose-progressive-spec-surface.md
- created: pose-mcp/internal/pose/spec_facts.go
- created: pose-mcp/internal/pose/spec_facts_test.go
- modified: pose-mcp/internal/cli/lintspec.go
- modified: pose-mcp/internal/cli/scaffold.go
- modified: pose-mcp/internal/cli/specs_cmd.go
- created: pose-mcp/internal/cli/spec_surface_test.go
- modified: .pose/templates/spec.md
- modified: locales/pt-BR/.pose/templates/spec.md
- modified: pose-mcp/internal/scaffold/dist/.pose/templates/spec.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/.pose/templates/spec.md
- modified: docs-site/docs/frontmatter.md
- modified: POSE.md
- modified: locales/pt-BR/POSE.md
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md
- modified: .pose/indexes/validation-matrix.json
- created: .pose/changelogs/unreleased/pose-progressive-spec-surface.md

Reconciled against the tree at activation.

### Delivery targets

- surface:progressive-spec-surface module:pose-mcp profile:cli-surface entrypoint:pose-mcp/cmd/pose/main.go

### Technical risks

- Hiding obligations; reduced view test.

## 4. Tasks

### Planning
- [x] Activate: confirm intent, re-read the cited sources at HEAD, reconcile Artifacts
- [ ] Run `pose assess discover --component pose-mcp` if the state is stale for the touched area

### Implementation
- [ ] Write the failing tests named in Validation first (the gate must fail before it passes)
- [ ] Implement incrementally, one requirement group per commit with `POSE-Spec: pose-progressive-spec-surface`

### Validation
- [ ] Run the deterministic checks below and retain results

## 5. Decisions

No material decision beyond the transversal ADR at planning time.

## 6. Validation

### Strategy

Minimal surface passes lint with adopted gates; facts generator compared against git;
no-invention test.

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
- Command: `pose lint-spec pose-progressive-spec-surface --strict` and `pose check --strict`
- Scope: this spec and the instance
- Expected: pass

### Requirement trace

- R1 [satisfied] surface:progressive-spec-surface evidence:integration test:TestMinimalSurfaceKeepsGovernanceAndDropsRitual check:delivery-integration test:TestLintAppliesTheSurfaceButNotToLifecycleGates
- R2 [satisfied] <implemented as pose specs facts <slug>> test:TestSpecFactsAreDerivedFromTheIndex check:spec-surface-integration
- R3 [satisfied] <facts carry paths and checks only> test:TestSpecFactsAreDerivedFromTheIndex check:spec-surface-integration
- R4 [satisfied] <trace, follow-up and amendment gates unchanged; facts report their limitations> test:TestLintAppliesTheSurfaceButNotToLifecycleGates check:delivery-integration
- R5 [satisfied] test:TestMinimalSurfaceKeepsGovernanceAndDropsRitual check:delivery-integration

## 7. Final Report

### Delivered scope

Delivers --surface, surface-aware lint and pose specs facts; R2 is implemented as specs facts. Each requirement is traced to its tests and matrix checks under Validation. Reviewed by agent:claude-opus-5-5 in the same session that implemented it; no separate execution or person reviewed it.

### Follow-ups

None recorded at planning time.
