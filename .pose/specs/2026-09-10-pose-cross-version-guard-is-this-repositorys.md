---
slug: pose-cross-version-guard-is-this-repositorys
status: in-progress
created_at: 2026-09-10
completed_at:
supersedes:
depends_on: pose-release-boundary-rehearsal
priority: 0
components: pose-mcp
delivers:
task_type: bugfix
---

# Spec: The cross-version guard belongs to this repository's CI

## 1. Intent

### Goal
Make the cross-version compatibility test fail on this repository's CI and skip
everywhere else, instead of failing on any CI at all.

### Business value
`pose-release-boundary-rehearsal` added a test that builds the previous release
from its tag, and a guard that turns a missing precondition into a failure when
`CI` is set — because a skip on the machine whose verdict gates a release is a
test quietly not running.

The guard keyed on `CI` alone. A repository that vendors this engine as a
submodule and runs its suite is also CI, and its submodule checkout has no tags:
it cannot build the previous release, it is not responsible for this engine's
release history, and there is nothing for it to configure. The guard told it to
fix something that is not its to fix.

That broke the adopting repository's build the same day v4.0.0 reached it, on a
pull request whose only content was the adoption.

### Constraints
- The protection must survive: a shallow checkout in this repository's own CI
  must still fail rather than skip.

### Non-goals
- Making consumers fetch tags for the submodule. Pushing this engine's test
  preconditions onto everyone who vendors it is the shape of the defect, not a
  fix for it.

---

## 2. Requirements

### Functional
- R1: A missing tag shall fail on this repository's own CI.
- R2: A missing tag shall skip on any other repository's CI.
- R3: A missing tag shall skip locally, as before.

### Non-functional
- The signal is the repository the workflow runs for, not a heuristic about the
  checkout.

---

## 3. Technical Plan

### Affected areas
- `pose-mcp/internal/cli/release_compatibility_test.go` — `skipUnlessCI`

### Artifacts
- created: .pose/specs/2026-09-10-pose-cross-version-guard-is-this-repositorys.md
- renamed: .pose/changelogs/unreleased/pose-cross-version-guard-is-this-repositorys.md -> .pose/changelogs/v4.0.1/pose-cross-version-guard-is-this-repositorys.md
- modified: pose-mcp/internal/cli/release_compatibility_test.go

### Technical risks
- `GITHUB_REPOSITORY` is provider-specific. On a provider that does not set it
  the guard skips, which is the safe direction for a consumer and the unsafe one
  for this repository — mitigated by this repository's CI being the one that
  sets it.

---

## 4. Tasks

### Implementation
- [x] Increment 1: The guard compares the repository, not only the presence of CI (R1, R2, R3)

### Validation
- [x] All three branches exercised

---

## 5. Decisions

### Decision 1
- Date: 2026-09-10
- Context: three ways to stop the guard breaking consumers — have consumers
  fetch tags, detect a shallow checkout, or scope the guard to this repository.
- Decision: scope it by `GITHUB_REPOSITORY` against `releaseRepo`.
- Rationale: making consumers fetch tags spreads this engine's test preconditions
  to everyone who vendors it. Detecting shallowness would skip silently in this
  repository's own CI when a checkout regressed, which is the hole the guard was
  written to close. The repository is the thing that actually differs, and the
  constant to compare against already exists.

---

## 6. Validation

### Strategy
Run the test under all three environments and require the answer each one
deserves.

### Deterministic checks

#### Test
- Command: `go test -count=1 ./...`
- Scope: `pose-mcp`
- Expected: exit 0

#### Gate
- Command: `pose check --strict`
- Scope: this instance
- Expected: exit 0

### Execution log
- Date: 2026-09-10
- Environment: local, Go 1.26
- Notes: with `CI=1 GITHUB_REPOSITORY=oseiaspereira88/pose` and the resolved tag
  forced to one that does not exist, the test fails naming the configuration.
  With `CI=1 GITHUB_REPOSITORY=oseiaspereira88/harne8` and the same forced tag,
  it skips. With tags present it runs and passes under either. The failure this
  fixes was observed on the adopting repository's CI: `no tag below v4.0.0 is
  present`.

### Results summary
- Successes: R1, R2, R3 verified.
- Failures: none.

### Requirement trace
- R1 [satisfied] <CI=1 with GITHUB_REPOSITORY set to this repository and an unresolvable tag fails with "on this repository's CI a missing tag is a configuration failure">
- R2 [satisfied] <the same forced tag under another repository's GITHUB_REPOSITORY skips>
- R3 [satisfied] <with CI unset the guard skips, unchanged from before>

### Known gaps
- A provider that does not set `GITHUB_REPOSITORY` skips even in this
  repository's own CI. Nothing reports that, and the release gates would be one
  silent skip away again.

---

## 7. Final Report

### Summary
The guard protects the repository that owns the release history and leaves
everyone who vendors it alone.

### Follow-ups

- [done] Assert in the workflow contract that this repository's CI sets the environment the guard keys on, so a provider or workflow change cannot turn the failure back into a silent skip. Done in `pose-the-guard-signal-is-declared-not-inherited`, and better than asked: rather than asserting the provider sets something, the workflow declares `POSE_RELEASE_HISTORY_AVAILABLE` and the contract test requires every job running the suite to declare it — which found a third such job, the release workflow's own. — owner:unowned crit:medium review:2026-12-10
