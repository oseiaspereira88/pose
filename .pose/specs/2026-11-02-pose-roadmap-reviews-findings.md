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
- R2: An empty `.git` directory shall not count as a repository, a `.git` file, a non-empty directory, a bare repository and a repository reached through a link shall stay one however broken, and a submodule marker that is no repository shall end the ignore walk.
- R3: A close whose transition took effect with durability unconfirmed shall report the close and say so on every path, `close --apply` included.
- R4: A state refresh that published with durability unconfirmed shall finish its later effects, log `unsynced`, and succeed saying so, except under `strict_refresh`, which asks for durability and fails.
- R5: An immutable publication shall fall back to an exclusive write only where a link is unavailable, and the fallback shall report its durability like any publication.
- R6: The cleanup plan shall hold a dry run of both corpora, with the content of each unchanged.

### Non-functional

- R1 reads a few more specs on `pose state --attention`.

### Security

- An empty directory holds no history for a gate to read, and git itself walks past it, so the engine and git now agree.

### Compatibility

- More `premise-stale` items can appear, advisory.

## 3. Technical Plan

### Affected areas

`internal/pose/premise_validity.go`, `internal/pose/git_snapshot_integrity.go`, `internal/pose/release_lifecycle.go`, `internal/pose/discovery.go`, `internal/pose/writer_publication.go`, `internal/cli/review_closeout.go`, `internal/cli/closeout_plan.go`, `internal/cli/state_refresh.go`, `internal/cli/state.go`, `internal/cli/state_hooks.go`.

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
- [ ] Independent review and closeout

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

- R1 [planned] capability:roadmap-reviews-findings evidence:integration check:roadmap-reviews-findings-integration test:TestAPremiseOutlivesItsDelivery test:TestAWithdrawnPremiseAsksForNothing
- R2 [planned] check:roadmap-reviews-findings-integration test:TestAnEmptyGitDirectoryIsNoRepository test:TestEveryRepositoryGitFindsIsARepository test:TestAMarkerThatIsNoRepositoryEndsTheIgnoreWalk
- R3 [planned] check:roadmap-reviews-findings-integration test:TestAnUnsyncedCloseSaysItClosed
- R4 [planned] check:roadmap-reviews-findings-integration test:TestAnUnsyncedRefreshDoesNotReadAsFailed test:TestAManualUnsyncedRefreshSaysSoAndSucceeds test:TestAStrictPolicyRefusesAnUnsyncedRefresh
- R5 [planned] check:roadmap-reviews-findings-integration test:TestOnlyAMissingLinkFallsBack test:TestTheFallbackAnswersForItsDirectory
- R6 [planned] check:roadmap-reviews-findings-integration test:TestTheCleanupPlanHoldsBothCorpora

## 7. Final Report

### Delivered scope

### Residual risks

### Follow-ups
