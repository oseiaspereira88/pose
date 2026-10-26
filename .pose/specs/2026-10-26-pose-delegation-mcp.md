---
slug: pose-delegation-mcp
status: draft
created_at: 2026-10-26
completed_at:
supersedes:
depends_on: pose-delegated-review-completion-signal, pose-delegated-review-round-protocol, pose-delegation-handoff
priority: 3
components: pose-mcp
task_type: feature
delivers:
---

# Spec: Agents operate delegated runs over MCP

## 1. Intent

### Goal

Expose bounded MCP tools to start, inspect, wait for, resume and hand off delegated runs, with exactly the refusals of the CLI.

### Business value

Decided by the [delegation ADR](../adr/2026-10-26-delegation-is-a-first-class-execution-surface.md), which reverses option 4 of the 2026-10-09 ADR. That option was rejected because an implementing agent could rerun a review until it was approved; the attempt ledger now refuses a rerun on an unchanged bundle without a reason, and a rejection stays visible. Agents that reach POSE through MCP should not have to shell out to act on `next_actions`.

### Constraints

No MCP call blocks beyond its own bounded timeout. Every write tool follows the server's existing authorization and `project_id` rules. MCP never returns raw transcripts or session references.

### Non-goals

Recording attestations over MCP (unchanged); streaming events over MCP.

## 2. Requirements

### Functional

- R1: `pose_delegate_start` shall start a run of any purpose, always detached, and return the run identifier, attempt, agent and achieved independence.
- R2: `pose_delegate_status` shall return the run's state, heartbeat age, decision when terminal and `next_actions`, without blocking; it replaces the `pose_review_run_status` tool named by the completion-signal spec.
- R3: `pose_delegate_wait` shall wait up to a timeout bounded by the server's maximum and return either the terminal result or `still-running` with the current status.
- R4: `pose_delegate_resume` and `pose_delegate_handoff` shall behave as their CLI commands and return the new run.
- R5: Every refusal of the CLI — rerun on an unchanged bundle without a reason, exhausted budget, containment incompatibility, unavailable agent — shall be returned by the MCP tools with the same reason code.
- R6: Each tool shall require `project_id` and the server's write authorization for writes, and shall refuse a project the caller cannot read.
- R7: Tool results shall carry the normalized summary only: no raw transcript, prompt, session reference or environment value.

### Non-functional

- A tool call returns within its declared bound even when the run never ends.

### Security

- The server's identity and authorization model applies unchanged; no tool widens what the CLI allows.

### Compatibility

- Existing MCP tools are unchanged.

## 3. Technical Plan

### Affected areas

MCP server tool catalog and handlers, shared refusal codes with the CLI, bounded wait.

### Artifacts

Implementation artifacts are declared when the spec starts.

### API/contract changes

Tools `pose_delegate_start`, `pose_delegate_status`, `pose_delegate_wait`, `pose_delegate_resume`, `pose_delegate_handoff`.

## 4. Tasks

- [ ] Tools over the shared run lifecycle
- [ ] Shared refusal codes
- [ ] Bounded wait
- [ ] Result redaction

## 5. Decisions

### Decision D1
- Basis: R5
- Minimal option: a reduced MCP surface without refusal parity
- Selected option: the same refusals and reason codes as the CLI
- Rationale: the safeguard that justified reversing the 2026-10-09 decision is the ledger's refusal; any MCP path around it reopens the approval loop
- Consequences: refusal codes become a shared contract of CLI and MCP
- Falsifier: an MCP sequence that obtains a second run on an unchanged bundle without a reason

## 6. Validation

### Strategy

Over a fake agent, each tool returns the documented shape; a rerun on an unchanged bundle is refused over MCP with the CLI's code; `pose_delegate_wait` returns `still-running` at its bound; results contain no transcript or session reference.

### Requirement trace

## 7. Final Report

### Delivered scope

Not started: milestone `agent-access` of roadmap `delegation`.

### Residual risks

None yet.

### Follow-ups
