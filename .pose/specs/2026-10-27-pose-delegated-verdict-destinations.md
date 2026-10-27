---
slug: pose-delegated-verdict-destinations
status: draft
created_at: 2026-10-27
completed_at:
supersedes:
depends_on: pose-delegated-review-conclusions
priority: 3
components: pose-mcp
task_type: feature
delivers:
---

# Spec: An adjudication or smoke run records its verdict where its origin expects it

## 1. Intent

### Goal

Give the verdict of a delegated adjudication and of a delegated smoke run a governed destination, and let the engine record it there under the rules a review conclusion follows.

### Business value

R2 of [`pose-delegated-review-conclusions`](2026-10-27-pose-delegated-review-conclusions.md) asks for these verdicts to be recorded "where the origin expects them" — the adjudicated spec's decision record, the smoke target's evidence — and neither destination exists. The maintainer chose on 2026-10-27 to deliver review conclusions first and to define these destinations in a spec of their own.

### Constraints

The same checks as a review conclusion apply: an intact, signed run of a current bundle, a complete typed verdict, idempotence, refusals recorded with their reason. A verdict never becomes an approval of another scope.

### Non-goals

Review conclusions (delivered by `pose-delegated-review-conclusions`); turning the capability on.

## 2. Requirements

### Functional

- R1: An `adjudication` run that passes every check shall record its verdict as a decision record of the adjudicated spec, readable by the spec's lint and closeout, and the adjudicated questions shall carry the verdict's answer for each.
- R2: A `smoke` run that passes every check shall record, for each delivery target it exercised, an observation that the surface checks read as evidence of that target, with the commands it ran and what it observed.
- R3: A verdict that fails a check, is incomplete or does not parse, shall record no governed effect and keep its reason, as a review conclusion does.

## 3. Technical Plan

### Artifacts

- created: .pose/specs/2026-10-27-pose-delegated-verdict-destinations.md

Implementation artifacts are declared when the spec starts.

## 4. Tasks

- [ ] Adjudication verdict as a decision record
- [ ] Smoke observation as surface evidence
- [ ] Refusals and idempotence

## 5. Decisions

No decision recorded yet.

## 6. Validation

### Strategy

Fixtures for each kind: a passing run records its verdict at its destination; each failed check, an incomplete verdict and a replay record nothing.

### Requirement trace

## 7. Final Report

### Delivered scope

Not started: split on 2026-10-27 from R2 of `pose-delegated-review-conclusions`.

### Residual risks

None yet.

### Follow-ups
