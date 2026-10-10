---
slug: pose-update-dry-run-reports-the-whole-update
status: in-progress
created_at: 2026-10-09
completed_at:
supersedes:
depends_on:
priority: 0
components: pose-mcp
task_type: bugfix
surface: minimal
changelog:
delivers: capability:update-dry-run-reports-the-whole-update
---

# Spec: The update dry-run reports the whole update, and a stale instance is visible

## 1. Intent

### Goal

Make `pose update --dry-run` show everything `pose update` would do, and make an instance last updated by an older engine visible in `doctor` and `check`.

### Business value

On 2026-10-09 four instances (Harne8, pose-dist, audio-relay, storageclose) were declared adopted on 7.1.0 because `pose update --no-self --dry-run` printed only `instance already at schema v1` and `DRY-RUN — no changes applied`. The real update then merged machinery into AGENTS.md, POSE.md, skills and the spec template, stamped `engine_version`, seeded `.pose/project.json` and opened a configuration review with up to eight decision requests. Every instance had stayed stamped by 6.1.0 since its earlier update; only `pose version` revealed it, and `doctor` and `check --strict` passed silently. Three adoption specs recorded a false conclusion as a result.

### Constraints

The dry-run writes nothing, as today.

## 2. Requirements

### Functional

- R1: `pose update --dry-run` shall list every file it would merge or create, the `engine_version` change, and each configuration-review request it would open, with the same wording the real run prints.
- R2: When nothing would change, the dry-run shall say so explicitly, distinct from "no schema migration".
- R3: `pose doctor` and `pose check` shall warn when the instance's `engine_version` is older than the running engine, naming `pose update`.

## 3. Technical Plan

### Artifacts

- created: .pose/specs/2026-10-09-pose-update-dry-run-reports-the-whole-update.md
- created: .pose/starts/pose-update-dry-run-reports-the-whole-update.json
- created: .pose/changelogs/unreleased/pose-update-dry-run-reports-the-whole-update.md
- created: pose-mcp/internal/cli/update_dryrun.go
- created: pose-mcp/internal/cli/update_dryrun_test.go
- modified: pose-mcp/internal/cli/maintenance.go
- modified: pose-mcp/internal/cli/check.go
- modified: pose-mcp/internal/cli/doctor.go
- modified: pose-mcp/internal/cli/compat_test.go
- modified: pose-mcp/internal/cli/upgrade_test.go
- modified: pose-mcp/internal/cli/testdata/direct-print-sites.json

### Delivery targets

- capability:update-dry-run-reports-the-whole-update module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go

## 5. Decisions

### Decision D1
- Date: 2026-10-10
- Context: a dry-run that describes the update is a second implementation of it, and the first one drifted.
- Decision: the dry-run runs the same `cmdUpdate`, without the binary self-update, on a disposable copy of `.pose`, `.agents`, `.claude`, `.codex`, `.github` and the top-level files, then reports the files it would create, modify or remove, the `engine_version` stamp, and the update's own output prefixed `[DRY-RUN]`.
- Rationale: fidelity by construction; the test compares the dry-run's list with the real update's changes on the same fixture.
- Consequences: an update step that reads source directories outside the copy (module discovery for absent seeds) sees none in the dry-run.

## 6. Validation

### Strategy

A fixture instance stamped 6.1.0: the dry-run output lists the merges, stamp and requests the real update then performs (compared line by line), and `doctor` warns before the update and not after. The test must fail on the current engine first.

### Execution log

2026-10-10: on storageclose (stamped 7.1.0), the dry-run reported the machinery merges, `would modify: .pose/state/machinery-manifest.json` and `would stamp engine_version: 7.1.0 -> 7.3.0-dev`, and left the working tree unchanged. The real update's "instance already at schema v1. Nothing to do." line, printed after merging machinery, now says the manuals and machinery are current for the engine.

2026-10-10, independent review (agent:independent-claude-opus-5-5-review, fallback reviewer): medium severity. `pose update --dry-run --force` reported "the update would fail" because the copy held an empty `.git` and `--force` runs install, which refuses a directory git does not recognise; the real `--force` update succeeded. The copy is now `git init`-ed; `TestUpdateDryRunWithForceRunsOnTheCopy` covers it (reproduced on audio-relay before the fix).

2026-10-10, independent review (agent:independent-gpt-6.1-sol-review): medium severity. The copy skipped symlinks, so with `.pose/policy` symlinked the dry-run predicted 23 files and 12 requests the real update does not create. The copy now follows symlinks as the update reads through them (cycles broken, broken links skipped); `TestUpdateDryRunFollowsASymlinkedPolicy` fails on the previous copy.

2026-10-10, second independent review (agent:independent-gpt-6.1-sol-review): medium severity. Following links turned them into plain directories, hiding the update's refusals: with `.pose/templates` symlinked the dry-run returned 0 and the real update 1. The copy now keeps links as links, pointing at copies of their targets (inside the shadow, or next to it for targets outside the instance), so the update meets the same links and nothing it writes reaches the instance; `hashTree` reads through links to compare content. `TestUpdateDryRunMeetsTheLinksTheUpdateMeets` fails on the previous copy.

### Requirement trace

- R1 [satisfied] capability:update-dry-run-reports-the-whole-update evidence:integration test:TestUpdateDryRunListsWhatTheUpdateChanges test:TestUpdateDryRunWithForceRunsOnTheCopy test:TestUpdateDryRunFollowsASymlinkedPolicy test:TestUpdateDryRunMeetsTheLinksTheUpdateMeets
- R2 [satisfied] capability:update-dry-run-reports-the-whole-update evidence:integration test:TestUpdateDryRunListsWhatTheUpdateChanges
- R3 [satisfied] capability:update-dry-run-reports-the-whole-update evidence:integration test:TestUpdateDryRunListsWhatTheUpdateChanges

## 7. Final Report

### Delivered scope

`pose update --dry-run` runs the update on a disposable copy and lists what it would change and stamp; `doctor` and `check` warn while the instance was last updated by an older engine.

### Residual risks

- Module discovery for absent seeds reads source directories the dry-run copy does not include.

### Follow-ups
