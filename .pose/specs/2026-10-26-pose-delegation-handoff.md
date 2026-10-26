---
slug: pose-delegation-handoff
status: draft
created_at: 2026-10-26
completed_at:
supersedes:
depends_on: pose-delegation-routing, pose-delegated-review-completion-signal
priority: 2
components: pose-mcp
task_type: feature
delivers:
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

Implementation artifacts are declared when the spec starts.

### API/contract changes

`pose delegate handoff`; run record field `hands_off_from`; `next_actions` in `ls` and `wait` JSON.

## 4. Tasks

- [ ] Continuation of another agent under the same attempt and budget
- [ ] Handoff brief section with digests
- [ ] Lineage in the record and in watch
- [ ] Terminal choices and `next_actions`
- [ ] Thread session update on vendor change

## 5. Decisions

### Decision D1
- Basis: R2
- Minimal option: a handoff opens a new attempt
- Selected option: a continuation of the same attempt sharing its budget
- Rationale: a new attempt would let alternating agents bypass `max_runs_per_bundle` and turn a rejection-free history into repeated tries
- Consequences: a long stop may exhaust the budget and escalate as the capability spec requires
- Falsifier: a chain of handoffs that runs longer than the attempt's continuation budget

### Decision D2
- Basis: R3, R6
- Minimal option: hand the new agent the stopped agent's whole transcript
- Selected option: the partial final message and the executed commands with exit status, labelled and bounded
- Rationale: the new reviewer gains the ground already covered without inheriting the stopped reviewer's reasoning wholesale, and re-verifies what it relies on
- Consequences: some reasoning is redone
- Falsifier: a handoff review that cites the predecessor's partial as its evidence

## 6. Validation

### Strategy

A fake agent stops on a recorded limit; a handoff to a second fake agent runs under the same attempt with the expected brief section and digests, and its conclusion alone is recorded; a chain of handoffs stops at the budget; a completed run is refused; a terminal session offers the three choices and a non-terminal run returns `next_actions`; watch shows both markers in order.

### Requirement trace

## 7. Final Report

### Delivered scope

Not started: milestone `continuity` of roadmap `delegation`.

### Residual risks

None yet.

### Follow-ups
