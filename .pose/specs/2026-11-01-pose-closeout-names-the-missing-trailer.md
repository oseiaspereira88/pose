---
slug: pose-closeout-names-the-missing-trailer
status: done
created_at: 2026-11-01
supersedes:
depends_on:
remediates:
priority: 1
components: pose-mcp
task_type: bugfix
delivers: capability:closeout-names-the-missing-trailer
completed_at: 2026-11-01
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

- [x] Report the cause alone, with its fix
- [x] Independent review and closeout

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

- R1 [satisfied] capability:closeout-names-the-missing-trailer evidence:integration check:closeout-names-trailer-integration test:TestCloseoutPlanNamesTheMissingTrailerOnce test:TestSpecWithoutAChangeSetStaysUnsealable report:.pose/results/delivery-validation.json commit:b73681d347fc11c923fbbbc6f3f563a394a92ca9
- R2 [satisfied] check:closeout-names-trailer-integration test:TestCloseoutPlanNamesTheMissingTrailerOnce report:.pose/results/delivery-validation.json commit:b73681d347fc11c923fbbbc6f3f563a394a92ca9
- R3 [satisfied] check:closeout-names-trailer-integration test:TestOnlyTheMissingChangeSetSilencesTheEvidenceGap test:TestExternalMilestoneKeepsItsEvidenceGate report:.pose/results/delivery-validation.json commit:b73681d347fc11c923fbbbc6f3f563a394a92ca9

## 7. Final Report

### Delivered scope

With no attributed change set the bundle reports that cause alone, with its fix: the `POSE-Spec:` line, `pose index` and `pose artifact-check`. The evidence gap that follows from it is no longer a second reason, as issue #208 asked. The test was seen failing with both messages on the previous code. A bundle with no change set stays unsealable, and a milestone of external members keeps its evidence gate. Independently reviewed by Codex in four rounds, with no blocker.

### Residual risks

The evidence blocker depends on the prefix of another blocker, through one constant.

### Follow-ups

- [covered: pose-evidence-gate-means-validation] F1: For a spec with a change set, `assess-design` evidence fills the list, so the empty-evidence blocker does not establish that validation ran. Decide whether it should distinguish them (review R2-F9).