---
slug: pose-release-prepare-guards-claimed-fragments
status: draft
created_at: 2026-10-09
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

# Spec: Release prepare does not break specs that claim unreleased fragments

## 1. Intent

### Goal

Stop `pose release prepare --apply` from archiving a changelog fragment that a spec still claims at its `unreleased/` path, without warning.

### Business value

The v7.1.0 freeze (`2cf8d4b1`) turned CI red on main: `pose-open-backlog-reconciliation` claimed three fragments under `.pose/changelogs/unreleased/` that it had written for other specs, and the freeze moved them to `.pose/changelogs/v7.1.0/`. `release check --strict` and `check --strict` passed locally before the commit; the error appeared only in CI's structural gate.

## 2. Requirements

### Functional

- R1: `pose release plan` and `pose release prepare` shall list every spec whose Artifacts claim a fragment path the freeze will move, other than the fragment's own spec.
- R2: `pose release prepare --apply` shall refuse while such a claim exists, naming the spec and the path, unless `--allow-moved-claims` is given.
- R3: `pose release check --strict` on the prepared tree shall fail on the same condition the CI structural gate failed on.

## 3. Technical Plan

### Artifacts

- created: .pose/specs/2026-10-09-pose-release-prepare-guards-claimed-fragments.md

Implementation artifacts are declared when the spec starts.

## 5. Decisions

No decision recorded yet; the spec is a draft.

## 6. Validation

### Strategy

Fixture with one spec claiming another spec's unreleased fragment: plan lists it, prepare refuses, and the test fails on the current engine.

### Requirement trace

## 7. Final Report

### Delivered scope

Not started: opened on 2026-10-09 from the POSE 7.1.0 adoption in Harne8, pose-dist, audio-relay and storageclose.

### Residual risks

None yet.

### Follow-ups
