---
slug: pose-delegated-review-action-fulfillment
status: draft
created_at: 2026-10-09
completed_at:
supersedes:
depends_on: pose-delegated-review-dispatch
priority: 3
components: pose-mcp
task_type: feature
changelog:
delivers:
---

# Spec: A delegated review requested as an action and fulfilled by any runner

## 1. Intent

### Goal

Let a project request a delegated review as an action request that a local adapter, CI or Harne8's Conductor fulfils through the same contract.

### Business value

The maintainer requires that a developer can work with the repository and POSE alone or with Harne8, neither excluding the other. A review request that any runner can fulfil keeps both paths equal.

Origin: the 2026-10-09 review of `pose-native-attestation-issuer` by Codex through a personal launcher; it found three real defects in three rounds and exposed what the engine lacks (ADR `2026-10-09-delegated-review-is-an-adapter`, roadmap `delegated-review`).

### Constraints

The runner produces the same run record and draft as `review dispatch`; the request is satisfied only by a valid run. Requirements assume the ADR's provisional answers until `pose-delegated-review-contract` closes.

## 2. Requirements

### Functional

- R1: `pose review request <bundle> --kind delegated-review` shall open an action request carrying the brief digest and the accepted adapters.
- R2: A runner shall fulfil it by attaching a run record whose brief digest matches; any other answer leaves it unsatisfied.
- R3: The same request shall be fulfillable by a local `pose review dispatch` and by a non-local runner, demonstrated with a CI job and documented for the Harne8 Conductor.

## 3. Technical Plan

### Affected areas

Action kind and satisfaction rule, the fulfilment command, and a CI example.

### Artifacts

- created: .pose/specs/2026-10-09-pose-delegated-review-action-fulfillment.md

Implementation artifacts are declared when the spec starts.

## 4. Tasks

- [ ] Action kind
- [ ] Fulfilment by run record
- [ ] CI example and Conductor documentation

## 5. Decisions

No decision recorded yet; the contract is ADR `2026-10-09-delegated-review-is-an-adapter`, pending acceptance.

## 6. Validation

### Strategy

Fixture: a request fulfilled by a local run is satisfied; one answered without a run, or with a run for another brief, stays unsatisfied.

### Requirement trace

## 7. Final Report

### Delivered scope

Not started: part of roadmap `delegated-review`, opened on 2026-10-09.

### Residual risks

None yet.

### Follow-ups
