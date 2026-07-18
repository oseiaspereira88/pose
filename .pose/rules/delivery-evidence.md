# Rule: Delivery Evidence

## When to consult

Consult this guide when writing or reviewing any document that **claims delivery,
completion, or readiness**: status reports, handoffs, summaries, module READMEs,
current-state sections in specs or proposals, and messages that say something is complete.

## Required conventions

- Claim delivery only with attached, verifiable gate evidence: command and output
  (`./pose validate`, `go test`, `tsc`, `vitest`) or a link to the corresponding POSE report.
- Use POSE lifecycle vocabulary: `draft`, `in-progress`, `done`, `blocked`,
  `superseded`, or `abandoned`. Do not invent labels such as `completed` or `100% COMPLETE`.
- Separate **implemented and verified** work from **planned or documented** work.
  A plan describes intent; do not present that intent as current reality.
- Reference the report or evidence that crossed the spec exit gate before using `done`.
- Convert relative dates to absolute dates and stamp the verification date.

## Blocking anti-patterns

- Claiming `100% COMPLETE` or production readiness without a green
  `./pose validate --strict` for every affected module.
- Publishing contradictory delivery documents for the same scope; reconcile them first.
- Merging code with completion documentation before the applicable POSE checks pass.
- Mixing aspiration and verified state in one paragraph without clear labels.

## Minimum checks

- Run `./pose check --strict` for structure and spec status enums.
- Run `./pose validate --strict` for every module the document claims to deliver.
- Run `./pose lint-spec` when the document is a spec.

## Precedence in multi-domain conflicts

- Prefer verifiable check evidence over progress narratives when rules conflict.
- When pressured to claim completion without a gate, record the real state
  (`in-progress` or `blocked`) and the remaining exit conditions.

## Recurrence traceability

> Also apply: [.pose/rules/_base-recurrence.md](_base-recurrence.md)
