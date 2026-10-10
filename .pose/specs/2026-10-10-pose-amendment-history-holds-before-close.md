---
slug: pose-amendment-history-holds-before-close
status: draft
created_at: 2026-10-10
completed_at:
supersedes:
depends_on:
priority: 1
components: pose-mcp
task_type: bugfix
surface: minimal
changelog:
delivers:
---

# Spec: The amendment history holds before a spec closes

## 1. Intent

### Goal

A started spec's first amendment shall not leave its earlier nodes unacknowledged, and a spec whose amendment history would fail once done shall be refused before `pose close`, not after.

### Business value

On 2026-10-10 two specs (pose-dist's `pose-review-subject-classifies-engine-records` and Harne8's `harne8-adopt-pose-v7-2-0`) recorded their first `pose amend` event as `semantic` or `added` with no baseline. Their atomic start records held a baseline, but the amendment log did not, so every node present since the start read as "added without an amendment event". The amendment gate in `lint-spec` runs only on done specs, so the failure appeared only after `pose close`, and an independent reviewer had to unwind an approved close.

### Constraints

The amendment log stays append-only; nothing rewrites recorded events.

## 2. Requirements

### Functional

- R1: When `pose amend` records the first event of a spec that has a recorded atomic start, the engine shall seed the log with a baseline taken from the start record before the change event.
- R2: `pose close` shall refuse a spec whose amendment history would fail `lint-spec --strict` once done, naming the findings.
- R3: `lint-spec` on an in-progress spec shall report, as a warning, the amendment findings it would raise once done.

## 3. Technical Plan

### Artifacts

- created: .pose/specs/2026-10-10-pose-amendment-history-holds-before-close.md

Implementation artifacts are declared when the spec starts.

## 5. Decisions

No decision recorded yet; the spec is a draft.

## 6. Validation

### Strategy

Reproduce both 2026-10-10 cases in fixtures: a started spec amended once fails the done lint today and passes with R1; a close over a failing history is refused with R2.

### Requirement trace

## 7. Final Report

### Delivered scope

Not started: opened on 2026-10-10 from the independent reviews of the review-subject classifier and Harne8's 7.2.0 adoption.

### Residual risks

None yet.

### Follow-ups
