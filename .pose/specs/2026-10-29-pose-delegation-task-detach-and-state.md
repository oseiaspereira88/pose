---
slug: pose-delegation-task-detach-and-state
status: in-progress
created_at: 2026-10-29
completed_at:
supersedes:
depends_on: pose-delegation-task, pose-delegated-review-completion-signal
remediates:
priority: 3
components: pose-mcp
task_type: feature
delivers:
---

# Spec: A delegated task detaches, and its private record lives on a path only its user controls

## 1. Intent

### Goal

Close the two gaps `pose-delegation-task` left open before the launchers can retire: `pose delegate task --detach` (its follow-up F3), and a task-state path whose ancestors no other user can replace, with an explicit contract for `XDG_STATE_HOME` (its follow-up F2).

### Business value

The [delegation ADR](../adr/2026-10-26-delegation-is-a-first-class-execution-surface.md) retires the personal launchers only at proven parity, and a background task is how the launchers are used most: the parity gate cannot map that capability to a native command that does not exist. The task record holds prompts and agent output; owner-only leaf permissions do not protect it when a directory above the leaf can be swapped by someone else.

### Constraints

A detached task reuses the supervisor of the review purposes (spec `pose-delegated-review-completion-signal`): no new daemon, the same hand-back, the same bound. The state path keeps honouring a person's own `XDG_STATE_HOME`, including one that is or passes through a symbolic link the person made. POSE cannot prevent the user or root from changing their own directories.

### Non-goals

Starting a task over MCP (`pose-delegation-mcp` keeps refusing it: its prompt and access are the caller's); a queue or scheduler; moving existing records; the parity matrix itself (`pose-delegation-parity-gate`).

## 2. Requirements

### Functional

- R1: `pose delegate task <prompt-file|-> --detach` shall start the task as a detached run, as the review purposes do, and return at once with its `tat-` attempt, the supervisor's process and log, and the place its record is kept; `wait`, `ls`, `watch` and `cancel` shall resolve the returned attempt.
- R2: A prompt read from stdin by a detached `task` or by a detached `resume --prompt -` shall be read by the caller before it detaches and written to the detached run's stdin through a pipe, never through a file; a detached run shall read stdin only when its caller hands a prompt over that way, and never the caller's own stdin.
- R3: A detached task whose route must first wait for a usage-limit reset shall be refused, naming the reset, as the detach contract of `pose-delegated-review-completion-signal` (its milestone exit gate and manual) requires for the review purposes.
- R4: A detached task, resume or handoff refused before it reserves an attempt shall return the refusal the foreground command returns: the same document or message, reason code and exit code (2 for a usage or policy refusal, 1 otherwise).
- R5: `XDG_STATE_HOME` shall be used as the state base only when it is an absolute path; an empty or relative value shall fall back to the platform default, as the XDG Base Directory specification requires.
- R6: The state base shall be resolved once, links included, and the resolved base and each of its ancestors shall be a directory owned by the user or by root and not writable by group or others unless sticky; a task shall refuse to record or read under a base that is not, naming the directory.
- R7: Each directory POSE creates below the base (`pose`, `pose/delegate`, the project's directory) shall be created owner-only and accepted only as a real directory owned by the user and not writable by group or others; a link, another owner or a writable component shall be refused, naming the path, and never followed or tightened.
- R8: The MCP refusal of a task start shall state why a task is not started over MCP without claiming that a task cannot run detached.
- R9: The delegate flag parser shall take a lone `-` as the task's prompt operand and as the value of `--prompt`, never as an unknown flag or a missing value, so `pose delegate task -` and `pose delegate resume --prompt -` read stdin in the foreground as `pose-delegation-task` R1 and R8 describe.
- R10: A detached call shall resolve the project root once and use only that resolved root for the supervisor directory, its own run and the reservation it checks; it shall keep its log and hand-back only in a supervisor directory that, with every directory up to and including the project root, is a real directory of the user's writable by no group or other user, the root's ancestors meeting R6, and shall refuse to detach otherwise, naming the directory; it shall believe a hand-back only once a piped prompt has gone through whole and a run has reserved the attempt it names, and shall otherwise stop the run and report it; a caller whose current directory cannot be read shall be refused, never started in the root or elsewhere.
- R11: A project's private task history shall be keyed by its resolved root, so every spelling of the same root — a link, the pinned root of a detached run — reaches one history; a history an earlier engine kept under an unresolved spelling shall still be read and never written, by a foreground command and by a detached run, which is given its caller's spelling only as a key to that history, never as a path it opens, inherits no such value from its caller's environment, and accepts a spelling only while it resolves to the run's own root.

### Non-functional

- The path checks add one `lstat` per path component to a task's start and to each listing of task runs.

### Security

- R2's prompt travels only through an anonymous pipe between the caller and its detached run: no file, argv or environment variable carries it, so no directory permission can expose or replace it; the environment carries only the flag `POSE_DELEGATE_PROMPT_PIPED`.
- On Windows, where ownership and mode bits are not the access model, R6, R7 and the directory part of R10 keep only the link and type refusal; ACL privacy is not checked, and the manual says so.

### Compatibility

- Every detached `pose delegate` call, not only a task, now refuses a checkout whose root, `.pose`, `review-private` or `supervisors` another user could write, or whose root has an ancestor another non-root user owns or could write without the sticky bit; the foreground commands are unaffected, and the refusal names the directory and says to detach from a checkout on a private path.
- Task records an earlier engine kept under a linked spelling of the root stay readable through that spelling (R11); new records go under the resolved root.
- A relative `XDG_STATE_HOME` previously placed task records relative to the working directory; those records are no longer listed (R5), and the changelog says so. A base or component that fails R6 or R7 makes `ls` report the loss instead of reading runs from it.

## 3. Technical Plan

### Affected areas

Task front end and detached supervisor (prompt pipe, trusted supervisor directory, believed hand-back, record place, reset refusal), the delegate flag parser's `-`, task-state path resolution, MCP task refusal text, help catalogue, the four manuals.

### Artifacts

- created: .pose/specs/2026-10-29-pose-delegation-task-detach-and-state.md
- created: .pose/specs/2026-10-29-pose-delegation-task-detach-and-state.amendments.jsonl
- created: .pose/changelogs/unreleased/pose-delegation-task-detach-and-state.md
- created: .pose/starts/pose-delegation-task-detach-and-state.json
- modified: .pose/roadmaps/delegation.md
- modified: .pose/specs/2026-10-26-pose-delegation-parity-gate.md
- modified: pose-mcp/internal/cli/delegate.go
- modified: pose-mcp/internal/cli/delegate_task.go
- modified: pose-mcp/internal/cli/delegate_wait.go
- modified: pose-mcp/internal/cli/help_catalog.go
- modified: pose-mcp/internal/cli/cli.go
- created: pose-mcp/internal/cli/delegate_task_detach_test.go
- modified: pose-mcp/internal/mcpserver/delegate_tools.go
- modified: pose-mcp/internal/cli/delegate_mcp_test.go
- modified: pose-mcp/internal/pose/delegation_task.go
- created: pose-mcp/internal/pose/delegation_task_state_unix.go
- created: pose-mcp/internal/pose/delegation_task_state_windows.go
- created: pose-mcp/internal/pose/delegation_task_state_test.go
- modified: POSE.md
- modified: locales/pt-BR/POSE.md
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md

### API/contract changes

`pose delegate task --detach`; the detach document of a task carries its `record` place; the state-path contract of R5–R7.

## 4. Tasks

- [ ] Detached task with prompt pipe and record place
- [ ] Detached `resume --prompt -` through the same pipe
- [ ] Supervisor directory trust and believed hand-back
- [ ] State base contract and ancestor checks
- [ ] Created components: owner, mode, no link
- [ ] Manuals, help, MCP refusal text, changelog
- [ ] A lone `-` read as stdin by the parser
- [ ] Mutations that fail each new test
- [ ] Independent review and closeout

## 5. Decisions

### Decision D1
- Basis: R2
- Minimal option: refuse `--detach` with a stdin prompt
- Selected option: the caller reads stdin and writes it to the detached run's stdin through a pipe (amended in review round 1, F2F3-R1-01 and -02: a private file in the supervisor directory could be replaced by a writer of that directory, and survived a publication error)
- Rationale: piping a prompt is how the launchers are driven; refusing it would make the parity row true only for prompt files; a pipe has no name to replace and nothing to clean up
- Consequences: the run reads the whole prompt before it hands its attempt back, so the caller returns only after the prompt is through
- Falsifier: a detached run that ran with a prompt other than the caller's stdin, or a prompt found in a file

### Decision D2
- Basis: R5, R6, R7
- Minimal option: refuse any link on the whole path
- Selected option: trust the resolved base the person chose, as OpenSSH does for a home directory, and refuse links only on the components POSE creates
- Rationale: links in a home or state directory are common and are the person's own; the risk is a directory someone else can write, which ownership and mode tell, not linkness
- Consequences: a group-writable home or state base refuses tasks until it is tightened or `XDG_STATE_HOME` names a private directory
- Falsifier: a record read or written through a directory another non-root user could rename

## 6. Validation

### Strategy

A built binary detaches a task with a prompt file and with stdin, waits for it and finds its record under the state directory; a detached `resume --prompt -` sends the piped prompt; a route limited until a reset refuses to detach. Path tests cover a relative `XDG_STATE_HOME`, a linked base, a group-writable ancestor, a linked or foreign `pose` component and a writable project directory. Each new test is shown failing with its production line reverted.

### Requirement trace

## 7. Final Report

### Delivered scope

Not started: closes follow-ups F2 and F3 of `pose-delegation-task` before milestone `retirement` of roadmap `delegation`.

### Residual risks

None yet.

### Follow-ups
