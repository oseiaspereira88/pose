---
slug: pose-setup-command
status: in-progress
created_at: 2026-10-05
completed_at:        # stamped on the transition to status: done
supersedes:          # slug of the superseded spec (when applicable)
depends_on: 
remediates: 
priority: 1
components: pose-mcp
task_type: feature
surface: minimal
delivers: capability:guided-setup
---

# Spec: One command shows what is in force, what is new and what is missing

## 1. Intent

### Goal

Add `pose setup`: the single place a person (or an agent on their behalf) sees the instance's configuration — identity, commit gate, capabilities in force, capabilities new since the last review, what needs setup — and the one next step, and, at a terminal, performs each step only after the person confirms it. `pose install` and `pose update` end by naming it.

### Business value

The 7.0.0 walk found configuration scattered across `doctor`, `adopt --list`, `state --governance` and hand-edited policy files, an install that ends with no next step, and an update that says "Nothing to do" while bringing capabilities. A newcomer cannot tell what to do next, and an existing instance never hears about what is new: features stay in limbo. Part of roadmap pose-v7-onboarding-and-consolidation (milestone entry-and-update).

### Constraints

Nothing is changed without a confirmation: without a terminal (or with `--json`) setup only reports, with the exact commands. It reads only what other commands already read; it never adopts a capability on its own.

### Non-goals

Replacing `doctor` (health) or `adopt` (the toggles): setup composes them.

## 2. Requirements

### Functional

- R1: `pose setup` shall report, in this order: project identity, principals and keys (with the git identity's suggested principal and whether it is registered), the pre-commit gate, capabilities in force, capabilities new since the last configuration review, capabilities that need setup, and one next step with its command; `--json` shall return the same plan.
- R2: A capability shall count as new when it is off, undecided and introduced after the version recorded as `reviewed_version` in `.pose/policy/adoption-decisions.json` (every off, undecided capability when none is recorded); a fresh install shall record its engine version, and deciding the last new capability shall advance it.
- R3: At a terminal, `pose setup` shall offer each open step — installing the hook, registering a key with a role, deciding each new capability (adopt, decline with a reason, defer with a reason, or skip) — and perform it only after an explicit yes; otherwise it shall change nothing.
- R4: `pose install` shall end with the next step `pose setup`, and `pose update` shall end by naming how many capabilities are new and `pose setup`, instead of "Nothing to do" when there are any.
- R5: `pose doctor` shall report pending capability decisions as a `next` step naming `pose setup`.

### Non-functional

- Setup without a terminal never blocks on input.

### Security

- Setup never reads a private key; registering a key reads only a `.pub` file.

### Compatibility

- `adoption-decisions.json` gains an optional `reviewed_version`.

## 3. Technical Plan

### Affected areas

A new `pose setup` command, the adoption decision record, install/update summaries, doctor.

### Artifacts

- created: .pose/specs/2026-10-05-pose-setup-command.md
- created: .pose/starts/pose-setup-command.json
- created: pose-mcp/internal/cli/setup.go
- created: pose-mcp/internal/cli/setup_test.go
- modified: pose-mcp/internal/pose/capability_catalog.go
- modified: pose-mcp/internal/cli/adopt.go
- modified: pose-mcp/internal/cli/install.go
- modified: pose-mcp/internal/cli/maintenance.go
- modified: pose-mcp/internal/cli/doctor.go
- modified: pose-mcp/internal/cli/cli.go
- modified: pose-mcp/internal/cli/help_catalog.go
- modified: .pose/indexes/validation-matrix.json
- modified: POSE.md
- modified: locales/pt-BR/POSE.md
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md
- modified: docs-site/docs/cli.md
- created: .pose/changelogs/unreleased/pose-setup-command.md

### Delivery targets

- capability:guided-setup module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go

### Technical risks

- A prompt flow is hard to test; the interactive path reads from an injected reader and is driven by tests with scripted answers.

## 6. Validation

### Strategy

A fresh install and an older instance read through `pose setup --json`; the interactive path driven with scripted answers; install, update and doctor outputs checked for the next step.

### Deterministic checks

#### Test
- Command: `cd pose-mcp && go test ./internal/cli -run 'Setup'`
- Expected: pass

### Requirement trace

### Known gaps

## 7. Final Report

### Delivered scope

Not started.

### Residual risks

None beyond the technical risk.

### Follow-ups
