---
slug: pose-monorepo-validation-recipes
status: draft
created_at: 2026-07-18
completed_at:
supersedes:
depends_on: pose-changed-scope-validation, pose-stack-catalog-expansion
priority: 19
---

# Spec: Monorepo validation recipes

## 1. Intent

### Goal
publish executable recipes for workspace, task-graph and mixed-language monorepositories.
### Business value
Turns the module model into practical complex-repo adoption guidance.
### Constraints
- Delegate to native build graphs rather than duplicating them.
### Non-goals
- Build a new monorepo orchestrator.

## 2. Requirements

### Functional
- R1: Fixtures shall cover JavaScript workspaces, Bazel-style graphs and mixed languages.
- R2: Recipes shall demonstrate metadata, changed scope, severity and shared dependencies.
- R3: CI shall execute every documented recipe against pinned fixtures.

### Non-functional
- Keep fixtures small but behaviorally realistic.

### Security
- Use structured commands and confined module roots.

### Compatibility
- Repos without monorepo metadata retain full-repo validation.

## 3. Technical Plan

### Affected areas
- Fixtures, docs, module metadata, tests and CI.

### API/contract changes
- Document supported patterns and non-guarantees.

### Data/storage changes
- Version fixture manifests and expected snapshots.

### Technical risks
- Examples rot unless executed on every relevant change.

### Primary references
- [Bazel concepts](https://bazel.build/basics)
- [Nx affected model](https://nx.dev/ci/features/affected)

## 4. Tasks

### Planning
- [ ] Confirm baseline and fixtures against [Bazel concepts](https://bazel.build/basics).

### Implementation
- [ ] Select representative fixture architectures. ([reference](https://bazel.build/basics))
- [ ] Implement recipes that delegate to native task graphs. ([reference](https://nx.dev/ci/features/affected))
- [ ] Run docs-as-tests for scope, widening and failure evidence. ([reference](https://bazel.build/basics))

### Validation
- [ ] Run `go test ./pose-mcp/internal/cli/... -run 'Monorepo|Affected|Module'` and retain the result artifact. ([reference](https://bazel.build/basics))
- [ ] Run `pose check --strict` and inspect readiness. ([reference](https://nx.dev/ci/features/affected))

## 5. Decisions

- Create an ADR before changing this contract; compare alternatives against [Bazel concepts](https://bazel.build/basics).

## 6. Validation

**Strategy:** validate unit behavior, negative/security cases, contract fixtures and an end-to-end consumer path.

### Planned deterministic checks
- Test: `go test ./pose-mcp/internal/cli/... -run 'Monorepo|Affected|Module'`.
- Structure: `pose check --strict`.
- Readiness: `pose lint-spec pose-monorepo-validation-recipes --ready-check`.

### Execution status
- Not executed. This planning spec remains `draft`; delivery requires gate evidence.

## 7. Final Report

- Delivered scope: none; this spec defines future implementation.
- Residual risk: Examples rot unless executed on every relevant change.
- Follow-ups: none until implementation starts.
