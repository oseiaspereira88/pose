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

The defects the reviews of `epistemic-lifecycle`, `agency-readiness`, `record-truthfulness` and `pose-state-consistency-and-recovery` left open are closed, and the suite means the same in a reviewer's sandbox as it does here.

### Business value

A premise stops being watched the moment its spec is done, which is when a decision starts to rely on it. A reviewer's sandbox creates an empty `.git` above the temporary directory, which failed 333 test identities in every review of this cycle, so no independent review ever saw the suite green. A milestone is refused over a changelog fragment its own commit created for another spec. A close reports refused over a spec it already moved to done, and a failed exclusive write leaves a partial record that later writes refuse.

### Constraints

- A premise signal on a done spec stays advisory.
- The hermetic change touches tests and one environment variable of the raw reader.

### Non-goals

- Migrating anything: the dry run of both corpora is measured, and writes nothing.

## 2. Requirements

### Functional

- R1: A spec that is done shall keep raising `premise-stale` when a pinned contract changes.
- R2: Tests shall not see a repository above the temporary directory, and the raw reader shall honor `GIT_CEILING_DIRECTORIES`.
- R3: A review subject path shall resolve through its release archival when a commit of the change set created the fragment, whichever spec the release archived it for.
- R4: A close whose transition took effect with durability unconfirmed shall report the close and say so.
- R5: A state refresh that published with durability unconfirmed shall not read as failed.
- R6: An immutable publication shall fall back to an exclusive write only where the filesystem has no hard links, and a write cut short shall leave no record.
- R7: The cleanup plan shall hold a dry run of both corpora, with the content of each unchanged.

### Non-functional

- R1 reads a few more specs on `pose state --attention`.

### Security

- `GIT_CEILING_DIRECTORIES` only narrows discovery, and a narrowed discovery reads as no repository, which gates refuse.

### Compatibility

- More `premise-stale` items can appear, advisory.

## 3. Technical Plan

### Affected areas

`internal/pose/premise_validity.go`, `internal/testgit/testgit.go`, `internal/pose/git_snapshot_integrity.go`, `internal/pose/review_bundle.go`, `internal/pose/writer_publication.go`, `internal/cli/review_closeout.go`, `internal/cli/state_hooks.go`.

### Artifacts

- created: .pose/specs/2026-11-02-pose-roadmap-reviews-findings.md
- created: .pose/starts/pose-roadmap-reviews-findings.json
- created: .pose/changelogs/unreleased/pose-roadmap-reviews-findings.md
- created: .pose/reports/2026-11-02-pose-v7-legacy-cleanup-both-corpora.md
- modified: .pose/indexes/validation-matrix.json
- modified: pose-mcp/internal/pose/premise_validity.go
- modified: pose-mcp/internal/testgit/testgit.go
- modified: pose-mcp/internal/pose/git_snapshot_integrity.go
- modified: pose-mcp/internal/pose/review_bundle.go
- modified: pose-mcp/internal/pose/writer_publication.go
- modified: pose-mcp/internal/cli/review_closeout.go
- created: pose-mcp/internal/pose/roadmap_reviews_findings_test.go
- created: pose-mcp/internal/cli/roadmap_reviews_findings_test.go

### Delivery targets

- capability:roadmap-reviews-findings module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go

### API/contract changes

`pose close` prints `close.unsynced` where it failed. `pose state --attention` lists premises of done specs.

## 4. Tasks

- [ ] Watch a premise after delivery
- [ ] Make the tests hermetic to a repository above them
- [ ] Resolve a fragment archived for another spec
- [ ] Say what an unsynced publication did
- [ ] Keep a partial record from being left
- [ ] Measure both corpora
- [ ] Independent review and closeout

## 5. Decisions

### Decision D1
- Basis: R2
- Minimal option: tell reviewers to ignore the failures
- Selected option: a ceiling at the temporary directory in `testgit.Isolate`, which the raw reader lets through
- Rationale: the failures are identical in every sandbox run and hid any real one among them
- Consequences: `GIT_CEILING_DIRECTORIES` leaves the list of redirecting variables
- Falsifier: a ceiling that makes a gate pass where discovery would have refused

## 6. Validation

### Strategy

Tests seen failing first, and the full suite with an empty `.git` above the temporary directory, which failed before.

### Requirement trace

- R1 [planned] capability:roadmap-reviews-findings evidence:integration check:roadmap-reviews-findings-integration test:TestAPremiseOutlivesItsDelivery
- R2 [planned] check:roadmap-reviews-findings-integration test:TestAnEmptyRepositoryAboveTheTestsChangesNothing
- R3 [planned] check:roadmap-reviews-findings-integration test:TestAFragmentArchivedForAnotherSpecResolves
- R4 [planned] check:roadmap-reviews-findings-integration test:TestAnUnsyncedCloseSaysItClosed
- R5 [planned] check:roadmap-reviews-findings-integration test:TestAnUnsyncedRefreshDoesNotReadAsFailed
- R6 [planned] check:roadmap-reviews-findings-integration test:TestOnlyAMissingLinkFallsBack test:TestACutWriteLeavesNoRecord
- R7 [planned] check:roadmap-reviews-findings-integration test:TestTheCleanupPlanHoldsBothCorpora

## 7. Final Report

### Delivered scope

### Residual risks

### Follow-ups
