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
- modified: pose-mcp/internal/cli/state_refresh.go
- modified: pose-mcp/internal/cli/compat_test.go
- modified: pose-mcp/internal/pose/delegated_review_adoption.go
- created: pose-mcp/internal/cli/effects_reported_test.go
- created: pose-mcp/internal/pose/effects_reported_test.go

### Delivery targets

- capability:effects-reported-for-what-they-did module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go

### API/contract changes

`pose state validate` gains a `refresh-effect-unfinished` finding in place of `refresh-pending` for that marker. `pose hooks install` writes where git reads. Refresh-log entries may include `durability_unconfirmed` for partial publications; readers preserve the warning across failed attempts and skipped replays. Delegation activation still checks its configuration after an unsynced publication, and a refused activation remains a failure even if its rollback is unsynced.

## 4. Tasks

- [x] Find a repository and its hooks as git does
- [x] Publish the second policy document after an unsynced first
- [x] Report a published policy as published
- [x] Keep an unresolved durability warning through a replay
- [x] Read a lost log for what it is
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

Focused integration tests and an isolated mutation corpus per changed guard. The failed-refresh warning regression was observed failing before its fix. The shared writer handoff was consulted as knowledge:roadmap-v7-independent-outcomes-20261102; its sandbox limitations are not accepted as a passing suite.

### Requirement trace

- R1 [satisfied] capability:effects-reported-for-what-they-did evidence:integration check:effects-reported-integration test:TestUpdateRefusesAnEmptyGitDirectory
- R2 [satisfied] check:effects-reported-integration test:TestHooksLiveWhereGitReadsThem test:TestHooksInstallWhereGitRunsThem
- R3 [satisfied] check:effects-reported-integration test:TestAnUnsyncedPolicyStillWritesItsSecondDocument test:TestUnsyncedDelegationActivationStillChecksItsConfiguration
- R4 [satisfied] check:effects-reported-integration test:TestAnUnsyncedInstallAdoptionSaysItAdopted test:TestAnUnsyncedPolicyIsPublishedForEveryCaller
- R5 [satisfied] check:effects-reported-integration test:TestAReplayKeepsAnUnresolvedDurabilityWarning test:TestAFailedRefreshKeepsAnUnresolvedDurabilityWarning test:TestAPartialRefreshReportsItsPublicationDurability
- R6 [satisfied] check:effects-reported-integration test:TestALostLogReadsAsAnUnfinishedEffect test:TestDoctorReadsALostLogAsAnUnfinishedEffect

## 7. Final Report

### Delivered scope

Implemented R1–R6 with focused integration tests. Hooks are resolved by git, updates refuse an empty `.git`, and policy publication continues to its second document after an unsynced first one. Callers report published policies with a warning. State validation preserves an unconfirmed publication through skipped and failed attempts and distinguishes partial publication durability. Lost-log markers report an unfinished effect. A separate Codex review found two blockers: failed refreshes hiding warnings and unsynced delegation activation bypassing its final configuration check. Both now have deterministic regression cases. Full validation and formal attestation remain required before closeout.

### Residual risks

A two-document policy publication is not a transaction: a hard failure of the second document leaves the first published and reports failure. Hook paths on Windows and macOS require platform validation beyond the local Linux tests. Validation of state is the default `pose state` command; this spec's references to `state validate` name that validation path, not a new subcommand.

### Follow-ups

No new backlog item is claimed resolved before the closeout gates pass. F4 and F5 of `pose-roadmap-reviews-findings` remain outside this spec.
