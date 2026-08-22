---
slug: pose-doc-command-reference
status: done
created_at: 2026-08-04
completed_at: 2026-08-04
changelog: none
supersedes:
depends_on:
priority: 10
---

# Spec: Document Command Reference Completeness

## 1. Intent

### Goal
Ensure `assess` and `doctor` command references are present in POSE.md and embedded in the scaffold dist of `v0.16.6` to preserve complete scope during `pose upgrade`.

### Business value
Guarantees zero documentation drift when upgrading POSE instances.

### Constraints
- Retain exact section parity and explicit CLI commands.

### Non-goals
- Change existing CLI behavior.

## 2. Requirements

### Functional
- R1: `POSE.md` and embedded scaffolds shall include `assess` and `doctor` command entries.

### Non-functional
- Deterministic validation pass.

## 3. Technical Plan

### Affected areas
- `POSE.md`, `locales/pt-BR/POSE.md`, `pose-mcp/internal/scaffold/dist/`.

### Artifacts
- none: documentation

### API/contract changes
- None.

## 4. Tasks

- [x] Add `assess` and `doctor` to `POSE.md` and `locales/pt-BR/POSE.md`.
- [x] Sync embedded scaffold dist.
- [x] Cut `v0.16.6`.

## 5. Decisions

- None.

## 6. Validation

**Strategy:** Run `pose check --strict` and `pose validate`.

### Planned deterministic checks
- Structure: `pose check --strict`.
- Validation: `pose validate`.

### Requirement trace
- R1 [satisfied] `POSE.md` and embedded scaffolds include `assess` and `doctor` command entries.

### Execution status
Executed on 2026-08-04:
- `pose check --strict` — SUCCESS.
- `pose validate` — SUCCESS.

## 7. Final Report

- Delivered scope: Documented `assess` and `doctor` command references across POSE.md and embedded scaffold dist.

