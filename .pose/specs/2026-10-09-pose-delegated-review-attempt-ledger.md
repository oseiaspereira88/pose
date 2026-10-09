---
slug: pose-delegated-review-attempt-ledger
status: draft
created_at: 2026-10-09
completed_at:
supersedes:
depends_on: pose-delegated-review-dispatch
priority: 2
components: pose-mcp
task_type: feature
changelog:
delivers:
---

# Spec: Every delegated review attempt stays on the record

## 1. Intent

### Goal

Keep every delegated review run, rejections included, so a review cannot be retried until it approves.

### Business value

Nothing on 2026-10-09 would have stopped the implementer from discarding the reviewer's "changes required" and running another reviewer. The three rejections that found real defects were valuable precisely because they stood.

Origin: the 2026-10-09 review of `pose-native-attestation-issuer` by Codex through a personal launcher; it found three real defects in three rounds and exposed what the engine lacks (ADR `2026-10-09-delegated-review-is-an-adapter`, roadmap `delegated-review`).

### Constraints

Append-only; no command deletes or rewrites a run. Requirements assume the ADR's provisional answers until `pose-delegated-review-contract` closes.

## 2. Requirements

### Functional

- R1: Runs under `.pose/review-runs/` shall be append-only and content-addressed, and `pose history-check` shall treat a rewritten or deleted run as a violation.
- R2: A new dispatch on a bundle that already has a completed run shall be refused unless the bundle changed or `--reason` is recorded with the new run.
- R3: `pose review verify` shall disclose earlier rejecting runs for the same scope next to the approving one.
- R4: `pose insights` shall report runs per bundle, approval rate per adapter, and reruns without a bundle change.

## 3. Technical Plan

### Affected areas

Ledger layout, the rerun rule in dispatch, verify disclosure and insights metrics.

### Artifacts

- created: .pose/specs/2026-10-09-pose-delegated-review-attempt-ledger.md

Implementation artifacts are declared when the spec starts.

## 4. Tasks

- [ ] Append-only run ledger
- [ ] Rerun rule
- [ ] Verify disclosure
- [ ] Insights metrics

## 5. Decisions

No decision recorded yet; the contract is ADR `2026-10-09-delegated-review-is-an-adapter`, pending acceptance.

## 6. Validation

### Strategy

Fixture with a rejecting run: a rerun on the same bundle without a reason is refused; verify shows the rejection; deleting a run fails history-check.

### Requirement trace

## 7. Final Report

### Delivered scope

Not started: part of roadmap `delegated-review`, opened on 2026-10-09.

### Residual risks

None yet.

### Follow-ups
