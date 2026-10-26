---
slug: pose-delegated-review-completion-signal
status: draft
created_at: 2026-10-18
completed_at:
supersedes:
depends_on: pose-delegated-review-dispatch, pose-delegated-review-attempt-ledger, pose-delegation-surface
priority: 2
components: pose-mcp
task_type: feature
changelog:
delivers:
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

Implementation artifacts are declared when the spec starts.

### API/contract changes

`pose delegate <purpose> --detach`, `pose delegate wait`, `ls` and `watch`, `pose_delegate_status`, `on_terminal`; documented exit codes.

## 4. Tasks

- [ ] Supervisor, heartbeat and stalled detection
- [ ] Atomic terminal transition and marker
- [ ] `wait` with distinguishing exit codes
- [ ] `ls` and the MCP status tool
- [ ] `watch` with chain replay, picker and colour rules
- [ ] `on_terminal` hook under containment

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
