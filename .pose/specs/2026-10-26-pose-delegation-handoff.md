---
slug: pose-delegation-handoff
status: done
created_at: 2026-10-26
completed_at: 2026-10-27
supersedes:
depends_on: pose-delegation-routing, pose-delegated-review-completion-signal
priority: 2
components: pose-mcp
task_type: feature
delivers: capability:delegation-handoff
---

# Spec: A stopped delegated run can continue with another agent

## 1. Intent

### Goal

Let a stopped delegated run continue with another agent in the same attempt, reusing what the stopped run did, and offer the person or agent who started it the choice between waiting, handing off and cancelling.

### Business value

Decided by the [delegation ADR](../adr/2026-10-26-delegation-is-a-first-class-execution-surface.md). `pose review resume` continues only the same agent; when a vendor hits its usage limit for hours, the only alternative is a new run that starts from nothing. The run-continuity ADR made a vendor change explicit and recorded; this spec gives it a command and keeps the work.

### Constraints

A partial output is never a review. The handoff run is a continuation of the same attempt and shares its budget. The new agent's independence is computed against the implementation, never against the stopped agent.

### Non-goals

Continuing the same session (`resume`, delivered); session reuse across rounds (`pose-delegated-review-round-protocol`).

## 2. Requirements

### Functional

- R1: `pose delegate handoff <run> --to <agent|auto>` shall accept a run that is `interrupted`, `stalled`, or `failed` without a complete conclusion, and shall refuse a completed run.
- R2: The handoff run shall belong to the same attempt and thread as its predecessor and shall consume the attempt's continuation budget (`resume.max_resumes`, `max_wall_time`), so alternating agents cannot extend a run without bound.
- R3: The handoff brief shall be the predecessor's generated brief followed by a labelled section carrying the predecessor's partial final message and the commands it ran with their exit status, taken from the normalized events, each with its digest, and an instruction to re-verify anything the new run relies on.
- R4: The run record shall bind the base brief digest and the handoff section digest, and provenance shall sign both when signing is adopted.
- R5: The run record shall name its predecessor in `hands_off_from`; `pose delegate watch` shall replay the chain with a distinct marker at each handoff, beside the marker for a resume.
- R6: The conclusion of a handoff run shall come only from that run; the predecessor's partial shall never be accepted as a review, a verdict or evidence.
- R7: `--to auto` shall route as `pose-delegation-routing` requires, excluding agents that are currently unavailable.
- R8: When a foreground `pose delegate` run stops interrupted in a terminal, it shall offer waiting for the reset (when the reset is known and the agent declares `resume`), handing off to the first available agent, or cancelling, and shall do nothing until answered.
- R9: Outside a terminal, and in `ls` and `wait` with `--json`, an interrupted run shall carry `next_actions`: each action with its command, whether it is available now, and why not.
- R10: After a handoff to another vendor, the thread's current session shall be the handoff run's session, so a later round continues the new vendor's session.

### Non-functional

- Building the handoff brief reads only the predecessor's recorded artifacts; it starts no process.

### Security

- The handoff section carries no session reference, no environment value and nothing outside the predecessor's recorded transcript and output.
- The size of the handoff section is bounded and truncation is disclosed in it.

### Compatibility

- Runs recorded before this spec can be handed off when their transcript is intact; they simply have no normalized commands, and the section says so.

## 3. Technical Plan

### Affected areas

Continuation planner, brief renderer (handoff section), run record (`hands_off_from`, handoff digest), watch renderer, terminal prompt, `next_actions` in JSON outputs.

### Artifacts

- modified: .pose/specs/2026-10-26-pose-delegation-handoff.md
- created: .pose/starts/pose-delegation-handoff.json
- created: .pose/changelogs/unreleased/pose-delegation-handoff.md
- created: pose-mcp/internal/pose/delegation_handoff.go
- created: pose-mcp/internal/pose/delegation_handoff_test.go
- created: pose-mcp/internal/pose/delegation_handoff_legacy.go
- modified: pose-mcp/internal/pose/review_dispatch.go
- modified: pose-mcp/internal/pose/review_run_continuity.go
- modified: pose-mcp/internal/pose/review_run_provenance.go
- modified: pose-mcp/internal/pose/review_attempt_ledger.go
- modified: pose-mcp/internal/pose/delegation_events.go
- modified: pose-mcp/internal/pose/delegation_live.go
- created: pose-mcp/internal/cli/delegate_handoff.go
- created: pose-mcp/internal/cli/delegate_handoff_test.go
- modified: pose-mcp/internal/cli/delegate.go
- modified: pose-mcp/internal/cli/delegate_wait.go
- modified: pose-mcp/internal/cli/delegate_watch.go
- modified: pose-mcp/internal/cli/delegate_test.go
- modified: pose-mcp/internal/cli/help_catalog.go
- modified: POSE.md
- modified: locales/pt-BR/POSE.md
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md

### Delivery targets

- capability:delegation-handoff module:pose-mcp profile:composed-capability entrypoint:pose-mcp/internal/cli/delegate_handoff.go

### API/contract changes

`pose delegate handoff`; run record field `hands_off_from`; `next_actions` in `ls` and `wait` JSON.

## 4. Tasks

- [x] Continuation of another agent under the same attempt and budget
- [x] Handoff brief section with digests
- [x] Lineage in the record and in watch
- [x] Terminal choices and `next_actions`
- [ ] Thread session update on vendor change: deferred to `pose-delegated-review-round-protocol` (D3)
- [x] Independent review and closeout

## 5. Decisions

### Decision D1
- Basis: R2
- Minimal option: a handoff opens a new attempt
- Selected option: a continuation of the same attempt sharing its budget
- Rationale: a new attempt would let alternating agents bypass `max_runs_per_bundle` and turn a rejection-free history into repeated tries
- Consequences: a long stop may exhaust the budget and escalate as the capability spec requires. The budget's ceiling is read from the agent that opened the attempt, for resumes and handoffs alike, so a successor that declares a larger budget or none does not widen it (review round 1, B01)
- Falsifier: a chain of handoffs that runs longer than the attempt's continuation budget

### Decision D2
- Basis: R3, R6
- Minimal option: hand the new agent the stopped agent's whole transcript
- Selected option: the partial final message and the executed commands with exit status, labelled and bounded
- Rationale: the new reviewer gains the ground already covered without inheriting the stopped reviewer's reasoning wholesale, and re-verifies what it relies on
- Consequences: some reasoning is redone
- Falsifier: a handoff review that cites the predecessor's partial as its evidence

### Decision D3
- Basis: R10
- Minimal option: record a thread session on the run now
- Selected option: leave R10 to `pose-delegated-review-round-protocol`, which introduces the thread and its current session; a handoff run captures its own session as every run does, so the protocol can name it the thread's
- Rationale: no thread exists in the engine yet; a field nothing reads would be a contract with no consumer
- Consequences: R10 is delivered with the round protocol, and the trace records it as deferred, not satisfied
- Falsifier: a round after a vendor handoff that resumes the stopped vendor's session

### Decision D4
- Basis: R1, R2, R5
- Minimal option: let a handoff continue only an interrupted run, and keep the reservation's agent for every run of the attempt
- Selected option: a handoff continues an interrupted or failed run, claims it as a resume does, and a run is continued once by either; the attempt reservation names the first agent, and each continuation's agent is checked against its own predecessor (a resume keeps it, a handoff changes it)
- Rationale: one claim per run keeps resume and handoff mutually exclusive without a second ledger; a failed run is as unfinished as an interrupted one
- Consequences: `resume` refuses a run that was handed off, naming its continuation; a run stalled without a record has nothing to hand off and is refused
- Falsifier: one run continued twice, or a continuation whose agent no chain check compares

## 6. Validation

### Strategy

A fake agent stops on a recorded limit; a handoff to a second fake agent runs under the same attempt with the expected brief section and digests, and its conclusion alone is recorded; a chain of handoffs stops at the budget; a completed run is refused; a terminal session offers the three choices and a non-terminal run returns `next_actions`; watch shows both markers in order.

### Requirement trace

- R1 [satisfied] Interrupted/failed handoffs and completed/same-agent refusals verified by test:TestDelegateHandoffRefusesAndIsBounded and check:delegation-surface-integration. A stall without a record is refused under D4; no orphan continuation is started.
- R2 [satisfied] Shared attempt identity and the opening agent's continuation ceiling verified by test:TestAttemptContinuationBudgetIsTheOpeningAgents and test:TestDelegateHandoffRefusesAndIsBounded; resume/handoff share the exclusive predecessor claim and accumulated wall-clock budget.
- R3 [satisfied] Recorded partial and normalized commands, conservative call correlation, quoting and disclosed truncation verified by test:TestHandoffSectionQuotesAndBounds and test:TestHandoffCommandsPairOnlyWhatIsKnown.
- R4 [satisfied] Base/section/wire digests and recorded renderer verified by test:TestHandoffRunProvenanceVerifies and test:TestHandoffSectionVerifiesByRecordedRenderer. Literal proofs from eb4ad73, 341aa5e and 0ac1cb6 each verify in all five independent upgrade scenarios.
- R5 [satisfied] hands_off_from, receipts and distinct watch markers verified by test:TestDelegateHandoffContinuesTheAttemptWithAnotherAgent and check:delegated-run-continuity-integration; independent probes replay both mixed-chain orders without history findings.
- R6 [satisfied] Quoted partial never supplies the successor's decision, as verified by test:TestHandoffSectionQuotesAndBounds and test:TestDelegateHandoffContinuesTheAttemptWithAnotherAgent; only the successor's own rejecting output determines wait exit 10.
- R7 [satisfied] Auto routing, availability and recorded-continuation retry verified by test:TestDelegateStoppedRunsOfferTheirWaysForward and test:TestDelegateHandoffAutoRetryReturnsTheContinuation.
- R8 [satisfied] Initial and repeated terminal choices, including cancel alone, verified by test:TestDelegateStoppedRunActionsAreExecutable; no action starts before the answer.
- R9 [satisfied] JSON and plain stopped-command results plus ls/wait actions, unavailable reasons and cancel semantics verified by test:TestDelegateStoppedRunActionsAreExecutable and check:delegation-surface-integration.
- R10 [deferred-integration: spec:pose-delegated-review-round-protocol] D3 defers the thread's current-session update and later-round session reuse to the round protocol. Each handoff captures its own session, but that integration is not claimed satisfied or executed here.

## 7. Final Report

### Delivered scope

Milestone `continuity` of roadmap `delegation`: cross-agent handoff of an interrupted or failed run under the same attempt, exclusive continuation claim and opening-agent budget; quoted bounded recorded partial and commands; lineage and renderer version in the run; reconstruction of immutable legacy proofs; terminal choices, machine-readable next_actions and mixed-chain watch replay. A stall without a record is refused under D4. R10 remains deferred to `pose-delegated-review-round-protocol` under D3, not satisfied.

### Independent review and evidence

Maintainer-authorized reviewer `agent:independent-gpt-6.1-sol-review` (Codex/OpenAI), another vendor than the implementer Claude/Anthropic, Claude Opus 5.5 session `f2a4d9f2-d557-4064-ae96-6ed52bb47cd2`. The `agent:independent-` prefix is a declaration required by the engine, not authenticated separation. Identity is declared and this instance's attestations are unsigned.

Bundle `rvb-7850daa8171184bc`, attestation `rva-19adaa30e9ae9f12`, approved over implemented commit `94e9b700a9de7fa2fbcbcbfa01c6a1ea392aee86`. Sealed explicitly with `--lead anthropic/claude-opus-5-5` from the reviewing Codex environment; the warning that no Anthropic session reference is available is expected, and no implementation execution digest was invented.

B01-B09 and F01-F03 are reconciled as fixed, including the unversioned conservative renderer introduced by 0ac1cb6. The complete review reports and independent probes are retained locally under `.review-work/round1/` through `.review-work/round5/` (deliberately untracked). The unrelated documentary commit 841fc2b was excluded from the implementation review.

### Validation executed

`pose validate --strict`: 158/158 checks passed, no failures, errors or skips; structured evidence is `.pose/results/delivery-validation.json`. Prior independent vet, Windows/Darwin builds, both locales, clean-session full suite and selected race checks passed. Current independent probes pass with -race, including claim exclusivity, mixed lineage, version isolation, forged/unknown-version refusals, stopped-run choices and all reconciled findings. Literal immutable proofs from eb4ad73, 341aa5e and 0ac1cb6 verify 5/5 each against the final binary. Earlier production omission mutations, including the always-current renderer mutation, failed the relevant regression tests without editing tracked code or tests.

Native artifact reconciliation, capability composition, rule selection, design/discovery/integration/debt assessment, approved review verification and completion gates were executed; their exact outputs are recorded in the local round-5 report. The roadmap remains nonterminal because subsequent milestones are not delivered by this spec.

### Residual risks

Identity and cognitive independence remain declarations, not vendor-authenticated receipts. Historical renderer selection is restricted to immutable unversioned runs and a matching bound digest; new runs record version 2 and never fall back to unsafe legacy pairing. The partial still requires independent re-verification and never becomes gate evidence on its own. R10's thread/session integration remains deferred under D3.

The integration assessment retains preexisting unobserved external consumers; the debt assessment retains incomplete coverage for two invalid historical specs and does not certify global absence of debt. No general task execution, thread protocol or external consumer adoption is claimed.

### Follow-ups

No open follow-ups remain for this spec. The round-protocol integration of R10 is the explicit D3 deferral to the existing spec, not a claim of delivered session reuse.
