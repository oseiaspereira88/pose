---
slug: pose-delegated-review-run-continuity
status: draft
created_at: 2026-10-18
completed_at:
supersedes:
depends_on: pose-delegated-review-attempt-ledger
priority: 2
components: pose-mcp
task_type: feature
changelog:
delivers:
---

# Spec: A delegated review that is interrupted is resumed, not restarted

## 1. Intent

### Goal

Let a delegated review that stopped before it finished (the reviewer's usage limit was reached, the process timed out or died, the host was lost) continue from where it stopped, on the same sealed commit and brief, as one attempt, instead of being restarted from nothing or mistaken for a completed review.

### Business value

Origin: the B06 consolidation reviews (2026-10-17). A review of a large diff takes tens of minutes of reviewer time. The reviewer's usage limit was reached during the programme, and the work was picked up by hand: the session was resumed with a personal launcher flag once the maintainer reported that the limit had reset, and nobody could say whether the half-finished run counted against the attempt budget or as a rerun. The engine knows none of this: `pose-delegated-review-dispatch` records an interruption as a failed run, and `pose-delegated-review-attempt-ledger` would refuse "rerun without reason" on a bundle whose only run was interrupted. The cost of an interruption should be the time to the reset, not the review.

### Constraints

Reuses the run record, the attempt reservation and the exclusive publisher of the ledger and state-consistency work; adds no runner. Vendor-specific behaviour (how a session is named, how a limit is worded) lives in the adapter declaration, never in the engine. Nothing here makes an interrupted run a review.

### Non-goals

Resuming across a changed sealed bundle (that is a new attempt, carrying prior findings: `pose-delegated-review-round-protocol`); choosing a different vendor on the maintainer's behalf; waiting for a limit that has no stated reset.

## 2. Requirements

### Functional

- R1: A run shall end in exactly one state of `completed`, `failed` or `interrupted`; an `interrupted` run shall record its cause (`usage-limit`, `timeout`, `process-lost`, `cancelled`, or `unknown`), whether it is resumable, and, when the adapter reports one, the instant the limit resets.
- R2: An adapter shall declare, in `.pose/policy/reviewers.json`, how its limit is recognised (exit codes and output patterns, and how the reset instant is read), how its session identifier is captured, and its resume command template. An adapter that declares none shall never have an interruption classified as `usage-limit`, and its interrupted runs shall be reported `unknown` and not resumable.
- R3: `pose review resume <run>` shall continue a resumable run on a worktree at the same sealed commit, with the same brief (digest compared), the same adapter and the vendor session captured by the interrupted run, and shall record the continuation as a new run linked `resumes: <run>` under the same attempt.
- R4: Resume shall be refused, with the reason, when the bundle or the brief digest changed, when the run already reached `completed` or `failed`, when the recreated worktree's tree digest differs from the one the run recorded, or when the resume budget (default 3 per attempt) or the attempt's accumulated wall-clock budget is spent.
- R5: A resume shall consume the reservation of the attempt it continues and shall not be a rerun: the ledger's rerun rule, its quota and `pose insights` count one attempt, and report the resumes of that attempt separately.
- R6: The draft and findings of an interrupted run shall be preserved in the run record by digest and disclosed as partial, and shall never be accepted as the review or counted as a rejection; only the draft of the run that reaches `completed` is the review.
- R7: `pose review resume <run> --wait-until-reset[=<max>]` shall sleep until the recorded reset instant (plus a bounded jitter) and then resume, shall never poll the vendor, and shall refuse when there is no recorded reset or the reset is later than `<max>`.
- R8: After a `usage-limit` interruption, dispatching the same bundle through a different adapter shall be allowed only with an explicit `--via` and shall be recorded as a new attempt with reason `limit-of:<run>`; the independence evidence shall name both runs, and the engine shall never switch adapter by itself.
- R9: `pose review status` shall show, for every interrupted run, its cause, whether it is resumable, the reset instant and the time left to it; `pose doctor` shall report an attempt whose last run is `interrupted` and not resumed.

### Non-functional

- A resume is idempotent under a crash: a second `resume` of the same run while a continuation exists shall return that continuation, not start another.
- Every transition is published through the exclusive publisher; a torn state is never visible.

### Security

- The captured session identifier is a reference to a vendor session and shall not be written to the transcript or to published reports; the resume template runs under the same containment as the first run.
- A run record is evidence: a resume that finds its predecessor rewritten or removed shall refuse.

### Compatibility

- Adapters without the new declarations keep working as they do now; their interruptions are classified `unknown`.

## 3. Technical Plan

### Affected areas

Run-state model and the supervisor's classification, adapter policy reader, `review resume`, `review status`, ledger accounting of resumes, doctor and insights.

### Artifacts

Implementation artifacts are declared when the spec starts.

### API/contract changes

New run states and fields; new `review resume` subcommand; additions to `reviewers.json` (`limit`, `session`, `resume`).

## 4. Tasks

- [ ] Run states, interruption causes and adapter declarations
- [ ] Resume with worktree and brief verification
- [ ] Attempt accounting and ledger interaction
- [ ] Wait-until-reset and explicit vendor fallback
- [ ] Status, doctor and insights

## 5. Decisions

### Decision D1
- Basis: R3, R5, R8
- Minimal option: treat a resume as a new dispatch with `--reason`
- Selected option: a resume is a continuation of one attempt, linked by `resumes:`
- Rationale: a reason-bearing rerun is exactly what the ledger exists to make rare and visible; an interruption is not the reviewer's verdict and must not be priced like one
- Consequences: attempts and runs become distinct counts; insights report both
- Falsifier: an interrupted run whose resume is counted as a rerun, or a rerun that is disguised as a resume

### Decision D2
- Basis: R8
- Minimal option: fall back automatically to the other vendor when a limit is reached
- Selected option: never; the maintainer chooses, and the choice is recorded
- Rationale: independence is evidence about who reviewed; changing reviewer silently changes the evidence
- Consequences: a limit can stop a review until the reset or a human decision
- Falsifier: an attestation whose run chain names an adapter nobody selected

## 6. Validation

### Strategy

A fake adapter that is interrupted at chosen points (limit with and without a reset, timeout, killed process) and resumes: the resumed run reaches the same draft as an uninterrupted run; a changed bundle, a changed brief, a rewritten predecessor and an exhausted budget are refused; the ledger counts one attempt; a fallback requires `--via` and is linked to the interrupted run.

### Requirement trace

## 7. Final Report

### Delivered scope

Not started: part of roadmap `delegated-review`, opened on 2026-10-18 from the operating experience of the B06 review rounds (see Origin).

### Residual risks

None yet.

### Follow-ups
