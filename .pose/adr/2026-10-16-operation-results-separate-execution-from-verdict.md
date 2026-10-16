---
title: Operation results separate execution from verdict
status: accepted
date: 2026-10-16
decision_type: architecture
---

# Operation results separate execution from verdict

## Context

Spec `pose-cli-mcp-operation-results` (B12) reproduced, on the 7.4.1 engine,
five ways a call reports something other than what happened (front 7 probes
P03, P06, P07, P08; front 8 PERF-07; front 9 CONS-13):

- `pose close --apply|--resume --json` printed `closeout_plan.result=closed`
  text, and printed it for a spec that was already done, where no step ran;
- `pose specs` accepted an unknown flag, an unparseable `--since` and a
  non-numeric `--recent` and returned every spec;
- `pose start --apply --status` ran the status and dropped the apply, and
  `pose close --apply --plan` ran whichever mode came last;
- `pose continuous-closeout status --json` printed text;
- an MCP gate (`pose_check`, `pose_lint_spec`, `pose_skills_check`) whose
  subprocess hit its deadline returned `passed=false, exit_code=-1`: the
  killed process was read as a failed gate, because `*exec.ExitError` was
  classified before the context;
- `pose close` exited 3 at a pending judgment and the usage journal recorded
  it as `failed`, and derived signal events were recorded with a 0 ms
  duration nobody measured.

ADR-003 (MCP is an adapter, not a fork) and the shared-domain ADR of
2026-10-14 fix where the decision lives; neither defines the shape of a
result. The recoverable closeout plan (2026-10-04) defines the steps and the
checkpoint, not what an apply reports.

## Options considered

1. **One result for every command.** Replace every JSON output with an
   envelope. Breaks every documented `--json` consumer (plans, start records,
   gates) for no new fact.
2. **A shared result for operations that run steps or wait, typed errors for
   execution failures, strict parsers elsewhere.** New JSON only where a call
   had none or printed text in place of the JSON it accepted; existing JSON
   documents keep their shape.
3. **Fix each probe locally.** Leaves `closed` meaning both "this call closed
   the spec" and "nothing to do", and the next route would re-invent it.

Option 2 is chosen.

## Decision

`internal/pose.OperationResult` (schema 1) is the result of an operation that
runs steps or can wait: `operation`, `scope`, `outcome`, the `step` it stopped
at or finished with, `steps_run` (what this call executed, in order),
`pending`, `next_action`, `effects` (confirmed by the domain), `unconfirmed`
(steps that failed after they may have written), `resumable`, `error` with a
typed `cause`, and `data` for a query's payload.

Outcomes are disjoint and name the execution, never a gate verdict:

| outcome | meaning | CLI exit |
|---|---|---|
| `completed` | every step this call ran succeeded; for a query, the query ran | 0 |
| `already-satisfied` | no step ran in this call; the stored state already says done. Not a fresh verification of any gate | 0 |
| `waiting` | stopped on another actor; `pending` and `next_action` say who and how to continue | 3 |
| `blocked` | the plan refuses the next step; nothing ran for it | 1 |
| `conflict` | the input digest no longer matches the repository | 1 |
| `failed` | a step ran and failed; `effects` lists what is confirmed, `unconfirmed` what may have been written | 1 |
| `inconclusive` | deadline or cancellation; no verdict | 1 |

Exit codes keep their documented meaning; only the outcome is new. A query's
readiness stays in its payload (`continuous-closeout status` keeps exit 1 for
a non-terminal scope), so a query that ran is `completed` whether or not the
scope is ready.

Rendering and exit mapping live in the adapters. The human renderer and the
JSON renderer read the same `OperationResult`; with `--json` stdout carries
exactly one JSON document and no progress text, on success, wait and failure.

Routes:

- `pose close --apply|--resume [--json]` returns an `OperationResult`. Its
  human output keeps the `closeout_plan.*` lines the quickstart reads, and a
  call that ran no step reports `closeout_plan.result=already-satisfied`
  instead of `closed`. `--plan --json` keeps the `CloseoutPlan` document.
- `pose continuous-closeout status --json` returns an `OperationResult` whose
  `data` holds `active`, `scope` and `terminal`.
- `pose start` keeps the plan, status and record documents; its modes become
  exclusive and a repeated `--apply` returns the record as JSON with `--json`.
- A route that supports `--json` and keeps its legacy success document
  (`start` in every mode, `close --plan`, `specs`, `specs facts`) reports an
  operational failure as an `OperationResult` with outcome `failed` and a
  typed cause on stdout, plus the human line on stderr; the exit stays 1.
- An effect is confirmed by observing the artifact, never the run's text
  (a failing check's output is replayed in it and can say anything): the
  evidence step confirms `results-written` only when the results path is a
  regular file holding a JSON document whose bytes differ from those it held
  before the run and whose `generated_at` is not older than the run. A write
  that failed leaves the earlier bytes and confirms nothing. `validate` exits
  1 both for a failing check and for a results file it could not write.
- A `pose start --apply` that fails after the domain published its baseline
  reads the start record back: the recorded phase is a confirmed effect, the
  spec transition is `unconfirmed`, and the result is resumable with the same
  digest. Repeating it resumes from the record (`Store.ResumeStart`), also
  after the spec already moved and its fresh preview has another digest. A
  spec changed under the start (`compare-and-swap-conflict`) is not
  resumable, and a record for another plan is a `conflict` this call did not
  write.
- `pose close <scope>` without a plan mode has no JSON result: `--json` there
  is refused with exit 2 and the help no longer advertises it.
- Parsers of `close`, `start`, `specs` and `continuous-closeout` refuse,
  with exit 2 and before any read that could widen the result: two different
  modes, a flag the mode does not take, an unknown flag, a missing value, an
  unparseable date, a negative day count and a non-positive number. Documented
  aliases (`--component`, a bare `--recent`) keep working; a day count is
  bounded (100000 days) so the window cannot overflow. `specs facts` takes one
  slug and `--json` in any position. MCP `pose_list_specs` decodes its
  arguments strictly and refuses the same values, an unknown argument, an
  explicit `0` for `recent` or `limit`, an explicit `null` for any filter, an
  empty `since` and a `sort` other than `date` or `slug` (the two orders its
  catalog advertises); omitted values keep the unfiltered, unpaginated
  listing.
- A gate subprocess is classified by its context first: a deadline or a
  cancellation returns `*GateExecutionError` (cause `deadline-exceeded` or
  `canceled`, the command, the partial output and a cleanup note), which MCP
  renders as `isError` with `error_code: execution_inconclusive`, never as
  `passed=false`. The process is stopped with a bounded wait for its pipes; a
  process it started may outlive it, and the result says so.
- The usage journal gains the execution outcome `waiting`, recorded when the
  operation reports it, and `duration_unknown` for events whose interval was
  not measured; aggregates skip those durations instead of averaging a zero.

## Consequences

- `pose close --apply --json` and `continuous-closeout status --json` gain a
  contract; consumers that parsed their text keep it without `--json`.
- Stricter refusals are a semantic change: invocations that used to run with
  an ignored flag now exit 2. Classified for a minor release, not a patch.
- A journal written by this engine can hold `waiting` events, which an older
  engine's reader rejects as invalid rows; the journal is local telemetry and
  never evidence.
- The envelope does not make an operation atomic. `effects` lists what the
  domain confirmed; an empty list after a failure does not prove a rollback,
  and `unconfirmed` names the step to inspect before a retry.
- Out of scope: a general CAS publisher (B14), the resident runtime (B35),
  context propagation into in-process CLI domain calls (a CLI process is
  ended by its signal), and paths whose JSON already has a contract.

## Falsifier

A call with `--json` whose stdout is not exactly one JSON document; an
invalid flag or value that returns data; a killed gate reported with a
`passed` verdict; or a `waiting` stop counted as `failed`.
