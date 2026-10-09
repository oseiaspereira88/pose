---
slug: pose-delegated-review-contract
status: in-progress
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

No code. The specs below stay draft until this one closes. Requirements follow the maintainer's decisions recorded in the accepted ADR on 2026-10-09.

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
- created: .pose/starts/pose-delegated-review-contract.json
- modified: .pose/specs/2026-10-09-pose-delegated-review-brief.md
- modified: .pose/specs/2026-10-09-pose-delegated-review-dispatch.md
- modified: .pose/specs/2026-10-09-pose-delegated-review-capability.md

Implementation artifacts are declared when the spec starts.

## 4. Tasks

- [x] Collect the maintainer's four answers
- [x] Accept the ADR
- [x] Reconcile the roadmap's specs

## 5. Decisions

The contract is ADR `2026-10-09-delegated-review-is-an-adapter`, accepted on 2026-10-09 with the maintainer's decisions: the engine records a verified run, people by exception through policy, agent independence by differing vendor or model (different vendor preferred), and reviews, adjudications and smoke runs from the first delivery.

## 6. Validation

### Strategy

`pose lint-spec --strict` on every spec of the roadmap after the reconciliation; the ADR lists no open decision.

### Requirement trace

- R1 [satisfied] evidence:manual <ADR section "Maintainer's decisions (2026-10-09)" records the four answers and their rationale>
- R2 [satisfied] evidence:manual <ADR status: accepted, no open decision>
- R3 [satisfied] evidence:manual <brief R4/R6, dispatch R5/R7, capability R4-R7 amended to the answers before any spec started; brief R4 corrected after the independent review found it still let the reviewer record>

## 7. Final Report

### Delivered scope

The ADR is accepted with the maintainer's four decisions and the roadmap's specs follow them.

### Residual risks

None yet.

### Follow-ups
