---
slug: pose-effects-reported-for-what-they-did
status: in-progress
created_at: 2026-11-02
supersedes:
depends_on: pose-roadmap-reviews-findings
remediates:
priority: 1
components: pose-mcp
task_type: bugfix
delivers: capability:effects-reported-for-what-they-did
completed_at:
---

# Spec: Effects reported for what they did

## 1. Intent

### Goal

The engine says what took effect wherever `pose-roadmap-reviews-findings` left a consumer behind, and it finds a repository and its hooks the way git does.

### Business value

`pose update` accepts an empty `.git` as a repository. `pose hooks install`, `pose doctor` and `pose setup` read `.git/hooks` directly, so in a linked worktree, or with `core.hooksPath`, they report a hook missing that git runs, or install one git never runs. A policy that published with durability unconfirmed is reported as not adopted, and its second document is not written. A default replay hides an unresolved durability warning, and a lost refresh log reads as a failed refresh.

### Constraints

- Containment and materialization checks of `.git` are not touched: they promise something stricter than a repository existing.
- A failed publication stays a failure.

### Non-goals

- Changing what a strict refresh refuses.

## 2. Requirements

### Functional

- R1: `pose update` shall refuse a root where git finds no repository, an empty `.git` included.
- R2: `pose hooks`, `pose doctor` and `pose setup` shall use the hooks directory git uses, in a linked worktree and under `core.hooksPath` too.
- R3: `PolicyDocs.Write` shall publish its second document after an unsynced first one and report the unsynced outcome.
- R4: Every caller of `PolicyDocs.Write` shall report a policy published with durability unconfirmed as published, with a warning.
- R5: `pose state validate` shall keep warning about an unresolved `unsynced` refresh through a deduplicated replay, until a publication confirms it.
- R6: `pose state validate` and `pose doctor` shall read the marker of a lost refresh log as a publication with an effect left to finish, not as a failed refresh.

### Non-functional

- R2 adds one git process to each command that reads the hooks directory.

### Security

- `git rev-parse --git-path hooks` reads through the raw reader, which strips the redirecting environment. A `core.hooksPath` of the repository is what git itself would run.

### Compatibility

- Where git cannot answer, the hooks directory stays `.git/hooks`, as before.

## 3. Technical Plan

### Affected areas

`internal/pose/capability_catalog.go`, `internal/pose/spec_trailers.go`, `internal/cli/maintenance.go`, `internal/cli/doctor.go`, `internal/cli/setup.go`, `internal/cli/adopt.go`, `internal/cli/configuration_review.go`, `internal/cli/issuer.go`, `internal/cli/init_delegation.go`, `internal/cli/state.go`.

### Artifacts

- created: .pose/specs/2026-11-02-pose-effects-reported-for-what-they-did.md
- created: .pose/starts/pose-effects-reported-for-what-they-did.json
- created: .pose/changelogs/unreleased/pose-effects-reported-for-what-they-did.md
- modified: .pose/indexes/validation-matrix.json
- modified: .pose/specs/2026-11-02-pose-roadmap-reviews-findings.md
- modified: pose-mcp/internal/pose/capability_catalog.go
- modified: pose-mcp/internal/pose/spec_trailers.go
- modified: pose-mcp/internal/cli/maintenance.go
- modified: pose-mcp/internal/cli/doctor.go
- modified: pose-mcp/internal/cli/setup.go
- modified: pose-mcp/internal/cli/adopt.go
- modified: pose-mcp/internal/cli/configuration_review.go
- modified: pose-mcp/internal/cli/issuer.go
- modified: pose-mcp/internal/cli/init_delegation.go
- modified: pose-mcp/internal/cli/state.go
- created: pose-mcp/internal/cli/effects_reported_test.go
- created: pose-mcp/internal/pose/effects_reported_test.go

### Delivery targets

- capability:effects-reported-for-what-they-did module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go

### API/contract changes

`pose state validate` gains a `refresh-effect-unfinished` finding in place of `refresh-pending` for that marker. `pose hooks install` writes where git reads.

## 4. Tasks

- [ ] Find a repository and its hooks as git does
- [ ] Publish the second policy document after an unsynced first
- [ ] Report a published policy as published
- [ ] Keep an unresolved durability warning through a replay
- [ ] Read a lost log for what it is
- [ ] Independent review and closeout

## 5. Decisions

### Decision D1
- Basis: R2
- Minimal option: follow a gitfile by hand
- Selected option: ask git, with `rev-parse --git-path hooks`
- Rationale: a gitfile, a common directory and `core.hooksPath` are three cases of one question git already answers
- Consequences: one git process per reader
- Falsifier: a repository where git runs hooks from a directory this does not name

## 6. Validation

### Strategy

Tests seen failing first, then a mutant per guard.

### Requirement trace

- R1 [planned] capability:effects-reported-for-what-they-did evidence:integration check:effects-reported-integration test:TestUpdateRefusesAnEmptyGitDirectory
- R2 [planned] check:effects-reported-integration test:TestHooksLiveWhereGitReadsThem test:TestHooksInstallWhereGitRunsThem
- R3 [planned] check:effects-reported-integration test:TestAnUnsyncedPolicyStillWritesItsSecondDocument
- R4 [planned] check:effects-reported-integration test:TestAnUnsyncedInstallAdoptionSaysItAdopted test:TestAnUnsyncedPolicyIsPublishedForEveryCaller
- R5 [planned] check:effects-reported-integration test:TestAReplayKeepsAnUnresolvedDurabilityWarning
- R6 [planned] check:effects-reported-integration test:TestALostLogReadsAsAnUnfinishedEffect

## 7. Final Report

### Delivered scope

### Residual risks

### Follow-ups
