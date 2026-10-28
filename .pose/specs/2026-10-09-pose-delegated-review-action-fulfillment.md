---
slug: pose-delegated-review-action-fulfillment
status: done
created_at: 2026-10-09
completed_at: 2026-10-28
supersedes:
depends_on: pose-delegated-review-dispatch, pose-delegated-review-conclusions, pose-delegated-review-run-provenance, pose-delegated-review-attempt-ledger, pose-delegated-adapter-containment, pose-action-target-and-node-scoping, pose-action-waiver-and-supersession-authority, pose-action-request-recovery-and-preflight, pose-review-approval-domain-preflight
priority: 3
components: pose-mcp
task_type: feature
changelog:
delivers: capability:delegated-review-action-fulfillment
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

- R4: (F1) The request shall bind its project, worktree and authorized actor, origin, scope, targets and kind, the bundle, brief and input digests, and the policy digest and revision, reusing `ActionRequest`; a configured agent is not by that fact able to fulfil it.
- R5: (F2) A run shall satisfy the request only when its signed proof verifies, its attempt ledger is intact, its containment outcome is complete, its conclusion was recorded by the engine and its destination is the one the request names; a matching brief digest or a completed status alone does not, and any other answer keeps the condition with its diagnostic.
- R6: (F3) Fulfilling the requested review shall be distinct from satisfying an approval: a valid rejecting run fulfils the service while approvals and gates stay unsatisfied and the rejection stays visible; a negative outcome never becomes an approval.
- R7: (F4) A criterion that policy reserves for a person shall not be satisfied by an agent's run; exhausted budget, reviewers in conflict and a business decision shall produce the question, its options, context and authority, as R6 of `pose-delegated-review-capability` defines.
- R8: (F5) The request digest, revision and subject shall bind to the run and to its effect at the destination; a replay with the same key and input shall be a faithful no-op, a changed or conflicting one refused, and a superseded bundle, subject or policy shall require revalidation.
- R9: (F6) Local, CI and the Harne8 boundary shall submit the same envelope to the same verifier under an authorized principal and scope; the runner chooses neither authority nor destination and records no person.
- R10: (F7) A fault before or after the conclusion, the answer or the receipt shall not duplicate an effect or a spawn; partial, deadline and cleanup outcomes stay distinct from a valid rejection, and no retry, alias or reason restores budget or lifts the block on an unchanged bundle.
- R11: (F8) The journey request → runner → conclusion and proof → effect → resumption shall be qualified per kind, with a negative outcome, a required person, budget, conflict and business escalations and a stale phase checkpoint, through CLI, MCP and a synthetic CI job; a real Harne8 integration is announced only with evidence from its boundary.

## 3. Technical Plan

### Affected areas

Action kind and satisfaction rule, the fulfilment command, and a CI example.

### Artifacts

- modified: .pose/adr/2026-10-25-bind-delegated-review-authority-to-native-signed-run-proofs.md
- created: .pose/changelogs/unreleased/pose-delegated-review-action-fulfillment.md
- created: .pose/specs/2026-10-09-pose-delegated-review-action-fulfillment.md
- created: .pose/starts/pose-delegated-review-action-fulfillment.json
- modified: POSE.md
- modified: docs-site/docs/mcp.md
- modified: docs-site/docs/review-and-trust.md
- modified: locales/pt-BR/POSE.md
- modified: pose-mcp/internal/cli/action.go
- modified: pose-mcp/internal/cli/delegate.go
- modified: pose-mcp/internal/cli/delegate_watch.go
- created: pose-mcp/internal/cli/delegated_review_request_test.go
- modified: pose-mcp/internal/cli/help_catalog.go
- modified: pose-mcp/internal/cli/review_closeout.go
- created: pose-mcp/internal/cli/review_request.go
- modified: pose-mcp/internal/cli/review_run_provenance_test.go
- created: pose-mcp/internal/mcpserver/action_fulfil_test.go
- modified: pose-mcp/internal/mcpserver/catalog.go
- modified: pose-mcp/internal/mcpserver/review_run_provenance_test.go
- modified: pose-mcp/internal/mcpserver/server.go
- modified: pose-mcp/internal/mcpserver/server_test.go
- modified: pose-mcp/internal/mcpserver/testdata/tool-catalog.golden.json
- modified: pose-mcp/internal/pose/action_preflight.go
- modified: pose-mcp/internal/pose/action_request.go
- created: pose-mcp/internal/pose/delegated_review_escalation.go
- created: pose-mcp/internal/pose/delegated_review_request.go
- created: pose-mcp/internal/pose/delegated_review_request_test.go
- modified: pose-mcp/internal/pose/governance_waits.go
- modified: pose-mcp/internal/pose/journal_integrity.go
- modified: pose-mcp/internal/pose/review_rounds.go
- modified: pose-mcp/internal/pose/review_conclusions.go
- modified: pose-mcp/internal/pose/review_run_provenance.go
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md

### Delivery targets

- capability:delegated-review-action-fulfillment module:pose-mcp profile:composed-capability entrypoint:pose-mcp/internal/cli/review_request.go

## 4. Tasks

- [x] Action kind
- [x] Fulfilment by run record
- [x] CI example and Conductor documentation
- [x] Escalations to a person
- [x] Portable run proofs
- [x] Independent review and closeout

## 5. Decisions

The contract is ADR `2026-10-09-delegated-review-is-an-adapter`, accepted on 2026-10-09, amended by the [delegation ADR](../adr/2026-10-26-delegation-is-a-first-class-execution-surface.md) of 2026-10-26 (command names) and its amendment of 2026-10-27 (phase B19 as a spec). R4–R11 reconcile F1–F8 of batch 11 before the start, as the roadmap gate requires.

### Decision D1
- Basis: R1, R4
- Minimal option: reuse the external-operation kind with review metadata
- Selected option: a new action kind `delegated-review`, opened by `pose review request <scope|bundle> --kind delegated-review`, carrying its material — purpose, scope, bundle and digest, brief digest and template, notes digest, policy digest, worktree, accepted agents and the destination of the conclusion — in the request digest; its subject is the scope's current bundle and the policies, read by the engine, so a superseded bundle or a policy change invalidates it. It admits no answer. The maintainer chose this on 2026-10-28
- Rationale: the generic satisfaction rule, an actor's answer, is exactly what must not satisfy it
- Consequences: other kinds and their digests are unchanged; the request's next step is to fulfil, not to answer
- Falsifier: a delegated-review request satisfied by an actor's answer

### Decision D2
- Basis: R2, R3, R5, R8, R9, R10
- Minimal option: only `pose delegate review --request` in the repository fulfils it
- Selected option: a verified envelope — request, request digest, expected revision, idempotency key, run, an `agent:` runner and a channel (cli, mcp, ci, conductor) — submitted by `pose action fulfil` (flags or `--envelope`) or the MCP tool `pose_action_fulfil`, and by `pose delegate review --request` locally. The engine reads everything else from the run — its kind, bundle, brief, agent, completion and containment, its signed proof, the attempt ledger, and the conclusion it recorded at the request's destination — and records a `fulfilled` event; it verifies it again inside the request's lock, and again on read while the request's subject is current. A rejecting run fulfils without approving. The maintainer chose this on 2026-10-28
- Rationale: one verifier for every runner keeps the local, CI and Harne8 paths equal; nothing the runner states decides the outcome
- Consequences: a fulfilment that no longer verifies satisfies nothing but never makes a journal unreadable; the event's header must say what its payload says; once the subject moves on a fulfilled request, like an open one, satisfies nothing — it stays visible as history and the new bundle is asked with a new request (review rounds 1 and 2). Such a request is retired under the action lifecycle's authority rules: superseded by the new request, or by the person's escalation when the plan now reserves the review for a person; when that escalation is already asked, it is waived, as an answer that no longer satisfies is (review rounds 3 to 6). Its wait ended at the fulfilment the engine counted
- Falsifier: a fulfilment recorded or counted for a run the engine cannot verify against the request

### Decision D3
- Basis: R3, R9
- Minimal option: keep run proofs bound to the reader's worktree and let only the runner's checkout verify
- Selected option: a run proof verifies in any checkout of the project: the origin worktree is recorded in the signed provenance, not required of the reader (the amendment of 2026-10-28 to ADR `2026-10-25-bind-delegated-review-authority-to-native-signed-run-proofs`). The maintainer chose this on 2026-10-28
- Rationale: a run a CI job or the Conductor performs in its own checkout could not fulfil a request, and a run-backed attestation failed in every other clone
- Consequences: what a proof binds otherwise — project, audience, sealed scope, kind, bundle and commit, adapter, brief, output and transcript — is unchanged and portable
- Falsifier: a run proof of one project or bundle that verifies for another

### Decision D4
- Basis: R7
- Minimal option: move R7 to `pose-delegated-review-capability`
- Selected option: the minimum here — a run never satisfies a review the plan reserves for a person (`mandatory-human`), and each exception opens one decision request to the maintainer role with the question, options and their consequences, context and the request's targets and effects: a plan reserving the review for a person (asked when the request is opened), a spent attempt budget (asked when `pose delegate --request` is refused for it), delegated reviewers in conflict on the bundle and a `business_decision` the reviewer raised in its draft (asked when the request is fulfilled). Opening one again returns the one open. The capability's question format refines this later without breaking it. The maintainer chose this on 2026-10-28
- Rationale: a person by exception, through policy, with the question ready, is the ADR's contract
- Consequences: the draft gains an optional `business_decision`; escalations are ordinary decision requests
- Falsifier: an agent's run satisfying a review reserved for a person, or an exception that reaches nobody

## 6. Validation

### Strategy

Fixture: a request fulfilled by a local run is satisfied; one answered without a run, or with a run for another brief, stays unsatisfied.

### Requirement trace

- R1 [satisfied] The typed request opens from the current sealed bundle with brief digest and accepted agents: test:TestReviewRequestIsFulfilledOverTheCLI test:TestDelegatedReviewMaterialIsInTheDigest evidence:integration check:pose-mcp/go/delegation-surface-integration check:pose-mcp/go/test
- R2 [satisfied] Only a verified matching run fulfils the service; wrong briefs, incomplete runs and actor answers are refused: test:TestDelegatedReviewRequestIsFulfilledByAVerifiedRun test:TestDelegatedReviewRequestRefusesOtherBriefsAndRefusedConclusions evidence:integration check:pose-mcp/go/delegation-surface-integration
- R3 [satisfied] Local delegate and the documented synthetic CI job transfer verified records between checkouts with a shared project identity; Conductor is a documented MCP boundary: test:TestDelegatedReviewRequestIsFulfilledByACIRunner test:TestDelegatedReviewCIJourneyAsDocumented test:TestToolsCall_ActionFulfil evidence:integration check:pose-mcp/go/delegation-surface-integration check:pose-mcp/go/mcp-action-resolve-integration
- R4 [satisfied] Review material, scope, targets, kind and policy bind to the request digest while other kinds retain their legacy digest: test:TestDelegatedReviewMaterialIsInTheDigest test:TestFulfillmentHeaderIsBound evidence:integration check:pose-mcp/go/delegation-surface-integration check:pose-mcp/go/test
- R5 [satisfied] Run proof, ledger, containment, conclusion and destination verify at admission, locked append and read; forged journal events satisfy nothing: test:TestDelegatedReviewRequestRefusesOtherRuns test:TestForgedFulfillmentsSatisfyNothing test:TestDelegatedReviewFulfillmentOnRead evidence:integration check:pose-mcp/go/delegation-surface-integration check:pose-mcp/go/test
- R6 [satisfied] A verified rejection fulfils the requested service without granting approval; the signed effect supplies its decision: test:TestDelegatedReviewRequestIsFulfilledByAVerifiedRun test:TestFulfillmentReadsTheSignedDecision evidence:integration check:pose-mcp/go/delegation-surface-integration check:pose-mcp/go/test
- R7 [satisfied] Mandatory person, budget, conflict and business exceptions create authority-bound decisions with stable question, targets and effects; unreadable inventories fail closed: test:TestDelegatedReviewRequestEscalatesToAPerson test:TestEscalationsKeepTheirTargetsAndEffects test:TestReplacingWithAnUnreadableInventoryIsRefused evidence:integration check:pose-mcp/go/delegation-surface-integration check:pose-mcp/go/test
- R8 [satisfied] Material changes invalidate satisfaction; exact replay is a no-op and authorized replacement or stale-service waiver retires the predecessor: test:TestReviewRequestReplacementReplays test:TestAStaleFulfilledRequestIsReplaced test:TestFulfilledPredecessorWithAPersonAlreadyAsked evidence:integration check:pose-mcp/go/delegation-surface-integration check:pose-mcp/go/action-lifecycle-authority-integration check:pose-mcp/go/test
- R9 [satisfied] CLI, CI and MCP submit the same canonical envelope to one verifier; portability retains project, audience and bundle boundaries: test:TestReviewRequestIsFulfilledOverTheCLI test:TestDelegatedReviewCIJourneyAsDocumented test:TestToolsCall_ActionFulfil evidence:integration check:pose-mcp/go/delegation-surface-integration check:pose-mcp/go/mcp-action-resolve-integration
- R10 [satisfied] Replay and incomplete-run recovery preserve one effect and the attempt budget; lifecycle authority remains required: test:TestDelegatedReviewRequestRevalidatesAndRefusesIncompleteRuns test:TestReviewRequestReplacementReplays test:TestReplacingWithAnUnreadableInventoryIsRefused evidence:integration check:pose-mcp/go/delegation-surface-integration check:pose-mcp/go/action-lifecycle-authority-integration check:pose-mcp/go/test
- R11 [satisfied] Per-kind destinations, rejecting outcomes, exceptions and stale checkpoints resume through native surfaces; the fulfilled predecessor with an existing person decision reaches zero closeout restrictions: test:TestDelegatedReviewRequestPerKind test:TestAStaleFulfilledRequestIsReplaced test:TestFulfilledPredecessorWithAPersonAlreadyAsked test:TestDelegatedReviewCIJourneyAsDocumented evidence:integration check:pose-mcp/go/delegation-surface-integration check:pose-mcp/go/test

## 7. Final Report

### Delivered scope

Delivered delegated-review action requests, verified fulfillment through one canonical CLI/MCP envelope, portable native run proofs, per-kind conclusion destinations and authority-bound escalation, supersession and stale-service waiver. A verified rejection fulfills the requested service without approving the scope. Historical fulfillments cannot satisfy a changed subject.

The local runner, CLI/MCP and documented synthetic CI journey are exercised by native integration tests. The Conductor boundary is documented; this delivery does not assert a live Conductor deployment or close the later capability, parity and adoption milestones.

### Validation executed

Independent round 8 reviewed implementation `b17960c835c9659d4b14f8b04e53c51dcc71d36d` on 2026-10-28, in a new Codex/OpenAI execution (`agent:independent-codex-fulfillment-round8`) distinct from the original Claude/Anthropic implementation. The review confirmed that R7-F01 is resolved: the native requirement parser recognizes eleven requirements and zero orphan trace entries, and all committed text outside the eight declaration prefixes is unchanged. The behavioral review and mutation evidence from round 7 concern identical production code and were read as historical context, not represented as new executions.

Current `pose validate --strict` passed 158/158 checks. The explicit component validation, artifact-check and surface-check passed; the latter observes containing-module integration coverage as inferred coverage, with a separate native synthetic CI journey. Sealed bundle `rvb-f65774ef08fcf69c` and attestation `rva-39b329bab795aac0` record the explicit judgments and tool evidence. Review context: knowledge:action-answer-proof-and-historical-time; knowledge:action-recovery-intent-and-refusal. `pose review verify`, `review-check`, native `close`, strict spec lint and strict structure checks were required for this transition.

### Residual risks

Native run proofs are signed executor declarations, not independent vendor receipts. The review identity and original lead are declared; this instance records unsigned attestations. Windows/Darwin runtime and a real Conductor deployment are not claimed by this closeout.

The first fresh validation passed 157/158: a temporary Git status failure (exit 128) in `TestReviewCheckNamesANegativeVerdictOnAClosedScope` interrupted the unrelated retention fixture. The same test passed in the complete suite, the isolated check passed five consecutive repetitions, and the subsequent full native validation passed. Its cause remains undetermined; no code fix is claimed. The failed attempt and successful rerun are preserved in the local round-8 review logs.

### Follow-ups

No new actionable product finding was identified in the final review. Existing work on capability composition, parity and adoption remains in its separately governed roadmap members.
