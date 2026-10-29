---
slug: pose-delegation-task-detach-and-state
status: done
created_at: 2026-10-29
completed_at: 2026-10-29
supersedes:
depends_on: pose-delegation-task, pose-delegated-review-completion-signal
remediates:
priority: 3
components: pose-mcp
task_type: feature
delivers: capability:delegation-task
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
- modified: .pose/specs/2026-10-26-pose-delegation-task.md
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
- modified: .pose/state/history.jsonl
- modified: .pose/state/project-state.md
- modified: .pose/state/refresh-log.jsonl
- modified: POSE.md
- modified: locales/pt-BR/POSE.md
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md

### Delivery targets

- capability:delegation-task module:pose-mcp profile:composed-capability entrypoint:pose-mcp/internal/cli/delegate_task.go

### API/contract changes

`pose delegate task --detach`; the detach document of a task carries its `record` place; the state-path contract of R5–R7.

## 4. Tasks

- [x] Detached task with prompt pipe and record place
- [x] Detached `resume --prompt -` through the same pipe
- [x] Supervisor directory trust and believed hand-back
- [x] State base contract and ancestor checks
- [x] Created components: owner, mode, no link
- [x] Manuals, help, MCP refusal text, changelog
- [x] A lone `-` read as stdin by the parser
- [x] Mutations that fail each new test
- [x] Independent review and closeout

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

- R1 [satisfied] check:delegation-surface-integration test:TestDelegateTaskDetaches test:TestDelegateTaskHistoryIsOneThroughARootLink report:.pose/results/delivery-validation.json commit:86aac5e4b7a0bd942e70ff769cebe3bcb7689048 — A task detaches with a prompt file or stdin, returns its tat- attempt, log and record place, and wait, ls and watch resolve it, through a root link too.
- R2 [satisfied] check:delegation-surface-integration test:TestDelegateTaskDetachedPromptArrivesWhole test:TestDelegateTaskDetachedResumeTakesAPipedPrompt report:.pose/results/delivery-validation.json commit:86aac5e4b7a0bd942e70ff769cebe3bcb7689048 — A piped prompt of 1 MiB reaches the detached task and resume whole, through a pipe; no file holds it.
- R3 [satisfied] check:delegation-surface-integration test:TestDelegateTaskDetachRefusesWhatItCannotStart report:.pose/results/delivery-validation.json commit:86aac5e4b7a0bd942e70ff769cebe3bcb7689048 — A route that must wait for a reset is refused detached, naming the reset.
- R4 [satisfied] check:delegation-surface-integration test:TestDelegateTaskDetachRefusesWhatItCannotStart test:TestDelegateTaskDetachIgnoresAnInheritedLegacyRoot report:.pose/results/delivery-validation.json commit:86aac5e4b7a0bd942e70ff769cebe3bcb7689048 — Refusals before reservation return the foreground's document and exit code (2 and 1), JSON and text.
- R5 [satisfied] check:test test:TestTaskStateBaseTakesOnlyAnAbsoluteXDGStateHome report:.pose/results/delivery-validation.json commit:86aac5e4b7a0bd942e70ff769cebe3bcb7689048 — Only an absolute XDG_STATE_HOME is the base; otherwise the platform default.
- R6 [satisfied] check:test test:TestTaskStateDirTrustsOnlyAPrivateBase test:TestTaskStateDirRefusesAnotherUsersDirectory report:.pose/results/delivery-validation.json commit:86aac5e4b7a0bd942e70ff769cebe3bcb7689048 — A linked base is resolved; a base or ancestor another user owns or can write without sticky is refused, root ancestors accepted.
- R7 [satisfied] check:test test:TestTaskStateDirRefusesALinkedOrWritableComponent test:TestTaskStateDirRefusesAnotherUsersDirectory report:.pose/results/delivery-validation.json commit:86aac5e4b7a0bd942e70ff769cebe3bcb7689048 — POSE-made components that are links, foreign or writable are refused, never followed or tightened.
- R8 [satisfied] check:delegation-surface-integration test:TestDelegateToolsOverMCPMatchTheCLI report:.pose/results/delivery-validation.json commit:86aac5e4b7a0bd942e70ff769cebe3bcb7689048 — The MCP refusal of a task start gives the reason and the CLI detach guidance, not the obsolete claim.
- R9 [satisfied] check:delegation-surface-integration test:TestDelegateTaskReadsAPromptFromStdin report:.pose/results/delivery-validation.json commit:86aac5e4b7a0bd942e70ff769cebe3bcb7689048 — A lone - is the task's prompt operand and the value of --prompt in the foreground.
- R10 [satisfied] check:delegation-surface-integration test:TestDelegateTaskDetachRefusesAWritableSupervisorDirectory test:TestDelegateTaskDetachDoesNotBelieveAForgedHandback test:TestDelegateTaskDetachStaysInTheCheckedRoot test:TestDelegateTaskDetachRefusesAMissingCurrentDirectory report:.pose/results/delivery-validation.json commit:86aac5e4b7a0bd942e70ff769cebe3bcb7689048 — Private supervisor chain, pinned resolved root, hand-back believed only after the prompt is through and the attempt reserved, missing cwd refused.
- R11 [satisfied] check:delegation-surface-integration test:TestDelegateTaskHistoryIsOneThroughARootLink test:TestDelegateTaskDetachedContinuationFindsTheLegacyHistory test:TestDelegateTaskDetachIgnoresAnInheritedLegacyRoot report:.pose/results/delivery-validation.json commit:86aac5e4b7a0bd942e70ff769cebe3bcb7689048 — One history per resolved root; legacy link-keyed history read by foreground and detached continuations; inherited or foreign spellings ignored (TestOnlyADetachedRunTakesItsCallersRootSpelling, TestADetachedRunInheritsNoControlVariable, TestTaskHistoryKeyedByTheResolvedRootReadsTheLegacyKey under check:test).

## 7. Final Report

### Delivered scope

Closes follow-ups F2 and F3 of `pose-delegation-task` on implementation head `86aac5e4b7a0bd942e70ff769cebe3bcb7689048`, ahead of milestone `retirement` of roadmap `delegation`: `pose delegate task --detach`, a piped prompt carried to a detached task or resume through a pipe, foreground refusal parity for detached calls, the `-` operand and `--prompt -` that `pose delegate task` and `resume` promised but never parsed (R9, a defect of the closed spec found here), an explicit `XDG_STATE_HOME` contract and ownership/permission checks on the task-state path, a private supervisor directory and a pinned resolved root for every detached purpose, and one task history per resolved root with legacy link-keyed history still read. The detach and supervisor rules apply to review, adjudicate, smoke, resume and handoff as well as task.

Independent review: seven rounds by Codex `gpt-6.1-sol` (OpenAI, reasoning medium), principal `agent:independent-f2f3-sol-20261029-r1`, a different vendor than the implementer Claude Opus 5.5 (Anthropic). Rounds 1–6 requested changes (F2F3-R1-01..06, R2-01..02, R3-01..03, R4-01..04, R5-01, R6-01), each fixed in its own commit with a regression that fails when its production line is reverted, and recorded in the amendment journal; round 7 approved with no blocker. The reports are local review evidence under `/tmp/pose-roadmaps-evidence/f2f3/` (untracked); the native bundle and attestation are the sealed record.

### Validation executed

- `go test ./...` (pose-mcp, without agent session markers, `GOMAXPROCS=2 GOFLAGS=-p=1`): passed on the implementation head; `go test -race` over the delegation, detach, wait, task-state and handoff selections passed; `go vet` and Windows/Darwin builds passed.
- Mutations: 30 production mutants (M1–M31, with M22+M23 and M30+M31 applied together because each alone is covered by the other layer) each fail an assertion after a passing baseline.
- The native strict validation, index, seal, attestation and verify of the closeout are recorded by the bundle and attestation linked from the closeout commit.

### Residual risks

On Windows, ownership, mode bits and ACLs of the task-state path and the supervisor directory are not checked (only links and non-directories are refused), and detached runs were build-checked, not run natively. A hand-back naming an attempt that exists is believed without proof that this run reserved it; inside a private supervisor chain only the user or root can write one. The user and root changing their own trusted tree are outside the guarantee. The observation and MCP residuals of knowledge:delegation-observation-round4 and knowledge:delegation-mcp-round5 are unchanged by this spec and keep their owners and review dates there.

### Follow-ups

- [open] F1: on Windows, fold the names of the detached control variables before filtering them and check ACL privacy of the task-state and supervisor directories, with native Windows runs of a detached task, before Windows directory privacy is claimed (F2F3-R7-01 and the ACL boundary). (owner:@oseiaspereira88 crit:medium review:2026-11-26)
- [open] F2: add a deterministic regression for the copy result arriving after the detached run exits, through a seam that holds the copy's report (F2F3-R4-03). (owner:@oseiaspereira88 crit:low review:2026-11-26)
- [open] F3: authenticate that the detached run that hands an attempt back reserved it, instead of checking that the attempt exists. (owner:@oseiaspereira88 crit:low review:2026-11-26)
- [open] F4: make history-check write an error verdict in its machine document on every early return, such as an unavailable Git repository, which now prints a default pass document with exit 2 (F2F3-R7-02, found by the review, not caused by this spec). (owner:@pose-maintainers crit:low review:2026-11-26)
- [open] F5: let routing learn a usage limit from task runs as it does from review runs, so --agent auto skips an agent a task found limited. (owner:@oseiaspereira88 crit:low review:2026-11-26)
