---
slug: pose-skills-harmonization
status: in-progress
created_at: 2026-08-22
supersedes:
depends_on: pose-review-policy-v2-migration-and-verify-compat
priority: 26
components: cli, scaffold, documentation
delivers: surface:skills-harmonization
---

# Spec: Harmonization of Agent Skills with Modern Spec Layout, Assessment Tools, and Review Bundles

## 1. Intent

### Goal
Harmonize the POSE Agent Skills across English and Portuguese locales (`.agents/skills/` and `locales/pt-BR/.agents/skills/`) to reflect modern engine capabilities:
1. Reference the default dated flat spec format `.pose/specs/YYYY-MM-DD-<slug>.md` (and `--folder` for multi-file envelopes) in `pose-feature` and `pose-spec-closeout`.
2. Include explicit execution timing for `pose assess discover` and `pose assess tech-debt` in `pose-feature`, `pose-review`, and `pose-spec-closeout`.
3. Include standard `pose review bundle --seal` and `pose review auto-attest` flows in review and closeout skills.
4. Maintain 100% command parity across all locales and verify via `pose skills-check` and `scaffold` unit tests.

### Business value
Prevents agent friction when following skills by providing up-to-date commands matching current engine defaults, ensuring seamless interoperability with Claude Code, Antigravity, and MCP clients.

### Constraints
- Zero command disparity between English and translated locales.
- Pass `pose skills-check --strict` and `pose check --strict`.

---

## 2. Requirements

### Functional
- R1: `pose-feature` skill shall teach dated flat spec format creation and `pose assess discover` discovery timing.
- R2: `pose-review` skill shall teach `pose assess tech-debt` and `pose assess integrate` during PR review, alongside `pose review bundle --seal` and `pose review auto-attest`.
- R3: `pose-spec-closeout` skill shall teach review bundle sealing, auto-attestation, and `pose assess discover --update-state` metrics refresh upon closure.
- R4: All changes shall maintain complete command parity between English and Portuguese (`locales/pt-BR`).

### Non-functional
- Full test pass in `internal/scaffold` and `internal/cli`.
- Embedded scaffold synchronized via `go generate ./internal/scaffold`.

---

## 3. Technical Plan

### Affected areas
- `.agents/skills/pose-feature/SKILL.md`
- `.agents/skills/pose-spec-closeout/SKILL.md`
- `.agents/skills/pose-review/SKILL.md`
- `locales/pt-BR/.agents/skills/pose-feature/SKILL.md`
- `locales/pt-BR/.agents/skills/pose-spec-closeout/SKILL.md`
- `locales/pt-BR/.agents/skills/pose-review/SKILL.md`
- `pose-mcp/internal/scaffold/dist/`

### Delivery targets
- surface:skills-harmonization module:pose-mcp profile:cli-surface entrypoint:pose-mcp/cmd/pose/main.go

---

## 4. Tasks

### Artifacts
- modified: .agents/skills/pose-feature/SKILL.md
- modified: .agents/skills/pose-spec-closeout/SKILL.md
- modified: .agents/skills/pose-review/SKILL.md
- modified: locales/pt-BR/.agents/skills/pose-feature/SKILL.md
- modified: locales/pt-BR/.agents/skills/pose-spec-closeout/SKILL.md
- modified: locales/pt-BR/.agents/skills/pose-review/SKILL.md
- modified: pose-mcp/internal/scaffold/dist/.agents/skills/pose-feature/SKILL.md
- modified: pose-mcp/internal/scaffold/dist/.agents/skills/pose-spec-closeout/SKILL.md
- modified: pose-mcp/internal/scaffold/dist/.agents/skills/pose-review/SKILL.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/.agents/skills/pose-feature/SKILL.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/.agents/skills/pose-spec-closeout/SKILL.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/.agents/skills/pose-review/SKILL.md

### Increments
- [x] Increment 1: Update English skill definitions (R1, R2, R3).
- [x] Increment 2: Update Portuguese skill translations with exact command parity (R4).
- [x] Increment 3: Regenerate embedded scaffold and verify with `TestSkillLocaleParity` and `TestEmbeddedDistMatchesPoseDist`.

---

## 5. Validation

### Automated
- `TMPDIR=/home/go/.cache/tmp go test ./internal/scaffold -v`
- `pose skills-check --strict`
- `pose check --strict`

### Requirement trace
- R1 [satisfied] surface:skills-harmonization check:delivery-integration test:TestSkillLocaleParity evidence:integration
- R2 [satisfied] surface:skills-harmonization check:delivery-integration test:TestSkillLocaleParity evidence:integration
- R3 [satisfied] surface:skills-harmonization check:delivery-integration test:TestSkillLocaleParity evidence:integration
- R4 [satisfied] surface:skills-harmonization check:delivery-integration test:TestLocaleCoverage evidence:integration

---

## 6. Delivery Evidence

### Artifact claims
- surface:skills-harmonization -> pose-mcp/cmd/pose/main.go

### Known gaps
None.

---

## 7. Final Report

### Delivered scope
- Updated skills to guide agents on modern flat spec naming, assessment tooling, and review bundle flows.
- Ensured 100% parity across English and Portuguese skill trees.
- Synchronized embedded distribution files.

### Follow-ups
- [done] All requirements delivered and verified.
