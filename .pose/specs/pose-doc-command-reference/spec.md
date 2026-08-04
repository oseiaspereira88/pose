---
slug: pose-doc-command-reference
status: draft
created_at: 2026-08-04
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

## 4. Tasks

- [x] Add `assess` and `doctor` to `POSE.md` and `locales/pt-BR/POSE.md`.
- [x] Sync embedded scaffold dist.
- [x] Cut `v0.16.6`.

## 5. Validation

- Run `pose check --strict` and `pose validate`.
