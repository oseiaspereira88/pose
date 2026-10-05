---
slug: pose-init-is-install
status: in-progress
created_at: 2026-10-05
completed_at:        # stamped on the transition to status: done
supersedes:          # slug of the superseded spec (when applicable)
depends_on: 
remediates: 
priority: 1
components: pose-mcp
task_type: bugfix
surface: minimal
delivers: capability:init-is-install
---

# Spec: `pose init` produces a working instance

## 1. Intent

### Goal

Make `pose init`, run where no POSE instance exists, perform the full installation into the current repository, so both entry commands produce the same working instance; on an existing instance it only ensures the structure and names the next step.

### Business value

Walked for the 7.0.0 review: `pose init` in an empty repository creates fifteen empty directories and says "Run: pose check"; that check fails with seven errors and `pose new-spec` fails with "template not found". The README's first delivery starts with `pose init --wizard --yes`, so a person following it from a bare repository gets a broken instance. Part of roadmap pose-v7-onboarding-and-consolidation (milestone entry-and-update).

### Constraints

An existing instance is never overwritten by `init`; refreshing it stays `pose update`. The installer's own flags keep their meaning.

### Non-goals

Changing what `pose install` installs.

## 2. Requirements

### Functional

- R1: `pose init` in a repository without a POSE instance shall run the full installation into it, after which `pose check --strict` passes and `pose new-spec` works.
- R2: `pose init` on an existing instance shall ensure the directory structure, write nothing else, and say the instance is already installed, naming `pose update` to refresh it.
- R3: `pose init --wizard [--yes]` shall install first when no instance exists, then seed the detected modules as before.
- R4: `pose init` shall accept the installer's `--locale`, `--project-name`, `--project-id`, `--skip-mcp` and `--allow-non-git` flags.
- R5: The command help, CLI reference and manual shall describe `init` as installing.

### Non-functional

- None beyond the installer's.

### Security

- None beyond the installer's.

### Compatibility

- On an existing instance the behaviour is unchanged.

## 3. Technical Plan

### Affected areas

CLI dispatch for `init`, the wizard, help, docs.

### Artifacts

- created: .pose/specs/2026-10-05-pose-init-is-install.md
- created: .pose/starts/pose-init-is-install.json
- modified: pose-mcp/internal/cli/cli.go
- modified: pose-mcp/internal/cli/init.go
- modified: pose-mcp/internal/cli/help_catalog.go
- created: pose-mcp/internal/cli/init_install_test.go
- modified: pose-mcp/internal/cli/cli_test.go
- modified: pose-mcp/internal/cli/testdata/direct-print-sites.json
- modified: docs-site/docs/cli.md
- modified: POSE.md
- modified: locales/pt-BR/POSE.md
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md
- modified: .pose/indexes/validation-matrix.json
- created: .pose/changelogs/unreleased/pose-init-is-install.md

### Delivery targets

- capability:init-is-install module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go

### Technical risks

- A script that relied on `init` creating bare directories in a repository without POSE now gets a full instance; that instance is what every later command needs.

## 6. Validation

### Strategy

Fresh git repositories: `init`, `init --wizard --yes`, flags passed through; a second `init` on the installed instance.

### Deterministic checks

#### Test
- Command: `cd pose-mcp && go test ./internal/cli -run 'InitIsInstall|InitNative'`
- Expected: pass

### Requirement trace

- R1 [satisfied] test:TestInitIsInstallInABareRepository check:init-is-install-integration
- R2 [satisfied] test:TestInitOnAnInstalledInstanceWritesNothingElse test:TestInitNativeCreatesStructure check:init-is-install-integration
- R3 [satisfied] test:TestInitIsInstallWithTheWizard check:init-is-install-integration
- R4 [satisfied] test:TestInitIsInstallInABareRepository check:init-is-install-integration
- R5 [satisfied] test:TestInitIsInstallIsDocumented check:init-is-install-integration

### Known gaps

None.

## 7. Final Report

### Delivered scope

Not started.

### Residual risks

None beyond the technical risk.

### Follow-ups

None.
