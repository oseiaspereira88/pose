---
slug: pose-delegated-review-contract
status: draft
created_at: 2026-10-09
completed_at:
supersedes:
depends_on: 
priority: 1
components: pose-mcp
task_type: feature
changelog:
delivers:
---

# Spec: Delegated review contract

## 1. Intent

### Goal

Accept the delegated-review ADR with the maintainer's answers to its four open decisions, and reconcile the roadmap's specs with them before any implementation.

### Business value

The ADR proposes that a delegated reviewer is an adapter turning a sealed bundle into a draft. Four choices change what the later specs build: who records the attestation, whether an agent run can satisfy `different-actor`, whether a different vendor is required, and the initial scope.

Origin: the 2026-10-09 review of `pose-native-attestation-issuer` by Codex through a personal launcher; it found three real defects in three rounds and exposed what the engine lacks (ADR `2026-10-09-delegated-review-is-an-adapter`, roadmap `delegated-review`).

### Constraints

No code. The specs below stay draft until this one closes. Requirements assume the ADR's provisional answers until `pose-delegated-review-contract` closes.

## 2. Requirements

### Functional

- R1: The maintainer shall answer the ADR's four open decisions, each recorded in the ADR with its rationale.
- R2: The ADR status shall move to accepted only when every open decision is answered.
- R3: Each spec of roadmap `delegated-review` shall be amended where an answer differs from the assumption it was written under, before it starts.

## 3. Technical Plan

### Affected areas

Record the answers in the ADR, flip its status and amend the affected specs' requirements.

### Artifacts

- created: .pose/specs/2026-10-09-pose-delegated-review-contract.md
- created: .pose/adr/2026-10-09-delegated-review-is-an-adapter.md
- created: .pose/roadmaps/delegated-review.md

Implementation artifacts are declared when the spec starts.

## 4. Tasks

- [ ] Collect the maintainer's four answers
- [ ] Accept the ADR
- [ ] Reconcile the roadmap's specs

## 5. Decisions

No decision recorded yet; the contract is ADR `2026-10-09-delegated-review-is-an-adapter`, pending acceptance.

## 6. Validation

### Strategy

`pose lint-spec --strict` on every spec of the roadmap after the reconciliation; the ADR lists no open decision.

### Requirement trace

## 7. Final Report

### Delivered scope

Not started: part of roadmap `delegated-review`, opened on 2026-10-09.

### Residual risks

None yet.

### Follow-ups
