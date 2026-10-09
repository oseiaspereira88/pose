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

Off by default; adopting it changes no earlier review. Requirements assume the ADR's provisional answers until `pose-delegated-review-contract` closes.

## 2. Requirements

### Functional

- R1: `pose adopt delegated-review` shall turn on the run-backed independence rule, dated the day it is applied, and refuse while no adapter is configured.
- R2: The documentation shall describe the contract, the Codex and Claude Code adapter examples, the cross-vendor preference and every safeguard of the ADR.
- R3: A `pose-delegated-review` skill shall tell an agent when to dispatch, that it must not write the reviewer's prompt, and that a rejection is answered by a change, not a rerun.
- R4: An end-to-end journey shall seal a fixture with a seeded defect, dispatch to a fake adapter, record the rejecting run, fix, rerun on the new bundle and close.

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

No decision recorded yet; the contract is ADR `2026-10-09-delegated-review-is-an-adapter`, pending acceptance.

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
