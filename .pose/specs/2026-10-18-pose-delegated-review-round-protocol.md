---
slug: pose-delegated-review-round-protocol
status: draft
created_at: 2026-10-18
completed_at:
supersedes:
depends_on: pose-delegated-review-attempt-ledger, pose-delegated-review-brief
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

### Constraints

The brief remains generated from the sealed bundle and the ledger, not written by the implementer; prior findings come from the ledger, the implementer's text stays in its labelled notes. A reviewer's independence is not traded for context unless a policy says so.

### Non-goals

Resuming an interrupted run (`pose-delegated-review-run-continuity`); deciding the verdict; merging findings across unrelated scopes.

## 2. Requirements

### Functional

- R1: The round of a bundle shall be derived from the supersession chain of its scope in the ledger (round 1 for a bundle that supersedes none) and shall be shown by `pose review verify`, `status` and insights.
- R2: For a bundle of round 2 or later, `pose review brief` shall add a section built mechanically from the previous round's completed run: every finding with its identifier, severity, disposition and evidence digest, and the commit it was made against. Text written by the implementer shall stay in its labelled notes.
- R3: The brief shall instruct the reviewer to state for each prior finding whether it is `fixed`, `partial`, `not-fixed` or `superseded`, with evidence, before reporting anything new, and to report every finding it can establish in this round, with a completeness statement naming what it swept.
- R4: A draft for a round after the first shall carry `prior_findings` (one entry per prior finding) and `sweep`; a draft that omits a prior finding that was open, or whose `fixed` entry cites no evidence, shall be invalid and shall not be accepted as an approval.
- R5: A prior finding reported `partial` or `not-fixed` shall keep its identifier in the next brief, so one defect has one identity across rounds.
- R6: By default each round runs a new vendor session. An adapter may declare `continue_across_rounds`, which resumes the previous round's session when the supersession chain is unbroken and the policy allows it; the run records the session lineage and `pose review verify` discloses it.
- R7: `pose insights` shall report, per scope, findings per round, the share of prior findings fixed, partial and not-fixed, and rounds to approval; `pose review verify` shall warn when a scope passes the policy's round threshold (default 5), recommending a different adapter or a human adjudication.
- R8: A scope whose rounds do not converge shall not be hidden by the round protocol: a rejection stays visible as the ledger requires, and the protocol shall never lower the number of rounds recorded.

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

Brief gains a prior-findings section; the draft gains `prior_findings` and `sweep`; adapter gains `continue_across_rounds`; policy gains a round threshold.

## 4. Tasks

- [ ] Round derivation and disclosure
- [ ] Prior-findings section of the brief
- [ ] Draft schema and validation
- [ ] Session lineage across rounds
- [ ] Insights and the round threshold

## 5. Decisions

### Decision D1
- Basis: R6
- Minimal option: always continue the previous vendor session so the reviewer remembers
- Selected option: new session by default, continuation only when declared and allowed
- Rationale: a reviewer that remembers its earlier verdict anchors on it; the cold session with the prior findings in its brief gets the context and keeps the independence
- Consequences: a round costs the full read of the subject
- Falsifier: rounds that converge faster with a continued session and no loss of defects found, measured by insights

### Decision D2
- Basis: R3, R4
- Minimal option: ask in prose for an exhaustive review
- Selected option: a draft that does not account for each prior finding and does not state its sweep is invalid
- Rationale: the instruction was given in prose in round 2 of B06 and the next round still returned new medium findings of the same class
- Consequences: a reviewer that stops early produces an invalid draft, not a quiet approval
- Falsifier: an invalid draft accepted, or a valid draft that omitted an open finding

## 6. Validation

### Strategy

A fixture scope with three seeded defects reviewed over three rounds by a fake adapter: round 2 must account for each prior finding, a draft that omits one is refused, a `partial` keeps its identity, the section is byte-stable, a continued session is disclosed, and insights report the convergence figures; a tampered prior run makes the brief refuse.

### Requirement trace

## 7. Final Report

### Delivered scope

Not started: part of roadmap `delegated-review`, opened on 2026-10-18 from the operating experience of the B06 review rounds (see Origin).

### Residual risks

None yet.

### Follow-ups
