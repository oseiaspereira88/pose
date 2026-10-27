---
slug: pose-delegation-mcp
status: done
created_at: 2026-10-26
completed_at: 2026-10-27
supersedes:
depends_on: pose-delegated-review-completion-signal, pose-delegated-review-round-protocol, pose-delegation-handoff
priority: 3
components: pose-mcp
task_type: feature
delivers: capability:delegation-mcp
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

- modified: .pose/specs/2026-10-26-pose-delegation-mcp.md
- created: .pose/starts/pose-delegation-mcp.json
- created: .pose/changelogs/unreleased/pose-delegation-mcp.md
- created: pose-mcp/internal/mcpserver/delegate_tools.go
- created: pose-mcp/internal/mcpserver/delegate_tools_test.go
- modified: pose-mcp/internal/cli/cli.go
- modified: pose-mcp/internal/cli/delegate_wait.go
- modified: pose-mcp/internal/mcpserver/server.go
- modified: pose-mcp/internal/mcpserver/catalog.go
- modified: pose-mcp/internal/mcpserver/server_test.go
- modified: pose-mcp/internal/mcpserver/testdata/tool-catalog.golden.json
- created: pose-mcp/internal/pose/delegation_refusal.go
- modified: pose-mcp/internal/pose/review_attempt_ledger.go
- modified: pose-mcp/internal/pose/review_run_continuity.go
- modified: pose-mcp/internal/pose/review_adapter_containment.go
- modified: pose-mcp/internal/pose/delegation_handoff.go
- modified: pose-mcp/internal/pose/delegation_task.go
- modified: pose-mcp/internal/pose/review_dispatch.go
- modified: pose-mcp/internal/cli/delegate.go
- modified: pose-mcp/internal/cli/delegate_handoff.go
- modified: pose-mcp/internal/cli/delegate_task.go
- created: pose-mcp/internal/cli/delegate_mcp_test.go
- modified: pose-mcp/internal/cli/help_catalog.go
- modified: docs-site/docs/mcp.md
- modified: POSE.md
- modified: locales/pt-BR/POSE.md
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md

### Delivery targets

- capability:delegation-mcp module:pose-mcp profile:composed-capability entrypoint:pose-mcp/internal/mcpserver/delegate_tools.go

### API/contract changes

Tools `pose_delegate_start`, `pose_delegate_status`, `pose_delegate_wait`, `pose_delegate_resume`, `pose_delegate_handoff`.

## 4. Tasks

- [x] Tools over the shared run lifecycle
- [x] Shared refusal codes
- [x] Bounded wait
- [x] Result redaction
- [x] Independent review and closeout

## 5. Decisions

### Decision D1
- Basis: R5
- Minimal option: a reduced MCP surface without refusal parity
- Selected option: the same refusals and reason codes as the CLI
- Rationale: the safeguard that justified reversing the 2026-10-09 decision is the ledger's refusal; any MCP path around it reopens the approval loop
- Consequences: refusal codes become a shared contract of CLI and MCP
- Falsifier: an MCP sequence that obtains a second run on an unchanged bundle without a reason

### Decision D2
- Basis: R1–R5
- Minimal option: reimplement planning, routing and refusals in the MCP server
- Selected option: each tool runs `pose delegate … --json` in the project the server resolved, with the binary that serves MCP, under the tool's bound, and reduces the CLI's document to the summary keys; refusals are typed in the engine (`DelegationRefusal`) and carry a `reason_code` in the CLI's error document, which the tool returns as is. A start first previews (`--dry-run`) to report the agent, routing and independence, then starts `--detach`. `pose delegate resume` and `handoff` gain `--detach`, which hands the continuation's attempt back once it starts; a continuation re-planned at execution keeps the hand-back and the routing the caller set
- Rationale: one implementation of planning and refusal is the only way an MCP path cannot drift around the ledger's refusal (D1); the server package cannot import the CLI, and the CLI already isolates a detached run in its own session
- Consequences: an MCP call costs a process start; the tool bounds are the CLI's detach bound (150 s for a start, resume or handoff), 30 s for a status and the wait's own bound plus 30 s
- Falsifier: a refusal over MCP that the CLI does not make, or one with another reason code

### Decision D3
- Basis: R6
- Minimal option: make `project_id` a required argument of each tool
- Selected option: the tools follow the server's shared project-scope contract (spec `pose-mcp-project-scope-contract`): `project_id` is optional with the shared description, the default project is used only when it is unambiguous, and an unknown or ambiguous project is refused with the server's typed errors; the server's authorization admits each call before it runs, and write-capable tools are classed `external-side-effect`
- Rationale: the catalog's conformance test forbids a required `project_id` (a default is convenience only); requiring it in five tools would break the contract every other tool keeps, while every call still runs in one resolved, authorized project
- Consequences: R6's "require `project_id`" is met as "resolve a single authorized project"; the trace says so
- Falsifier: a delegate tool that ran in a project the caller could not read, or in an ambiguous selection

### Decision D4
- Basis: R1
- Minimal option: start tasks over MCP
- Selected option: `pose_delegate_start` starts review, adjudicate and smoke runs and refuses a task, which has no detached form yet (follow-up F3 of `pose-delegation-task`) and whose prompt and access are the caller's to choose on the CLI
- Rationale: R1 requires every start to be detached; a task started in the foreground would block the call past its bound
- Consequences: R1 holds for the review purposes; a task over MCP waits for `pose delegate task --detach`
- Falsifier: an MCP call that blocks on a task

## 6. Validation

### Strategy

Over a fake agent, each tool returns the documented shape; a rerun on an unchanged bundle is refused over MCP with the CLI's code; `pose_delegate_wait` returns `still-running` at its bound; results contain no transcript or session reference.

### Requirement trace

- R1 [satisfied] Within D4, review/adjudicate/smoke start detached; task start is deferred. D2 returns an attempt handle first and wait resolves the run (F02). test:TestDelegateToolsOverMCPMatchTheCLI capability:delegation-mcp evidence:integration check:delegation-surface-integration commit:f0514d01404e7bed0cfc602f2db065f3e0c6289e
- R2 [satisfied] Status reports active/terminal state, heartbeat, decision and safe next_actions. test:TestDelegateToolsOverMCPMatchTheCLI test:TestDelegateToolsStatusKeepsFailedAndHidesRunnerErrors evidence:integration check:delegation-surface-integration
- R3 [satisfied] Wait returns still-running at its bound and the terminal result afterward. test:TestDelegateToolsOverMCPMatchTheCLI test:TestDelegateToolsBoundIsTheOperations evidence:integration check:delegation-surface-integration
- R4 [satisfied] Resume/handoff share CLI behavior for review and task runs, including repeated predecessor requests with and without task --record; the attempt identifies the successor through wait (F02). test:TestDelegateToolsOverMCPContinueStoppedRuns test:TestDelegateToolsOverMCPReturnAnExistingContinuation test:TestDelegateToolsOverMCPReturnAnExistingTaskContinuation evidence:integration check:delegation-surface-integration
- R5 [satisfied] Shared typed refusals preserve reason codes across foreground, detach and MCP, including nested routing refusals. test:TestDelegateToolsOverMCPMatchTheCLI test:TestDelegateToolsOverMCPKeepADetachedRoutingRefusal evidence:integration check:delegation-surface-integration
- R6 [satisfied] D3 resolves one authorized project under the shared default/explicit project contract; denial, unknown and ambiguous selection stop before execution, and the resolved root is carried to the child. test:TestDelegateToolsOverMCPHoldAgainstTheFirstReview evidence:integration check:delegation-surface-integration
- R7 [satisfied] Results project safe summaries and fixed diagnostics, including failed after repeated projection and runner startup errors. test:TestDelegateToolsProjectWhatTheyReturn test:TestDelegateToolsStatusKeepsFailedAndHidesRunnerErrors test:TestDelegateToolsOverMCPHoldAgainstTheFirstReview evidence:integration check:delegation-surface-integration

Unnumbered non-functional clause (review coverage label R8, not a published requirement ID): Non-functional bound: one shared deadline per call, operation-specific bounds and bounded wait. test:TestDelegateToolsShareOneDeadlinePerCall test:TestDelegateToolsBoundIsTheOperations evidence:integration check:delegation-surface-integration

Unnumbered security clause (review coverage label R9, not a published requirement ID): Security: existing policy/project authorization and shared CLI ledger/containment refusals remain in force; partial output is not an approval. test:TestDelegateToolsOverMCPHoldAgainstTheFirstReview test:TestDelegateToolsOverMCPMatchTheCLI evidence:integration check:delegation-surface-integration

Unnumbered compatibility clause (review coverage label R10, not a published requirement ID): Compatibility: catalog/golden and the full module suite preserve existing MCP tools while adding delegation. test:TestDelegateToolsOverMCPMatchTheCLI report:.pose/results/delivery-validation.json evidence:integration check:delegation-surface-integration

## 7. Final Report

### Delivered scope

Milestone `agent-access` of roadmap `delegation`: five bounded MCP tools start, inspect, wait for, resume and hand off delegated runs through the shared CLI/domain lifecycle, attempt ledger, authorized project root and typed refusal codes. Results project the normalized summary without private transcript, prompt, session, environment values or host diagnostics. Existing review and task continuations return idempotently through MCP and detached CLI, with task history both private and recorded; a consumed handback is not recreated at terminal.

D3 follows the shared explicit/default project selection contract. D4 limits MCP start to review, adjudicate and smoke; starting a task over MCP remains deferred. F02 preserves the attempt-first identity contract: wait returns the run. This delivery does not close the broader roadmap or its remaining members.

### Independent review and evidence

Approved implementation commit `f0514d01404e7bed0cfc602f2db065f3e0c6289e`, sealed bundle `rvb-557e630823d2fe7a`, attestation `rva-7d020e952840064f`. The maintainer-authorized reviewer `agent:independent-gpt-6.1-sol-review` declares Codex/OpenAI, another vendor than implementer Claude/Anthropic, Claude Opus 5.5. The engine-required independent prefix is a declaration, not authenticated separation; this instance uses unsigned attestations and does not observe cognitive independence. The seal explicitly names `anthropic/claude-opus-5-5`; its missing Anthropic execution/session warning is disclosed and no execution digest was invented. The native domain accepted the explicit review; verify returned fresh, approved and ready-to-close before the guarded lifecycle transition.

All blockers B01–B07 are fixed in the reexecuted cases. F01/F04/F05/F06/F07/F08 are reconciled as fixed; F02 and F03 are accepted only at low severity below and in the operability judgment. Their native finding IDs are `f02` and `f03`. Expiring context and reproduction recipes are retained in knowledge:delegation-mcp-round5, consuming knowledge:delegation-observation-round4, knowledge:delegation-routing-round3 and knowledge:delegated-rounds-legacy-status. Local reports, probes and overlays remain in `.review-work/round1/` through `.review-work/round5/`, deliberately untracked.

### Validation executed

Global strict validation passed 158/158 checks at the implementation head; separate strict pose-mcp validation passed 155/155, with no failures, errors or skips. Canonical machine evidence is `.pose/results/delivery-validation.json`. docs-site declares no validation producer: its selector executed zero checks and its tool disposition explicitly records not-used rather than borrowing Go evidence. Catalog, help and root/scaffold English/Portuguese manuals were compared with runtime behavior.

Independent vet, Windows/Darwin builds, both locale full Go suites (locale C without session markers), selected race checks and redacted Gitleaks passed. All prior probe families and TestRound4 cases were reexecuted. Seven production reversions failed the expected assertions, including omission of existing task handback and removal of the once-only guard. Replay probes preserve attempt, successor and two task runs; the late-handback probe leaves zero residual handbacks.

Strict artifact reconciliation observed all 28 declared artifacts in five implementing commits. Strict surface-check verified capability:delegation-mcp through its production handler and current integration evidence with zero findings. Rule selection, bounded design, component discovery, provider/consumer and debt assessments were executed. The broad debt scan remains incomplete for two unchanged old invalid specs; static unobserved-consumer diagnostics are disclosed limits, not claims of complete repository-wide assurance. Output from discovery and assessments is retained locally and excluded from this closeout commit; only state files produced by native close are included.

The spec publishes R1–R7. The three unnumbered non-functional/security/compatibility clauses are covered in Requirement trace using review labels R8–R10, explicitly not invented published requirement IDs; native close refused such orphan IDs before the trace representation was corrected. Requirements, Technical Plan and Decisions were preserved.

### Residual risks

F02 limits initial identity presentation; the attempt remains observable and wait resolves the run. F03 is a recovery window of intentional detached execution: cancellation of the transport after reservation can leave the caller without a response while the discoverable run continues. Neither permits duplicate execution beyond the ledger, budget bypass or acceptance of task/partial output as governance evidence. Fake-agent checks establish engine behavior, not a provider receipt or guaranteed reviewer cognition.

### Follow-ups

- [open] F02: Align the public identity contract with the initial attempt handle or return the new run identity at handback, retaining continuation retry regressions. See knowledge:delegation-mcp-round5. (owner:@oseiaspereira88 crit:low review:2026-11-26)
- [open] F03: Document recovery and correlate retries after transport timeout/cancellation; consult ls/status before repeating and keep explicit cancellation separate. See knowledge:delegation-mcp-round5. (owner:@oseiaspereira88 crit:low review:2026-11-26)
