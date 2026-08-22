---
slug: pose-locales-alignment
status: done
created_at: 2026-08-04
completed_at: 2026-08-04
changelog: none
supersedes:
depends_on:
priority: 10
---

# Spec: Locales alignment and delivery profile registration

## 1. Intent

### Goal
Align Portuguese localized rules, workflows and skills with the English canonical source, register the `backend-go` governance profile in the validation matrix, and release `v0.16.5`.

### Business value
Guarantees zero scope loss when upgrading localized POSE instances and ensures strict delivery integrity validation.

### Constraints
- Retain exact section parity and explicit CLI commands across all localized artifacts.

### Non-goals
- Change existing CLI behavior or break existing release policy contracts.

## 2. Requirements

### Functional
- R1: All 38 Portuguese rules, workflows and skills shall match the canonical English structure.
- R2: The `backend-go` profile shall be registered in `deliveryProfiles` of the validation matrix.

### Non-functional
- High fidelity and full deterministic validation pass.

### Security
- Maintain clean release provenance and immutable tags.

### Compatibility
- Full backward compatibility across `v0.9.0+` releases.

## 3. Technical Plan

### Affected areas
- `locales/pt-BR/`, `.pose/indexes/validation-matrix.json`, version literals and scaffold dist.

### API/contract changes
- None.

### Data/storage changes
- None.

### Technical risks
- None.

### Primary references
- [POSE Architecture](../../POSE.md)

### Artifacts
- none: documentation

### Risk mitigation
- None.

## 4. Tasks

### Planning
- [x] Audit section parity between English and Portuguese artifacts.

### Implementation
- [x] Align Portuguese rules, workflows and skills with English canonical source.
- [x] Register `backend-go` profile in validation-matrix.json.

### Validation
- [x] Run `pose check --strict` and `pose validate`.

## 5. Decisions

- Register `backend-go` as a governance profile to support existing spec delivery targets without breaking spec frontmatter contracts.

## 6. Validation

**Strategy:** Run `pose check --strict` and `pose validate`.

### Planned deterministic checks
- Structure: `pose check --strict`.
- Validation: `pose validate`.

### Requirement trace
- R1 [satisfied] `locales/pt-BR/` rules, workflows and skills aligned.
- R2 [satisfied] `backend-go` profile registered in `validation-matrix.json`.

### Execution status
Executed on 2026-08-04:
- `pose check --strict` — SUCCESS.
- `pose validate` — SUCCESS.

## 7. Final Report

- Delivered scope: Aligned Portuguese localization files and registered `backend-go` profile.
