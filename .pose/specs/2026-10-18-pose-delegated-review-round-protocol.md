---
slug: pose-delegated-review-round-protocol
status: draft
created_at: 2026-10-18
completed_at:
supersedes:
depends_on: pose-delegated-review-attempt-ledger, pose-delegated-review-brief, pose-delegation-handoff
priority: 2
components: pose-mcp
task_type: feature
changelog:
delivers:
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

Implementation artifacts are declared when the spec starts.

### API/contract changes

Brief gains a prior-findings section; the draft gains `prior_findings` and `sweep`; `delegation.json` gains `routing.rounds` (`continue-thread` or `cold-session`) and a round threshold.

## 4. Tasks

- [ ] Round derivation and disclosure
- [ ] Prior-findings section of the brief
- [ ] Draft schema and validation
- [ ] Thread session continuation, forced new sessions and lineage
- [ ] Insights split between continued and cold rounds
- [ ] Insights and the round threshold

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
