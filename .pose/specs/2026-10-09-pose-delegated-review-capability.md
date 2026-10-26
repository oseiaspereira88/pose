---
slug: pose-delegated-review-capability
status: draft
created_at: 2026-10-09
completed_at:
supersedes:
depends_on: pose-delegated-review-run-provenance, pose-delegated-review-attempt-ledger, pose-delegation-routing, pose-delegation-handoff, pose-delegated-review-round-protocol, pose-delegated-review-action-fulfillment, pose-delegation-mcp
priority: 3
components: pose-mcp
task_type: feature
changelog:
delivers:
---

# Spec: Delegated review as an adoptable capability

## 1. Intent

### Goal

Make delegated review a catalog capability that `pose init` offers with yes as the default, with documentation, a skill and agent examples, so a project adopts it like any other contract, and so a project that declines it still reaches delivery with an honest record.

### Business value

The maintainer asked for this to become POSE's standard way to hand reviews, attestations and similar work to a secondary agent, without the mechanisation turning into a rubber stamp.

Origin: the 2026-10-09 review of `pose-native-attestation-issuer` by Codex through a personal launcher; it found three real defects in three rounds and exposed what the engine lacks (ADR `2026-10-09-delegated-review-is-an-adapter`, roadmap `delegated-review`).

Amended on 2026-10-26 by the [delegation ADR](../adr/2026-10-26-delegation-is-a-first-class-execution-surface.md): the capability is offered at init instead of opt-in only, runs through `pose delegate`, and when it is off a plan that requires `different-actor` degrades to a disclosed same-actor review (R1–R5 reworded, R8–R11 added).

### Constraints

Offered by `pose init`; an existing instance stays as it is until it adopts the capability; adopting it changes no earlier review. Requirements follow the maintainer's decisions recorded in the 2026-10-09 ADR and amended by the 2026-10-26 ADR. `mandatory-human` is never degraded.

## 2. Requirements

### Functional

- R1: `pose adopt delegated-review` shall turn on the run-backed independence rule, dated the day it is applied, and refuse while `delegation.json` configures no agent.
- R2: The documentation shall describe the contract under `pose delegate`, the Codex and Claude Code agent examples, routing and its fallback steps, the degradation of R10 and every safeguard of both ADRs.
- R3: A `pose-delegate` skill shall tell an agent when to delegate a review and when a task, that it must not write the reviewer's prompt, that a rejection is answered by a change, not a rerun, and how to act on `next_actions`.
- R4: An end-to-end journey shall seal a fixture with a seeded defect, delegate its review to a fake agent, have the engine record the rejecting run, fix, run the next round on the new bundle and close with no manual step; a second journey escalates a person-required criterion to an action request; a third, with the capability off, closes with a disclosed same-actor review.
- R5: When a `pose delegate review` run passes every check — brief generated from the sealed bundle, disposable copy unchanged, run record intact and signed, `different-vendor` or `different-model` independence against the implementation recorded in the bundle, conclusion complete and accepted by the attestation preflight, no earlier rejection on an unchanged bundle — the engine shall record the attestation from the run with no further step; a rejecting conclusion shall be recorded and block the scope until the bundle changes.
- R6: Review policy shall declare which criteria require a person, and the engine shall turn each escalation — such a criterion, reviewers in conflict, a reviewer asking for a business decision, or the run budget exhausted without approval — into an action request with the question and options ready, answered as a confirmation or approval; nothing else waits on a person.
- R7: An `adjudication` or `smoke` run that passes the same checks shall record its verdict where its origin expects it (the adjudicated spec's decision record, the smoke target's evidence), under the same rules.
- R8: `pose init` shall offer delegation with yes as the default answer; on yes it shall write `delegation.json` with an agent for each supported CLI found on the path and record `delegated_review: true` dated the day of the init; on a refusal, or when the init is not interactive, it shall record `delegated_review: false` and write no `delegation.json`.
- R9: `pose update` shall never turn the capability on; while it is off, it shall print one suggestion naming `pose adopt delegated-review`.
- R10: While `delegated_review` is false, a plan that requires `different-actor` shall be satisfied by an attestation with `same-actor-separate-execution`, and `pose review verify`, the closeout and the review record shall disclose the limitation `delegated-review-disabled`; plans that require `mandatory-human` shall be unchanged.
- R11: While the capability is on and routing finds no available agent for a review after its fallback steps, the engine shall raise the escalation of R6 with the question ready, and shall not degrade as R10 does.

## 3. Technical Plan

### Affected areas

Catalog entry, `pose init` and `pose update` prompts, independence degradation in the review plan and its disclosure, docs page, skill, journey tests.

### Artifacts

- created: .pose/specs/2026-10-09-pose-delegated-review-capability.md

Implementation artifacts are declared when the spec starts.

## 4. Tasks

- [ ] Catalog entry
- [ ] Init offer and update suggestion
- [ ] Degradation when off, escalation when no agent is available
- [ ] Docs and agent examples
- [ ] Skill
- [ ] Journeys

## 5. Decisions

The contract is ADR `2026-10-09-delegated-review-is-an-adapter`, accepted on 2026-10-09 with the maintainer's decisions: the engine records a verified run, people by exception through policy, agent independence by differing vendor or model (different vendor preferred), and reviews, adjudications and smoke runs from the first delivery.

The [delegation ADR](../adr/2026-10-26-delegation-is-a-first-class-execution-surface.md), accepted on 2026-10-26, amends it: delegation is offered by `pose init` with yes as the default instead of being opt-in only; with the capability off, `different-actor` degrades to a disclosed `same-actor-separate-execution` so one agent of one vendor can reach delivery; the implementation identity comes from the sealed bundle; runs go through `pose delegate`.

### Decision D1
- Basis: R10, R11
- Minimal option: keep `different-actor` mandatory whether or not the capability is on
- Selected option: degrade to a disclosed same-actor review when the capability is off; escalate, never degrade, when it is on and no agent is available
- Rationale: an instance that declined delegation has no second actor to name, and requiring one only invites a declared reviewer nobody verified; an instance that adopted it has promised independence, so a missing agent is a question for a person, not a silent downgrade
- Consequences: the review record of a declining instance always shows the limitation
- Falsifier: a closeout with the capability on that recorded `same-actor-separate-execution` for a `different-actor` plan without an escalation

## 6. Validation

### Strategy

The journey tests fail before the capability exists and pass after. An interactive init fixture writes `delegation.json` from fake CLIs on the path; a non-interactive init records the capability off; `pose update` suggests without enabling; with the capability off a `different-actor` plan closes with the disclosed limitation and a `mandatory-human` plan does not; with it on and every agent unavailable, the escalation is raised.

### Requirement trace

## 7. Final Report

### Delivered scope

Not started: opened on 2026-10-09 under roadmap `delegated-review`; moved on 2026-10-26 to milestone `adoption` of roadmap `delegation`.

### Residual risks

None yet.

### Follow-ups
