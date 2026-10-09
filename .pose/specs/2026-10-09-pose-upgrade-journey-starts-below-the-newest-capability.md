---
slug: pose-upgrade-journey-starts-below-the-newest-capability
status: done
created_at: 2026-10-09
completed_at: 2026-10-09
supersedes:
depends_on: pose-upgrade-journey-starts-below-current
priority: 0
components: pose-mcp
task_type: bugfix
surface: minimal
changelog: none
delivers: governance:upgrade-journey-starts-below-the-newest-capability
---

# Spec: The upgrade journey starts below the newest capability

## 1. Intent

### Goal

Make the install-and-upgrade journey start from a published release that predates the engine's newest governed capability, so the update it exercises always has a configuration review to open.

### Business value

Found while preparing 7.1.0: with the version bumped, the journey installed 7.0.0, the newest release older than the engine, and failed with "the update did not open a configuration review". 7.0.0 and 7.1.0 offer the same capabilities, so no review is owed: the engine was right and the journey's premise was wrong. Without the fix the journey would fail on every release that adds no capability.

### Constraints

The journey still starts from a published, checksum-verified release and still asserts the configuration review, the setup decisions and the idempotent second update.

## 2. Requirements

### Functional

- R1: The journey shall install the newest published release older than both the engine under test and the version that introduced the engine's newest governed capability.
- R2: When no such release exists, the journey shall fail naming the bound it used.

## 3. Technical Plan

### Artifacts

- created: .pose/specs/2026-10-09-pose-upgrade-journey-starts-below-the-newest-capability.md
- modified: tests/journeys/install-and-upgrade.sh

### Delivery targets

- governance:upgrade-journey-starts-below-the-newest-capability module:pose-mcp profile:release-governance entrypoint:tests/journeys/install-and-upgrade.sh

## 5. Decisions

### Decision D1
- Date: 2026-10-09
- Context: the newest capability's `introduced_in` is 7.0.0, so any engine from 7.0.0 to the next capability release upgrades from 6.2.0.
- Options considered: (a) bound by the newest capability's `introduced_in`; (b) assert no review when nothing is new.
- Decision: (a).
- Rationale: (b) would stop exercising the review, setup and second-update path on every release that adds no capability.

## 6. Validation

### Strategy

At engine 7.1.0 the journey failed in validate (`install-and-upgrade-journeys-integration`, "the update did not open a configuration review"); with the fix `POSE_JOURNEYS=1 go test ./internal/cli -run TestJourneysInstallAndUpgradeFromThePublishedRelease` passes, installing 6.2.0.

### Requirement trace

- R1 [satisfied] governance:upgrade-journey-starts-below-the-newest-capability check:install-and-upgrade-journeys-integration evidence:integration test:TestJourneysInstallAndUpgradeFromThePublishedRelease
- R2 [satisfied] governance:upgrade-journey-starts-below-the-newest-capability evidence:manual <the script fails with "no published release is older than <bound>" when the list is empty>

## 7. Final Report

### Delivered scope

The journey reads the newest `introduced_in` from `pose adopt --list --json` and starts below it.

### Residual risks

- Reviewed in the same session that implemented it, with no separate reviewer execution: declared, not independent.

### Follow-ups
