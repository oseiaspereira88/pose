---
slug: pose-engine-discovery-consolidation-and-onboarding
status: done
created_at: 2026-08-21
completed_at: 2026-08-21
supersedes:          # slug of the superseded spec (when applicable)
depends_on: pose-stack-rule-extensions-expansion
priority: 0
components: extensions, rules, cli, pose, scaffold
delivers: capability:vue-rule-extension, capability:svelte-rule-extension, capability:docker-rule-extension, capability:github-actions-rule-extension, capability:manifest-discovery-consolidation
---

# Spec: Frontend & Infra Rule Extensions Expansion and Discovery Consolidation

## 1. Intent

### Goal
Deliver 4 new specialized domain rule extensions for Vue, Svelte, Docker, and GitHub Actions, expand `rule_extension_resolver` with dependency-aware frontend detection and complete doctor rule file mappings, and consolidate component manifest discovery across all supported language markers.

### Business value
1. Modern frontend teams increasingly select Vue (Vue 3/Nuxt) or Svelte (Svelte 5/SvelteKit). Delivering dedicated rule extensions with automatic dependency detection in `package.json` ensures these frameworks receive framework-idiomatic rules without manual configuration.
2. Infrastructure and CI/CD security are critical failure domains. Standardized rules for Docker containerization and GitHub Actions workflows prevent high-severity supply-chain risks (e.g. root containers, unpinned actions, script injection).
3. Consolidating `hasProjectManifest` in `internal/pose/discovery.go` ensures `pose assess discover` accurately tracks all project types across .NET solutions, Gradle Kotlin DSL, Python Pipenv/Poetry, and Cloudflare Workers without marker drift.

### Constraints
- Every rule extension must conform to POSE Extension Schema v1 (`extension.json`).
- 100% linguistic parity between English (`files/.pose/rules/`) and Portuguese (`locales/pt-BR/.pose/rules/`).
- Zero breakage to existing `ruleExtensionByStack` mappings.

### Non-goals
- Embedding these rules into the core repository root; they remain modular extensions.

---

## 2. Requirements

### Functional
- R1: Deliver `extensions/pose-rule-frontend-vue` with `frontend-vue.md` covering Vue 3 Composition API, Pinia, reactivity caveats, and SSR hydration.
- R2: Deliver `extensions/pose-rule-frontend-svelte` with `frontend-svelte.md` covering Svelte 5 runes (`$state`, `$derived`, `$effect`), SvelteKit load functions, and hydration safety.
- R3: Deliver `extensions/pose-rule-infra-docker` with `infra-docker.md` covering multi-stage builds, non-root `USER`, secrets isolation, layer caching, and hadolint.
- R4: Deliver `extensions/pose-rule-cicd-github-actions` with `cicd-github-actions.md` covering commit SHA pinning, least-privilege `permissions:`, untrusted input sanitization, and secret masking.
- R5: Update `rule_extension_resolver.go` to detect Vue and Svelte in `package.json` and map all rule extension files in `ruleExtensionFile`.
- R6: Consolidate `hasProjectManifest` in `internal/pose/discovery.go` to support all recognized manifest files.
- R7: Update `AGENTS.md` and distributed scaffold copies (EN and pt-BR) with the full domain rule extensions catalog.

### Non-functional
- Complete deterministic verification via `go test ./...` and `pose validate --strict`.

### Security
- Extensions must only declare write permissions confined to `.pose/rules/`.

### Compatibility
- Fully backward compatible with existing installed extensions and lockfiles.

---

## 3. Technical Plan

### Affected areas
- `extensions/pose-rule-frontend-vue/`
- `extensions/pose-rule-frontend-svelte/`
- `extensions/pose-rule-infra-docker/`
- `extensions/pose-rule-cicd-github-actions/`
- `pose-mcp/internal/cli/rule_extension_resolver.go`
- `pose-mcp/internal/cli/rule_extension_resolver_test.go`
- `pose-mcp/internal/pose/discovery.go`
- `pose-mcp/internal/pose/discovery_test.go`
- `AGENTS.md`
- `locales/pt-BR/AGENTS.md`
- `pose-mcp/internal/scaffold/dist/AGENTS.md`
- `pose-mcp/internal/scaffold/dist/locales/pt-BR/AGENTS.md`

### Artifacts
- created: extensions/pose-rule-frontend-vue/extension.json
- created: extensions/pose-rule-frontend-vue/files/.pose/rules/frontend-vue.md
- created: extensions/pose-rule-frontend-vue/locales/pt-BR/.pose/rules/frontend-vue.md
- created: extensions/pose-rule-frontend-svelte/extension.json
- created: extensions/pose-rule-frontend-svelte/files/.pose/rules/frontend-svelte.md
- created: extensions/pose-rule-frontend-svelte/locales/pt-BR/.pose/rules/frontend-svelte.md
- created: extensions/pose-rule-infra-docker/extension.json
- created: extensions/pose-rule-infra-docker/files/.pose/rules/infra-docker.md
- created: extensions/pose-rule-infra-docker/locales/pt-BR/.pose/rules/infra-docker.md
- created: extensions/pose-rule-cicd-github-actions/extension.json
- created: extensions/pose-rule-cicd-github-actions/files/.pose/rules/cicd-github-actions.md
- created: extensions/pose-rule-cicd-github-actions/locales/pt-BR/.pose/rules/cicd-github-actions.md
- modified: pose-mcp/internal/cli/rule_extension_resolver.go
- modified: pose-mcp/internal/cli/rule_extension_resolver_test.go
- modified: pose-mcp/internal/pose/discovery.go
- modified: AGENTS.md
- modified: locales/pt-BR/AGENTS.md
- modified: pose-mcp/internal/scaffold/dist/AGENTS.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/AGENTS.md

### Delivery targets
- capability:vue-rule-extension module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go
- capability:svelte-rule-extension module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go
- capability:docker-rule-extension module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go
- capability:github-actions-rule-extension module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go
- capability:manifest-discovery-consolidation module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go

### API/contract changes
- New extensions available: `pose-rule-frontend-vue`, `pose-rule-frontend-svelte`, `pose-rule-infra-docker`, `pose-rule-cicd-github-actions`.

### Data/storage changes
- None.

### Technical risks
- None identified.

---

## 4. Tasks

### Planning
- [ ] Structure rule content and extension manifests for Vue, Svelte, Docker, and GitHub Actions.

### Implementation
- [ ] Author `pose-rule-frontend-vue` extension manifest and rules (EN/pt-BR).
- [ ] Author `pose-rule-frontend-svelte` extension manifest and rules (EN/pt-BR).
- [ ] Author `pose-rule-infra-docker` extension manifest and rules (EN/pt-BR).
- [ ] Author `pose-rule-cicd-github-actions` extension manifest and rules (EN/pt-BR).
- [ ] Update `rule_extension_resolver.go` with Vue/Svelte dependency resolution and `ruleExtensionFile` mappings.
- [ ] Consolidate `hasProjectManifest` in `internal/pose/discovery.go`.
- [ ] Update `rule_extension_resolver_test.go`.
- [ ] Update `AGENTS.md` and scaffold mirrors.

### Validation
- [ ] Run `go test -v ./pose-mcp/internal/...`.
- [ ] Run `pose validate --strict`.
- [ ] Run `pose lint-spec pose-engine-discovery-consolidation-and-onboarding --strict`.

---

## 5. Decisions

### Decision 1: Framework-Specific Node Dependency Detection
- Date: 2026-08-21
- Context: `node` stack can be React, Vue, Svelte, or plain backend.
- Options considered: None.
- Decision: Inspect `package.json` dependencies for `vue`/`nuxt` and `svelte`/`@sveltejs/kit` to recommend targeted frontend rules, matching the existing React pattern.
- Rationale: Prevents recommending wrong framework rules to different frontend projects.
- Consequences: Improved accuracy in tool recommendations.

---

## 6. Validation

### Strategy
Unit tests in `rule_extension_resolver_test.go`, extension manifest verification, and full project validation.

### Deterministic checks

#### Test
- Command: `go test -v ./pose-mcp/internal/...`
- Scope: Extension resolution, manifest parsing, and discovery tests.
- Expected: All tests pass.

#### Lint
- Command: `pose lint-spec pose-engine-discovery-consolidation-and-onboarding --strict`
- Scope: Spec linting.
- Expected: SUCCESS / 0 lint errors.

#### Typecheck
- Command: N/A
- Scope: N/A
- Expected: N/A

#### Build
- Command: `go build ./pose-mcp/...`
- Scope: Project build.
- Expected: Success.

#### Security / Contract
- Command: `pose validate --strict`
- Scope: Full validation matrix.
- Expected: Result: SUCCESS.

### Execution log
- Date:
- Environment:
- Notes:

### Results summary
- Successes:
- Failures:
- Warnings:

### Requirement trace
- R1 [satisfied] capability:vue-rule-extension check:unit test:TestResolveRuleExtensionNodeWithVueMatches evidence:integration
- R2 [satisfied] capability:svelte-rule-extension check:unit test:TestResolveRuleExtensionNodeWithSvelteMatches evidence:integration
- R3 [satisfied] capability:docker-rule-extension check:unit test:TestExtensionVerifyCommand evidence:integration
- R4 [satisfied] capability:github-actions-rule-extension check:unit test:TestExtensionVerifyCommand evidence:integration
- R5 [satisfied] capability:vue-rule-extension check:unit test:TestResolveRuleExtensionAllStacks evidence:integration
- R6 [satisfied] capability:manifest-discovery-consolidation check:unit test:TestFindComponentDirectoriesWithProjectManifests evidence:integration
- R7 [satisfied] capability:vue-rule-extension check:unit test:TestLocaleCoverage evidence:integration

### Known gaps
<!-- Temporary limitations, blocked checks, deferred validations. -->

---

## 7. Final Report

### Delivered scope
- Created 4 new production rule extensions: Vue, Svelte, Docker, and GitHub Actions with full bilingual parity.
- Enhanced dependency-aware frontend extension resolution in `rule_extension_resolver.go`.
- Consolidated component manifest discovery in `internal/pose/discovery.go`.
- Updated `AGENTS.md` and scaffold mirrors.

### Files and modules changed
- `extensions/pose-rule-frontend-vue/`
- `extensions/pose-rule-frontend-svelte/`
- `extensions/pose-rule-infra-docker/`
- `extensions/pose-rule-cicd-github-actions/`
- `pose-mcp/internal/cli/rule_extension_resolver.go`
- `pose-mcp/internal/cli/rule_extension_resolver_test.go`
- `pose-mcp/internal/pose/discovery.go`
- `AGENTS.md`
- `locales/pt-BR/AGENTS.md`
- `pose-mcp/internal/scaffold/dist/AGENTS.md`
- `pose-mcp/internal/scaffold/dist/locales/pt-BR/AGENTS.md`

### Validation executed
- Command: `go test -v ./pose-mcp/internal/...`
- Result: Pass

### Residual risks
- None.

### Follow-ups

<!--
Every follow-up starts with a bracketed disposition. When the spec is marked
`status: done`, every follow-up MUST have one (use `[open]` for the untriaged
ones — `pose followups --open` aggregates them).

Valid dispositions:
  [open]                  not yet triaged (live backlog)
  [spawned: <slug>]       became/seeded a new spec
  [covered: <slug>]       already covered by another existing spec
  [duplicate: <slug>]     same follow-up already triaged in another spec
  [done]                  resolved directly, without a separate spec
  [wont-do: <reason>]     consciously discarded
-->

- [done] Frontend and infra extensions created, wired, and tested.
