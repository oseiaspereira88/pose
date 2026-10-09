---
slug: pose-lint-validates-remediation-references
status: draft
created_at: 2026-10-09
completed_at:
supersedes:
depends_on:
priority: 2
components: pose-mcp
task_type: bugfix
surface: minimal
changelog:
delivers:
---

# Spec: lint-spec validates remediation references

## 1. Intent

### Goal

Report a malformed `remediates:` value when the spec is linted, not only when its review bundle is sealed.

### Business value

On 2026-10-09 `pose-red-signal-clears-only-at-the-head` declared `remediates: pose-red-signals-reach-a-person`. `lint-spec --strict` passed; the seal failed with `remediation-lineage/syntax: expected reference@category`, and the closeout script reported only "scope has no current sealed bundle".

## 2. Requirements

### Functional

- R1: `pose lint-spec` shall validate each `remediates:` entry with the parser the review bundle uses and fail with the expected form (`spec:<slug>@<category>`) and the accepted categories.
- R2: `pose review bundle --seal` shall keep its own check.

## 3. Technical Plan

### Artifacts

- created: .pose/specs/2026-10-09-pose-lint-validates-remediation-references.md

Implementation artifacts are declared when the spec starts.

## 5. Decisions

No decision recorded yet; the spec is a draft.

## 6. Validation

### Strategy

A spec with `remediates: <bare-slug>` fails lint-spec; `spec:<slug>@defect-fix` passes. The test fails on the current engine first.

### Requirement trace

## 7. Final Report

### Delivered scope

Not started: opened on 2026-10-09 from the POSE 7.1.0 adoption in Harne8, pose-dist, audio-relay and storageclose.

### Residual risks

None yet.

### Follow-ups
