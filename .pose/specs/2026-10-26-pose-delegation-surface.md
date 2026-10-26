---
slug: pose-delegation-surface
status: draft
created_at: 2026-10-26
completed_at:
supersedes:
depends_on: pose-delegated-adapter-containment, pose-delegated-review-run-continuity, pose-delegated-review-attempt-ledger
priority: 2
components: pose-mcp
task_type: feature
delivers:
---

# Spec: `pose delegate` is the one surface for delegated executions

## 1. Intent

### Goal

Give every delegated execution one command tree, one vocabulary, one policy file and one run record, so a person and an agent start, inspect and continue a delegated run the same way whatever its purpose.

### Business value

Decided by the [delegation ADR](../adr/2026-10-26-delegation-is-a-first-class-execution-surface.md). The engine's delegated review already reserves attempts, contains the reviewer, continues interrupted runs and proves provenance, yet no review has run through it: execution operations are scattered across `pose review dispatch`, `resume` and `status`, beside the commands that govern the review subject, and the personal launchers that fill the gap stay in use. This spec builds the frame every later delegation spec plugs into.

### Constraints

Reuse the delivered dispatch, ledger, continuity, provenance and containment paths; this spec moves and names them, it does not reimplement them. No command string reaches a shell. No new daemon.

### Non-goals

Routing between vendors (`pose-delegation-routing`); handoff (`pose-delegation-handoff`); detach, wait, heartbeat and live watch (`pose-delegated-review-completion-signal`, amended by the ADR); general tasks (`pose-delegation-task`); MCP (`pose-delegation-mcp`).

## 2. Requirements

### Functional

- R1: `pose delegate` shall offer the purposes `review`, `adjudicate`, `smoke` and `task` and the lifecycle commands `ls`, `watch`, `wait`, `resume`, `handoff`, `cancel` and `agents`; with no argument it shall print its help, and an unknown subcommand shall exit 2.
- R2: Every subcommand shall give `--agent`, `--access`, `--label`, `--lead`, `--dry-run` and `--json` the same meaning, and the help shall define the terms agent (a configured CLI profile), run (one execution), attempt (a run and its continuations) and thread (the rounds of one scope).
- R3: `review`, `adjudicate` and `smoke` shall run through the delivered dispatch path — sealed bundle or target, generated brief, disposable worktree, attempt ledger, containment — with `read` access; any other `--access` shall be refused with exit 2.
- R4: When `--agent` is omitted it shall mean `auto`; until `pose-delegation-routing` replaces the resolution, `auto` shall resolve to the only configured agent of the purpose, and with several it shall refuse and list them.
- R5: `pose review dispatch`, `pose review resume` and `pose review status` shall keep their behaviour and exit codes, print one line on stderr naming the `pose delegate` replacement, and be absent from the help listing.
- R6: `.pose/policy/delegation.json` schema 2 shall declare `agents` (vendor, model, CLI, `events`, per-access command templates, session, resume, limit, effort, timeout, executor, containment), `routing` and per-purpose budgets (`purposes.<name>.max_runs_per_bundle`, `timeout`), validated with the rules the schema 1 loader applies today.
- R7: `pose update` shall convert `reviewers.json` schema 1 into `delegation.json` schema 2 deterministically, each adapter becoming an agent whose command is its `read` template, and shall show the conversion in its preview; the engine shall read schema 1 while schema 2 is absent and shall refuse when both files exist.
- R8: The child process shall not receive the parent agent's session variables: a built-in list per known vendor plus the policy's `env_unset`; the run record shall list the removed names, never their values.
- R9: An agent's `events` (`codex-jsonl`, `claude-stream-json` or `plain`) shall select a parser that turns its stream into a versioned normalized event schema — turn, message, tool call, tool output, usage limit, final message, error — kept beside the raw transcript.
- R10: Every subcommand shall emit one versioned JSON document (or NDJSON for streams) with `--json`, and text for a person otherwise.
- R11: A run reference shall accept a run identifier, a `--label` or a session prefix; an ambiguous reference shall exit 2 and list the candidates.
- R12: `--dry-run` shall print the resolved agent, purpose, access, working directory and argv with secrets redacted, and shall reserve no attempt.
- R13: `pose delegate cancel <run>` shall stop the run through its supervisor and record it `failed` with the cause `cancelled`; the attempt shall stay consumed.
- R14: `pose delegate agents` shall list each configured agent with vendor, model, whether its CLI is on the path, and the reset instant of its last recorded usage limit.

### Non-functional

- Help and messages exist in English and pt-BR, as for every command.
- Parsing the policy and the normalized events costs no network access.

### Security

- Command templates are argv arrays; placeholders (`{output}`, `{session}`, `{effort}`) are whole argv elements and never interpolated into code, as the continuity rules require for `{session}`.
- `--dry-run` and `--json` never print secrets, session references or environment values.

### Compatibility

- Instances with `reviewers.json` keep working until they run `pose update`; the aliases stay for one minor version after this spec is released.

## 3. Technical Plan

### Affected areas

CLI command tree and help catalog, dispatch front end, policy loader and migration in `pose update`, child environment, event parsers, run record fields (`label`, `purpose`, removed variable names).

### Artifacts

Implementation artifacts are declared when the spec starts.

### API/contract changes

New command tree `pose delegate`; `delegation.json` schema 2; normalized event schema 1; deprecated aliases.

## 4. Tasks

- [ ] Command tree, vocabulary and help
- [ ] Purposes over the delivered dispatch path
- [ ] Policy schema 2 and migration
- [ ] Environment removal and event normalization
- [ ] `ls`, `cancel`, `agents`, run references, `--dry-run`, `--json`
- [ ] Aliases with deprecation notice

## 5. Decisions

### Decision D1
- Basis: R1, R3
- Minimal option: `pose review dispatch --kind`
- Selected option: purposes as subcommands of `pose delegate`
- Rationale: each purpose carries different guarantees (a task writes and never attests); a flag hides that difference and keeps execution under the subject's command
- Consequences: `pose review` stops starting runs
- Falsifier: a purpose whose guarantees cannot be stated without consulting another purpose's flags

### Decision D2
- Basis: R13
- Minimal option: a new terminal state `cancelled`
- Selected option: `failed` with the cause `cancelled`
- Rationale: the exit-code table of `wait` and every reader of terminal states stay as they are; cancelling is never a way to recover budget
- Consequences: insights distinguish cancellation by cause, not by state
- Falsifier: a consumer that must treat cancellation differently from failure to stay correct

## 6. Validation

### Strategy

Each subcommand parses and rejects as specified; a review run through `pose delegate review` and through the alias produces identical records apart from `label` and `purpose`; a schema 1 fixture migrates to the expected schema 2 bytes and both files together are refused; a fake agent receives none of the listed variables; recorded Codex and Claude streams normalize to the same event sequence; `--dry-run` reserves nothing.

### Requirement trace

## 7. Final Report

### Delivered scope

Not started: milestone `surface` of roadmap `delegation`.

### Residual risks

None yet.

### Follow-ups
