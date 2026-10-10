---
slug: pose-same-actor-level-admits-a-different-actor
status: done
created_at: 2026-10-10
completed_at: 2026-10-10
supersedes:
depends_on:
priority: 0
components: pose-mcp
task_type: bugfix
surface: minimal
changelog:
delivers: governance:same-actor-level-admits-a-different-actor
---

# Spec: The lowest independence level admits a different actor

## 1. Intent

### Goal

A review by another principal, in another execution, shall satisfy a policy that requires `same-actor-separate-execution`.

### Business value

On 2026-10-10 an independent Codex review of Harne8's `pose-configuration-review-7-2-0` approved the scope, and the engine refused the signed attestation: `review policy requires the same actor and the authority claim names different principals`. Harne8's policy requires `same-actor-separate-execution` for specs, so under verified identity the lowest level refused exactly the cross-vendor reviews the delegated-review roadmap makes standard, while `different-actor` would have accepted them. The engine's own comment says the three values are ordered; this check contradicted it.

### Constraints

The implementation's own execution must still be refused, and the higher levels keep their checks.

## 2. Requirements

### Functional

- R1: Under `same-actor-separate-execution`, an authority claim whose reviewer differs from the implementation principal, in a different execution, shall verify.
- R2: Under `same-actor-separate-execution`, a claim naming the implementation's own execution as the review execution shall still be refused, whoever reviews.

## 3. Technical Plan

### Artifacts

- created: .pose/specs/2026-10-10-pose-same-actor-level-admits-a-different-actor.md
- created: .pose/starts/pose-same-actor-level-admits-a-different-actor.json
- created: .pose/changelogs/unreleased/pose-same-actor-level-admits-a-different-actor.md
- modified: pose-mcp/internal/pose/review_bundle.go
- modified: pose-mcp/internal/pose/review_authority_test.go

### Delivery targets

- governance:same-actor-level-admits-a-different-actor module:pose-mcp profile:release-governance entrypoint:pose-mcp/cmd/pose/main.go

## 5. Decisions

### Decision D1
- Date: 2026-10-10
- Context: R5 of `pose-abm-review-authority` reads "`same-actor-separate-execution` exige principal e executions distintos", which the implementation read as "same principal, distinct executions". The same spec orders the three values, and the code comment above the check says the checks must be ordered with them.
- Decision: drop the equality of principals; keep the distinct-execution check.
- Rationale: a different actor is more separation than a separate run of the same actor; refusing it makes the floor stricter than the level above it.

## 6. Validation

### Strategy

`TestSameActorLevelAdmitsADifferentActor` checks a different actor, the same actor in a separate run, and the implementation's own run under `same-actor-separate-execution`; it fails on the previous check with the refusal Harne8 met.

### Requirement trace

- R1 [satisfied] governance:same-actor-level-admits-a-different-actor evidence:unit test:TestSameActorLevelAdmitsADifferentActor
- R2 [satisfied] governance:same-actor-level-admits-a-different-actor evidence:unit test:TestSameActorLevelAdmitsADifferentActor

## 7. Final Report

### Delivered scope

Under `same-actor-separate-execution`, a signed review by another principal in another execution verifies; the implementation's own run is still refused.

### Residual risks

Harne8 needs a pose-dist pin that contains this fix before `pose-configuration-review-7-2-0` can close with the Codex review.

### Follow-ups
