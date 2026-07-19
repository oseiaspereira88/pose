---
slug: pose-changed-scope-validation
status: draft
created_at: 2026-07-18
completed_at:
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
- [ ] Confirm baseline and fixtures against [Nx affected model](https://nx.dev/ci/features/affected).

### Implementation
- [ ] Define deterministic revision, dependency and policy inputs. ([reference](https://nx.dev/ci/features/affected))
- [ ] Implement affected traversal with conservative fallback. ([reference](https://git-scm.com/docs/gitrevisions))
- [ ] Test rename, delete, shared library and uncertain dependency cases. ([reference](https://nx.dev/ci/features/affected))

### Validation
- [ ] Run `go test ./pose-mcp/internal/cli/... -run 'Changed|Affected|Module'` and retain the result artifact. ([reference](https://nx.dev/ci/features/affected))
- [ ] Run `pose check --strict` and inspect readiness. ([reference](https://git-scm.com/docs/gitrevisions))

## 5. Decisions

- Create an ADR before changing this contract; compare alternatives against [Nx affected model](https://nx.dev/ci/features/affected).

## 6. Validation

**Strategy:** validate unit behavior, negative/security cases, contract fixtures and an end-to-end consumer path.

### Planned deterministic checks
- Test: `go test ./pose-mcp/internal/cli/... -run 'Changed|Affected|Module'`.
- Structure: `pose check --strict`.
- Readiness: `pose lint-spec pose-changed-scope-validation --ready-check`.

### Execution status
- Not executed. This planning spec remains `draft`; delivery requires gate evidence.

## 7. Final Report

- Delivered scope: none; this spec defines future implementation.
- Residual risk: Incomplete graphs can create false negatives; expose confidence.
- Follow-ups: none until implementation starts.
