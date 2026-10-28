---
slug: pose-delegated-review-conclusions
status: done
created_at: 2026-10-27
completed_at: 2026-10-28
supersedes:
depends_on: pose-delegated-review-round-protocol, pose-delegated-review-run-provenance, pose-delegated-review-attempt-ledger, pose-delegated-adapter-containment, pose-review-approval-domain-preflight, pose-delegation-routing
priority: 3
components: pose-mcp
task_type: feature
delivers: capability:delegated-review-conclusions
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
- R2: An `adjudication` or `smoke` run that passes the same checks shall record its verdict where its origin expects it (the adjudicated spec's decision record, the smoke target's evidence), under the same rules. Delivered by [`pose-delegated-verdict-destinations`](2026-10-27-pose-delegated-verdict-destinations.md): no such destination exists yet, and defining them is a feature of its own (decision D2, the maintainer's choice of 2026-10-27).
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
- created: .pose/specs/2026-10-27-pose-delegated-verdict-destinations.md
- modified: .pose/roadmaps/delegation.md
- created: .pose/starts/pose-delegated-review-conclusions.json
- created: .pose/changelogs/unreleased/pose-delegated-review-conclusions.md
- created: pose-mcp/internal/pose/review_conclusions.go
- created: pose-mcp/internal/pose/review_conclusions_test.go
- modified: pose-mcp/internal/pose/review_rounds.go
- modified: pose-mcp/internal/pose/review_brief.go
- modified: pose-mcp/internal/pose/review_brief_test.go
- modified: pose-mcp/internal/pose/delegation_policy.go
- modified: pose-mcp/internal/pose/review_bundle.go
- modified: pose-mcp/internal/pose/spec_transfer_locks.go
- modified: pose-mcp/internal/pose/spec_transfer_lock_other.go
- modified: pose-mcp/internal/pose/review_run_provenance.go
- modified: pose-mcp/internal/pose/review_run_provenance_test.go
- modified: pose-mcp/internal/cli/review_run_provenance_test.go
- modified: pose-mcp/internal/mcpserver/review_run_provenance_test.go
- created: pose-mcp/internal/cli/review_conclude.go
- created: pose-mcp/internal/cli/review_conclude_test.go
- modified: pose-mcp/internal/cli/review_closeout.go
- modified: pose-mcp/internal/cli/delegate.go
- modified: pose-mcp/internal/cli/delegate_handoff.go
- modified: pose-mcp/internal/cli/help_catalog.go
- modified: POSE.md
- modified: locales/pt-BR/POSE.md
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md

### Delivery targets

- capability:delegated-review-conclusions module:pose-mcp profile:composed-capability entrypoint:pose-mcp/internal/cli/review_conclude.go

## 4. Tasks

- [x] Review conclusion recorded as the attestation
- [ ] Adjudication and smoke destinations: moved to `pose-delegated-verdict-destinations` (D2)
- [x] Refusals, idempotence and supersession
- [x] Independent review and native closeout: round 4 approved implementation 153d54bb2690bbe6b96d8da9760b1f97b6502728; bundle rvb-07212529dbde2aa2, attestation rva-ac02fc0441f45356.

## 5. Decisions

Phase B19 as a spec of its own is recorded by the amendment of 2026-10-27 to the delegation ADR, by the maintainer's choice of 2026-10-27.

### Decision D1
- Basis: R1
- Minimal option: a person runs `pose review run-sign` and `pose review attest --run --run-proof --sign` with the criteria and tools as flags
- Selected option: the conclusion is typed in the run's pose-draft block — `criteria` and `tools` beside the findings, which the brief of template `review-brief/3` asks for, and `action`, `owner` and `review_by` on a follow-up to accept it as a risk — and the engine signs the run's proof and the attestation with the native issuer the delegation policy names (`conclusions.issuer`, the key outside the project) or `--issuer`, recording it through the same domain writer and preflight as every attestation. It records on its own as a review run completes only when review policy `delegated_review` is true and the issuer is set; otherwise `pose review conclude <run>` does it on demand. The maintainer chose this on 2026-10-27
- Rationale: one writer and one preflight keep the conclusion to the bar every attestation meets; a typed conclusion leaves nothing for the engine to answer on the reviewer's behalf
- Consequences: a criterion or tool the draft does not answer is refused by the preflight, never filled in; runs briefed with `review-brief/1` and `/2` keep verifying with their template; whoever signs it, an attestation of a run briefed with `review-brief/3` states that run's conclusion — criteria, tools that state a conclusion, and findings — or verify refuses it, so the manual `review attest --run --run-proof --sign` path is bound as the engine's is (review round 1, B1)
- Falsifier: an attestation whose criterion, tool or finding the run's draft did not state

### Decision D2
- Basis: R2
- Minimal option: give adjudication and smoke verdicts a file of their own under this spec
- Selected option: move R2 to `pose-delegated-verdict-destinations`, which defines where each verdict goes (the adjudicated spec's decision record, the smoke target's surface evidence) and records it under the same rules; the maintainer chose this on 2026-10-27
- Rationale: neither destination exists, and defining them is a feature of its own; the review conclusion needs none of it
- Consequences: R2 is traced as delivered by that spec, not here
- Falsifier: an adjudication or smoke verdict recorded with no destination its origin reads


## 6. Validation

### Strategy

Fixtures for each kind: a passing run records its conclusion; each failed check, an incomplete conclusion, a replay and a superseded bundle record nothing; a rejection blocks until the bundle changes.

### Requirement trace

- R1 [satisfied] Native engine records the signed review attestation and rejecting conclusion through the shared admission: test:TestConcludeReviewRunRecordsTheAttestation test:TestConcludeReviewRunRecordsARejection check:pose-mcp/go/test commit:153d54bb2690bbe6b96d8da9760b1f97b6502728
- R2 [deferred-integration: spec:pose-delegated-verdict-destinations] Delivered by `pose-delegated-verdict-destinations` under D2; adjudication and smoke destinations remain in that draft and are not claimed implemented by this spec.
- R3 [satisfied] Invalid conclusions and unreadable records preserve refusal reasons; incomplete recovery history preserves concluding and reports losses: test:TestConcludeReviewRunRefusesWhatIsNotAConclusion test:TestConcludeReviewRunRecordsAnUnreadableRun test:TestConcludeReviewRunRefusesAnIncompleteRecovery check:pose-mcp/go/test
- R4 [satisfied] Replay, supersession, concurrent conclusion and interrupted reservation recovery preserve one governed conclusion: test:TestConcludeReviewRunRecordsTheAttestation test:TestConcludeReviewRunConcurrently test:TestConcludeReviewRunRecoversAReservation check:pose-mcp/go/test report:2026-10-28-delegated-conclusions-round4.md

## 7. Final Report

### Delivered scope

Delivered R1, R3 and R4 in implementation commit `153d54bb2690bbe6b96d8da9760b1f97b6502728`: the engine signs and records a review run's typed conclusion through the shared attestation writer and preflight, records rejecting conclusions, preserves refusal reasons, and serializes replay/recovery with a process-owned lock. Recovery requires an authenticated matching attestation and complete readable history. Explicit `review conclude` and the dual opt-in automatic delegate hook reach the production path for capability:delegated-review-conclusions. D2 assigns R2 delivery to `pose-delegated-verdict-destinations`; those draft destinations and capability adoption are not claimed delivered here.

### Independent review and evidence

Reviewer `agent:independent-gpt-6.1-sol-review`, declared Codex/OpenAI, reviewed the Claude/Anthropic implementation in a new execution. Identity is declared and the local attestation is unsigned; synthetic fixture signatures do not authenticate this review or prove cognitive independence. Knowledge consumed: knowledge:delegated-rounds-legacy-status and knowledge:module-metadata-discovery-invalidates-review-provenance.

Sealed bundle `rvb-07212529dbde2aa2`, digest `sha256:07212529dbde2aa2f8e1c04648b6b7aa6876489a9816feaefe47f439f7a64252`; plan `sha256:3abb8224d37f53956931a291e816b8fe0fedd1a5b715979873bce2d3482585d4`; attestation `rva-ac02fc0441f45356` approved all required coverage with explicit judgment and accepted low finding `f-r4-01`. Native `review verify` and `review-check` reported fresh/approved, and `close` applied the guarded transition on 2026-10-28 UTC after the requirement trace was supplied.

Current canonical strict validation passed 158/158 checks at the implemented commit. Independent full Go suites in C and pt_BR.UTF-8, vet, focused race tests, Windows/Darwin cross-builds and gitleaks passed. Reused rounds 1–3 probes include separate processes and actual SIGKILL recovery, repeated five times with one published proof and attestation per run; mutation counterexamples demonstrate the lock, reservation, typed-binding and Coverage protections. The obsolete preview/apply equality assertion remains intentionally in the historical probe source; current preview explicitly defers admission and recovery preview changes no bytes.

Strict artifact-check reconciled 28 claims with 31 attributed paths and no errors; broad orphan warnings were reviewed. Strict surface-check connected the declared capability and production entrypoint to fresh module-wide integration evidence; the separate-process probes exercise the conclusion path on Linux. The shared ADR structural fact is explicitly mapped to R1. Assessment coverage losses and seal warnings are disclosed in report:2026-10-28-delegated-conclusions-round4.md. Native results: `.pose/results/delivery-validation.json`; local probes/overlays/logs: `.review-work/round4/` (untracked).

### Residual risks

Accept the bounded low operability risk `f-r4-01` below. The unsupported-platform sentinel fails immediately, while other permanent backend errors can still wait two minutes and lose their cause as apparent contention. Failure remains closed; the injected EOPNOTSUPP probe published no proof or attestation. Linux process locking and owner death were exercised; Windows and Darwin were compiled, not run. Assessment tech-debt coverage remains incomplete for two pre-existing invalid specs, and broad integration consumer gaps are not asserted as new regressions. No production, tests, policy, matrix, capability or adoption configuration was changed by the reviewer. Handoff: knowledge:delegated-conclusions-round4.

### Follow-ups

- [open] f-r4-01: distinguish permanent filesystem/backend lock errors from contention and retain the original cause; injected EOPNOTSUPP currently waits until the deadline and reports another owner, with no governed effects. See knowledge:delegated-conclusions-round4. (owner:@oseiaspereira88 crit:low review:2026-11-27)
