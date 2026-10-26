---
slug: pose-delegation-surface
status: in-progress
created_at: 2026-10-26
completed_at:
supersedes:
depends_on: pose-delegated-adapter-containment, pose-delegated-review-run-continuity, pose-delegated-review-attempt-ledger
priority: 2
components: pose-mcp
task_type: feature
delivers: capability:delegation-surface
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
- R6: `.pose/policy/delegation.json` schema 2 shall declare `agents` (vendor, model, CLI, `events`, per-access command templates, session, resume, limit, effort, timeout, executor, containment, and an optional own `max_runs_per_bundle`), `routing` and per-purpose budgets (`purposes.<name>.max_runs_per_bundle`, `timeout`), validated with the rules the schema 1 loader applies today.
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

- modified: .pose/specs/2026-10-26-pose-delegation-surface.md
- created: .pose/starts/pose-delegation-surface.json
- created: .pose/changelogs/unreleased/pose-delegation-surface.md
- modified: .pose/indexes/validation-matrix.json
- created: pose-mcp/internal/pose/delegation_policy.go
- created: pose-mcp/internal/pose/delegation_policy_test.go
- created: pose-mcp/internal/pose/delegation_env.go
- created: pose-mcp/internal/pose/delegation_events.go
- created: pose-mcp/internal/pose/delegation_events_test.go
- created: pose-mcp/internal/pose/delegation_active.go
- created: pose-mcp/internal/pose/delegation_list.go
- modified: pose-mcp/internal/pose/review_dispatch.go
- modified: pose-mcp/internal/pose/review_run_continuity.go
- modified: pose-mcp/internal/pose/review_run_provenance.go
- created: pose-mcp/internal/cli/delegate.go
- created: pose-mcp/internal/cli/delegate_test.go
- modified: pose-mcp/internal/cli/cli.go
- modified: pose-mcp/internal/cli/help_catalog.go
- modified: pose-mcp/internal/cli/review_closeout.go
- modified: pose-mcp/internal/cli/review_run_continuity.go
- modified: pose-mcp/internal/cli/review_dispatch_test.go
- modified: pose-mcp/internal/cli/stack_seed.go
- modified: pose-mcp/internal/cli/check.go
- modified: POSE.md
- modified: locales/pt-BR/POSE.md
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md
- modified: docs-site/docs/cli.md
- modified: docs-site/docs/review-and-trust.md

### Delivery targets

- capability:delegation-surface module:pose-mcp profile:composed-capability entrypoint:pose-mcp/internal/cli/delegate.go

### API/contract changes

New command tree `pose delegate`; `delegation.json` schema 2; normalized event schema 1; deprecated aliases.

## 4. Tasks

- [x] Command tree, vocabulary and help
- [x] Purposes over the delivered dispatch path
- [x] Policy schema 2 and migration
- [x] Environment removal and event normalization
- [x] `ls`, `cancel`, `agents`, run references, `--dry-run`, `--json`
- [x] Aliases with deprecation notice
- [ ] Independent review and closeout

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

### Decision D3
- Basis: R13
- Minimal option: signal the running `pose` process (SIGTERM) and classify like any termination
- Selected option: a private marker per running attempt, naming its process as a start lock does, and a cancellation request file the running process watches; the run is stopped through its own context
- Rationale: a signal reads as an interruption and the delivered continuity records it `interrupted` and resumable, which a deliberate cancel is not; a file request works the same on every platform and leaves an accidental Ctrl-C resumable as before
- Consequences: `.pose/review-private/active/` holds runtime markers; a marker whose owner is proven dead is reported, never cancelled
- Falsifier: a cancelled run recorded `interrupted`, or a run stopped by a request meant for another attempt

### Decision D4
- Basis: R6, R7 and the security rule on placeholders
- Minimal option: convert every schema 1 adapter as it is
- Selected option: schema 2 accepts `{output}`, `{session}` and `{effort}` only as whole argv elements; a schema 1 adapter that interpolates one into an argument is not converted, and `pose update` warns and keeps `reviewers.json`
- Rationale: interpolating a path or a session reference into code is how a placeholder becomes an injection; schema 2 is the point where the stricter rule can be applied without breaking a recorded run
- Consequences: such an instance stays on schema 1, which keeps working, until it rewrites the template
- Falsifier: a schema 2 policy that runs an argument containing a placeholder inside it

### Decision D5
- Basis: R6, R7
- Minimal option: budgets only per purpose
- Selected option: per-purpose budgets, plus an optional budget on the agent that wins over the purpose's; the conversion moves a budget all adapters share to the purposes and keeps unshared budgets on each agent
- Rationale: every converted agent must project exactly the adapter it came from, because run provenance verifies the digest of that configuration; budgets that differ per adapter cannot be expressed per purpose
- Consequences: two places can hold a budget, and the agent's wins
- Falsifier: a converted policy whose projection differs from the schema 1 adapter, or a run whose provenance stops verifying after `pose update`

### Decision D6
- Basis: R9
- Minimal option: the draft of an agent without an `{output}` file is the transcript's tail
- Selected option: when the agent declares a streaming `events` format and writes no output file, its draft is the final message its stream carries (Claude's result, Codex's last agent message); provenance records the source as `stream-final`
- Rationale: a streaming agent's transcript is the event stream itself, so its tail is JSON rather than a conclusion, and a stream longer than the tail bound would fail capture
- Consequences: a stream that carries no final message keeps the delivered behaviour
- Falsifier: a streaming run whose recorded draft is not the agent's final message

## 6. Validation

### Strategy

Each subcommand parses and rejects as specified; a review run through `pose delegate review` and through the alias produces identical records apart from `label` and `purpose`; a schema 1 fixture migrates to the expected schema 2 bytes and both files together are refused; a fake agent receives none of the listed variables; recorded Codex and Claude streams normalize to the same event sequence; `--dry-run` reserves nothing.

### Requirement trace

- R1 [satisfied] check:delegation-surface-integration test:TestDelegateTreeParsesAndRefusesAsSpecified
- R2 [satisfied] check:delegation-surface-integration test:TestDelegateTreeParsesAndRefusesAsSpecified test:TestDelegateAutoAgentAndDryRun
- R3 [satisfied] check:delegation-surface-integration test:TestDelegateTreeParsesAndRefusesAsSpecified test:TestReviewDispatchRunsAndRecordsAdapters
- R4 [satisfied] check:delegation-surface-integration test:TestDelegateAutoAgentAndDryRun
- R5 [satisfied] check:delegation-surface-integration test:TestDelegateReviewRecordsWhatTheAliasRecords
- R6 [satisfied] check:delegation-surface-integration test:TestDelegationPolicyProjectsPerPurposeBudgets test:TestDelegationPolicyRefusesWhatItCannotStateSafely
- R7 [satisfied] check:delegation-surface-integration test:TestDelegationMigrationProjectsTheSameAdapters test:TestDelegationMigrationIsDeterministic test:TestDelegationMigrationKeepsUnsharedBudgetsPerAgent test:TestDelegationPolicyFilesAreExclusive test:TestUpdateConvertsTheReviewersPolicy test:TestUpdateKeepsAReviewersPolicyItCannotConvertSafely
- R8 [satisfied] check:delegation-surface-integration test:TestDelegateChildReceivesNoParentSession
- R9 [satisfied] capability:delegation-surface evidence:integration check:delegation-surface-integration test:TestDelegationEventsAreTheSameForBothVendors test:TestDelegationEventsReportUsageLimits test:TestDelegationEventsArePlainAndBounded test:TestDelegationEventsAreBoundToTheRecord test:TestDelegateListsAndNamesRuns
- R10 [satisfied] check:delegation-surface-integration test:TestDelegateTreeParsesAndRefusesAsSpecified test:TestDelegateReviewRecordsWhatTheAliasRecords test:TestDelegateAgentsShowsAvailability
- R11 [satisfied] check:delegation-surface-integration test:TestDelegateListsAndNamesRuns
- R12 [satisfied] check:delegation-surface-integration test:TestDelegateAutoAgentAndDryRun
- R13 [satisfied] check:delegation-surface-integration test:TestDelegateCancelRecordsAFailedCancelledRun
- R14 [satisfied] check:delegation-surface-integration test:TestDelegateAgentsShowsAvailability

## 7. Final Report

### Delivered scope

Not started: milestone `surface` of roadmap `delegation`.

### Residual risks

None yet.

### Follow-ups
