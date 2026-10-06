---
slug: pose-onboarding-spec
status: done
created_at: 2026-10-05
completed_at: 2026-10-06
supersedes:          # slug of the superseded spec (when applicable)
depends_on: 
remediates: 
priority: 1
components: pose-mcp
task_type: feature
surface: minimal
delivers: capability:onboarding-spec
---

# Spec: A new instance starts with a governed onboarding spec

## 1. Intent

### Goal

`pose install` scaffolds a first spec, `pose-onboarding`, in a new instance: adopting POSE in this project is itself the first piece of governed work, with requirements (identity declared, a maintainer who can prove answers, the commit gate, the capability defaults reviewed, the install committed), tasks that point to `pose setup`, and checks a newcomer can run. It is the first exemplar of how POSE works, in the project's own repository.

### Business value

The maintainer asked for onboarding to be guided by a first spec rather than by documentation alone: a person learns the lifecycle — draft, start, evidence, close — on work whose subject is POSE itself, and the result stays in the repository as the record of how the project adopted it. Part of roadmap pose-v7-onboarding-and-consolidation (milestone guided-onboarding).

### Constraints

Scaffolded only by an install that creates the instance, never by `pose update`, and never over an existing file. The spec must pass `pose lint-spec`, `pose check --strict` and the Definition of Ready, so `pose start` accepts it as is.

### Non-goals

Closing the spec automatically: the person drives it through the lifecycle.

## 2. Requirements

### Functional

- R1: `pose install` on a new instance shall create `.pose/specs/<date>-pose-onboarding.md` (status draft), in the install's locale, with requirements for project identity, a maintainer able to prove answers, the commit gate, the reviewed capability defaults and the committed install, tasks naming the `pose setup` steps, and deterministic checks.
- R2: The scaffolded spec shall pass lint, the strict check and the Definition of Ready on the fresh instance, so `pose start spec:pose-onboarding` reports it ready.
- R3: `pose update`, and an install over an existing instance, shall not create it, and no install shall overwrite it.
- R4: `pose setup` shall show the onboarding spec's state and, once its steps are done, name the lifecycle commands that take it to done.

### Non-functional

- None.

### Security

- None.

### Compatibility

- New instances only.

## 3. Technical Plan

### Affected areas

Install scaffolding, a localized onboarding template, setup.

### Artifacts

- created: .pose/specs/2026-10-05-pose-onboarding-spec.md
- created: .pose/starts/pose-onboarding-spec.json
- created: pose-mcp/internal/cli/onboarding.go
- created: pose-mcp/internal/cli/onboarding_test.go
- modified: pose-mcp/internal/cli/install.go
- modified: pose-mcp/internal/cli/setup.go
- modified: pose-mcp/internal/cli/setup_test.go
- modified: pose-mcp/internal/cli/stack_seed_test.go
- modified: .pose/indexes/validation-matrix.json
- modified: POSE.md
- modified: locales/pt-BR/POSE.md
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md
- renamed: .pose/changelogs/unreleased/pose-onboarding-spec.md -> .pose/changelogs/v7.0.0/pose-onboarding-spec.md

### Delivery targets

- capability:onboarding-spec module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go

### Technical risks

- A template that drifts from the spec contract would fail the install's own gate; the test lints and starts it on a fresh install.

## 6. Validation

### Strategy

Fresh installs in both locales: the spec exists, lints, passes the strict check and is ready to start; an update and a re-install leave it alone.

### Deterministic checks

#### Test
- Command: `cd pose-mcp && go test ./internal/cli -run 'Onboarding'`
- Expected: pass

### Requirement trace

- R1 [satisfied] test:TestOnboardingSpecIsScaffoldedReadyToStartInEitherLocale check:onboarding-spec-integration
- R2 [satisfied] test:TestOnboardingSpecIsScaffoldedReadyToStartInEitherLocale check:onboarding-spec-integration
- R3 [satisfied] test:TestOnboardingSpecIsNeverCreatedByUpdateNorOverwritten check:onboarding-spec-integration
- R4 [satisfied] test:TestSetupDrivesTheOnboardingSpecToClose test:TestSetupOnAFreshInstallNamesTheNextStep check:onboarding-spec-integration

### Known gaps

- The drive-to-close test needs `ssh-keygen` to register a key and skips without it.

## 7. Final Report

### Delivered scope

Localized onboarding spec scaffolded by a new install (requirements, tasks, a decision, checks), lint-clean and ready to start; `pose setup` names its start first and its close last.

### Residual risks

None beyond the technical risk.

### Follow-ups
