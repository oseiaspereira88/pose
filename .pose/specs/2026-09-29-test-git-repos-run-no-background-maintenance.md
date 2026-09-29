---
slug: test-git-repos-run-no-background-maintenance
status: done
created_at: 2026-09-29
completed_at: 2026-09-29
supersedes:
depends_on:
priority: 1
components: pose-mcp
task_type: bugfix
delivers: governance:test-git-isolation
---

# Spec: Test repositories start no background git maintenance

## 1. Intent

### Goal

A test that creates Git repositories cannot fail its temporary-directory cleanup
because a git process is still writing into `.git`.

### Business value

`TestReviewReleaseArchiveRejectsUnattestedContent/archive_escapes_root` failed on
`main`'s CI, and `TestSpecTrailerAttributionSurvivesUnrelatedCommits` failed
locally, both with `TempDir RemoveAll cleanup: unlinkat .../.git...: directory not
empty`. Neither failure concerned what the test asserts, and both passed on the
next run. A gate that fails at random teaches people to rerun it instead of reading
it. Now that the release waits for CI (spec `release-runs-the-ci-gates`), it can
also block a release for nothing.

### Constraints

Production behaviour is unchanged: the engine's own git invocations keep the user's
configuration. Only the test binaries of the packages that create repositories are
isolated.

### Non-goals

Rewriting test fixtures, or retrying cleanup until it succeeds. A retry would hide
the writer instead of removing it.

## 2. Requirements

- R1: The `cli`, `pose` and `mcpserver` test binaries run every git process with a
  global configuration that disables automatic maintenance.
- R2: A test proves, from git's own trace, that a commit under that configuration
  starts no `git maintenance` process, and that the same probe detects one under
  an empty global configuration.
- R3: The full suite still passes with the developer's `~/.gitconfig` out of the
  tests.

## 3. Technical Plan

After every commit, git 2.55 starts `git maintenance run --auto --quiet --detach`.
The detached process outlives the command, so a test's `t.TempDir()` cleanup can
race it while it repacks objects. A new internal package, `testgit`, writes a
config with `maintenance.auto = false` and `gc.auto = 0` and points
`GIT_CONFIG_GLOBAL` at it. Each affected package's `TestMain` calls it before
`m.Run`. `GIT_CONFIG_GLOBAL` is used rather than `GIT_CONFIG_COUNT` because one
test already sets the latter and would silently drop the setting.

### Artifacts

- created: .pose/specs/2026-09-29-test-git-repos-run-no-background-maintenance.md
- created: .pose/changelogs/unreleased/test-git-repos-run-no-background-maintenance.md
- created: pose-mcp/internal/testgit/testgit.go
- created: pose-mcp/internal/testgit/testgit_test.go
- created: pose-mcp/internal/pose/main_test.go
- modified: pose-mcp/internal/cli/cli_test.go
- modified: pose-mcp/internal/mcpserver/server_test.go
- modified: .pose/indexes/validation-matrix.json
- modified: .pose/results/delivery-validation.json
- modified: .pose/indexes/delivery-integrity.json
- modified: .pose/indexes/spec-graph.json

### Delivery targets

- governance:test-git-isolation module:pose-mcp profile:backend-go entrypoint:pose-mcp/cmd/pose/main.go

### Rollout and reversal

Test-only. Reverting restores the race.

## 4. Tasks

- [x] Identify the writer from git's trace instead of from timing.
- [x] Disable automatic maintenance for the test binaries that create repositories.
- [x] Prove the probe detects maintenance by default and none under isolation.
- [x] Run the full suite repeatedly under isolation.

## 5. Decisions

### Decision D1

- Status: active
- Remove the background writer rather than retrying the cleanup. A retry makes the
  symptom rarer and leaves a process mutating a repository a later assertion might
  read.

## 6. Validation

| Scenario | Command (from pose-mcp) | Expected evidence |
| --- | --- | --- |
| Maintenance probe | `go test ./internal/testgit -count=1` | default config starts maintenance; isolated config does not |
| Probe detects a regression | same, with `maintenance.auto = true` in the isolated config | the isolated case fails |
| Full suite under isolation | `go test ./... -count=1`, three runs | pass |

### Execution log

2026-09-29: `GIT_TRACE2_EVENT` on a plain `git commit` in a fresh repository
recorded the child `git maintenance run --auto --quiet --detach`. With the
isolated global config, the same commit recorded none. The flaky test alone did
not reproduce in 40 runs, so it only races under the full suite's load; the trace,
not the flake, is the evidence.

2026-09-29, implemented. With `maintenance.auto` flipped back to true, the probe's
isolated case fails. The full suite passed three consecutive runs under isolation.

### Closeout

2026-09-29 UTC. Full matrix 40/40 into the results path; bundle
`rvb-12471147fe662418`, 37 evidence items; attestation `rva-be7e0f9f7ef3f2c4`,
`agent:claude-opus-5-5`, approved with five explicit judgments.

### Requirement trace

- R1 [satisfied] governance:test-git-isolation evidence:integration check:test-git-isolation-integration test:TestIsolatedCommitStartsNoBackgroundMaintenance — the
  cli, pose and mcpserver TestMain hooks call testgit.Isolate before m.Run
- R2 [satisfied] governance:test-git-isolation evidence:integration check:test-git-isolation-integration test:TestIsolatedCommitStartsNoBackgroundMaintenance — git's
  trace shows maintenance started under an empty global config and none under
  isolation; flipping maintenance.auto back fails the isolated case
- R3 [satisfied] governance:test-git-isolation evidence:integration check:test-git-isolation-integration check:test — the full suite passed three consecutive runs
  under isolation, and the registered `test` check passed in both closeout runs

## 7. Final Report

### Scope delivered

The test binaries that create Git repositories start no detached maintenance, so
their temporary-directory cleanup no longer races a background writer.

### Residual risks

Shell test scripts (`tests/**/*.sh`) create repositories outside Go and are not
covered. None has shown the race, and their cleanup does not fail a gate when a
directory is left behind.

### Follow-ups

None.
