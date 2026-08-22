---
slug: pose-release-closeout-skill-policy-link
status: in-progress
created_at: 2026-08-22
supersedes:
depends_on: pose-review-bundle-component-discovery-and-governance-paths
priority: 25
components: skills, locales, scaffold
delivers: surface:release-closeout-skill-link
---

# Spec: Release Closeout Skill Canonical Policy Link Correction

## 1. Intent

### Goal
Fix issue #32 where the `pose-release-closeout` skill contained a relative link to `.pose/release-policy.json`, causing `pose skills-check --strict` to fail on freshly initialized schema-v1 instances where that legacy file is not installed.

### Business value
Ensures all 22 shipped skills pass strict conformance checks out-of-the-box in new or updated schema-v1 POSE repositories without errors or manual workarounds.

### Constraints
- Every relative link in shipped skills must resolve to an installed resource.
- Complete parity between English and Portuguese (`locales/pt-BR/`).
- Scaffold dist copies must match root skills.

### Non-goals
- Changing the release workflow or verification steps.

---

## 2. Requirements

### Functional
- R1: `.agents/skills/pose-release-closeout/SKILL.md` and `locales/pt-BR/.agents/skills/pose-release-closeout/SKILL.md` shall reference `[Changelog and release policy](../../../.pose/policy/changelog.json)` instead of the legacy non-installed `.pose/release-policy.json`.
- R2: `pose skills-check --strict` shall pass with zero errors in clean schema-v1 instances.
- R3: Embedded scaffold distribution files shall be synced to match.

### Non-functional
- Zero warnings or errors in `pose check --strict` and `pose skills-check --strict`.

---

## 3. Technical Plan

### Affected areas
- `.agents/skills/pose-release-closeout/SKILL.md`
- `locales/pt-BR/.agents/skills/pose-release-closeout/SKILL.md`
- `pose-mcp/internal/scaffold/dist/...`

### Delivery targets
- surface:release-closeout-skill-link module:pose-mcp profile:cli-surface entrypoint:pose-mcp/cmd/pose/main.go

---

## 4. Tasks

### Artifacts
- modified: .agents/skills/pose-release-closeout/SKILL.md
- modified: locales/pt-BR/.agents/skills/pose-release-closeout/SKILL.md
- modified: pose-mcp/internal/scaffold/dist/.agents/skills/pose-release-closeout/SKILL.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/.agents/skills/pose-release-closeout/SKILL.md

### Increments
- [x] Increment 1: Update relative policy links in EN and pt-BR skills (R1).
- [x] Increment 2: Sync embedded scaffold assets (R3).
- [x] Increment 3: Validate skills conformance (R2).

---

## 5. Validation

### Automated
- `pose skills-check --strict`
- `go test ./internal/scaffold/...`
- `pose check --strict`
- `pose validate --strict`

### Requirement trace
- R1 [satisfied] surface:release-closeout-skill-link check:delivery-integration test:TestClaudeSkillLinksMatchAgentsSkills evidence:integration
- R2 [satisfied] surface:release-closeout-skill-link check:delivery-integration test:TestSkillsCheckDiscoveryAndBoundedWorkflowFixture evidence:integration
- R3 [satisfied] surface:release-closeout-skill-link check:delivery-integration test:TestSkillLocaleParity evidence:integration

---

## 6. Delivery Evidence

### Artifact claims
- surface:release-closeout-skill-link -> pose-mcp/cmd/pose/main.go

### Known gaps
None.

---

## 7. Final Report

### Delivered scope
- Corrected release policy link in `pose-release-closeout` skill to point to canonical `.pose/policy/changelog.json` in English and Portuguese, resolving issue #32.
- Synchronized distribution scaffold and verified strict skill conformance.

### Follow-ups
- [done] All requirements delivered and verified.
