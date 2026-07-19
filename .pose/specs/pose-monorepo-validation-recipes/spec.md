---
slug: pose-monorepo-validation-recipes
status: done
created_at: 2026-07-18
completed_at: 2026-07-19
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
- [x] Confirm baseline and fixtures against [Bazel concepts](https://bazel.build/basics): the module graph, changed-scope selection and severity/policy machinery all shipped in the two prior roadmap milestones with no realistic multi-language proof; this spec is a proof-and-documentation delivery, not new engine surface.

### Implementation
- [x] Select representative fixture architectures: JS/npm workspace (single-hop dependency), a 3-hop declared graph in the style of fine-grained Bazel targets, and a mixed go+node+python repo with a `criticality: high` shared module. ([reference](https://bazel.build/basics))
- [x] Implement recipes that delegate to native task graphs: every recipe uses only `module-metadata.json` (`dependsOn`, `criticality`) and `validation-matrix.json` — no new orchestrator, no BUILD-file parsing; the doc explicitly shows how a real Bazel/Nx repo fronts its own graph as a structured check instead. ([reference](https://nx.dev/ci/features/affected))
- [x] Run docs-as-tests for scope, widening and failure evidence: `monorepo_recipes_test.go` builds each fixture exactly as documented in `docs-site/docs/monorepo-recipes.md` and asserts the exact `--explain` output shown there, plus severity composition and skip-reason recording in the structured JSON result — the doc and the test cannot drift without one of them failing. ([reference](https://bazel.build/basics))

### Validation
- [x] Run `go test ./pose-mcp/internal/cli/... -run 'Monorepo|Affected|Module'` and retain the result artifact (see §6 and `.pose/reports/`; matched via `-run Recipe`, the actual test-name prefix). ([reference](https://bazel.build/basics))
- [x] Run `pose check --strict` and inspect readiness. ([reference](https://nx.dev/ci/features/affected))

## 5. Decisions

- No ADR: this spec adds fixtures, tests and documentation proving the changed-scope, module-metadata and severity contracts already accepted by ADRs `2026-07-19-explainable-changed-scope-selection` and `2026-07-19-versioned-validation-result-contract`. No new public contract, command or matrix field is introduced.

## 6. Validation

**Strategy:** validate unit behavior, negative/security cases, contract fixtures and an end-to-end consumer path.

### Planned deterministic checks
- Test: `go test ./pose-mcp/internal/cli/... -run 'Monorepo|Affected|Module'`.
- Structure: `pose check --strict`.
- Readiness: `pose lint-spec pose-monorepo-validation-recipes --ready-check`.

### Requirement trace
- R1 [satisfied] fixtures cover JS workspace, Bazel-style declared graph and mixed-language layouts; check:test (TestRecipeJSWorkspaceDependencyWidening, TestRecipeDeclaredGraphTransitiveWidening, TestRecipeMixedLanguageStacksDetection)
- R2 [satisfied] recipes demonstrate metadata, changed scope, severity and shared dependencies together; check:test (TestRecipeMixedLanguageSharedDependencyAlwaysIncluded) report:2026-07-19-standard-validate-native.md
- R3 [satisfied] every documented recipe is executed against its pinned fixture in the same test that asserts the doc's exact output; check:test (TestRecipeJSWorkspaceRootManifestChangeRunsEverything)

### Execution status
Executed on 2026-07-19 with a development build (`pose 0.9.0-dev`, rebuilt from this change):

- `go -C pose-mcp test ./internal/cli -run 'Recipe' -count=1` — SUCCESS (six tests, all passed on first run — output was drafted to match the actual engine, not the reverse).
- `go -C pose-mcp test ./... -count=1` — SUCCESS (full suite).
- `pose check --strict` — SUCCESS; `pose lint-spec pose-monorepo-validation-recipes --strict` — SUCCESS.
- `pose validate --strict --module pose-mcp --report` — SUCCESS (report retained under `.pose/reports/`).
- `mkdocs build --strict` was not run locally (no mkdocs in this environment); the docs CI workflow (`docs.yml`) validates the new nav entry and internal links on push.

## 7. Final Report

### Delivered scope

`docs-site/docs/monorepo-recipes.md` with three docs-as-tests recipes (JS
workspace, declared dependency graph, mixed-language with a shared
high-criticality module); `monorepo_recipes_test.go` executing every
documented command against its exact fixture; mkdocs nav entry. Proves the
validation-platform roadmap's outcome — "representative polyglot and
monorepo fixtures pass under the same result contract" — with executable
evidence instead of prose claims.

### Residual risks

- Examples rot unless executed on every relevant change — mitigated
  structurally: the fixture and the doc are the same source of truth
  (the test builds what the doc shows), so a change to either without the
  other fails CI, not silently drifts.

### Follow-ups

- [open] Verify mkdocs build --strict picks up the new nav entry and internal links cleanly on the next docs.yml CI run. (owner:@pose-maintainers crit:low review:2026-08-14)
