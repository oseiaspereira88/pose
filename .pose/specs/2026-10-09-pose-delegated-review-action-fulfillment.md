---
slug: pose-delegated-review-action-fulfillment
status: draft
created_at: 2026-10-09
completed_at:
supersedes:
depends_on: pose-delegated-review-dispatch, pose-delegated-review-conclusions, pose-delegated-review-run-provenance, pose-delegated-review-attempt-ledger, pose-delegated-adapter-containment, pose-action-target-and-node-scoping, pose-action-waiver-and-supersession-authority, pose-action-request-recovery-and-preflight, pose-review-approval-domain-preflight
priority: 3
components: pose-mcp
task_type: feature
changelog:
delivers:
---

# Spec: A delegated review requested as an action and fulfilled by any runner

## 1. Intent

### Goal

Let a project request a delegated review as an action request that a local agent, CI or Harne8's Conductor fulfils through the same contract.

### Business value

The maintainer requires that a developer can work with the repository and POSE alone or with Harne8, neither excluding the other. A review request that any runner can fulfil keeps both paths equal.

Origin: the 2026-10-09 review of `pose-native-attestation-issuer` by Codex through a personal launcher; it found three real defects in three rounds and exposed what the engine lacks (ADR `2026-10-09-delegated-review-is-an-adapter`, roadmap `delegated-review`). Command names follow the [delegation ADR](../adr/2026-10-26-delegation-is-a-first-class-execution-surface.md) of 2026-10-26.

### Constraints

The runner produces the same run record and draft as `pose delegate review`; the request is satisfied only by a valid run. Requirements assume the ADR's provisional answers until `pose-delegated-review-contract` closes.

## 2. Requirements

### Functional

- R1: `pose review request <bundle> --kind delegated-review` shall open an action request carrying the brief digest and the accepted agents.
- R2: A runner shall fulfil it by attaching a run record whose brief digest matches; any other answer leaves it unsatisfied.
- R3: The same request shall be fulfillable by a local `pose delegate review` and by a non-local runner, demonstrated with a CI job and documented for the Harne8 Conductor.

R4–R11 reconcile the extensions F1–F8 of the [batch 11 report](../reports/2026-10-13-pose-batch-11-authoring.md) (amendment of 2026-10-27); R1–R3 are unchanged. The phase B19 acceptance that this milestone requires is [`pose-delegated-review-conclusions`](2026-10-27-pose-delegated-review-conclusions.md) being done, by the amendment of 2026-10-27 to the delegation ADR; this spec does not require the capability to be done.

- R4 (F1): The request shall bind its project, worktree and authorized actor, origin, scope, targets and kind, the bundle, brief and input digests, and the policy digest and revision, reusing `ActionRequest`; a configured agent is not by that fact able to fulfil it.
- R5 (F2): A run shall satisfy the request only when its signed proof verifies, its attempt ledger is intact, its containment outcome is complete, its conclusion was recorded by the engine and its destination is the one the request names; a matching brief digest or a completed status alone does not, and any other answer keeps the condition with its diagnostic.
- R6 (F3): Fulfilling the requested review shall be distinct from satisfying an approval: a valid rejecting run fulfils the service while approvals and gates stay unsatisfied and the rejection stays visible; a negative outcome never becomes an approval.
- R7 (F4): A criterion that policy reserves for a person shall not be satisfied by an agent's run; exhausted budget, reviewers in conflict and a business decision shall produce the question, its options, context and authority, as R6 of `pose-delegated-review-capability` defines.
- R8 (F5): The request digest, revision and subject shall bind to the run and to its effect at the destination; a replay with the same key and input shall be a faithful no-op, a changed or conflicting one refused, and a superseded bundle, subject or policy shall require revalidation.
- R9 (F6): Local, CI and the Harne8 boundary shall submit the same envelope to the same verifier under an authorized principal and scope; the runner chooses neither authority nor destination and records no person.
- R10 (F7): A fault before or after the conclusion, the answer or the receipt shall not duplicate an effect or a spawn; partial, deadline and cleanup outcomes stay distinct from a valid rejection, and no retry, alias or reason restores budget or lifts the block on an unchanged bundle.
- R11 (F8): The journey request → runner → conclusion and proof → effect → resumption shall be qualified per kind, with a negative outcome, a required person, budget, conflict and business escalations and a stale phase checkpoint, through CLI, MCP and a synthetic CI job; a real Harne8 integration is announced only with evidence from its boundary.

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

The contract is ADR `2026-10-09-delegated-review-is-an-adapter`, accepted on 2026-10-09, amended by the [delegation ADR](../adr/2026-10-26-delegation-is-a-first-class-execution-surface.md) of 2026-10-26 (command names) and its amendment of 2026-10-27 (phase B19 as a spec). R4–R11 reconcile F1–F8 of batch 11 before the start, as the roadmap gate requires.

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
