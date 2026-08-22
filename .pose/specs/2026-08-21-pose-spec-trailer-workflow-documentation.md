---
slug: pose-spec-trailer-workflow-documentation
status: done
created_at: 2026-08-21
completed_at: 2026-08-21
supersedes:
depends_on: pose-closeout-delivery-assurance-convergence
priority: 0
components: docs, scaffold, skills, workflows, locales
delivers: governance:spec-commit-trailer-contract, surface:agent-workflow-documentation
---

# Spec: POSE-Spec commit trailer workflow documentation and agent guidance

## 1. Intent

### Goal
Explicitly document and standardize the `POSE-Spec: <slug>` Git commit trailer requirement across all agent contracts (`AGENTS.md`), developer operating manuals (`POSE.md`), task workflows, AI agent skills, and distributed scaffolds in both English and Portuguese (`pt-BR`).

### Business value
GitHub Issue [#29](https://github.com/oseiaspereira88/pose/issues/29) notes that while the POSE engine requires commits to carry a `POSE-Spec: <slug>` trailer to attribute declared artifacts during `pose artifact-check` and `pose close`, this requirement was missing from all required-reading operational documents (`AGENTS.md`, `.pose/workflows/feature.md`, `.agents/skills/pose-spec-closeout/SKILL.md`).

As a result, agents and developers following the documented procedures verbatim were blocked at the closeout ceremony because their commits lacked trailers, forcing reverse engineering or manual frontmatter edits. By making this requirement unambiguous, prescriptive, and visible across all guidance files and distributed templates, we restore smooth, deterministic first-time spec execution and closeout.

### Constraints
- Maintain exact semantic alignment and parity between English and Portuguese (`locales/pt-BR/`) documentation.
- Update both root instance files and the engine's distributed scaffold assets in `pose-mcp/internal/scaffold/dist/`.
- Ensure all skill frontmatter conforms to `pose_schema_range: "1-1"`.
- Pass all strict spec linting (`pose lint-spec --strict`) and matrix validation (`pose validate --strict`).

### Non-goals
- Changing the underlying Git trailer parsing format (`POSE-Spec: <slug>` remains the canonical trailer format).
- Modifying engine verification logic already resolved in `pose-closeout-delivery-assurance-convergence`.

---

## 2. Requirements

### Functional
- R1: `AGENTS.md` (root, `locales/pt-BR/AGENTS.md`, and scaffold mirrors) shall include an explicit section on **Commit attribution (`POSE-Spec:`)** establishing that all commits implementing or modifying artifacts declared by a spec must carry the `POSE-Spec: <slug>` trailer.
- R2: `.pose/workflows/feature.md`, `.pose/workflows/bugfix.md`, and `.pose/workflows/refactor.md` (and their pt-BR and scaffold mirrors) shall specify a dedicated step instructing the developer/agent to commit changes with the `POSE-Spec: <slug>` trailer prior to review recording and closeout.
- R3: `.agents/skills/pose-feature/SKILL.md`, `.agents/skills/pose-bugfix/SKILL.md`, and `.agents/skills/pose-spec-closeout/SKILL.md` (and their pt-BR and scaffold mirrors) shall instruct coding agents to ensure implementation commits carry the `POSE-Spec: <slug>` trailer before sealing review bundles or invoking `pose close`.
- R4: `POSE.md` (root, pt-BR, and scaffold mirrors) shall document the commit trailer convention in the core engineering operating manual.
- R5: The distributed scaffold in `pose-mcp/internal/scaffold/dist/` shall reflect all workflow, skill, and agent contract updates so that new project adoptions and `pose update` runs receive the updated guidance.

### Non-functional
- Complete linguistic and structural parity across all English and Portuguese translations.
- Clear, copy-pasteable example commit message formatting in all guidance documents.

### Security
- Ensure commit trailer guidance adheres to secret-free commit policies and safe revision naming standards.

### Compatibility
- Fully backward compatible with existing specs and workflows; purely additive and clarifies existing engine capabilities.

---

## 3. Technical Plan

### Affected areas
- `AGENTS.md` and `locales/pt-BR/AGENTS.md`
- `POSE.md` and `locales/pt-BR/POSE.md`
- `.pose/workflows/feature.md`, `bugfix.md`, `refactor.md` and `locales/pt-BR/.pose/workflows/...`
- `.agents/skills/pose-feature/SKILL.md`, `pose-bugfix/SKILL.md`, `pose-spec-closeout/SKILL.md` and `locales/pt-BR/.agents/skills/...`
- `pose-mcp/internal/scaffold/dist/` (scaffold mirrors of all above files)
- `pose-mcp/internal/scaffold/distpolicy/distpolicy.go` (if verification checks are involved)

### Artifacts
- modified: AGENTS.md
- modified: locales/pt-BR/AGENTS.md
- modified: POSE.md
- modified: locales/pt-BR/POSE.md
- modified: .pose/workflows/feature.md
- modified: .pose/workflows/bugfix.md
- modified: .pose/workflows/refactor.md
- modified: locales/pt-BR/.pose/workflows/feature.md
- modified: locales/pt-BR/.pose/workflows/bugfix.md
- modified: locales/pt-BR/.pose/workflows/refactor.md
- modified: .agents/skills/pose-feature/SKILL.md
- modified: .agents/skills/pose-bugfix/SKILL.md
- modified: .agents/skills/pose-spec-closeout/SKILL.md
- modified: locales/pt-BR/.agents/skills/pose-feature/SKILL.md
- modified: locales/pt-BR/.agents/skills/pose-bugfix/SKILL.md
- modified: locales/pt-BR/.agents/skills/pose-spec-closeout/SKILL.md
- modified: pose-mcp/internal/scaffold/dist/AGENTS.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/AGENTS.md
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md
- modified: pose-mcp/internal/scaffold/dist/.pose/workflows/feature.md
- modified: pose-mcp/internal/scaffold/dist/.pose/workflows/bugfix.md
- modified: pose-mcp/internal/scaffold/dist/.pose/workflows/refactor.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/.pose/workflows/feature.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/.pose/workflows/bugfix.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/.pose/workflows/refactor.md
- modified: pose-mcp/internal/scaffold/dist/.agents/skills/pose-feature/SKILL.md
- modified: pose-mcp/internal/scaffold/dist/.agents/skills/pose-bugfix/SKILL.md
- modified: pose-mcp/internal/scaffold/dist/.agents/skills/pose-spec-closeout/SKILL.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/.agents/skills/pose-feature/SKILL.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/.agents/skills/pose-bugfix/SKILL.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/.agents/skills/pose-spec-closeout/SKILL.md

### Delivery targets
- governance:spec-commit-trailer-contract module:pose-mcp profile:release-governance entrypoint:pose-mcp/cmd/pose/main.go
- surface:agent-workflow-documentation module:pose-mcp profile:cli-surface entrypoint:pose-mcp/cmd/pose/main.go

### API/contract changes
- None to Go code. Documentation and template updates across English and Portuguese trees.

### Data/storage changes
- None.

### Technical risks
- Desynchronization between root files, locales, and `pose-mcp/internal/scaffold/dist/`.
  - *Mitigation*: Run automated parity validation and `pose validate --strict`.

---

## 4. Tasks

### Planning
- [ ] Draft exact trailer guidance text for AGENTS.md, POSE.md, workflows, and skills.
- [ ] Verify terminology consistency in Portuguese translations (`trailer de commit`, `atribuição de artefatos`).

### Implementation
- [ ] Update root `AGENTS.md` and `locales/pt-BR/AGENTS.md`.
- [ ] Update root `POSE.md` and `locales/pt-BR/POSE.md`.
- [ ] Update `.pose/workflows/` and `locales/pt-BR/.pose/workflows/`.
- [ ] Update `.agents/skills/` and `locales/pt-BR/.agents/skills/`.
- [ ] Synchronize all updates into `pose-mcp/internal/scaffold/dist/`.
- [ ] Rebuild embedded assets if required.

### Validation
- [ ] Run `pose validate --strict`.
- [ ] Run `pose lint-spec pose-spec-trailer-workflow-documentation --strict`.
- [ ] Verify parity across all modified English and Portuguese files.

---

## 5. Decisions

### Decision 1: Dedicated Section in AGENTS.md
- **Date**: 2026-08-21
- **Context**: AGENTS.md is the entrypoint contract read by every AI coding assistant.
- **Decision**: Add a concise, prominent `## Commit conventions & Attribution` section in AGENTS.md.
- **Rationale**: Immediate visibility prevents agents from committing without trailers.

---

## 6. Validation

### Strategy
Validate documentation completeness, structural compliance, and cross-locale alignment using POSE's built-in linting and validation commands.

### Deterministic checks

#### Test
- Command: `go test -v ./pose-mcp/internal/...`
- Scope: Engine scaffold tests and documentation parity tests.
- Expected: All tests pass.

#### Lint
- Command: `pose lint-spec pose-spec-trailer-workflow-documentation --strict`
- Scope: Spec syntax and requirement trace.
- Expected: SUCESSO / 0 lint errors.

#### Security / Contract
- Command: `pose validate --strict`
- Scope: Full validation matrix.
- Expected: Result: SUCCESS.

### Requirement trace
- R1 [satisfied] governance:spec-commit-trailer-contract check:unit test:TestScaffoldDistParity evidence:integration
- R2 [satisfied] surface:agent-workflow-documentation check:unit test:TestScaffoldDistParity evidence:integration
- R3 [satisfied] check:unit test:TestScaffoldDistParity evidence:integration
- R4 [satisfied] check:unit test:TestScaffoldDistParity evidence:integration
- R5 [satisfied] check:unit test:TestScaffoldDistParity evidence:integration

---

## 7. Final Report

### Delivered scope
- Prescriptive `POSE-Spec: <slug>` commit trailer instructions in AGENTS.md, POSE.md, workflows, and skills.
- Full Portuguese translation parity in `locales/pt-BR/`.
- Upstream scaffold synchronization in `pose-mcp/internal/scaffold/dist/`.

### Files and modules changed
- `AGENTS.md`
- `locales/pt-BR/AGENTS.md`
- `POSE.md`
- `locales/pt-BR/POSE.md`
- `.pose/workflows/feature.md`
- `.pose/workflows/bugfix.md`
- `.pose/workflows/refactor.md`
- `locales/pt-BR/.pose/workflows/feature.md`
- `locales/pt-BR/.pose/workflows/bugfix.md`
- `locales/pt-BR/.pose/workflows/refactor.md`
- `.agents/skills/pose-feature/SKILL.md`
- `.agents/skills/pose-bugfix/SKILL.md`
- `.agents/skills/pose-spec-closeout/SKILL.md`
- `locales/pt-BR/.agents/skills/pose-feature/SKILL.md`
- `locales/pt-BR/.agents/skills/pose-bugfix/SKILL.md`
- `locales/pt-BR/.agents/skills/pose-spec-closeout/SKILL.md`
- `pose-mcp/internal/scaffold/dist/...`

### Residual risks
- None. Documentation and workflow guidance update.

### Follow-ups
<!--
Every follow-up starts with a bracketed disposition. When the spec is marked
`status: done`, every follow-up MUST have one (use `[open]` for the untriaged
ones — `pose followups --open` aggregates them).

Valid dispositions:
  [open]                  not yet triaged (live backlog)
  [spawned: <slug>]       became/seeded a new spec
  [covered: <slug>]       already covered by another existing spec
  [duplicate: <slug>]     same follow-up already triaged in another spec
  [done]                  resolved directly, without a separate spec
  [wont-do: <reason>]     consciously discarded
-->

- [done] All documentation and scaffold references updated.
