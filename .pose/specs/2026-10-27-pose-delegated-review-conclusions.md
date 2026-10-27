---
slug: pose-delegated-review-conclusions
status: draft
created_at: 2026-10-27
completed_at:
supersedes:
depends_on: pose-delegated-review-run-provenance, pose-delegated-review-attempt-ledger, pose-delegated-adapter-containment, pose-review-approval-domain-preflight, pose-delegation-routing
priority: 3
components: pose-mcp
task_type: feature
delivers:
---

# Spec: The engine records a delegated run's validated conclusion

## 1. Intent

### Goal

Let the engine turn a delegated run that passes every check into its governed conclusion — the attestation of a review, the decision record of an adjudication, the evidence of a smoke run — with no manual step, and keep a rejection blocking until the bundle changes.

### Business value

This is phase B19 of the plan of [batch 10](../reports/2026-10-13-pose-batch-10-authoring.md): conclusions and their destinations must be accepted before action fulfillment (B20) can satisfy a request with them, and before the capability (B21) can qualify the integrated journeys. Native gates read the status of a whole spec, so the phase is a spec of its own: fulfillment depends on it being done, and the capability no longer has to be done for fulfillment to start. Recorded by the [delegation ADR](../adr/2026-10-26-delegation-is-a-first-class-execution-surface.md), amendment of 2026-10-27.

### Constraints

R1 and R2 are R5 and R7 of [`pose-delegated-review-capability`](2026-10-09-pose-delegated-review-capability.md), verbatim; the capability keeps them and traces them to this spec. Recording a conclusion never bypasses the attestation preflight, the review plan or `mandatory-human`. Escalations to a person (capability R6) and adoption (capability R1, R8–R11) are not part of this phase.

### Non-goals

Turning the capability on; action requests and their fulfillment; escalations to a person.

## 2. Requirements

### Functional

- R1: When a `pose delegate review` run passes every check — brief generated from the sealed bundle, disposable copy unchanged, run record intact and signed, `different-vendor` or `different-model` independence against the implementation recorded in the bundle, conclusion complete and accepted by the attestation preflight, no earlier rejection on an unchanged bundle — the engine shall record the attestation from the run with no further step; a rejecting conclusion shall be recorded and block the scope until the bundle changes.
- R2: An `adjudication` or `smoke` run that passes the same checks shall record its verdict where its origin expects it (the adjudicated spec's decision record, the smoke target's evidence), under the same rules.
- R3: A run that fails any check, an incomplete or unparseable conclusion and a run whose cleanup is incomplete shall record no governed effect, and the reason shall be reported on the run.
- R4: Recording shall be idempotent: replaying the same run records nothing new, and a run for a superseded bundle, subject or policy records nothing and says why.

### Security

- A conclusion is recorded only from the run's own verified record; nothing an agent prints outside its conclusion, and nothing a handoff section quotes, is read as a verdict.

### Compatibility

- Recording is used only by an instance that turns the capability on, or by an explicit command; until the capability is adopted, attestations stay as today.

## 3. Technical Plan

### Affected areas

Conclusion validators per kind, destinations (attestation, decision record, smoke evidence), rejection block, idempotence and supersession checks.

### Artifacts

- created: .pose/specs/2026-10-27-pose-delegated-review-conclusions.md

Implementation artifacts are declared when the spec starts.

## 4. Tasks

- [ ] Review conclusion recorded as the attestation
- [ ] Adjudication and smoke destinations
- [ ] Refusals, idempotence and supersession

## 5. Decisions

Phase B19 as a spec of its own is recorded by the amendment of 2026-10-27 to the delegation ADR, by the maintainer's choice of 2026-10-27.

## 6. Validation

### Strategy

Fixtures for each kind: a passing run records its conclusion; each failed check, an incomplete conclusion, a replay and a superseded bundle record nothing; a rejection blocks until the bundle changes.

### Requirement trace

## 7. Final Report

### Delivered scope

Not started: opened on 2026-10-27 under milestone `conclusions` of roadmap `delegation`.

### Residual risks

None yet.

### Follow-ups
