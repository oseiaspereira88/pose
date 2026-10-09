---
slug: pose-locale-obligation-flags
status: in-progress
created_at: 2026-10-09
completed_at:
supersedes:
depends_on: pose-locale-coverage-contract
priority: 1
components: pose-mcp, docs
task_type: feature
surface: minimal
changelog: none
delivers: governance:locale-obligation-flags
---

# Spec: Locale parity compares the flags that change an obligation

## 1. Intent

### Goal

Make the locale coverage gate compare, per command, the flags that change what a command obliges or does (`--strict`, `--apply`, `--seal`), so a source and its translation cannot prescribe different behaviour while teaching the same command.

### Business value

Origin: the open follow-up of `pose-locale-coverage-contract` (crit medium, review 2026-10-10), prioritized by the maintainer on 2026-10-09. The gate treated `pose surface-check --strict` and `pose surface-check` as the same command, and only one of them is the doctrine.

### Constraints

Other flags stay out: example lists differ legitimately between a terse source and an example-rich translation, which is why flags were ignored. A flag counts only in the same code span, or fenced code line, as the command it qualifies.

### Non-goals

Comparing prose, or flags that only shape output.

## 2. Requirements

### Functional

- R1: The locale coverage gate shall fail when a source teaches a command with an obligation flag that the translation teaches without it, or the reverse.
- R2: An obligation flag outside its command's code span shall not count.
- R3: The skills that differed shall prescribe the same obligations in both languages.

### Compatibility

- Five skills differed when the comparison was first run: the pt-BR `pose-review` asked for `pose validate` "proportional to risk" where the source requires `pose validate --strict`, and the English `pose-feature`, `pose-knowledge`, `pose-recurrence-escalation` and `pose-spec-closeout` omitted obligation flags their translations taught. Each was aligned to the stricter or more explicit side.

## 3. Technical Plan

### Artifacts

- created: .pose/specs/2026-10-09-pose-locale-obligation-flags.md
- modified: pose-mcp/internal/scaffold/skill_locale_parity_test.go
- modified: pose-mcp/internal/scaffold/locale_coverage_test.go
- modified: .agents/skills/pose-feature/SKILL.md
- modified: .agents/skills/pose-knowledge/SKILL.md
- modified: .agents/skills/pose-recurrence-escalation/SKILL.md
- modified: .agents/skills/pose-spec-closeout/SKILL.md
- modified: locales/pt-BR/.agents/skills/pose-review/SKILL.md
- modified: pose-mcp/internal/scaffold/dist/.agents/skills/pose-feature/SKILL.md
- modified: pose-mcp/internal/scaffold/dist/.agents/skills/pose-knowledge/SKILL.md
- modified: pose-mcp/internal/scaffold/dist/.agents/skills/pose-recurrence-escalation/SKILL.md
- modified: pose-mcp/internal/scaffold/dist/.agents/skills/pose-spec-closeout/SKILL.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/.agents/skills/pose-review/SKILL.md
- modified: .pose/indexes/validation-matrix.json
- modified: .pose/specs/2026-08-10-pose-locale-coverage-contract.md

### Delivery targets

- governance:locale-obligation-flags module:pose-mcp profile:release-governance entrypoint:pose-mcp/cmd/pose/main.go

## 5. Decisions

### Decision D1
- Date: 2026-10-09
- Context: the follow-up asked whether flags that change an obligation should be compared, and how without reintroducing noise.
- Options considered: (a) compare a closed set of obligation flags per command span; (b) compare every flag; (c) keep ignoring flags.
- Decision: (a).
- Rationale: (b) is the noise the original gate avoided; (c) leaves the doctrine unguarded. A closed set keeps the comparison about behaviour.
- Consequences: a new flag that changes an obligation must be added to `obligationFlags`.

## 6. Validation

### Strategy

Measured first against the current tree: the comparison reported five skills, each a real difference in prescribed behaviour, and none in the manuals, rules, workflows or templates. With the documents restored to their previous text, `TestLocaleCoverage` fails on those five; with them aligned it passes. `TestLocaleObligationFlagsDistinguishAGateFromAReport` reports `surface-check --strict` taught as a plain report and ignores a flag mentioned outside the command's span.

### Deterministic checks

#### Test
- Command: `cd pose-mcp && go test ./internal/scaffold -run 'LocaleCoverage|LocaleObligationFlags'`
- Scope: pose-mcp
- Expected: pass

### Requirement trace

- R1 [satisfied] governance:locale-obligation-flags check:locale-obligation-flags-integration evidence:integration test:TestLocaleCoverage
- R2 [satisfied] governance:locale-obligation-flags check:locale-obligation-flags-integration evidence:integration test:TestLocaleObligationFlagsDistinguishAGateFromAReport
- R3 [satisfied] governance:locale-obligation-flags check:locale-obligation-flags-integration evidence:integration test:TestLocaleCoverage

## 7. Final Report

### Delivered scope

The locale coverage gate compares `command --strict|--apply|--seal` per code span in both directions. Five skills were aligned: pt-BR `pose-review` now requires `pose validate --strict`, and the English `pose-feature`, `pose-knowledge`, `pose-recurrence-escalation` and `pose-spec-closeout` name the obligation flags their translations already taught.

### Residual risks

- Reviewed in the same session that implemented it, with no separate reviewer execution: declared, not independent.

### Follow-ups
