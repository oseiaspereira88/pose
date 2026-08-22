---
slug: pose-changed-scope-validation
status: done
created_at: 2026-07-18
completed_at: 2026-07-19
supersedes:
depends_on: pose-structured-validation-results
priority: 16
---

# Spec: Explainable changed-scope validation

## 1. Intent

### Goal
select minimum safe checks from changes, dependency metadata and policy.
### Business value
Reduces monorepo feedback time without hiding skipped work.
### Constraints
- Selection must be deterministic for the same base/head and config.
### Non-goals
- Promise perfect semantic impact analysis from paths.

## 2. Requirements

### Functional
- R1: The command shall accept explicit revisions and emit affected modules plus reasons.
- R2: Policy may widen selection; uncertainty shall prefer safe execution.
- R3: Every unselected check shall have a machine-readable skip reason.

### Non-functional
- Keep selection fast and cache only immutable inputs.

### Security
- Confine Git paths and reject unsafe revisions.

### Compatibility
- Without changed-scope flags, preserve full validation.

## 3. Technical Plan

### Affected areas
- Repository index, module graph, validation selection, CLI and results.

### API/contract changes
- Add revision and changed-scope options with explain output.

### Data/storage changes
- Extend indexes with explicit dependency edges.

### Technical risks
- Incomplete graphs can create false negatives; expose confidence.

### Primary references
- [Nx affected model](https://nx.dev/ci/features/affected)
- [Git revision selection](https://git-scm.com/docs/gitrevisions)

## 4. Tasks

### Planning
- [x] Confirm baseline and fixtures against [Nx affected model](https://nx.dev/ci/features/affected): `pose validate` always ran the full matrix; no revision-based selection existed; `module-metadata.json` had no dependency edges yet.

### Implementation
- [x] Define deterministic revision, dependency and policy inputs: `--changed-from <rev> [--changed-to <rev>]` with a safe revision grammar rejecting option injection before Git runs; `dependsOn` edges added to `module-metadata.json`; policy widening for `criticality: high` (ADR `2026-07-19-explainable-changed-scope-selection`). ([reference](https://nx.dev/ci/features/affected))
- [x] Implement affected traversal with conservative fallback: longest-prefix module match for changed (tracked + untracked worktree) files, fixed-point transitive dependency widening, and a root-level/unmapped change selecting every module — uncertainty always prefers safe execution. ([reference](https://git-scm.com/docs/gitrevisions))
- [x] Test rename, delete, shared library and uncertain dependency cases: `validate_scope_test.go` covers direct selection with `--explain` reasons, dependency-chain widening, root-level change running everything, all-skipped-with-reasons when nothing changed, and unsafe-revision rejection; unselected checks are recorded `skipped` in the structured result with a machine-readable reason (composes with the result contract from milestone 1). ([reference](https://nx.dev/ci/features/affected))

### Validation
- [x] Run `go test ./pose-mcp/internal/cli/... -run 'Changed|Affected|Module'` and retain the result artifact (see §6 and `.pose/reports/`). ([reference](https://nx.dev/ci/features/affected))
- [x] Run `pose check --strict` and inspect readiness. ([reference](https://git-scm.com/docs/gitrevisions))

## 5. Decisions

- ADR `.pose/adr/2026-07-19-explainable-changed-scope-selection.md` (Accepted): path prefixes + declared dependency edges + policy widening with reasons everywhere, over per-language build-graph analysis (contradicts the thin deterministic core and the non-goal against perfect semantic impact analysis).

## 6. Validation

**Strategy:** validate unit behavior, negative/security cases, contract fixtures and an end-to-end consumer path.

### Planned deterministic checks
- Test: `go test ./pose-mcp/internal/cli/... -run 'Changed|Affected|Module'`.
- Structure: `pose check --strict`.
- Readiness: `pose lint-spec pose-changed-scope-validation --ready-check`.

### Requirement trace
- R1 [satisfied] explicit revisions resolved into affected modules plus reasons via --explain; check:test (TestChangedScopeSelectsAffectedModule, TestChangedScopeDependencyWidening)
- R2 [satisfied] policy (criticality high) widens selection; uncertainty (root-level/unmapped change) selects everything; check:test (TestChangedScopeRootChangeRunsEverything)
- R3 [satisfied] every unselected check recorded skipped with a machine-readable reason; check:test (TestChangedScopeNoChangesSkipsAllWithReasons, TestChangedScopeRejectsUnsafeRevision) report:2026-07-19-standard-validate-native.md

### Execution status
Executed on 2026-07-19 with a development build (`pose 0.9.0-dev`, rebuilt from this change):

- `go -C pose-mcp test ./internal/cli -run 'ChangedScope' -count=1` — SUCCESS (five tests over a real git fixture repo).
- `go -C pose-mcp test ./... -count=1` — SUCCESS (full suite).
- `pose check --strict` — SUCCESS; `pose lint-spec pose-changed-scope-validation --strict` — SUCCESS.
- `pose validate --strict --module pose-mcp --report` — SUCCESS (report retained under `.pose/reports/`).

## 7. Final Report

### Delivered scope

Deterministic changed-scope selection (`--changed-from/--changed-to`) over
declared dependency edges and policy widening, with a safe-execution
fallback for root-level or unmapped changes; `--explain` prints every
decision; unselected checks surface as machine-readable skips in the
structured result; revision confinement rejects option injection;
operating-manual documentation and ADR.

### Residual risks

- Path-based selection cannot see semantic coupling outside declared
  `dependsOn` edges — the non-goal accepts this; teams own edge declaration.

### Follow-ups

- [open] Seed dependsOn edges for this repository's real modules (pose-mcp, mcp-enforce) once cross-module coupling needs scoped validation. (owner:@pose-maintainers crit:low review:2026-10-16)
