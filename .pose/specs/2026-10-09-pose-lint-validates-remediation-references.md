---
slug: pose-lint-validates-remediation-references
status: abandoned
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

### Decision D1
- Date: 2026-10-10
- Context: before implementing, the premise was measured: a copy of the 2026-10-09 spec with `remediates: pose-red-signals-reach-a-person` fails `pose lint-spec` with `remediation-lineage/syntax: expected reference@category`, the same error the seal gave. `lintspec.go` already validates every `remediates:` entry through `ValidateRemediationLineage`.
- Decision: abandon the spec.
- Rationale: the premise was wrong. On 2026-10-09 lint-spec was not run on that spec before sealing; the spec attributed to the engine a step the implementer skipped.

## 6. Validation

### Strategy

A spec with `remediates: <bare-slug>` fails lint-spec; `spec:<slug>@defect-fix` passes. The test fails on the current engine first.

### Requirement trace

## 7. Final Report

### Delivered scope

Nothing to deliver: `lint-spec` already validates remediation references. Abandoned after measuring the premise (D1).

### Residual risks

None yet.

### Follow-ups
