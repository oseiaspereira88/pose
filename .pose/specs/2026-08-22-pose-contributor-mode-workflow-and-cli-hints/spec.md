---
slug: pose-contributor-mode-workflow-and-cli-hints
status: in-progress
created_at: 2026-08-22
supersedes:
depends_on: pose-contributor-mode-protocol, pose-manual-and-cli-command-parity
priority: 25
components: cli, skills, workflows, scaffold, locales
delivers: surface:contributor-mode-cli-hints
---

# Spec: Contributor Mode Workflow and CLI Contextual Guidance Integration

## 1. Intent

### Goal
Provide end-to-end integration for POSE Contributor Mode by:
1. Embedding conditional contributor reminders into key AI agent skills (`pose-bugfix`, `pose-review`, `pose-spec-closeout`, `pose-recurrence-escalation`, `pose-feature`) and core workflows (`bugfix.md`, `review.md`, `feature.md`, `refactor.md`) in English and Brazilian Portuguese (`pt-BR`).
2. Adding non-intrusive, localized CLI contextual hints when contributor mode is active (`.pose/state/contributor.json` with `active: true`) upon validation/check failures (`pose check`, `pose lint-spec`, `pose validate`, `pose artifact-check`, `pose close`), within `pose doctor` diagnostics, and integrating `pose report-limitation`.
3. Ensuring `pose update` strictly preserves contributor mode state and doc sections across upgrades without data loss.

### Business value
Empowers AI agents and developers to naturally capture and stage engine friction, missing stack rules, and false-positive diagnostics at the exact moment they occur, while keeping contributor mode completely opt-in, privacy-preserving, and non-intrusive.

### Constraints
- Hints in CLI MUST ONLY appear when Contributor Mode is explicitly ACTIVE.
- Hints MUST NEVER be emitted when `--json` is supplied (strict machine-readability invariant).
- Invariant: Zero proprietary code in staged reports (`.pose/contributions/`).
- Full bilingual parity across English and `pt-BR` for all skills, workflows, docs, and CLI hints.
- `pose update` and `MergeManagedDoc` must preserve contributor sections indefinitely.

### Non-goals
- Automatic external submission over the network (staging remains local; submission is developer-adjudicated).

---

## 2. Requirements

### Functional
- R1: When Contributor Mode is active, CLI validation/check/close failure summaries (`pose check`, `pose lint-spec`, `pose validate`, `pose artifact-check`, `pose close`) shall emit an informational hint suggesting `pose contribute stage --type bug` to report potential engine defects or false positives.
- R2: `pose doctor` shall display Contributor Mode status and an actionable staging hint in its diagnostic output when active.
- R3: `pose report-limitation` shall integrate with Contributor Mode by automatically staging or mirroring limitation reports into `.pose/contributions/` when active.
- R4: CLI hints shall strictly suppress extra text when `--json` flag is provided.
- R5: Key skills (`pose-bugfix`, `pose-review`, `pose-spec-closeout`, `pose-recurrence-escalation`, `pose-feature`) shall include explicit guidance instructing agents to stage synthetic reproductions under `.pose/contributions/` if Contributor Mode is active.
- R6: Core task workflows (`bugfix.md`, `review.md`, `feature.md`, `refactor.md`) shall document the contributor feedback step.
- R7: Full linguistic parity in `locales/pt-BR/` and upstream embedded scaffold assets in `pose-mcp/internal/scaffold/dist/`.
- R8: `pose update` shall preserve Contributor Mode state and doc sections without regression.

### Non-functional
- Zero impact on CLI execution performance.
- Deterministic, unit-tested CLI hint output and doc preservation.

### Security
- Staged contributions remain purely local with synthetic examples. No network transmission.

### Compatibility
- Existing repositories without contributor mode maintain identical behavior and zero hint noise.

---

## 3. Technical Plan

### Affected areas
- `pose-mcp/internal/cli/contribute.go`
- `pose-mcp/internal/cli/check.go`
- `pose-mcp/internal/cli/doctor.go`
- `pose-mcp/internal/cli/lintspec.go`
- `pose-mcp/internal/cli/report_limitation.go`
- `pose-mcp/internal/cli/artifact_integrity.go`
- `pose-mcp/internal/cli/review_closeout.go`
- `pose-mcp/internal/cli/contribute_hints_test.go`
- `.agents/skills/pose-bugfix/SKILL.md`
- `.agents/skills/pose-review/SKILL.md`
- `.agents/skills/pose-spec-closeout/SKILL.md`
- `.agents/skills/pose-recurrence-escalation/SKILL.md`
- `.agents/skills/pose-feature/SKILL.md`
- `locales/pt-BR/.agents/skills/...`
- `.pose/workflows/bugfix.md`
- `.pose/workflows/review.md`
- `.pose/workflows/feature.md`
- `.pose/workflows/refactor.md`
- `locales/pt-BR/.pose/workflows/...`
- `pose-mcp/internal/scaffold/dist/...`

### Delivery targets
- surface:contributor-mode-cli-hints module:pose-mcp profile:cli-surface entrypoint:pose-mcp/cmd/pose/main.go

---

## 4. Tasks

### Artifacts
- created: pose-mcp/internal/cli/contribute_hints_test.go
- modified: pose-mcp/internal/cli/contribute.go
- modified: pose-mcp/internal/cli/check.go
- modified: pose-mcp/internal/cli/doctor.go
- modified: pose-mcp/internal/cli/lintspec.go
- modified: pose-mcp/internal/cli/report_limitation.go
- modified: pose-mcp/internal/cli/artifact_integrity.go
- modified: pose-mcp/internal/cli/review_closeout.go
- modified: .agents/skills/pose-bugfix/SKILL.md
- modified: .agents/skills/pose-review/SKILL.md
- modified: .agents/skills/pose-spec-closeout/SKILL.md
- modified: .agents/skills/pose-recurrence-escalation/SKILL.md
- modified: .agents/skills/pose-feature/SKILL.md
- modified: locales/pt-BR/.agents/skills/pose-bugfix/SKILL.md
- modified: locales/pt-BR/.agents/skills/pose-review/SKILL.md
- modified: locales/pt-BR/.agents/skills/pose-spec-closeout/SKILL.md
- modified: locales/pt-BR/.agents/skills/pose-recurrence-escalation/SKILL.md
- modified: locales/pt-BR/.agents/skills/pose-feature/SKILL.md
- modified: .pose/workflows/bugfix.md
- modified: .pose/workflows/review.md
- modified: .pose/workflows/feature.md
- modified: .pose/workflows/refactor.md
- modified: locales/pt-BR/.pose/workflows/bugfix.md
- modified: locales/pt-BR/.pose/workflows/review.md
- modified: locales/pt-BR/.pose/workflows/feature.md
- modified: locales/pt-BR/.pose/workflows/refactor.md
- modified: pose-mcp/internal/scaffold/distpolicy/distpolicy.go
- modified: pose-mcp/internal/scaffold/gen/main.go
- modified: pose-mcp/internal/scaffold/manual_locale_parity_test.go
- modified: pose-mcp/internal/scaffold/scaffold_test.go
- modified: pose-mcp/internal/scaffold/dist/.agents/skills/pose-bugfix/SKILL.md
- modified: pose-mcp/internal/scaffold/dist/.agents/skills/pose-feature/SKILL.md
- modified: pose-mcp/internal/scaffold/dist/.agents/skills/pose-recurrence-escalation/SKILL.md
- modified: pose-mcp/internal/scaffold/dist/.agents/skills/pose-review/SKILL.md
- modified: pose-mcp/internal/scaffold/dist/.agents/skills/pose-spec-closeout/SKILL.md
- modified: pose-mcp/internal/scaffold/dist/.pose/workflows/bugfix.md
- modified: pose-mcp/internal/scaffold/dist/.pose/workflows/feature.md
- modified: pose-mcp/internal/scaffold/dist/.pose/workflows/refactor.md
- modified: pose-mcp/internal/scaffold/dist/.pose/workflows/review.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/.agents/skills/pose-bugfix/SKILL.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/.agents/skills/pose-feature/SKILL.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/.agents/skills/pose-recurrence-escalation/SKILL.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/.agents/skills/pose-review/SKILL.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/.agents/skills/pose-spec-closeout/SKILL.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/.pose/workflows/bugfix.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/.pose/workflows/feature.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/.pose/workflows/refactor.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/.pose/workflows/review.md

### Increments
- [x] Increment 1: CLI Failure and Diagnostic Hints Implementation (R1, R2, R3, R4)
- [x] Increment 2: AI Agent Skills & Core Workflows Integration (R5, R6, R7)
- [x] Increment 3: Automated Unit Testing and Scaffold Parity Verification (R8)

---

## 5. Validation

### Automated
- `go test ./internal/cli -run "TestContributorModeHints|TestReportLimitationStagesContributionWhenActive"`
- `go test ./internal/scaffold/...`
- `pose check --strict`
- `pose validate --strict`

---

## 6. Delivery Evidence

### Artifact claims
- capability:contributor-mode-cli-hints -> pose-mcp/cmd/pose/main.go
- surface:contributor-workflow-guidance -> POSE.md

### Requirement trace
- R1 (CLI failure hints): Covered by `PrintContributorFailureHint` in `check.go`, `lintspec.go`, `artifact_integrity.go`, `review_closeout.go`.
- R2 (Doctor status hint): Covered by `PrintContributorDoctorHint` in `doctor.go`.
- R3 (Report-limitation staging): Covered in `report_limitation.go`.
- R4 (JSON suppression): Covered by `!jsonOut` and `!jsonOutput` guard clauses in `doctor.go` and `artifact_integrity.go`.
- R5 (Key skills integration): Embedded in `pose-bugfix`, `pose-review`, `pose-spec-closeout`, `pose-recurrence-escalation`, `pose-feature`.
- R6 (Workflows guidance): Embedded in `bugfix.md`, `review.md`, `feature.md`, `refactor.md`.
- R7 (Bilingual parity): Verified in `locales/pt-BR/` and embedded dist.
- R8 (Update preservation): Tested and verified in `TestMergeManagedDocPreservesContributorMode`.

### Known gaps
None.

---

## 7. Final Report

### Delivered scope
- Integrated conditional Contributor Mode guidance across 5 key AI agent skills and 4 core task workflows in both English and Portuguese.
- Implemented non-intrusive, localized CLI hints in `pose check`, `pose lint-spec`, `pose doctor`, `pose close`, `pose artifact-check`, and `pose report-limitation`.
- Maintained strict JSON format safety and preserved contributor configuration across `pose update`.

### Follow-ups
- [done] All requirements delivered and verified.
