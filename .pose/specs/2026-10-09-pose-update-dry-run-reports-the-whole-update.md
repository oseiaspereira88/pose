---
slug: pose-update-dry-run-reports-the-whole-update
status: draft
created_at: 2026-10-09
completed_at:
supersedes:
depends_on:
priority: 0
components: pose-mcp
task_type: bugfix
surface: minimal
changelog:
delivers:
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

Implementation artifacts are declared when the spec starts.

## 5. Decisions

No decision recorded yet; the spec is a draft.

## 6. Validation

### Strategy

A fixture instance stamped 6.1.0: the dry-run output lists the merges, stamp and requests the real update then performs (compared line by line), and `doctor` warns before the update and not after. The test must fail on the current engine first.

### Requirement trace

## 7. Final Report

### Delivered scope

Not started: opened on 2026-10-09 from the POSE 7.1.0 adoption in Harne8, pose-dist, audio-relay and storageclose.

### Residual risks

None yet.

### Follow-ups
