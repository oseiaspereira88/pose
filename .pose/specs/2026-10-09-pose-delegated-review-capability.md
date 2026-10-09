---
slug: pose-delegated-review-capability
status: draft
created_at: 2026-10-09
completed_at:
supersedes:
depends_on: pose-delegated-review-run-provenance, pose-delegated-review-attempt-ledger
priority: 3
components: pose-mcp
task_type: feature
changelog:
delivers:
---

# Spec: Delegated review as an adoptable capability

## 1. Intent

### Goal

Make delegated review a catalog capability, off by default, with documentation, a skill and adapter examples, so a project adopts it like any other contract.

### Business value

The maintainer asked for this to become POSE's standard way to hand reviews, attestations and similar work to a secondary agent, without the mechanisation turning into a rubber stamp.

Origin: the 2026-10-09 review of `pose-native-attestation-issuer` by Codex through a personal launcher; it found three real defects in three rounds and exposed what the engine lacks (ADR `2026-10-09-delegated-review-is-an-adapter`, roadmap `delegated-review`).

### Constraints

Off by default; adopting it changes no earlier review. Requirements follow the maintainer's decisions recorded in the accepted ADR on 2026-10-09.

## 2. Requirements

### Functional

- R1: `pose adopt delegated-review` shall turn on the run-backed independence rule, dated the day it is applied, and refuse while no adapter is configured.
- R2: The documentation shall describe the contract, the Codex and Claude Code adapter examples, the cross-vendor preference and every safeguard of the ADR.
- R3: A `pose-delegated-review` skill shall tell an agent when to dispatch, that it must not write the reviewer's prompt, and that a rejection is answered by a change, not a rerun.
- R4: An end-to-end journey shall seal a fixture with a seeded defect, dispatch to a fake adapter, have the engine record the rejecting run, fix, rerun on the new bundle and close with no manual step; a second journey escalates a person-required criterion to an action request.
- R5: When a dispatch run of kind `review` passes every check — brief generated from the sealed bundle, disposable copy unchanged, run record intact and signed, vendor or model different from the implementation's, conclusion complete and accepted by the attestation preflight, no earlier rejection on an unchanged bundle — the engine shall record the attestation from the run with no further step; a rejecting conclusion shall be recorded and block the scope until the bundle changes.
- R6: Review policy shall declare which criteria require a person, and the engine shall turn each escalation — such a criterion, reviewers in conflict, a reviewer asking for a business decision, or the run budget exhausted without approval — into an action request with the question and options ready, answered as a confirmation or approval; nothing else waits on a person.
- R7: An `adjudication` or `smoke` run that passes the same checks shall record its verdict where its origin expects it (the adjudicated spec's decision record, the smoke target's evidence), under the same rules.

## 3. Technical Plan

### Affected areas

Catalog entry, docs page, skill, journey test.

### Artifacts

- created: .pose/specs/2026-10-09-pose-delegated-review-capability.md

Implementation artifacts are declared when the spec starts.

## 4. Tasks

- [ ] Catalog entry
- [ ] Docs and adapter examples
- [ ] Skill
- [ ] Journey

## 5. Decisions

The contract is ADR `2026-10-09-delegated-review-is-an-adapter`, accepted on 2026-10-09 with the maintainer's decisions: the engine records a verified run, people by exception through policy, agent independence by differing vendor or model (different vendor preferred), and reviews, adjudications and smoke runs from the first delivery.

## 6. Validation

### Strategy

The journey test fails before the capability exists and passes after.

### Requirement trace

## 7. Final Report

### Delivered scope

Not started: part of roadmap `delegated-review`, opened on 2026-10-09.

### Residual risks

None yet.

### Follow-ups
