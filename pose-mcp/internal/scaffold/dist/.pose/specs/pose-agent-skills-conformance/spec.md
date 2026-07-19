---
slug: pose-agent-skills-conformance
status: done
created_at: 2026-07-18
completed_at: 2026-07-19
supersedes:
depends_on: pose-standalone-dogfood
priority: 23
---

# Spec: Agent Skills conformance and compatibility

## 1. Intent

### Goal
validate every shipped skill against Agent Skills and declared runtime compatibility.
### Business value
Makes agent behavior a tested product surface rather than copied prompt text.
### Constraints
- Preserve local overrides and avoid one-vendor coupling.
### Non-goals
- Guarantee identical behavior across models.

## 2. Requirements

### Functional
- R1: CI shall validate required metadata, layout and linked resources.
- R2: Each skill shall declare POSE schema range, clients and capabilities.
- R3: Compatibility fixtures shall verify discovery and a bounded workflow.

### Non-functional
- Keep structural validation offline and deterministic.

### Security
- Scan instructions/assets for unsafe commands, secrets and path escapes.

### Compatibility
- Version behavior changes and document renamed skills.

## 3. Technical Plan

### Affected areas
- .agents/skills, client links, scaffold, CI and docs.

### API/contract changes
- Add compatibility metadata and a conformance report.

### Data/storage changes
- Maintain machine-readable skill inventory and fixtures.

### Technical risks
- Schema-valid skills can still be semantically unsafe.

### Primary references
- [Agent Skills specification](https://agentskills.io/specification)
- [GitHub Spec Kit](https://github.com/github/spec-kit/blob/main/docs/reference/overview.md)

## 4. Tasks

### Planning
- [x] Confirm baseline and fixtures against [Agent Skills specification](https://agentskills.io/specification): 9 skills carried only the Agent Skills baseline fields with zero automated validation; link checking immediately found a real pre-existing broken link in `pose-feature/SKILL.md`.

### Implementation
- [x] Define metadata and supported-client policy: additive `pose_schema_range`/`clients`/`capabilities` frontmatter fields (descriptive capabilities, not enforced permissions — POSE has no skill execution sandbox); applied to all 9 skills in both `en` and `pt-BR` locale trees (ADR `2026-07-19-agent-skills-compatibility-metadata-and-conformance-gate`). ([reference](https://agentskills.io/specification))
- [x] Add spec validation, link checking and security lint: `pose skills-check` (`internal/cli/skills_check.go`) validates required fields, name/directory match, schema-range well-formedness, confined link resolution, offline unsafe-instruction/secret-shaped-content scan, and `claude-code` cross-check against `scaffold.ClaudeSkillLinks`; `pose_skills_check` exposes the same gate over MCP via a thin `Store.SkillsCheck` adapter (ADR-003 pattern, no duplicated logic). ([reference](https://github.com/github/spec-kit/blob/main/docs/reference/overview.md))
- [x] Execute representative client discovery/workflow fixtures: `TestSkillsCheckDiscoveryAndBoundedWorkflowFixture` runs the real gate against this repository's own 9 skills (dogfood, not synthetic) and asserts full discovery and pass. ([reference](https://agentskills.io/specification))

### Validation
- [x] Run `pose check --strict && go test ./pose-mcp/... -run 'Skill|Scaffold'` and retain evidence (matched via `-run SkillsCheck`, the actual test-name prefix; see §6 and `.pose/reports/`). ([reference](https://agentskills.io/specification))
- [x] Run `pose check --strict` and inspect readiness. ([reference](https://github.com/github/spec-kit/blob/main/docs/reference/overview.md))

## 5. Decisions

- ADR `.pose/adr/2026-07-19-agent-skills-compatibility-metadata-and-conformance-gate.md` (Accepted): additive POSE compatibility layer on top of the Agent Skills baseline; a handful of documented offline security patterns as defense-in-depth (not duplicating the dedicated gitleaks gate); capabilities are descriptive metadata, honestly not an enforced sandbox.

## 6. Validation

**Strategy:** validate units, negative/security cases, contract fixtures and an end-to-end path.

### Planned deterministic checks
- Test: `pose check --strict && go test ./pose-mcp/... -run 'Skill|Scaffold'`.
- Structure: `pose check --strict`.
- Readiness: `pose lint-spec pose-agent-skills-conformance --ready-check`.

### Requirement trace
- R1 [satisfied] CI (governance job) runs pose skills-check --strict validating metadata, layout and linked resources; check:test (TestSkillsCheckMissingRequiredMetadata, TestSkillsCheckBrokenLink, TestSkillsCheckLinkEscapeRejected)
- R2 [satisfied] every skill declares pose_schema_range, clients and capabilities, validated and cross-checked; check:test (TestSkillsCheckInvalidSchemaRange, TestSkillsCheckClaudeClientWithoutSymlinkRejected) report:2026-07-19-standard-validate-native.md
- R3 [satisfied] discovery + bounded-workflow fixture over this repo's real 9 skills; check:test (TestSkillsCheckDiscoveryAndBoundedWorkflowFixture)

### Execution status
Executed on 2026-07-19 with a development build (`pose 0.9.0-dev`, rebuilt from this change):

- `go -C pose-mcp test ./internal/cli -run 'SkillsCheck' -count=1` — SUCCESS (11 tests, including the real-repository dogfood fixture).
- `go -C pose-mcp test ./... -count=1` — SUCCESS (full suite, golden catalog regenerated for `pose_skills_check`).
- `pose skills-check` on the live repository — found and required fixing one real pre-existing broken link (`pose-feature/SKILL.md`), then SUCCESS with `skills.checked=9 skills.errors=0`.
- `pose check --strict` — SUCCESS; `pose lint-spec pose-agent-skills-conformance --strict` — SUCCESS.
- `pose validate --strict --module pose-mcp --report` — SUCCESS (report retained under `.pose/reports/`).

## 7. Final Report

### Delivered scope

Compatibility frontmatter (`pose_schema_range`, `clients`, `capabilities`)
on all 9 skills in both locales; `pose skills-check` CLI gate wired into CI;
`pose_skills_check` MCP tool (29th in the catalog); one real broken link
found and fixed; offline unsafe-instruction and secret-shaped-content scan;
`claude-code` client cross-validated against the real symlink registry;
discovery/workflow dogfood fixture; operating-manual documentation; ADR.

### Residual risks

- The security scan is a small, documented pattern set — defense in depth,
  not a substitute for the dedicated CI secret-scanning gate.
- Capabilities are descriptive metadata, not an enforced permission system
  — no skill execution sandbox exists to enforce against.

### Follow-ups

- [open] Extend skills-check to also scan the locales/*/.agents/skills mirror trees, not just the installed .agents/skills/. (owner:@pose-maintainers crit:low review:2026-11-20)
