---
slug: pose-delegated-review-expired-session
status: in-progress
created_at: 2026-10-31
supersedes:
depends_on: pose-delegated-review-round-protocol, pose-delegation-parity-gate
remediates:
priority: 1
components: pose-mcp
task_type: bugfix
delivers: capability:delegated-review-expired-session
---

# Spec: A session that cannot continue gives way to a new one

## 1. Intent

### Goal

A review round whose attempt failed while continuing a session opens a new session on its next attempt, with the reason recorded, and the gates of the delegation roadmaps say what was delivered.

### Business value

Independent integration review of the milestones found the `rounds` exit gate unmet: a session the provider expired fails the round, and every next attempt resumes the same dead session, because the plan only asks whether the local saved session can be read. It also found three gates that stopped matching their delivery: `dispatch` and `surface` still name the review aliases that `pose-delegation-parity-gate` retired, and `retirement` promises on every platform what its member spec disclaims on Windows.

### Constraints

- Attempts and budgets stay as they are: a failed attempt counts, and the next one is the operator's to start.
- The new session gets the full brief, so no prior finding is lost.
- No vendor's error text is parsed: the signal is a continuation that failed.

### Non-goals

- Retrying inside one dispatch.
- Windows ACL protection of the task state, which stays F1 of `pose-delegation-task-detach-and-state`.
- A run interrupted by a limit, which `pose delegate resume` continues.

## 2. Requirements

### Functional

- R1: When a round's run that continued a session has failed for the same bundle, the next attempt shall open a new session and record why, naming the run that failed.
- R2: A failed run that did not continue a session, and an interrupted run, shall not change how the next attempt is planned.
- R3: The exit gates of `dispatch` and `surface` shall state the delivered command and the retired aliases, the gate of `dispatch` the interrupted state of a timed-out run, and the gate of `retirement` its supported platforms; `rounds` shall list this spec as a member.

### Non-functional

- The plan reads the bundle's runs it already reads, with no new Git or provider call.

### Security

- The reason recorded is bounded and comes from the run's own recorded failure.

### Compatibility

- A continuation that fails then opens a new session instead of failing again.

## 3. Technical Plan

### Affected areas

`internal/pose/review_rounds.go`, `.pose/roadmaps/delegation.md`, `.pose/roadmaps/delegated-review.md`.

### Artifacts

- created: .pose/specs/2026-10-31-pose-delegated-review-expired-session.md
- created: .pose/starts/pose-delegated-review-expired-session.json
- created: .pose/changelogs/unreleased/pose-delegated-review-expired-session.md
- modified: .pose/roadmaps/delegation.md
- modified: .pose/roadmaps/delegated-review.md
- modified: .pose/indexes/validation-matrix.json
- modified: pose-mcp/internal/pose/review_rounds.go
- created: pose-mcp/internal/cli/delegated_review_expired_session_test.go

### Delivery targets

- capability:delegated-review-expired-session module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go

### API/contract changes

A review thread's reason can name a failed continuation.

## 4. Tasks

- [x] Open a new session after a failed continuation
- [x] Reconcile the gates
- [x] Independent review and closeout

## 5. Decisions

### Decision D1
- Basis: R1
- Minimal option: match the provider's words for an expired session
- Selected option: any continuation that failed gives way to a new session on the next attempt
- Rationale: vendors word it differently and change it, and resuming a session that just failed is not worth another attempt
- Consequences: a transient failure also costs its session; the full brief keeps the findings
- Falsifier: a next attempt that resumes a session a run of its bundle failed to continue

### Decision D2
- Basis: R3
- Minimal option: restore the aliases, or keep the Windows promise in the gate
- Selected option: state in each gate what was delivered, as the maintainer's retirement of the aliases and the task spec's platform limit already decided
- Rationale: a gate that names retired commands cannot be met, and qualifying a platform states what is proved
- Consequences: Windows protection of the task state stays an open follow-up of its spec, not a gate
- Falsifier: a gate of these milestones that the delivered system does not meet

## 6. Validation

### Strategy

The reviewer's fake provider that rejects a resume, through two real attempts of a round; a failed cold run and the normal continuation as controls; independent re-judgment of the gates.

### Requirement trace

- R1 [satisfied] capability:delegated-review-expired-session evidence:integration check:delegated-review-expired-session-integration test:TestFailedContinuationOpensANewSession test:TestFailedContinuationReasonIsBounded report:.pose/results/delivery-validation.json commit:c5eb59617874007d4b0dcbd337288ffa12176fb0 — The next attempt after a failed continuation opens a new session and names the run.
- R2 [satisfied] check:delegated-review-expired-session-integration test:TestOnlyAFailedContinuationGivesWay test:TestContinuationAcrossRoundsKeepsItsSession report:.pose/results/delivery-validation.json commit:c5eb59617874007d4b0dcbd337288ffa12176fb0 — Cold, interrupted and other bundles' runs plan as before.
- R3 [satisfied] check:delegated-review-expired-session-integration report:.pose/results/delivery-validation.json commit:c5eb59617874007d4b0dcbd337288ffa12176fb0 — The gates of dispatch, surface, rounds and retirement state what was delivered.

## 7. Final Report

### Delivered scope

A review attempt that failed while continuing a session gives way to a new session on the next attempt, with the reason recorded and no vendor text parsed. The gates of `dispatch`, `surface`, `rounds` and `retirement` now state what was delivered. Independently reviewed by Codex in one round, by the reviewer who found the gaps.

### Residual risks

A transient failure of a continuation also costs its session; the full brief keeps the findings. Windows protection of the task state stays with `pose-delegation-task-detach-and-state` F1.

### Follow-ups

- [open] F1: offer the next attempt at a failed run: `delegate ls` and the dispatch result show handoff and cancel, not how to run again (review F1). (owner:@oseiaspereira88 crit:low review:2026-11-27)
