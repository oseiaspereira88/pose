---
slug: pose-spec-surface-single-key
status: done
created_at: 2026-10-05
completed_at: 2026-10-05
supersedes:          # slug of the superseded spec (when applicable)
depends_on: 
remediates: spec:pose-progressive-spec-surface@defect-fix
priority: 2
components: pose-mcp
task_type: bugfix
surface: minimal
delivers: surface:progressive-spec-surface
---

# Spec: A minimal spec scaffold carries one surface key

## 1. Intent

### Goal

`pose new-spec --surface` shall fill the template's `surface:` key instead of adding a second one.

### Business value

The first spec scaffolded with `--surface minimal` (pose-attention-federated-parity) had two `surface:` keys: the template gained a commented `surface:` line in pose-progressive-spec-surface, and the scaffold kept inserting its own. A duplicate key leaves which one a reader honours to the parser.

### Constraints

Templates without the key still receive it.

### Non-goals

Changing the surfaces.

## 2. Requirements

### Functional

- R1: A spec scaffolded with `--surface` shall have exactly one `surface:` key, set to the requested surface.

### Non-functional

- None beyond the shared constraints.

### Security

- None.

### Compatibility

- Specs already scaffolded keep their content; the duplicate in pose-attention-federated-parity was removed by hand.

## 3. Technical Plan

### Affected areas

`pose-mcp/internal/cli/scaffold.go`.

### Artifacts

- created: .pose/specs/2026-10-05-pose-spec-surface-single-key.md
- renamed: .pose/changelogs/unreleased/pose-spec-surface-single-key.md -> .pose/changelogs/v7.0.0/pose-spec-surface-single-key.md
- modified: pose-mcp/internal/cli/scaffold.go
- modified: pose-mcp/internal/cli/spec_surface_test.go

### Delivery targets

- surface:progressive-spec-surface module:pose-mcp profile:cli-surface entrypoint:pose-mcp/cmd/pose/main.go

### Technical risks

- None.

## 6. Validation

### Strategy

The minimal-surface scaffold test counts `surface:` keys; it failed with two before the fix.

### Deterministic checks

#### Test
- Command: `cd pose-mcp && go test ./internal/cli -run MinimalSurface`
- Expected: pass

### Requirement trace

- R1 [satisfied] surface:progressive-spec-surface evidence:integration <the scaffold fills the template key> test:TestMinimalSurfaceKeepsGovernanceAndDropsRitual check:spec-surface-integration check:spec-surface-reachability

### Known gaps

None.

## 7. Final Report

### Delivered scope

The scaffold replaces the template's `surface:` line; a template without it still gets the key before `task_type:`.

### Residual risks

None.

### Follow-ups

