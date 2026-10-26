---
slug: pose-delegated-review-completion-signal
status: in-progress
created_at: 2026-10-18
completed_at:
supersedes:
depends_on: pose-delegated-review-dispatch, pose-delegated-review-attempt-ledger, pose-delegation-surface
priority: 2
components: pose-mcp
task_type: feature
changelog:
delivers: capability:delegation-observation
---

# Spec: A delegated run can be watched and announces that it finished

## 1. Intent

### Goal

Let whoever started a delegated run (a person, an agent, a CI step, a runner such as Harne8's Conductor) start it without holding the process, follow it live, and be woken by a plain process exit when it reaches a terminal state, with the result in the exit code and in one machine-readable line.

### Business value

Origin: the B06 review rounds (2026-10-17). The lead agent launched the reviewer as a background process and expected to be resumed when it ended. On the later rounds it was not: the launching command returned at once, the host saw a finished task, and the real review ran unobserved until the maintainer reported that it had finished. The behaviour the lead needs exists in every host as "a tracked process exits", but nothing in POSE offers a process whose lifetime equals the review's, so each launcher reinvented it, and got it wrong in a different way.

Amended on 2026-10-26 by the [delegation ADR](../adr/2026-10-26-delegation-is-a-first-class-execution-surface.md): the commands move under `pose delegate` and cover every purpose, and the spec gains the live watch, the run listing and the picker that the personal watch launcher provides today (R9–R12), so that launcher can retire.

### Constraints

POSE shall not depend on any host's notification feature; it offers a process to wait on and state to read, and a documented recipe per host belongs to the capability spec. Terminal transitions use the exclusive publisher. No new daemon: the supervisor of a detached run is an ordinary process the run itself started.

### Non-goals

Push notifications to chat or email; resuming an interrupted run (`pose-delegated-review-run-continuity`); choosing the reviewer.

## 2. Requirements

### Functional

- R1: `pose delegate <purpose> --detach` shall start the run under a supervisor that survives the caller's terminal, shall return the run identifier and the paths of its record at once, and shall exit 0 without waiting.
- R2: `pose delegate wait <run> [--timeout <d>]` shall block until the run is terminal and shall exit with a code that distinguishes `approved` (0), `changes-requested` or `rejected` (10), `failed` (20), `interrupted` (30), `stalled` (40) and `timeout of the wait` (50), after printing one line of JSON (`--json`) or text with the run, state, decision, resumable flag and reset instant. Usage errors exit 2.
- R3: `pose delegate wait` shall be a plain foreground process whose lifetime is the review's, so that a host tracking the process is woken by its exit; it shall not poll the network and shall not depend on a host-specific feature.
- R4: The supervisor shall write a heartbeat to the run record at a declared interval; a run whose heartbeat is older than three intervals and whose supervisor process identity (pid and start time) is gone shall be recorded `stalled` by the next reader, with the transition published by compare-and-swap so concurrent readers agree.
- R5: The terminal transition shall be published atomically, state before the `done` marker; a reader that sees the marker shall find the terminal state and the digest of the draft.
- R6: `pose delegate ls [<run>|<bundle>]` and the MCP tool `pose_delegate_status` (`pose-delegation-mcp`) shall report the state without blocking, including heartbeat age, so a caller can tell a running run from a stalled one; the MCP tool shall never block longer than its own bounded timeout.
- R7: `delegation.json` may declare `on_terminal` for an agent or a purpose, a command run once after the terminal transition with the run identifier and state in its environment and nothing else; its failure shall be recorded on the run and shall not change the run's state; it runs under the adapter containment policy.
- R8: Many waiters on one run shall all be released; waiting on an already terminal run shall return at once with the same code.
- R9: `pose delegate watch <run|label>` shall follow the run's normalized events live and stop by itself at the terminal transition; `--no-follow` shall print what was recorded so far and exit; `-v` shall add the tail of each tool output, bounded by `--lines <n>` (default 12); `--json` shall stream the normalized events as NDJSON.
- R10: `watch` shall show an attempt as one task: the runs it resumed or handed off from are replayed first, with a distinct marker at each resume and each handoff, and an interrupted run shall name the run that continued it.
- R11: In a terminal, `watch` with no argument shall follow the only active run and otherwise open a picker — fzf when it is on the path, with a live preview of the run under the cursor refreshed at a configurable interval (0 turns it off), a numbered menu without it; outside a terminal, an ambiguous or missing reference shall exit 2 and list the candidates.
- R12: `pose delegate ls` shall list attempts by their latest run with state, purpose, agent, label, turns, age, last step and heartbeat age; colours shall follow `NO_COLOR` and be off when the output is not a terminal, in `ls` and in `watch`.

### Non-functional

- The wait costs no CPU while idle and holds no lock on the run record.
- A killed `pose delegate wait` or `watch` leaves the run running.

### Security

- `on_terminal` receives no secrets, no draft content and no path outside the run directory.
- The status of a run in another project is not readable without that project's read authority.

### Compatibility

- A foreground `pose delegate` run, and the deprecated `pose review dispatch`, keep their present behaviour and exit codes; `--detach`, `wait`, `watch` and the extended `ls` are additions.

## 3. Technical Plan

### Affected areas

Supervisor, run record heartbeat and terminal marker, `pose delegate wait`, `ls` and `watch`, the picker, the MCP status tool, the `on_terminal` policy key.

### Artifacts

- modified: .pose/specs/2026-10-18-pose-delegated-review-completion-signal.md
- created: .pose/starts/pose-delegated-review-completion-signal.json
- modified: pose-mcp/internal/pose/delegation_active.go
- modified: pose-mcp/internal/pose/delegation_list.go
- created: pose-mcp/internal/pose/delegation_wait.go
- created: pose-mcp/internal/pose/delegation_wait_test.go
- created: pose-mcp/internal/pose/delegation_supervision_linux_test.go
- created: pose-mcp/internal/pose/delegation_live.go
- created: pose-mcp/internal/pose/delegation_live_test.go
- created: pose-mcp/internal/pose/delegation_observation_review_test.go
- created: pose-mcp/internal/cli/delegate_observation_review_test.go
- modified: .gitleaksignore
- modified: pose-mcp/internal/pose/delegation_routing_review_test.go
- created: pose-mcp/internal/pose/delegation_hook.go
- modified: pose-mcp/internal/pose/delegation_events.go
- modified: pose-mcp/internal/pose/delegation_policy.go
- created: pose-mcp/internal/cli/delegate_watch.go
- created: pose-mcp/internal/cli/delegate_watch_test.go
- modified: pose-mcp/internal/cli/cliout/state.go
- modified: pose-mcp/internal/pose/review_dispatch.go
- modified: pose-mcp/internal/cli/delegate.go
- created: pose-mcp/internal/cli/delegate_wait.go
- created: pose-mcp/internal/cli/delegate_wait_test.go
- created: pose-mcp/internal/cli/delegate_detach_unix.go
- created: pose-mcp/internal/cli/delegate_detach_windows.go
- modified: pose-mcp/internal/cli/delegate_test.go
- modified: pose-mcp/internal/cli/cli_test.go
- modified: pose-mcp/internal/cli/help_catalog.go
- modified: POSE.md
- modified: locales/pt-BR/POSE.md
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md

### Delivery targets

- capability:delegation-observation module:pose-mcp profile:composed-capability entrypoint:pose-mcp/internal/cli/delegate.go

### API/contract changes

`pose delegate <purpose> --detach`, `pose delegate wait`, `ls` and `watch`, `pose_delegate_status`, `on_terminal`; documented exit codes.

## 4. Tasks

- [x] Supervisor, heartbeat and stalled detection
- [x] Atomic terminal transition and marker
- [x] `wait` with distinguishing exit codes
- [x] `ls` with heartbeat age and stall; the MCP status tool is delivered by `pose-delegation-mcp`, as R6 names
- [x] `watch` with chain replay, picker and colour rules
- [x] `on_terminal` hook under containment
- [x] First independent review: changes requested (B01–B09; F01–F04), B01–B09 and F02–F04 addressed with regression tests that fail on 2e015c4; F01 (redacted live events a failed capture later withholds) stays a disclosed follow-up of D7
- [x] Second independent review: B01–B07, B09 and F02–F04 accepted as fixed and the exact gitleaks entry accepted; B08 partial and the new follow-up N01 addressed with regression tests that fail on 939c5ab; F01 stays disclosed under D7
- [x] Third independent review (Claude Sonnet 5.5, same vendor, another model): B08 and N01 accepted as fixed; the new blocker N02 and the follow-ups F-R3-01, F-R3-03, F-R3-04, F-R3-06, F-R3-08 and F-R3-10 addressed, with regression tests that fail on 9377ac7 where the behaviour is testable; F01, F-R3-02, F-R3-05, F-R3-07 and F-R3-09 stay follow-ups for the closeout
- [ ] Fourth independent review and closeout

## 5. Decisions

### Decision D1
- Basis: R2, R3
- Minimal option: document that a caller should poll `review status`
- Selected option: a blocking `review wait` whose exit is the signal
- Rationale: every agent host and CI system already wakes on a process exit; polling makes each caller invent an interval, and a short-lived launcher is how the review was lost on 2026-10-17
- Consequences: the exit-code table is a public contract
- Falsifier: a host that tracks processes and is not woken, or a wait that exits while the review still runs

### Decision D2
- Basis: R4
- Minimal option: pid file only
- Selected option: pid plus start time plus heartbeat
- Rationale: a pid is reused; a heartbeat alone cannot tell a slow reviewer from a dead one
- Consequences: a long silent reviewer step must still heartbeat from the supervisor, not from the vendor
- Falsifier: a live run recorded `stalled`, or a dead run never recorded

### Decision D3
- Basis: R1
- Minimal option: return the run identifier
- Selected option: return the attempt identifier, a run reference every lifecycle command takes
- Rationale: a run identifier is derived from the run's recorded content and exists only when the run ends; the attempt is reserved before the agent starts
- Consequences: `wait`, `ls`, `watch` and `cancel` take the attempt; the run identifier appears in their output once the run ends
- Falsifier: a detached run whose returned reference no lifecycle command resolves

### Decision D4
- Basis: R4, R5
- Minimal option: rewrite the marker with the stall under a digest expectation
- Selected option: publish an exclusive stall claim beside the marker (a hard link that fails when it exists); the marker is never removed by a reader, and the record is published before the running process removes its marker
- Rationale: a digest expectation is checked before a rename and two readers can both pass it; an exclusive link cannot be won twice, so every reader returns the first reader's instant
- Consequences: a stalled marker and its claim stay in the private directory; the attempt stays consumed and is not resumable, as a lost host is
- Falsifier: two readers returning different stall instants, or a run recorded stalled while its process runs

### Decision D5
- Basis: R1, routing R7
- Minimal option: detach a run whose route must first wait for a usage-limit reset, and hand back nothing until it starts
- Selected option: refuse to detach such a run, naming the reset
- Rationale: R1 promises the reference at once; a reset can be hours away, and a supervisor waiting with nothing to hand back is the unobserved run this spec exists to end
- Consequences: the caller runs it in the foreground, where the wait is visible, or detaches after the reset
- Falsifier: a detached run that waited for a reset before handing back its attempt

### Decision D6
- Basis: R7
- Minimal option: write the hook's outcome into the run record
- Selected option: record it beside the run, in the engine's private state, keyed by the run identifier, and show it in `ls`
- Rationale: the run record is published immutable and bound by digest to its ledger receipt and proof; the hook runs after that publication and may never change the run
- Consequences: the outcome is local to the instance that ran the run, like the supervisor's log
- Falsifier: a hook whose failure changed a run's state, its record or its digest

### Decision D7
- Basis: R9, security
- Minimal option: follow the raw stream and redact at the end
- Selected option: normalize and redact each complete line as it arrives, with the run's credentials and every session reference the stream has declared so far, into a private file removed when the record is published
- Rationale: a watcher must never see a credential the record would withhold; a session reference is known only once the stream declares it
- Consequences: a session reference printed before the line that declares it can reach the live file, which is private and short-lived; the events bound to the record are derived again from the whole transcript and redacted with it; a line is read whole whatever its length, within the capture's stream budget, so the live events keep the record's numbering; and a watcher may already have seen redacted events that a capture failing later withholds from the record
- Falsifier: a credential the record redacts appearing in a live event

## 6. Validation

### Strategy

A fake adapter exercises approved, changes-requested, failed, interrupted and killed runs; `wait` returns the documented code for each, is released together with a second waiter, and survives being killed itself; a killed supervisor is recorded `stalled` by exactly one of two concurrent readers; the hook failing leaves the state unchanged. Recorded Codex and Claude streams replay through `watch` with the same output, a resumed and a handed-off chain show their markers in order, follow mode ends at the terminal transition, the picker falls back to a numbered menu without fzf, an ambiguous reference outside a terminal exits 2, and `NO_COLOR` removes every colour.

### Requirement trace

## 7. Final Report

### Delivered scope

Not started: opened on 2026-10-18 under roadmap `delegated-review` from the operating experience of the B06 review rounds (see Origin); moved on 2026-10-26 to milestone `observation` of roadmap `delegation`.

### Residual risks

None yet.

### Follow-ups
