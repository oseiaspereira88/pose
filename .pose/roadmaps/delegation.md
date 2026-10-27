---
slug: delegation
status: draft
created_at: 2026-10-26
depends_on: delegated-review
---

# Roadmap: Delegation

**Outcome:** every delegated execution — a review, an adjudication, a smoke
run or a general task handed to another agent — starts, runs, is watched,
continues and ends through `pose delegate`, with routing between vendors,
recorded independence and one run record; and the personal launchers it
replaces retire with nothing lost.

Decision: [Delegation is a first-class execution surface](../adr/2026-10-26-delegation-is-a-first-class-execution-surface.md),
which amends [the 2026-10-09 ADR](../adr/2026-10-09-delegated-review-is-an-adapter.md).
Foundation: the contract, brief, dispatch, attempt ledger, run continuity, run
provenance and adapter containment delivered under
[`delegated-review`](delegated-review.md). Its four remaining draft specs move
here: `completion-signal`, `round-protocol` and `capability` are amended by the
ADR; `action-fulfillment` keeps its requirements and takes the new command
names.

A milestone starts only after its specs reconcile with the ADR; the three
amended specs carry the ADR's changes before their milestones start.

## Milestone: surface
- after: spec:pose-delegated-adapter-containment, spec:pose-delegated-review-run-continuity
- target_start:
- target_due:
- specs: pose-delegation-surface

**Exit gate:** `pose delegate review|adjudicate|smoke|task|ls|watch|wait|resume|handoff|cancel|agents`
parse with one vocabulary (`--agent`, `--access`, `--label`, `--lead`,
`--json`); `pose review dispatch|resume|status` print their replacement and
behave identically; `delegation.json` schema 2 loads, schema 1 migrates
deterministically through `pose update`; the child environment carries none
of the parent agent's session variables; both vendors' event streams normalize
to one NDJSON shape.

## Milestone: routing
- after: surface, spec:pose-delegated-review-run-provenance
- target_start:
- target_due:
- specs: pose-delegation-routing

**Exit gate:** a sealed bundle records its implementation identity and
`provenance.implementation` is no longer read for new bundles; `--agent auto`
picks the opposite vendor of the implementation, falls back through
`routing.on_unavailable` when `pose delegate agents` reports a blocked vendor,
and every run records the independence it achieved; a same-model run never
satisfies `different-actor`.

## Milestone: observation
- after: surface
- target_start:
- target_due:
- specs: pose-delegated-review-completion-signal

**Exit gate:** `--detach` returns the run at once; `wait` releases every waiter
with the documented exit codes; `ls` reports every state with heartbeat age;
`watch` follows a run live until it ends, replays resumed and handed-off runs
with a marker at each hand-over, and opens a picker in a terminal; stalled
runs are detected by heartbeat and supervisor identity.

## Milestone: continuity
- after: routing, observation
- target_start:
- target_due:
- specs: pose-delegation-handoff

**Exit gate:** `handoff --to` continues an interrupted attempt with another
agent, carrying the labelled partial and its digest, with the lineage
recorded and the partial never accepted as a review; a stopped run offers
wait, handoff and cancel in a terminal and as `next_actions` in `--json`.

## Milestone: rounds
- after: continuity, spec:pose-delegated-review-brief, spec:pose-delegated-review-attempt-ledger
- target_start:
- target_due:
- specs: pose-delegated-review-round-protocol

**Exit gate:** with `continue-thread`, the round after a rejection resumes the
thread's first session; a vendor change, a broken chain, an expired session or
the round threshold opens a new session with the reason recorded; prior
findings are accounted for and swept as R2–R5 require; insights measure
convergence.

## Milestone: task
- after: surface, observation
- target_start:
- target_due:
- specs: pose-delegation-task

**Exit gate:** `pose delegate task` runs with each access level, outside a Git
repository too, keeps its record under the user's state directory unless
`--record`, prints the final message and returns the agent's exit code, and
no gate, attestation or closeout ever reads a task run.

## Milestone: agent-access
- after: observation, rounds
- target_start:
- target_due:
- specs: pose-delegation-mcp

**Exit gate:** MCP tools start a delegated run, read its status and wait within
a bounded timeout; a rerun on an unchanged bundle without a reason is refused
over MCP exactly as on the CLI.

## Milestone: conclusions
- after: routing, spec:pose-delegated-review-run-provenance, spec:pose-delegated-adapter-containment, spec:pose-review-approval-domain-preflight
- target_start:
- target_due:
- specs: pose-delegated-review-conclusions

**Exit gate:** phase B19 of the batch 10 plan: a passing review run is
recorded as the attestation by the engine, an adjudication or smoke run as its
verdict at its destination, a rejection blocks until the bundle changes, and
a failed check, an incomplete conclusion, a replay or a superseded bundle
record nothing. Made a spec by the amendment of 2026-10-27 to the delegation
ADR, so fulfillment can require it done without requiring the capability.

## Milestone: fulfillment
- after: surface, conclusions, spec:pose-delegated-review-run-provenance, spec:pose-delegated-adapter-containment, spec:pose-action-target-and-node-scoping, spec:pose-action-waiver-and-supersession-authority, spec:pose-action-request-recovery-and-preflight, spec:pose-review-approval-domain-preflight
- target_start:
- target_due:
- specs: pose-delegated-review-action-fulfillment

**Exit gate:** the gate recorded for this milestone under `delegated-review`
holds unchanged, with `pose delegate` as the local fulfiller; its current
acceptance of phase B19 is the `conclusions` milestone.

## Milestone: adoption
- after: routing, continuity, rounds, conclusions, fulfillment, agent-access, spec:pose-review-approval-domain-preflight
- target_start:
- target_due:
- specs: pose-delegated-review-capability

**Exit gate:** `pose init` offers delegation with yes as the default and writes
`delegation.json` from the CLIs found; a refusal or a non-interactive init
records `delegated_review: false`; `pose update` only suggests; with the
capability off, `different-actor` degrades to a disclosed
`same-actor-separate-execution`; with it on, a passing review run is recorded
as the attestation by the engine and a rejection blocks until the bundle
changes.

## Milestone: retirement
- after: task, adoption
- target_start:
- target_due:
- specs: pose-delegation-parity-gate

**Exit gate:** the parity matrix maps every launcher capability listed in the
ADR to a native command and a test; its check failed against a matrix with an
uncovered row before passing; documentation, skills and help name only
`pose delegate`; the launchers and the operational references to them are
retired, and historical records are left as written.
