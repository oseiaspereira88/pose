---
slug: review-subject-classifies-capabilities
status: in-progress
created_at: 2026-09-29
completed_at:
supersedes:
depends_on:
priority: 1
components: pose-mcp
task_type: bugfix
delivers: capability:review-subject-capabilities
---

# Spec: The review subject classifies the capability assessment

## 1. Intent

### Goal

A change set that touched `.pose/capabilities/` can be sealed for review.

### Business value

`pose capability` writes `.pose/capabilities/assessment.md` and appends to
`.pose/capabilities/history.jsonl`. Neither path matched a review subject rule,
so any bundle whose change set contained them refused to seal with
`unclassified review subject path`. It surfaced when closing
`pose-stdio-server-honours-sigterm`, whose R3 reassessed a capability: the
requirement was met, and the closeout could not proceed because of it. Two other
specs from the same squash commit were blocked the same way.

### Constraints

Unknown paths keep failing closed. Only these two paths gain a classification.

### Non-goals

Reclassifying any other path.

## 2. Requirements

- R1: A change set containing `.pose/capabilities/assessment.md` or
  `.pose/capabilities/history.jsonl` seals without an unclassified-path blocker.
- R2: The assessment is in the review subject as `governance`, since its bullets
  are the authority on each mechanism's state.
- R3: The history is not in the subject and is recorded as an excluded
  `derived-evidence` input, since it carries only snapshots of the assessment.

## 3. Technical Plan

`reviewBundlePathClass` returns `derived-evidence` for the history file, checked
before the general rules, and lists `.pose/capabilities/` with the governance
prefixes for everything else under it.

### Artifacts

- created: .pose/specs/2026-09-29-review-subject-classifies-capabilities.md
- created: .pose/changelogs/unreleased/review-subject-classifies-capabilities.md
- created: pose-mcp/internal/pose/review_subject_capabilities_test.go
- modified: pose-mcp/internal/pose/review_bundle.go
- modified: .pose/indexes/validation-matrix.json

### Delivery targets

- capability:review-subject-capabilities module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go

### Rollout and reversal

The subject accepts two paths it refused before; nothing that sealed before
seals differently. Reverting restores the refusal.

## 4. Tasks

- [x] Reproduce the refusal on a real change set.
- [x] Pin the classification in a test that fails without the fix.
- [x] Classify the assessment as governance and the history as derived evidence.
- [ ] Run the checks, obtain review and close.

## 5. Decisions

### Decision D1

- Status: active
- The assessment is governance, not derived evidence. It is written by commands,
  but what it records is a judgement about each mechanism that a reviewer must
  see change; `.pose/knowledge/` and `.pose/roadmaps/` are classified the same
  way for the same reason. The history adds nothing the assessment does not
  carry, so it is excluded like `.pose/state/`.

## 6. Validation

| Scenario | Command | Expected evidence |
| --- | --- | --- |
| Classification | `go test ./internal/pose -run ReviewSubjectClassifiesTheCapabilityAssessment -count=1` (from pose-mcp) | no blocker; assessment governance; history excluded as derived-evidence |
| Real change set | `pose review bundle spec:pose-stdio-server-honours-sigterm --seal` | seals |

### Execution log

2026-09-29: `pose review bundle spec:pose-stdio-server-honours-sigterm --seal`
refused with `unclassified review subject path .pose/capabilities/assessment.md`
and the same for `history.jsonl`; so did the two other specs of squash commit
`20c6565`.

2026-09-29, implemented. With the classification reverted, the new test fails on
the unclassified-path blocker for both files. With it, the test and the whole
`internal/pose` package pass.

### Requirement trace

- R1 [satisfied] capability:review-subject-capabilities evidence:integration check:review-subject-capabilities-integration test:TestReviewSubjectClassifiesTheCapabilityAssessment — no blocker names a `.pose/capabilities/` path
- R2 [satisfied] capability:review-subject-capabilities evidence:integration check:review-subject-capabilities-integration test:TestReviewSubjectClassifiesTheCapabilityAssessment — the assessment's subject entry has class governance
- R3 [satisfied] capability:review-subject-capabilities evidence:integration check:review-subject-capabilities-integration test:TestReviewSubjectClassifiesTheCapabilityAssessment — the history is absent from the subject and present as an excluded derived-evidence input

## 7. Final Report

### Scope delivered

Pending closeout.

### Residual risks

None known.

### Follow-ups

None.
