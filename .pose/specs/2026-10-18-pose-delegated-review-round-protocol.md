---
slug: pose-delegated-review-round-protocol
status: in-progress
created_at: 2026-10-18
completed_at:
supersedes:
depends_on: pose-delegated-review-attempt-ledger, pose-delegated-review-brief, pose-delegation-handoff
priority: 2
components: pose-mcp
task_type: feature
changelog:
delivers: capability:delegated-review-rounds
---

# Spec: Each review round starts from the findings of the last one

## 1. Intent

### Goal

Make the rounds of a delegated review one process: the brief of a round after the first carries the previous round's findings, the reviewer must account for each of them and sweep the subject in one pass, and the engine measures whether the rounds converge.

### Business value

Origin: B06 (2026-10-17), the largest consolidation unit so far. Four review rounds returned 19, 12, 10 and 6 findings. Each round reviewed the new commit afresh, so a finding "partially fixed" came back as a new finding two rounds later, and the reviewer stopped at the first medium finding each time, handing the rest to the next round. Nothing in the record says which findings of round N were verified fixed in round N+1, how many rounds a scope has taken, or whether they converge. That is exactly the information a maintainer needs to decide between another round, a different reviewer and a human adjudication.

Amended on 2026-10-26 by the [delegation ADR](../adr/2026-10-26-delegation-is-a-first-class-execution-surface.md): the rounds of a scope form a thread, and by default a round after a rejection continues the thread's session instead of starting a cold one (R6, R9, D1).

### Constraints

The brief remains generated from the sealed bundle and the ledger, not written by the implementer; prior findings come from the ledger, the implementer's text stays in its labelled notes. A continued session receives the same generated brief as a cold one, so the safeguards against anchoring do not depend on the session's memory.

### Non-goals

Resuming an interrupted run (`pose-delegated-review-run-continuity`); deciding the verdict; merging findings across unrelated scopes.

## 2. Requirements

### Functional

- R1: The round of a bundle shall be derived from the supersession chain of its scope in the ledger (round 1 for a bundle that supersedes none) and shall be shown by `pose review verify`, `status` and insights.
- R2: For a bundle of round 2 or later, `pose review brief` shall add a section built mechanically from the previous round's completed run: every finding with its identifier, severity, disposition and evidence digest, and the commit it was made against. Text written by the implementer shall stay in its labelled notes.
- R3: The brief shall instruct the reviewer to state for each prior finding whether it is `fixed`, `partial`, `not-fixed` or `superseded`, with evidence, before reporting anything new, and to report every finding it can establish in this round, with a completeness statement naming what it swept.
- R4: A draft for a round after the first shall carry `prior_findings` (one entry per prior finding) and `sweep`; a draft that omits a prior finding that was open, or whose `fixed` entry cites no evidence, shall be invalid and shall not be accepted as an approval.
- R5: A prior finding reported `partial` or `not-fixed` shall keep its identifier in the next brief, so one defect has one identity across rounds.
- R6: When `routing.rounds` is `continue-thread` (the default), a round after the first shall resume the thread's current session; it shall open a new session, recording the reason, when the agent's vendor changes, the supersession chain is broken, the session can no longer be resumed, or the scope reaches the round threshold. When `routing.rounds` is `cold-session`, every round shall open a new session. The run shall record the session lineage and `pose review verify` shall disclose it.
- R7: `pose insights` shall report, per scope, findings per round, the share of prior findings fixed, partial and not-fixed, and rounds to approval; `pose review verify` shall warn when a scope passes the policy's round threshold (default 5), recommending a different agent or a human adjudication.
- R8: A scope whose rounds do not converge shall not be hidden by the round protocol: a rejection stays visible as the ledger requires, and the protocol shall never lower the number of rounds recorded.
- R9: A continued session shall receive the full generated brief of its round, including the prior-findings section of R2, and its draft shall satisfy R3 and R4 exactly as a cold session's draft must; remembering an earlier round shall never replace accounting for it.
- R10: `pose insights` shall report the R7 figures separately for continued and cold rounds, so the default of R6 can be judged on measured convergence.

### Non-functional

- The prior-findings section is byte-stable for the same ledger state.

### Security

- Prior findings are carried by reference to the ledger digest; a tampered prior run makes the brief refuse.

### Compatibility

- Round 1 briefs are unchanged; drafts of round 1 need neither new field.

## 3. Technical Plan

### Affected areas

Brief generator, draft schema and validation, supersession chain, verify disclosure, insights.

### Artifacts

- modified: .pose/specs/2026-10-18-pose-delegated-review-round-protocol.md
- created: .pose/starts/pose-delegated-review-round-protocol.json
- created: .pose/changelogs/unreleased/pose-delegated-review-round-protocol.md
- created: pose-mcp/internal/pose/review_rounds.go
- created: pose-mcp/internal/pose/review_rounds_test.go
- modified: pose-mcp/internal/pose/review_brief.go
- modified: pose-mcp/internal/pose/review_brief_test.go
- modified: pose-mcp/internal/pose/review_dispatch.go
- modified: pose-mcp/internal/pose/review_run_provenance.go
- modified: pose-mcp/internal/pose/review_run_continuity.go
- modified: pose-mcp/internal/pose/delegation_handoff.go
- modified: pose-mcp/internal/pose/delegation_policy.go
- modified: pose-mcp/internal/pose/delegation_list.go
- modified: pose-mcp/internal/pose/delegation_wait.go
- modified: pose-mcp/internal/pose/review_attempt_ledger.go
- modified: pose-mcp/internal/pose/review_bundle.go
- created: pose-mcp/internal/cli/delegate_rounds_test.go
- modified: pose-mcp/internal/cli/review_closeout.go
- modified: pose-mcp/internal/cli/delegate.go
- modified: pose-mcp/internal/cli/insights.go
- modified: pose-mcp/internal/cli/help_catalog.go
- modified: POSE.md
- modified: locales/pt-BR/POSE.md
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md

### Delivery targets

- capability:delegated-review-rounds module:pose-mcp profile:composed-capability entrypoint:pose-mcp/internal/pose/review_rounds.go

### API/contract changes

Brief gains a prior-findings section; the draft gains `prior_findings` and `sweep`; `delegation.json` gains `routing.rounds` (`continue-thread` or `cold-session`) and a round threshold.

## 4. Tasks

- [x] Round derivation and disclosure
- [x] Prior-findings section of the brief
- [x] Draft schema and validation
- [x] Thread session continuation, forced new sessions and lineage
- [x] Insights split between continued and cold rounds
- [x] Insights and the round threshold
- [ ] Independent review and closeout

## 5. Decisions

### Decision D1
- Basis: R6
- Minimal option: a new session for every round
- Selected option: continue the thread's session by default (`continue-thread`), with `cold-session` available by policy and a new session forced on a vendor change, a broken chain, an unresumable session or the round threshold
- Rationale: the maintainer's decision of 2026-10-26 in the delegation ADR. A reviewer that already read the subject verifies its own findings without a full re-read, and the risk that it anchors on its earlier verdict is carried by R2–R5 and R9, which make every prior finding an explicit, evidenced entry whatever the session remembers. The original choice of a cold session is kept as the policy alternative
- Consequences: a continued round costs less than a full read; the comparison of R10 decides whether the default holds
- Falsifier: insights showing continued rounds accept more prior findings as `fixed` that a later round or a cold reviewer finds still open, or converge no faster than cold rounds

### Decision D2
- Basis: R3, R4
- Minimal option: ask in prose for an exhaustive review
- Selected option: a draft that does not account for each prior finding and does not state its sweep is invalid
- Rationale: the instruction was given in prose in round 2 of B06 and the next round still returned new medium findings of the same class
- Consequences: a reviewer that stops early produces an invalid draft, not a quiet approval
- Falsifier: an invalid draft accepted, or a valid draft that omitted an open finding

### Decision D3
- Basis: R2, R4
- Minimal option: read findings from the draft's prose
- Selected option: the draft is the fenced code block whose info string is `pose-draft`, at the end of the reviewer's final message: JSON with schema `pose.review.draft/v1`, `decision`, `findings` (`id`, `severity`, `disposition`, `file`, `line`, `summary`) and, from round 2, `prior_findings` and `sweep`, decoded strictly; the brief of every review round asks for it (template `review-brief/2`), and a decision line that contradicts the block makes the draft invalid. Where there is a block, its decision is the run's decision for every reader (wait, insights, verify, run provenance); the decision line is read only for drafts without one. A message carries one block; a finding's own identifier never takes the reserved round prefix `R<n>-`, so the identity a finding is carried under is injective. The maintainer chose this format on 2026-10-27, for this spec and for `pose-delegated-review-conclusions`
- Rationale: an identity kept across rounds needs findings the engine can read without interpreting prose; the conclusions phase needs the same structure
- Consequences: round 1 briefs change, against the compatibility note "Round 1 briefs are unchanged": they ask for the block so that round 2 has identified findings to carry. A round-1 draft without the block is still valid; it only leaves the next round with nothing to carry, which the next brief says. Runs briefed with `review-brief/1` keep verifying: the brief a run was sent is re-rendered with the template whose digest it recorded
- Falsifier: a finding of round N absent from round N+1's brief while the draft of round N reported it

### Decision D4
- Basis: R1, R5
- Minimal option: order the bundles of a scope by their seal time
- Selected option: a round is a sealed bundle of the scope a completed, unsuperseded review run concluded on; the rounds before a bundle are those whose concluding run ended no later than the second the bundle was sealed in, ordered by when those runs ended. A finding first reported in round N is identified `R<N>-<id>`; it leaves the open set only when a later draft reports it `fixed` with evidence, or `superseded`: unaccounted, reported `fixed` without evidence, `partial` or `not-fixed`, it stays open under that identifier, so an invalid account never closes a finding. A draft of the chain that cannot be read closes nothing and adds nothing; the brief says so and the next round opens a new session. A loss in the run history or in the sealed bundles that may concern the scope, a run of the scope whose bundle is missing, or a previous round whose record is not intact make the derivation, and so the brief, refuse; the earlier reviewers' text is quoted in the brief as data. The number of rounds a metric reports is the highest any run of the scope recorded, and a run records its round, and a continuation its predecessor's thread, from the start
- Rationale: seals have second precision and runs nanoseconds, so two seals in one second cannot order rounds; a bundle is sealed after the rejection it answers
- Consequences: a reseal of the same content is not a new round; a bundle a review never concluded on is not a round
- Falsifier: two rounds in the wrong order, or a carried finding under a new identifier

### Decision D5
- Basis: R6
- Minimal option: continue the session whenever the vendor is the same
- Selected option: continue the session the previous round's concluding run ended with only when the same agent runs the next round; another agent of the same vendor opens a new session, as a vendor change, a broken chain, an unresumable session and the threshold do. The threshold applies from the round that reaches it (`routing.round_threshold`, 5 by default). After a handoff the session the handoff run captured is the one a later round continues, which delivers R10 of `pose-delegation-handoff`
- Rationale: a session belongs to the CLI profile that opened it; resuming it from another profile of the same vendor could run another model or another configuration against it
- Consequences: changing agents always costs a full read
- Falsifier: a round that resumed a session another agent opened

## 6. Validation

### Strategy

A fixture scope with three seeded defects reviewed over three rounds by a fake adapter: round 2 must account for each prior finding, a draft that omits one is refused, a `partial` keeps its identity, the section is byte-stable, rounds 2 and 3 continue the thread's session and are disclosed, a vendor change and the round threshold each open a new session with its reason, `cold-session` opens one every round, a continued session's draft that omits a prior finding is refused like a cold one, and insights report the convergence figures for continued and cold rounds; a tampered prior run makes the brief refuse.

### Requirement trace

## 7. Final Report

### Delivered scope

Not started: opened on 2026-10-18 under roadmap `delegated-review` from the operating experience of the B06 review rounds (see Origin); moved on 2026-10-26 to milestone `rounds` of roadmap `delegation`.

### Residual risks

None yet.

### Follow-ups
