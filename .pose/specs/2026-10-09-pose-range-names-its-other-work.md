---
slug: pose-range-names-its-other-work
status: in-progress
created_at: 2026-10-09
completed_at:
supersedes:
depends_on: pose-abm-subject-evidence
priority: 1
components: pose-mcp
task_type: feature
surface: minimal
delivers: capability:range-names-its-other-work
---

# Spec: A contaminated range names its other work

## 1. Intent

### Goal

When a change set's range spans commits it does not attribute, say whose they are: the specs whose `POSE-Spec:` trailer is on them, and how many carry no trailer.

### Business value

Origin: the open follow-up of `pose-abm-subject-evidence` (crit medium), prioritized by the maintainer on 2026-10-09. The observation counted unattributed commits, so a reviewer knew the range was contaminated but not by what; the ABM field pilot's false positives came from exactly that mixing.

### Constraints

The observation stays outside the sealed payload. When Git cannot answer, nothing new is reported and the counts stand.

### Non-goals

Changing what the subject attributes; that is `pose-structural-facts-stay-in-their-commits`.

## 2. Requirements

### Functional

- R1: A contaminated range observation shall list the other specs whose trailer is on a commit the range spans but the change set does not attribute, and count those with no trailer.
- R2: The review bundle warning shall name them.

### Compatibility

- Two optional fields on an unsealed observation; the schema documents them.

## 3. Technical Plan

### Artifacts

- created: .pose/specs/2026-10-09-pose-range-names-its-other-work.md
- modified: pose-mcp/internal/pose/review_bundle.go
- modified: pose-mcp/internal/pose/subject_evidence_test.go
- modified: pose-mcp/schemas/v1/review-bundle.schema.json
- modified: .pose/indexes/validation-matrix.json
- modified: .pose/specs/2026-09-19-pose-abm-subject-evidence.md

### Delivery targets

- capability:range-names-its-other-work module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go

## 5. Decisions

No material decision: the follow-up named the remedy, and the trailer scan already exists in this package for structural attribution.

## 6. Validation

### Strategy

`TestSubjectRangeObservationNamesTheOtherWork` builds a range holding another spec's commit and a merged commit with no trailer; the observation names `other`, counts the untrailed commit, and the warning says both.

### Deterministic checks

#### Test
- Command: `cd pose-mcp && go test ./internal/pose -run SubjectRange`
- Scope: pose-mcp
- Expected: pass

### Requirement trace

- R1 [satisfied] capability:range-names-its-other-work check:range-ownership-integration evidence:integration test:TestSubjectRangeObservationNamesTheOtherWork
- R2 [satisfied] capability:range-names-its-other-work check:range-ownership-integration evidence:integration test:TestSubjectRangeObservationNamesTheOtherWork

## 7. Final Report

### Delivered scope

A contaminated range observation carries `unattributed_specs` and `untrailed_commits`, read by one bounded `git log` of the range, and the bundle warning names them.

### Residual risks

- Reviewed in the same session that implemented it, with no separate reviewer execution: declared, not independent.

### Follow-ups
