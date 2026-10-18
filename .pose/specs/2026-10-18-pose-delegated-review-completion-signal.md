---
slug: pose-delegated-review-completion-signal
status: draft
created_at: 2026-10-18
completed_at:
supersedes:
depends_on: pose-delegated-review-dispatch, pose-delegated-review-attempt-ledger
priority: 2
components: pose-mcp
task_type: feature
changelog:
delivers:
---

# Spec: A delegated review announces that it finished

## 1. Intent

### Goal

Let whoever started a delegated review (an agent, a CI step, a runner such as Harne8's Conductor) start it without holding the process, and be woken by a plain process exit when it reaches a terminal state, with the result in the exit code and in one machine-readable line.

### Business value

Origin: the B06 review rounds (2026-10-17). The lead agent launched the reviewer as a background process and expected to be resumed when it ended. On the later rounds it was not: the launching command returned at once, the host saw a finished task, and the real review ran unobserved until the maintainer reported that it had finished. The behaviour the lead needs exists in every host as "a tracked process exits", but nothing in POSE offers a process whose lifetime equals the review's, so each launcher reinvented it, and got it wrong in a different way.

### Constraints

POSE shall not depend on any host's notification feature; it offers a process to wait on and state to read, and a documented recipe per host belongs to the capability spec. Terminal transitions use the exclusive publisher. No new daemon: the supervisor of a detached run is an ordinary process the run itself started.

### Non-goals

Push notifications to chat or email; resuming an interrupted run (`pose-delegated-review-run-continuity`); choosing the reviewer.

## 2. Requirements

### Functional

- R1: `pose review dispatch --detach` shall start the run under a supervisor that survives the caller's terminal, shall return the run identifier and the paths of its record at once, and shall exit 0 without waiting.
- R2: `pose review wait <run> [--timeout <d>]` shall block until the run is terminal and shall exit with a code that distinguishes `approved` (0), `changes-requested` or `rejected` (10), `failed` (20), `interrupted` (30), `stalled` (40) and `timeout of the wait` (50), after printing one line of JSON (`--json`) or text with the run, state, decision, resumable flag and reset instant. Usage errors exit 2.
- R3: `pose review wait` shall be a plain foreground process whose lifetime is the review's, so that a host tracking the process is woken by its exit; it shall not poll the network and shall not depend on a host-specific feature.
- R4: The supervisor shall write a heartbeat to the run record at a declared interval; a run whose heartbeat is older than three intervals and whose supervisor process identity (pid and start time) is gone shall be recorded `stalled` by the next reader, with the transition published by compare-and-swap so concurrent readers agree.
- R5: The terminal transition shall be published atomically, state before the `done` marker; a reader that sees the marker shall find the terminal state and the digest of the draft.
- R6: `pose review status <run|bundle>` and the MCP tool `pose_review_run_status` shall report the state without blocking, including heartbeat age, so a caller can tell a running review from a stalled one; the MCP tool shall never block longer than its own bounded timeout.
- R7: `reviewers.json` may declare `on_terminal`, a command run once after the terminal transition with the run identifier and state in its environment and nothing else; its failure shall be recorded on the run and shall not change the run's state; it runs under the adapter containment policy.
- R8: Many waiters on one run shall all be released; waiting on an already terminal run shall return at once with the same code.

### Non-functional

- The wait costs no CPU while idle and holds no lock on the run record.
- A killed `pose review wait` leaves the review running.

### Security

- `on_terminal` receives no secrets, no draft content and no path outside the run directory.
- The status of a run in another project is not readable without that project's read authority.

### Compatibility

- A foreground `pose review dispatch` keeps its present behaviour and exit codes; `--detach` and `wait` are additions.

## 3. Technical Plan

### Affected areas

Dispatch front end and supervisor, run record heartbeat and terminal marker, `review wait` and `review status`, MCP tool, `reviewers.json` reader.

### Artifacts

Implementation artifacts are declared when the spec starts.

### API/contract changes

`review dispatch --detach`, `review wait`, `review status`, `pose_review_run_status`, `on_terminal`; documented exit codes.

## 4. Tasks

- [ ] Supervisor, heartbeat and stalled detection
- [ ] Atomic terminal transition and marker
- [ ] `review wait` with distinguishing exit codes
- [ ] `review status` and the MCP tool
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

A fake adapter exercises approved, changes-requested, failed, interrupted and killed runs; `wait` returns the documented code for each, is released together with a second waiter, and survives being killed itself; a killed supervisor is recorded `stalled` by exactly one of two concurrent readers; the hook failing leaves the state unchanged.

### Requirement trace

## 7. Final Report

### Delivered scope

Not started: part of roadmap `delegated-review`, opened on 2026-10-18 from the operating experience of the B06 review rounds (see Origin).

### Residual risks

None yet.

### Follow-ups
