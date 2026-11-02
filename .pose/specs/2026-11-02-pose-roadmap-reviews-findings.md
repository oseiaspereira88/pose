---
slug: pose-roadmap-reviews-findings
status: in-progress
created_at: 2026-11-02
supersedes:
depends_on: pose-milestone-review-proportional
remediates:
priority: 1
components: pose-mcp
task_type: bugfix
delivers: capability:roadmap-reviews-findings
completed_at:
---

# Spec: What the roadmap reviews found

## 1. Intent

### Goal

The defects the reviews of `epistemic-lifecycle`, `agency-readiness` and `pose-state-consistency-and-recovery` left open are closed, and the suite means the same in a reviewer's sandbox as it does here.

### Business value

A premise stops being watched the moment its spec is done, which is when a decision starts to rely on it. A reviewer's sandbox creates an empty `.git` above the temporary directory, which failed 333 tests in the full suite of the `pose-v7-onboarding-and-consolidation` review and the registered checks of `agency-readiness`, so an independent review could not see the suite green. A close reports refused over a spec it already moved to done, a refresh that published records failure, and an immutable publication falls back to a visible write after any link failure.

### Constraints

- A premise signal on a done spec stays advisory.
- The environment gets no new way to hide a repository from a gate.

### Non-goals

- Migrating anything: the dry run of both corpora is measured, and writes nothing.

## 2. Requirements

### Functional

- R1: A spec that is done shall keep raising `premise-stale` when a pinned contract changes, and a withdrawn premise or decision shall ask for nothing.
- R2: An empty `.git` directory shall not count as a repository, a `.git` file, a non-empty directory, a bare repository and a repository reached through a link shall stay one however broken, and the ignore walk shall tell a repository by file identity and enter each one once.
- R3: A close whose transition took effect with durability unconfirmed shall report the close and say so on every path, `close --apply` included.
- R4: A state refresh that published with durability unconfirmed shall finish its later effects, log `unsynced`, and succeed saying so, except under `strict_refresh`, which asks for durability, fails, and fails a replay until a publication is durable. A later effect that fails shall not hide the publication or its durability, and where the refresh log itself failed, `refresh_pending` shall say what took effect.
- R5: An immutable publication shall fall back to an exclusive write only where a link is unavailable, and the fallback shall report its durability like any publication, a complete record included.
- R6: The cleanup plan shall hold a dry run of both corpora, with the content of each unchanged.

### Non-functional

- R1 reads a few more specs on `pose state --attention`.

### Security

- An empty directory holds no history for a gate to read, and git itself walks past it, so the engine and git now agree.

### Compatibility

- More `premise-stale` items can appear, advisory.

## 3. Technical Plan

### Affected areas

`internal/pose/premise_validity.go`, `internal/pose/git_snapshot_integrity.go`, `internal/pose/release_lifecycle.go`, `internal/pose/discovery.go`, `internal/pose/operation_result.go`, `internal/pose/writer_publication.go`, `internal/cli/review_closeout.go`, `internal/cli/closeout_plan.go`, `internal/cli/state_refresh.go`, `internal/cli/state.go`, `internal/cli/state_hooks.go`.

### Artifacts

- created: .pose/specs/2026-11-02-pose-roadmap-reviews-findings.md
- created: .pose/starts/pose-roadmap-reviews-findings.json
- created: .pose/changelogs/unreleased/pose-roadmap-reviews-findings.md
- created: .pose/reports/2026-11-02-pose-v7-legacy-cleanup-both-corpora.md
- modified: .pose/indexes/validation-matrix.json
- modified: pose-mcp/internal/pose/premise_validity.go
- modified: pose-mcp/internal/pose/git_snapshot_integrity.go
- modified: pose-mcp/internal/pose/release_lifecycle.go
- modified: pose-mcp/internal/pose/discovery.go
- modified: pose-mcp/internal/pose/operation_result.go
- modified: pose-mcp/internal/pose/writer_publication.go
- modified: pose-mcp/internal/cli/review_closeout.go
- modified: pose-mcp/internal/cli/state_hooks.go
- modified: pose-mcp/internal/cli/closeout_plan.go
- modified: pose-mcp/internal/cli/closeout_terminal_verification_test.go
- modified: pose-mcp/internal/cli/state_refresh.go
- modified: pose-mcp/internal/cli/state.go
- modified: pose-mcp/internal/pose/writer_publication_test.go
- created: pose-mcp/internal/pose/roadmap_reviews_findings_test.go
- created: pose-mcp/internal/cli/roadmap_reviews_findings_test.go

### Delivery targets

- capability:roadmap-reviews-findings module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go

### API/contract changes

`pose close` reports `transition-durability` where it failed, and `pose state refresh` prints `state.unsynced` where it failed, and logs `unsynced`. `pose state --attention` lists premises of done specs.

## 4. Tasks

- [x] Watch a premise after delivery
- [x] Stop reading an empty `.git` as a repository
- [x] Say what an unsynced publication did
- [x] Fall back only where a link is unavailable
- [x] Measure both corpora
- [x] Independent review and closeout

## 5. Decisions

### Decision D2
- Basis: spec pose-review-archive-coattributed-fragments R3
- Minimal option: accept any archived owner when a commit of the change set changed the pending path
- Selected option: keep the guard, and let the roadmap name the milestone it blocks
- Rationale: that spec requires a disjoint commit to stay blocked, and commit `7220bc64` is such a commit: its trailer named `pose-open-backlog-reconciliation` while its three fragments belong to other specs, so the record is wrong and the guard is right
- Consequences: `lifecycle-truth` of `record-truthfulness` cannot seal a bundle of its own
- Falsifier: a second commit with a correct trailer refused the same way

### Decision D1
- Basis: R2
- Minimal option: tell reviewers to ignore the failures
- Selected option: one walk, in `hasGitMetadata`, which skips an empty `.git` directory as git does, and reads the physical path and bare repositories
- Rationale: `GIT_CEILING_DIRECTORIES` would have let the environment hide a repository from the gates, which `gitRedirectingEnv` exists to prevent; the defect was the walks disagreeing with git
- Consequences: a project under an empty `.git` directory reads as having no repository
- Falsifier: a state where git finds a repository and either walk says there is none

## 6. Validation

### Strategy

Each guard is seen failing under a mutant. The full suite passes with an empty `.git` above the temporary directory. In a user namespace without a root mapping, where the reviewer runs, root reads as uid 65534 and 59 delegation tests still fail on the ownership guard of the task state directory: that guard refuses correctly, so this spec names them and does not touch it.

### Requirement trace

- R1 [satisfied] capability:roadmap-reviews-findings evidence:integration check:roadmap-reviews-findings-integration test:TestAPremiseOutlivesItsDelivery test:TestAWithdrawnPremiseAsksForNothing report:.pose/results/delivery-validation.json commit:21e17d61da9644f3b57b287fc5989eb6b43c3d96
- R2 [satisfied] check:roadmap-reviews-findings-integration test:TestAnEmptyGitDirectoryIsNoRepository test:TestEveryRepositoryGitFindsIsARepository test:TestAMarkerThatIsNoRepositoryEndsTheIgnoreWalk test:TestTheIgnoreWalkKnowsARepositoryByIdentity test:TestTheIgnoreWalkEntersARepositoryOnce report:.pose/results/delivery-validation.json commit:21e17d61da9644f3b57b287fc5989eb6b43c3d96
- R3 [satisfied] check:roadmap-reviews-findings-integration test:TestAnUnsyncedCloseSaysItClosed report:.pose/results/delivery-validation.json commit:21e17d61da9644f3b57b287fc5989eb6b43c3d96
- R4 [satisfied] check:roadmap-reviews-findings-integration test:TestAnUnsyncedRefreshDoesNotReadAsFailed test:TestAManualUnsyncedRefreshSaysSoAndSucceeds test:TestAStrictPolicyRefusesAnUnsyncedRefresh test:TestAStrictReplayDoesNotReadAsDurable test:TestAPolicyMadeStrictLaterAsksAgain test:TestACombinedRefreshFailureSaysBoth test:TestASkippedReplayIsNoDurabilityUnderAStrictPolicy report:.pose/results/delivery-validation.json commit:21e17d61da9644f3b57b287fc5989eb6b43c3d96
- R5 [satisfied] check:roadmap-reviews-findings-integration test:TestOnlyAMissingLinkFallsBack test:TestTheFallbackAnswersForItsDirectory test:TestACompleteFallbackReportsItsDurability report:.pose/results/delivery-validation.json commit:21e17d61da9644f3b57b287fc5989eb6b43c3d96
- R6 [satisfied] check:roadmap-reviews-findings-integration test:TestTheCleanupPlanHoldsBothCorpora report:.pose/results/delivery-validation.json commit:21e17d61da9644f3b57b287fc5989eb6b43c3d96

## 7. Final Report

### Delivered scope

A premise is watched after delivery, and a withdrawn one asks for nothing. An empty `.git` directory is no repository, while a bare repository and one reached through a link are, and the ignore walk tells a repository by file identity and enters each once. A publication whose durability is unconfirmed is reported for what it did by `close --apply`, by every refresh entry point and by the immutable fallback, a strict policy refuses it on replay too, and the fallback runs only where a link is unavailable. The cleanup plan holds a dry run of both corpora.

### Residual risks

The in-place fallback stays non-atomic: a write cut short leaves a partial record, which readers of immutable records refuse and a retry of complete bytes cannot replace, so it has to be removed by hand. It also lets an identical retry see bytes the first writer has not synced yet. Removing the record automatically was tried and dropped: the name may no longer be this write's by then, and it deleted another writer's record under review. Under default policy, a deduplicated replay leaves `skipped` as the last entry, which hides the `durability-unconfirmed` warning of the entry before it. `EPERM` from a link is handled as this attempt's answer, not as proof the filesystem has no links. The 59 delegation tests named in round 2 of the review fail in a user namespace without a root mapping, where root reads as uid 65534: the ownership guard refuses correctly, and the rest of the suite passes there. `lifecycle-truth` cannot seal a bundle of its own (decision D2). Windows and macOS paths were cross-built and reviewed, not run.

### Follow-ups

- [open] F1: Stop `internal/cli/maintenance.go` accepting an empty `.git`, and resolve hook paths through git so a gitfile and a linked worktree work in `maintenance.go`, `doctor.go` and `setup.go` (review R2-F1). (owner:@oseiaspereira88 crit:medium review:2026-11-27)
- [open] F2: Report installation adoption for what it did when a policy publishes unsynced, including whether a second document is owed (review R2-F2). (owner:@oseiaspereira88 crit:medium review:2026-11-27)
- [open] F3: Keep an unresolved `unsynced` visible to `pose state validate` through a default deduplicated replay (review R2-F3). (owner:@oseiaspereira88 crit:low review:2026-11-27)
- [open] F4: Replace the prose test of the cleanup report with a deterministic inventory fixture, and retain the raw inventories (review F6). (owner:@oseiaspereira88 crit:low review:2026-11-27)
- [open] F5: Fix the flake of `TestTheWaitAcknowledgementNeverTouchesAnExistingEntry` under load, a cleanup race not seen in 60 isolated runs. (owner:@oseiaspereira88 crit:low review:2026-11-27)
- [open] F6: Render the `refresh_pending` marker of a lost refresh log as what it is in `pose state validate` and `pose doctor`, which read it as a failed refresh (review R4-F1). (owner:@oseiaspereira88 crit:low review:2026-11-27)
