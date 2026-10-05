---
slug: pose-flat-spec-amendments
status: done
created_at: 2026-10-04
completed_at: 2026-10-05
supersedes:
depends_on: pose-legacy-contract-cutoffs
priority: 1
components: pose-mcp
delivers: surface:flat-spec-amendments
task_type: bugfix
---

# Spec: Give flat dated specs the same amendment integrity as folder specs

## 1. Intent

### Goal

Resolve the amended spec through the canonical store so `pose amend` works for flat
dated specs, with a collision-free journal per spec.

### Business value

`pose amend` builds only `.pose/specs/<slug>/spec.md` while the flow creates flat dated
specs, so material changes to those end up as free text.

### Constraints

Use the existing spec resolution; do not create a second catalog. Old journals stay
readable without silent migration.

Program source: backlog items POSE-07 (P1, wave 0) of the
[third consolidated analysis](../reports/2026-10-03-pose-consolidated-analysis.md) (findings F03, F10; sources
E19, E26, E29). Cross-cutting decisions: [ADR](../adr/2026-10-04-obligations-are-projected-action-requests-are-persisted.md). Owner proposed:
@pose-maintainers. This is a planning spec: no requirement is declared satisfied.

### Non-goals

Reformatting existing specs; activating contract nodes without the pilot.

### Anti-mechanization guardrail

Usar resolução canônica; não inventar um segundo catálogo de specs.

## 2. Requirements

### Functional

- R1: `pose amend` shall resolve a flat dated spec and a folder spec through the same store resolver and record the same amendment for both.
- R2: Two flat specs in the same directory shall never share an `amendments.jsonl`; the journal path is derived from the spec identity.
- R3: Amendment IDs, hashes and R/A/D history shall be preserved as the applicable capability defines.
- R4: Existing folder journals shall keep being read unchanged; no command migrates them implicitly.
- R5: The MCP amend surface, if present, shall share the same domain function.

### Non-functional

- None specific beyond the shared constraints.

### Security

- Path derivation rejects traversal and symlinks leaving `.pose/specs`.

### Compatibility

- Folder layout unchanged.

## 3. Technical Plan

### Affected areas

Amend CLI, amendments domain, store resolution.

### Artifacts

- created: .pose/specs/2026-10-04-pose-flat-spec-amendments.md
- modified: pose-mcp/internal/pose/amendments.go
- created: pose-mcp/internal/pose/amendments_flat_test.go
- modified: pose-mcp/internal/cli/amend.go
- modified: pose-mcp/internal/cli/spec_format.go
- created: pose-mcp/internal/cli/amend_flat_test.go
- modified: POSE.md
- modified: locales/pt-BR/POSE.md
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md
- modified: .pose/indexes/validation-matrix.json
- created: .pose/changelogs/unreleased/pose-flat-spec-amendments.md

Reconciled against the tree at activation.

### Delivery targets

- surface:flat-spec-amendments module:pose-mcp profile:cli-surface entrypoint:pose-mcp/cmd/pose/main.go

### Technical risks

- Journal path choice for flat specs must be stable; decided once and documented.

## 4. Tasks

### Planning
- [x] Activate: confirm intent, re-read the cited sources at HEAD, reconcile Artifacts
- [ ] Run `pose assess discover --component pose-mcp` if the state is stale for the touched area

### Implementation
- [ ] Write the failing tests named in Validation first (the gate must fail before it passes)
- [ ] Implement incrementally, one requirement group per commit with `POSE-Spec: pose-flat-spec-amendments`

### Validation
- [ ] Run the deterministic checks below and retain results

## 5. Decisions

No material decision beyond the transversal ADR at planning time.

## 6. Validation

### Strategy

Failing test first: amend on a flat fixture spec errors today. Then parity and isolation
tests.

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
- Command: `pose lint-spec pose-flat-spec-amendments --strict` and `pose check --strict`
- Scope: this spec and the instance
- Expected: pass

### Requirement trace

- R1 [satisfied] surface:flat-spec-amendments evidence:integration test:TestAmendWorksForFlatSpecsWithAJournalPerSpec check:flat-spec-amendments-integration
- R2 [satisfied] test:TestAmendmentsPathIsPerSpecForFlatAndUnchangedForFolders check:flat-spec-amendments-integration
- R3 [satisfied] test:TestAmendWorksForFlatSpecsWithAJournalPerSpec check:flat-spec-amendments-integration
- R4 [satisfied] test:TestAmendmentsPathIsPerSpecForFlatAndUnchangedForFolders check:flat-spec-amendments-integration
- R5 [satisfied] <pose_spec_amendments reads through the same store resolver> test:TestAmendWorksForFlatSpecsWithAJournalPerSpec check:flat-spec-amendments-integration

## 7. Final Report

### Delivered scope

Delivers per-spec journals for flat specs through the store resolver. Each requirement is traced to its tests and matrix checks under Validation. Reviewed by agent:claude-opus-5-5 in the same session that implemented it; no separate execution or person reviewed it.

### Follow-ups

None recorded at planning time.
