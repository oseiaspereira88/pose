---
slug: retained-review-survives-an-invalid-newer-approval
status: in-progress
created_at: 2026-09-29
completed_at:
supersedes:
depends_on: review-verify-retains-completed-scopes
priority: 0
components: pose-mcp
task_type: bugfix
delivers: capability:completed-review-retention-precedence
---

# Spec: An invalid newer approval does not void a closed scope's standing approval

## 1. Intent

### Goal

A closed scope whose newest attestation is an approval that no longer validates
keeps the older approval it was closed with. Only a newer rejection or request for
changes voids it.

### Business value

6.0.2 made the newest attested bundle decide, to stop an older approval from
hiding a later rejection. It stopped at any newest attestation that did not
validate. In Harne8, `spec:harne8-pose-launch-surfaces` has two approvals: the one
from 2026-09-07 validates under its evidence waiver; the one from 2026-09-09 cites
`site` evidence for the `scripts` component, which a later evidence rule refuses.
Before 6.0.2 `review-check` fell back to the first; under 6.0.2 the scope lost its
approval, `pose check --strict` failed and so did Harne8's real-engine governance
test. The work was approved twice and never rejected.

### Constraints

A newer `rejected` or `changes-requested` attestation still wins over any older
approval, as 6.0.2 intended.

### Non-goals

Revalidating or migrating old attestations.

## 2. Requirements

- R1: When the newest attestation of a closed scope is an approval that no longer
  validates, an older standing approval is retained.
- R2: When the newest attestation rejects or requests changes, no older approval is
  retained.

## 3. Technical Plan

In `retainedCompletedReview`, pass over a bundle whose attestation does not
validate unless its decision is `rejected` or `changes-requested`, which ends the
search without a retained approval.

### Artifacts

- created: .pose/specs/2026-09-29-retained-review-survives-an-invalid-newer-approval.md
- created: .pose/changelogs/unreleased/retained-review-survives-an-invalid-newer-approval.md
- modified: pose-mcp/internal/pose/review_closeout.go
- modified: pose-mcp/internal/pose/completed_review_retention_test.go
- modified: .pose/results/delivery-validation.json
- modified: .pose/indexes/delivery-integrity.json
- modified: .pose/indexes/spec-graph.json

### Delivery targets

- capability:completed-review-retention-precedence module:pose-mcp/internal/pose profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go

### Rollout and reversal

Restores the pre-6.0.2 outcome for closed scopes with an invalid newer approval and
keeps 6.0.2's rule for newer negative decisions.

## 4. Tasks

- [x] Measure why Harne8's launch-surfaces scope lost its approval under 6.0.2.
- [x] Stop only on a newer negative decision.
- [x] Cover the invalid newer approval in a test that fails under 6.0.2.

## 5. Decisions

### Decision D1

- Status: active
- Distinguish a verdict from a validity failure. A rejection is a reviewer's
  conclusion about the work; an approval that fails an evidence rule introduced
  after it says nothing against the work, so it should not void an approval that
  still stands.

## 6. Validation

| Scenario | Command (from pose-mcp) | Expected evidence |
| --- | --- | --- |
| Retention precedence | `go test ./internal/pose -run CompletedReviewRetention -count=1` | invalid newer approval falls back; newer rejection does not |
| Full suite | `go test ./... -count=1` | pass |

### Execution log

2026-09-29: probing Harne8 with the engine's own validation, bundle
`rvb-95a91fb2b4f61558` / `rva-51bb7458cafe1660` validates (waived), and bundle
`rvb-5c52e77ba769c97b` / `rva-e2100d48dbb02eef`, approved, fails with `review tool
validate (component scripts) cites evidence from site`. With the fix, Harne8's
`pose check --strict` at pin `d6fb824` succeeds and `closeout-check
spec:harne8-abm-governed-execution` stays terminal. The new test case fails with
the 6.0.2 rule and passes with this one; the full suite passes.

### Requirement trace

- R1 [pending] test:TestCompletedReviewRetention
- R2 [pending] test:TestCompletedReviewRetention

## 7. Final Report

### Scope delivered

Pending closeout.

### Residual risks

A scope whose newest approval is invalid keeps an older approval that predates the
invalid one. That was the behaviour before 6.0.2 as well.

### Follow-ups

None.
