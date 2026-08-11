---
slug: pose-managed-doc-project-identity-regression
status: done
created_at: 2026-08-10
completed_at: 2026-08-11
supersedes:
depends_on: pose-manual-distribution-merge
priority: 0
components: pose-mcp, scaffold, docs
delivers: governance:managed-doc-project-identity
---

# Spec: Preserve project identity in managed manuals

## 1. Intent

### Goal
Restore project-name placeholders in distributed manual sources so install and upgrade render each instance's own identity.

### Business value
Prevent a POSE upgrade from rewriting every consuming repository as `pose-dist`.

### Constraints
- Preserve instance-owned sections and locale selection.
- Use the existing placeholder restoration path; do not add a second identity mechanism.

### Non-goals
- Change merge semantics or add new scaffold placeholders.

## 2. Requirements

### Functional
- R1: The distributed `AGENTS.md` and `POSE.md` sources shall retain `{{PROJECT_NAME}}` until install or managed-doc refresh resolves it for the target repository.
- R2: Refreshing a stale installed manual shall preserve its existing project name and instance-owned sections.

### Non-functional
- Keep source and embedded scaffold byte-for-byte aligned after generation.

### Security
- Resolve identity only from the existing installed manual, confined `.mcp.json` or target directory fallback.

### Compatibility
- Preserve English and pt-BR refresh behavior and all existing placeholder names.

## 3. Technical Plan

### Affected areas
- Canonical `AGENTS.md`/`POSE.md` templates and the generated embedded scaffold.

### Delivery targets
- governance:managed-doc-project-identity module:pose-mcp profile:release-governance entrypoint:pose-mcp/cmd/pose/main.go

### Artifacts
- created: .pose/reports/2026-08-10-standard-pose-managed-doc-project-identity-regression.md
- created: .pose/reports/history/standard-pose-managed-doc-project-identity-regression.jsonl
- created: .pose/specs/pose-managed-doc-project-identity-regression/spec.md
- modified: AGENTS.md
- modified: POSE.md

### API/contract changes
- None; restore the documented template contract.

### Data/storage changes
- None.

### Technical risks
- A literal placeholder could leak if generation or install substitution regresses; existing tests reject both drift and unresolved placeholders.

## 4. Tasks

### Planning
- [x] Reproduce the managed-doc refresh failure.
- [x] Isolate the hardcoded template identity as root cause.

### Implementation
- [x] Restore canonical project-name placeholders.
- [x] Regenerate the embedded scaffold.

### Validation
- [x] Run managed-doc, locale, scaffold and full module regression checks.

## 5. Decisions

- Reuse the established placeholder contract rather than teaching refresh to replace the product repository's literal name. This keeps one source of truth and the smallest rollback (`git revert` of the two template lines). Consulted `knowledge:contract-baseline-handoff` for the mandatory scaffold regeneration rule.

## 6. Validation

### Strategy
Run the previously failing refresh test, adjacent locale/placeholder tests and embedded drift guard, then the full Go suite.

### Deterministic checks

#### Test
- Command: `go -C pose-mcp test ./internal/cli -run 'RefreshManagedDocs|ManagedDoc' -count=1`
- Scope: refresh identity and instance-owned content.
- Expected: pass.

#### Lint
- Command: `go -C pose-mcp vet ./...`
- Scope: module.
- Expected: pass.

#### Typecheck
- Command: `go -C pose-mcp test ./... -run '^$'`
- Scope: module.
- Expected: pass.

#### Build
- Command: `go -C pose-mcp build ./cmd/pose`
- Scope: native CLI.
- Expected: pass.

#### Security / Contract
- Command: `go -C pose-mcp test ./internal/scaffold -run 'EmbeddedDist|ManualLocaleParity' -count=1`
- Scope: distributed manuals.
- Expected: pass.

### Execution log
- Date: 2026-08-10
- Environment: local Linux workspace.
- Notes: initial full-suite failure reproduced at `TestRefreshManagedDocsUpdatesAnInstalledManual`.

### Results summary
- Successes: the formerly failing refresh test, adjacent managed-doc tests, locale parity, embedded drift, full Go suite and tolerant module validation passed.
- Failures: none after restoring the placeholders.
- Warnings: none.

### Requirement trace
- R1 [satisfied] governance:managed-doc-project-identity evidence:integration test:TestEmbeddedDistMatchesPoseDist test:TestRefreshManagedDocsUpdatesAnInstalledManual — canonical and embedded manuals retain the project placeholder until target rendering.
- R2 [satisfied] test:TestRefreshManagedDocsUpdatesAnInstalledManual test:TestRefreshManagedDocsKeepsTheInstalledLocale — project identity, instance-owned body and locale survive refresh.

### Known gaps
- None.

## 7. Final Report

### Delivered scope
Restored the established project-name placeholder contract in both distributed manual templates and their embedded copies.

### Files and modules changed
- Root/embedded `AGENTS.md` and `POSE.md`, bugfix spec and changelog fragment.

### Validation executed
- Command: managed-doc/scaffold tests, full Go suite, vet, build and tolerant POSE module validation.
- Result: all passed.

### Residual risks
- Future manual template edits still require scaffold regeneration; the embedded drift test enforces this mechanically.

### Follow-ups
- [done] Root cause corrected directly; no systemic decision log is needed because the existing placeholder design remains valid.
