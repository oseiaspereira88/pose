---
slug: pose-localization-docs-contract
status: draft
created_at: 2026-07-18
completed_at:
supersedes:
depends_on: pose-public-install-contract, pose-release-compatibility-matrix
priority: 29
---

# Spec: Localization and documentation contract

## 1. Intent

### Goal
keep locales, command examples and public promises synchronized with release behavior.
### Business value
Turns product polish into a release-tested contract.
### Constraints
- Safety and compatibility text cannot ship partially translated.
### Non-goals
- Automatically translate unsupported community content.

## 2. Requirements

### Functional
- R1: Every documented command shall run or parse against the candidate binary.
- R2: Locale key parity and fallback shall be tested for CLI and scaffold.
- R3: Docs shall separate tutorial, how-to, reference and explanation with version applicability.

### Non-functional
- Build docs with strict links and deterministic snippets.

### Security
- Scan examples for secrets, unsafe downloads and permissions.

### Compatibility
- Preserve stable anchors and redirect moved pages where practical.

## 3. Technical Plan

### Affected areas
- MkDocs, README, locales, CLI help, scaffold and docs CI.

### API/contract changes
- Define locale completeness and docs compatibility policy.

### Data/storage changes
- Maintain snippet fixtures and translation key inventories.

### Technical risks
- Generated snippets can hide editorial context.

### Primary references
- [Diátaxis](https://diataxis.fr/)
- [Unicode CLDR](https://cldr.unicode.org/)

## 4. Tasks

### Planning
- [ ] Confirm baseline and fixtures against [Diátaxis](https://diataxis.fr/).

### Implementation
- [ ] Classify pages and define locale/release gates. ([reference](https://diataxis.fr/))
- [ ] Execute snippets and validate links, anchors and fallback keys. ([reference](https://cldr.unicode.org/))
- [ ] Review rendered security-critical and compatibility text. ([reference](https://diataxis.fr/))

### Validation
- [ ] Run `pose check --strict && mkdocs build --strict -f docs-site/mkdocs.yml` and retain evidence. ([reference](https://diataxis.fr/))
- [ ] Run `pose check --strict` and inspect readiness. ([reference](https://cldr.unicode.org/))

## 5. Decisions

- Create an ADR before changing this contract; compare [Diátaxis](https://diataxis.fr/).

## 6. Validation

**Strategy:** validate units, negative/security cases, contract fixtures and an end-to-end path.

### Planned deterministic checks
- Test: `pose check --strict && mkdocs build --strict -f docs-site/mkdocs.yml`.
- Structure: `pose check --strict`.
- Readiness: `pose lint-spec pose-localization-docs-contract --ready-check`.

### Execution status
- Not executed. This planning spec remains `draft`; delivery requires gate evidence.

## 7. Final Report

- Delivered scope: none; this spec defines future implementation.
- Residual risk: Generated snippets can hide editorial context.
- Follow-ups: none until implementation starts.

