---
slug: pose-delegation-task
status: in-progress
created_at: 2026-10-26
completed_at:
supersedes:
depends_on: pose-delegation-surface, pose-delegated-review-completion-signal
priority: 3
components: pose-mcp
task_type: feature
delivers: capability:delegation-task
---

# Spec: A general task can be delegated to another agent

## 1. Intent

### Goal

Let a person or an agent hand any task to another agent through `pose delegate task`, with the access it needs, the same lifecycle as every delegated run, and no path from a task to a gate.

### Business value

Decided by the [delegation ADR](../adr/2026-10-26-delegation-is-a-first-class-execution-surface.md). The personal launchers survive mostly because they also delegate work that is not a review: implementing, fixing, committing and pushing with another vendor's agent. Without a native task purpose they cannot retire, and the work they run stays unrecorded and unobservable by POSE's own commands.

### Constraints

A task runs in the real working directory, not in a disposable worktree; containment still bounds its resources, and the filesystem is not isolated beyond what the agent's own access mode provides. A task never produces an attestation, a verdict or evidence.

### Non-goals

Scheduling or queueing tasks; tasks across projects; reviewing what a task changed (that is a review of its own bundle).

## 2. Requirements

### Functional

- R1: `pose delegate task <prompt-file|->` shall run the chosen agent with the prompt on stdin in `--cd <dir>` (default: the current directory).
- R2: `--access read|sandboxed|write|full` shall select the agent's matching command template; the default shall be `purposes.task.default_access`, `full` shall never be a default, and choosing it shall print a warning naming what it disables.
- R3: `--add-dir`, `--effort` and `--no-network` shall reach the agent through its template; `--no-network` with `write` or `full`, or on an agent that cannot enforce it, shall be refused with exit 2.
- R4: A task shall run outside a Git repository, using the agent's `outside_git_args` when declared.
- R5: Without `--record`, the run shall be kept under the user's state directory for the project — prompt, raw and normalized events, final message, stderr, argv, removed variable names, exit code, label — with owner-only permissions; with `--record`, under `.pose/delegate-runs/`.
- R6: The final message shall go to stdout and the agent's exit code shall be returned; with `--json`, one result document shall replace both.
- R7: Every reader of review runs, attestations, closeouts and gates shall ignore task runs, and `pose review attest --run` shall refuse a task run.
- R8: `ls`, `watch`, `wait`, `resume`, `handoff` and `cancel` shall work on task runs; `resume` without a prompt shall send the default continuation message (continue where you stopped, redo the interrupted command, do not repeat completed steps unless their result must be fresh).
- R9: A task stopped by a usage limit shall report `next_actions` with the resume and handoff commands.
- R10: Routing shall treat the launching session as the implementation, as `pose-delegation-routing` R10 requires.

### Non-functional

- Starting a task adds no measurable overhead beyond the agent's own start-up and the record writes.

### Security

- `write` and `full` act on the real working directory; the help and the warning say so, and containment resource bounds still apply.
- Prompts are stored with owner-only permissions and never copied into `.pose/` without `--record`.

### Compatibility

- None: a new purpose.

## 3. Technical Plan

### Affected areas

Task front end, access templates, record location and permissions, exclusion from review readers, default continuation message.

### Artifacts

- modified: .pose/specs/2026-10-26-pose-delegation-task.md
- created: .pose/starts/pose-delegation-task.json
- created: .pose/changelogs/unreleased/pose-delegation-task.md
- created: pose-mcp/internal/pose/delegation_task.go
- modified: pose-mcp/internal/pose/delegation_policy.go
- modified: pose-mcp/internal/pose/delegation_list.go
- modified: pose-mcp/internal/pose/delegation_events.go
- modified: pose-mcp/internal/pose/delegation_routing.go
- modified: pose-mcp/internal/pose/delegation_wait.go
- modified: pose-mcp/internal/pose/review_dispatch.go
- modified: pose-mcp/internal/pose/review_run_continuity.go
- modified: pose-mcp/internal/pose/review_run_provenance.go
- modified: pose-mcp/internal/pose/review_bundle.go
- created: pose-mcp/internal/cli/delegate_task.go
- created: pose-mcp/internal/cli/delegate_task_test.go
- modified: pose-mcp/internal/cli/delegate.go
- modified: pose-mcp/internal/cli/delegate_handoff.go
- modified: pose-mcp/internal/cli/delegate_watch.go
- modified: pose-mcp/internal/cli/delegate_test.go
- modified: pose-mcp/internal/cli/review_closeout.go
- modified: pose-mcp/internal/cli/help_catalog.go
- modified: POSE.md
- modified: locales/pt-BR/POSE.md
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md

### Delivery targets

- capability:delegation-task module:pose-mcp profile:composed-capability entrypoint:pose-mcp/internal/cli/delegate_task.go

### API/contract changes

`pose delegate task`; policy keys `purposes.task.default_access`, `agents.<name>.outside_git_args`, `add_dir_args`, `no_network_args` and `resume.commands`, and the template placeholders `{outside_git}`, `{add_dirs}` and `{no_network}` (D3); record location; `pose delegate resume --prompt` for tasks; `wait` exits 0 for a task its agent completed.

## 4. Tasks

- [x] Task front end over the shared run lifecycle
- [x] Access templates, warnings and refusals
- [x] Record location, permissions and `--record`
- [x] Exclusion from every review and gate reader
- [x] Continuation message and `next_actions`
- [ ] Independent review and closeout

## 5. Decisions

### Decision D1
- Basis: R5
- Minimal option: record every task under `.pose/`
- Selected option: the user's state directory unless `--record`
- Rationale: a task is not governance evidence; recording it in the repository would mix work logs with evidence and leak prompts into history
- Consequences: a task's record does not travel with a clone
- Falsifier: a gate or report that needs a task record to be correct

### Decision D2
- Basis: R7
- Minimal option: trust callers not to attest task runs
- Selected option: every review reader ignores task runs and attestation refuses them
- Rationale: a write-capable run must never become an approval, whatever the caller intends
- Consequences: a task that reviews something has to be run as a review
- Falsifier: an attestation, verdict or closeout that cites a task run

### Decision D3
- Basis: R3, R4, R8
- Minimal option: append `--add-dir`, `--no-network` and the outside-Git arguments at the end of the agent's argv
- Selected option: the agent's template places `{add_dirs}`, `{no_network}` and `{outside_git}` as whole elements, filled from `add_dir_args` (with `{dir}`), `no_network_args` and `outside_git_args`; a flag whose placeholder or fragment is missing is refused with exit 2; a task resumes with `resume.commands` for its access, and a read task may use `resume.command`
- Rationale: the agents' CLIs read flags at fixed positions (before a trailing `-` for stdin, after a subcommand), so only the template knows where a fragment goes; resuming a write task with the read command would silently drop its access
- Consequences: an agent without the fragments cannot take those flags, and says so; existing review templates are unchanged
- Falsifier: a task that ran without the network restriction or the extra directory it was asked for, or a resumed task with another access

### Decision D4
- Basis: R5, R7
- Minimal option: record tasks under `.pose/review-runs` with a purpose field every reader filters
- Selected option: task runs are stored apart (state directory, or `.pose/delegate-runs` with `--record`), read only by the lifecycle commands, with their own identifiers (`tat-` attempts, `trn-` runs) and no attempt reservation or bundle; attestation and provenance refuse a task run by name
- Rationale: a reader that forgets the filter would turn a task into evidence; storage the review readers never open cannot
- Consequences: a task has no bundle budget; its continuation budget is the attempt's, as for reviews
- Falsifier: a review reader, a gate or a closeout that lists a task run

## 6. Validation

### Strategy

A fake agent runs a task with each access level, inside and outside a Git repository; `full` warns and is never defaulted; `--no-network` with `write` is refused; the record lands in the state directory with owner-only permissions, and under `.pose/` only with `--record`; `attest --run` refuses a task run and `review verify` ignores it; a limited task resumes with the default message.

### Requirement trace

## 7. Final Report

### Delivered scope

Not started: milestone `task` of roadmap `delegation`.

### Residual risks

None yet.

### Follow-ups
