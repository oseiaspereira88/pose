---
slug: pose-closeout-names-the-missing-trailer
status: in-progress
created_at: 2026-11-01
supersedes:
depends_on:
remediates:
priority: 1
components: pose-mcp
task_type: bugfix
delivers: capability:closeout-names-the-missing-trailer
completed_at:
---

# Spec: Closeout names the missing trailer once

## 1. Intent

### Goal

When no commit is attributed to a spec, the closeout plan reports that cause and its fix, and not the evidence gap that follows from it.

### Business value

Issue #208: with the `POSE-Spec:` line missing, `pose close spec:<slug> --plan` reports two reasons, no attributed change set and no attributed validation evidence. Only the first is a cause. The second sends the operator to validate again, which cannot help, and the evidence appears as soon as the line is added.

### Constraints

- The bundle stays blocked: one blocker replaces two, and nothing that blocked now passes.
- A milestone of external members, which has no change set by design, keeps its evidence gate.

### Non-goals

- Inferring the attribution of a commit without the line.

## 2. Requirements

### Functional

- R1: When a spec has no attributed change set, its bundle shall report that blocker alone, without the evidence blocker.
- R2: The blocker of a spec shall name the `POSE-Spec:` line, `pose index` and `pose artifact-check --spec <slug>`.
- R3: The evidence blocker shall still apply to every scope without that blocker.

### Non-functional

- No new read: the decision uses the blockers already computed.

### Security

- A bundle without a change set stays unsealable, so no approval gains a path.

### Compatibility

- A consumer that matched the old text of the blocker by prefix still matches it.

## 3. Technical Plan

### Affected areas

`internal/pose/review_bundle.go`.

### Artifacts

- created: .pose/specs/2026-11-01-pose-closeout-names-the-missing-trailer.md
- created: .pose/starts/pose-closeout-names-the-missing-trailer.json
- created: .pose/changelogs/unreleased/pose-closeout-names-the-missing-trailer.md
- modified: .pose/indexes/validation-matrix.json
- modified: pose-mcp/internal/pose/review_bundle.go
- created: pose-mcp/internal/pose/missing_change_set_test.go
- created: pose-mcp/internal/cli/closeout_names_missing_trailer_test.go

### Delivery targets

- capability:closeout-names-the-missing-trailer module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go

### API/contract changes

The text of the blocker gains its fix after the same prefix.

## 4. Tasks

- [ ] Report the cause alone, with its fix
- [ ] Independent review and closeout

## 5. Decisions

### Decision D1
- Basis: R1, R3
- Minimal option: mark the evidence blocker as derived and keep both
- Selected option: omit it when the blocker of the change set is present
- Rationale: a derived line still reads as a second task, and the first blocker keeps the bundle from sealing, so nothing is lost
- Consequences: the evidence blocker depends on the text of another blocker, through one constant
- Falsifier: a spec with no change set whose evidence would stay missing after the line is added, reported only after that second attempt

## 6. Validation

### Strategy

An integration test over a validation result pinned to a commit no change set attributes, seen failing on the code before the fix with both messages.

### Requirement trace

- R1 [planned] capability:closeout-names-the-missing-trailer evidence:integration check:closeout-names-trailer-integration test:TestCloseoutPlanNamesTheMissingTrailerOnce test:TestSpecWithoutAChangeSetStaysUnsealable
- R2 [planned] check:closeout-names-trailer-integration test:TestCloseoutPlanNamesTheMissingTrailerOnce
- R3 [planned] check:closeout-names-trailer-integration test:TestOnlyTheMissingChangeSetSilencesTheEvidenceGap test:TestExternalMilestoneKeepsItsEvidenceGate

## 7. Final Report

### Delivered scope

### Residual risks

### Follow-ups
