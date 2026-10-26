# ADR: Delegation is a first-class execution surface

## Status
Accepted (2026-10-26) by the maintainer. Amends [A delegated reviewer is an adapter that turns a sealed bundle into a draft](2026-10-09-delegated-review-is-an-adapter.md)
where stated below; every other decision of that ADR, and of the ADRs of the
attempt ledger, run continuity, run provenance and adapter containment, stands.
Roadmap: [`delegation`](../roadmaps/delegation.md).

## Context

The 2026-10-09 ADR made the engine the owner of delegated review. Since then
the brief, dispatch, attempt ledger, run continuity, run provenance and adapter
containment were delivered. In practice no review has run through them: every
delegated review, adjudication and closeout attestation still goes through two
personal launchers, `secondary-agent` and `secondary-agent-watch`, which the
2026-10-09 ADR already called examples rather than the mechanism. Two
structures keep the launchers alive:

- **They do more than review.** The launcher also hands general tasks to the
  other vendor in write mode (edits, commits, pushes). Nothing in POSE covers
  that, and `pose review` is the wrong home for it.
- **They own the run, not the subject.** Vendor choice by who is leading,
  session reuse, logs, live watch, picker, resume and limit hints are execution
  concerns. The engine spreads its share of them across `pose review dispatch`,
  `pose review resume` and `pose review status`, beside commands that govern
  the review subject (`bundle`, `attest`, `verify`).

Reading the delivered code against the launchers also found gaps the
launchers do not have, and one defect in the delivered contract:

- `--via <adapter>` is mandatory: there is no routing by vendor, no fallback
  when the preferred vendor is out of quota, and no way to know availability
  before a run fails.
- `reviewers.json` declares the implementation identity statically
  (`provenance.implementation`, with a fixed `execution`). The implementation
  is whoever produced the sealed subject, which varies per scope; a static
  identity makes "the other vendor" unanswerable and compares the reviewer with
  the wrong party.
- `pose review resume` continues only the same adapter; switching vendor after
  an interruption discards the partial work.
- `pose review status` lists interrupted runs only; there is no live watch.
- `round-protocol` R6 makes a cold session the default for each round.
- The capability is opt-in (`pose adopt delegated-review`), and turning it off
  does not relax `different-actor`, so a project with one vendor cannot reach
  delivery without declaring a reviewer it does not have.

## Options considered

1. **Keep `pose review dispatch` and add flags.** Leaves general tasks outside
   POSE and execution operations mixed with subject operations.
2. **A `pose secondary-agent` command.** Names a role relative to a primary; a
   CI job or Harne8's Conductor is nobody's secondary, and "agent" is already
   overloaded (`agent:` principals, `AGENTS.md`).
3. **A `pose agent` noun.** Short, but overloaded for the same reason, and
   `pose agent review` reads as an agent reviewing itself.
4. **A `pose delegate` verb owning every delegated execution**, with the
   purpose as a subcommand and one lifecycle for all purposes.

## Decision

Option 4.

### Separation of concerns

- `pose review` governs the subject: bundle, brief, attestation, verify,
  closeout. It stops starting or observing executions.
- `pose delegate` governs executions: who runs, how, with which access, and
  what happened. Every purpose shares one run record, ledger and lifecycle.

### Command surface

```
pose delegate review     <scope|bundle> [--agent auto|<name>] [--detach]
pose delegate adjudicate <scope|bundle> [--agent auto|<name>] [--detach]
pose delegate smoke      <target>       [--agent auto|<name>] [--detach]
pose delegate task       <prompt-file|-> [--agent auto|<name>] [--access read|sandboxed|write|full]
                         [--cd <dir>] [--add-dir <dir>] [--no-network] [--effort <level>] [--record] [--detach]

pose delegate ls      [<run>|<bundle>] [--active] [--json]
pose delegate watch   [<run>|<label>] [-v] [--lines <n>] [--no-follow] [--json]
pose delegate wait    <run> [--timeout <d>] [--json]
pose delegate resume  <run> [--wait-until-reset[=<max>]]
pose delegate handoff <run> --to <agent>
pose delegate cancel  <run>
pose delegate agents  [--json]
```

Common flags: `--label <name>`, `--lead <vendor>`, `--dry-run`, `--json`.

- Use one term per concept: **agent** (a configured CLI profile), **run** (one
  execution), **attempt** (a run and its continuations), **thread** (the rounds
  of one scope). `--agent` replaces `--via`, `--target` and "adapter" in the
  user-facing surface.
- Make the purpose a subcommand, not `--kind`; each purpose carries its
  guarantees. `review`, `adjudicate` and `smoke` force `read` access, the
  disposable worktree at the sealed commit, the generated brief and the ledger.
  `task` accepts every access level and never produces an attestation, verdict
  or evidence a gate reads.
- Keep `pose review dispatch`, `pose review resume` and `pose review status`
  as deprecated aliases for one minor version, printing the replacement; then
  remove them.
- Give every command `--json`. `watch --json` emits normalized events as
  NDJSON, identical for every vendor. `wait` uses the exit codes of
  `completion-signal` (0 approved, 10 changes requested or rejected, 20 failed,
  30 interrupted, 40 stalled, 50 wait timeout, 2 usage).
- In a terminal, `watch` with no argument opens a picker (fzf when present, a
  numbered menu otherwise). When a run stops on a limit, the terminal offers
  the choices — wait for the reset, hand off to another agent, cancel — and
  `--json` returns the same choices as `next_actions`, so an agent decides
  without guessing.

### One policy: `.pose/policy/delegation.json`

- Replace `reviewers.json` (schema 1) with `delegation.json` (schema 2):
  `agents` (vendor, model, CLI, event format, per-access command templates,
  session, resume, limit, effort), `routing` and per-purpose budgets.
- Migrate schema 1 deterministically in `pose update`; read schema 1 until the
  aliases are removed.
- Remove the parent agent's session variables from the child environment as
  engine behaviour, so a child never believes it is nested in its parent.
- Parse each vendor's event stream in the engine, selected by the agent's
  `events` field.

### Routing and independence

- Record the implementation identity (vendor, model, principal, execution)
  when the bundle is sealed, detected from the sealing session and overridable
  with `--lead`; remove `provenance.implementation` from the policy. Compare the
  reviewer with the implementation recorded in the bundle, not with whoever
  started the run.
- Route with `routing.prefer` (default `opposite-vendor-of-implementation`)
  and the ordered `routing.on_unavailable` list (default
  `same-vendor-other-model`, `wait`, `same-model`).
- Record the independence a run achieved: another vendor or another model
  satisfies `different-actor`, as decided on 2026-10-09; the same model yields
  `same-actor-separate-execution` and is disclosed at closeout.
- Read availability from `pose delegate agents`: the CLI on the path and the
  reset instant of the last recorded limit, so routing never spends a run to
  discover a blocked vendor.

### Continuity and rounds

- Keep `resume` as delivered: same agent, same session, same attempt.
- Add `handoff --to <agent>`: a new run in the same attempt and thread whose
  brief carries the predecessor's partial output, labelled and bound by
  digest, and what it had verified. Record the lineage. A handoff never turns
  the partial into a review.
- Amend `round-protocol` R6: the default is `continue-thread` — a round after
  a rejection resumes the session of the thread's first round. R2–R5 (prior
  findings in the brief, an account for each, a full sweep) remain mandatory.
  Open a new session, with the reason recorded, on a vendor change, a broken
  supersession chain, an expired session or the round threshold.

### Adoption and degradation

- Amend "opt-in": `pose init` offers delegation with yes as the default answer
  and writes `delegation.json` from the agent CLIs found on the path. A refusal
  or a non-interactive run records `delegated_review: false`. `pose update`
  suggests, never enables.
- When `delegated_review` is false, a plan that requires `different-actor`
  degrades to `same-actor-separate-execution` and the closeout discloses the
  limitation `delegated-review-disabled`. One agent of one vendor can take a
  scope from spec to delivery, and the record says what happened.
- `pose delegate task` does not depend on the capability: it is a tool, not a
  review rule.

### Agent access over MCP

- Reverse option 4 of the 2026-10-09 ADR: expose bounded MCP tools to start a
  delegated run, read its status and wait with a bounded timeout. The attempt
  ledger now refuses a rerun on an unchanged bundle without a reason, which
  removes the approval loop that option was rejected for.

### Retirement by parity

- Retire the personal launchers only when a versioned parity matrix maps every
  capability below to a native command and a test, and the matrix check fails
  while any row is uncovered.

`secondary-agent`: lead detection and opposite vendor, `--lead`, `--target`;
same-vendor warning; `--read-only`, `--sandboxed`, `--full-access`,
`--no-network`; write mode with the vendor's automatic approver; `--cd`,
`--add-dir`, `--effort`, models by configuration; running outside a Git
repository; `--resume` from a run or a raw session, reusing its settings, with
a default continuation message; usage-limit detection with the resume hint;
`--label`; `--dry-run`; prompt, events, final message, stderr, command, session
and exit code kept per run; final message on stdout and the agent's exit code;
removal of the parent's session variables.

`secondary-agent-watch`: list with state, turns, age and last step; selection
by label, run name or session prefix; picker with live preview and refresh;
follow that stops on exit, and no-follow; verbose output tail with a line
bound; resumed chains replayed with a marker at each hand-over; both vendors'
event formats; `NO_COLOR` and no colour off a terminal.

## Maintainer's decisions (2026-10-26)

1. **Name:** `pose delegate`.
2. **Task records:** kept outside the repository, under the user's state
   directory, unless `--record` asks for them under `.pose/`; a task is not
   governance evidence.
3. **MCP:** expose start, status and bounded wait.
4. **Rounds:** `continue-thread` is the default, accepting the anchoring risk
   under the R2–R5 safeguards.

## Consequences

- `pose review dispatch`, `resume` and `status` become aliases and then
  disappear; documentation, skills and help move to `pose delegate`.
- `reviewers.json` becomes `delegation.json`; instances migrate through
  `pose update`.
- The bundle gains the implementation identity; bundles sealed before the
  change keep reading the policy declaration until they are superseded.
- The remaining draft specs of [`delegated-review`](../roadmaps/delegated-review.md)
  move to [`delegation`](../roadmaps/delegation.md); `round-protocol`,
  `completion-signal` and `capability` are amended there.
- The personal launchers stay in use until the parity check passes; their
  retirement and the removal of operational references to them follow it.
  Historical records that name them (this ADR, the 2026-10-09 ADR, reports)
  stay as written.
- Instances with one vendor reach delivery with a disclosed same-actor review
  instead of a declared reviewer they do not have.
