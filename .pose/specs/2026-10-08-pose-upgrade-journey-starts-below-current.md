---
slug: pose-upgrade-journey-starts-below-current
status: done
created_at: 2026-10-08
completed_at: 2026-10-08
supersedes:
depends_on: pose-install-and-upgrade-journeys
remediates:
priority: 1
components: tests
task_type: bugfix
surface: minimal
changelog: none
delivers:
---

# Spec: The upgrade journey starts from a release older than the engine it tests

## 1. Intent

### Goal

Make journey 2 of `tests/journeys/install-and-upgrade.sh` install the newest published release that is older than the engine under test, instead of the latest published release.

### Business value

Since v7.0.0 was published (2026-10-07 23:05 UTC), the latest release equals a build of the same version, an update between them has nothing to review, and the journey failed with "the update did not open a configuration review" on every commit of main, turning CI red without any defect in the code under test. The last green run, on 25ed81e2, ran before the publication, when the latest release was v6.2.0.

### Constraints

`--previous` and `POSE_PREVIOUS_RELEASE` keep overriding the choice. Drafts and pre-releases are never chosen.

### Non-goals

Changing what the journey asserts after the update.

## 2. Requirements

### Functional

- R1: When no previous release is given, the journey shall install the newest published, non-draft, non-prerelease release whose version is lower than the version of the engine under test.
- R2: When no published release is older than the engine under test, the journey shall fail naming that reason instead of testing an update with nothing to review.

### Non-functional

- None.

### Security

- None: the journey still verifies the downloaded archive against `checksums.txt`.

### Compatibility

- Test-only change.

## 3. Technical Plan

### Affected areas

The install-and-upgrade journey script.

### Artifacts

- created: .pose/specs/2026-10-08-pose-upgrade-journey-starts-below-current.md
- modified: tests/journeys/install-and-upgrade.sh

### Technical risks

- The GitHub releases API is unauthenticated here; the journey already depends on GitHub for the download.

## 5. Decisions

### Decision D1
- Date: 2026-10-08
- Context: the journey picked `/releases/latest`, which stops being older than the build once the build's version is published.
- Options considered: (a) pin a fixed previous version in CI; (b) choose the newest release below the engine's own version.
- Decision: (b).
- Rationale: (a) goes stale with every release; (b) keeps testing the real upgrade path a user of the previous release takes.
- Consequences: right after a release, the journey upgrades from the release before it.

## 6. Validation

### Strategy

Run the journey check with the journeys enabled before and after the change: it fails at main 7f67249 with "the update did not open a configuration review" and passes after it, upgrading from v6.2.0 to the 7.0.0 build.

### Deterministic checks

#### Test
- Command: `cd pose-mcp && POSE_JOURNEYS=1 go test ./internal/cli -run Journeys -count=1`
- Scope: tests/journeys
- Expected: pass

### Requirement trace

- R1 [satisfied] check:install-and-upgrade-journeys-integration evidence:integration test:TestJourneysInstallAndUpgradeFromThePublishedRelease
- R2 [satisfied] check:install-and-upgrade-journeys-integration evidence:integration

### Known gaps

- R2's failure path is not driven by a test: it needs a release list with nothing older, which the published history does not offer.

## 7. Final Report

### Delivered scope

Journey 2 reads the version of the engine under test and installs the newest published release below it, read from the GitHub releases API; with none older it fails with that reason.

### Residual risks

Reviewed in the same session that implemented it, with no separate reviewer execution: declared, not independent.

### Follow-ups
