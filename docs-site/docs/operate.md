# Operate a governed instance

**Doc type:** How-to · **Applies to:** POSE v7.4.1 · **Reviewed:** 2026-10-20

Use this guide after the [quickstart](quickstart.md). It answers three daily
questions: what policy is actually in force, what is waiting, and who must act.

## Inspect before adopting

```bash
pose setup --json
pose adopt --list
pose state --governance --json
pose doctor
```

`setup --json` is read-only. At a terminal, `pose setup` can guide identity,
keys, the commit hook and capability decisions one confirmed step at a time.
The capability catalog distinguishes on, off, declined, deferred and
needs-setup. Effective governance distinguishes supported, configured,
applicable and effective; a shipped profile need not be active in this project.

Inspect a particular scope with:

```bash
pose state --governance --scope spec:customer-export --json
pose review-plan spec:customer-export --json --explain
```

Dated adoption can preserve the boundary for work already under way. A new
engine does not silently adopt every optional contract. `pose update` opens
configuration-review requests for undecided capabilities; the maintainer's
answer and `pose adopt --request <act-id> --apply` govern their application.
Record a reason when declining or deferring a capability.

## See what is owed

```bash
pose state --attention --json
pose state --attention --scope spec:customer-export --phase review --json
pose action list --present
```

Attention groups obligations by actor and affected phase: start, execution,
review, closeout and release. It projects action requests, follow-ups,
findings, stale premises, assessments, docs, capabilities and release work.
Check coverage first: an incomplete projection cannot establish that nothing
is owed. Attention explains work; the named gate enforces it.

Use `--actor <principal-or-role>` to focus on a recipient. A settled action
stays in the audit history and leaves the actionable groups unless its answer
is invalidated. A release-only question need not interrupt implementation.

## Ask for a material decision once

An action request persists a consequential decision, approval, input,
external operation or acceptance. It records options, consequences, recipient,
targets and the phases it restricts. Use `pose action open --help` to construct
the request, then inspect the exact record:

```bash
pose action show <act-id> --json
pose action list --present
```

An answer binds to the request digest and revision. Recipient roles determine
who may answer; idempotency makes a retry safe. Declining an approval records
a decision and grants no authorization. Ordinary clarification questions do
not need an action journal.

Under verified assurance, use a registered SSH Ed25519 key or a trusted
issuer's claim. Register the public key through `pose identity add`; produce
the statement with `pose action statement` and sign through
`pose action resolve … --sign <key>`. The [CLI reference](cli.md#signed-action-answers)
describes flags and security-key presence. A `human:` prefix alone proves no
person answered.

## Preserve the baseline and reassess only stale inputs

Preview readiness with `pose lint-spec <slug> --ready-check`, then use
`pose start spec:<slug>` and its digest-bound apply. Under adopted atomic
start, the engine records the R/A/D baseline and moves the spec to
`in-progress`. A baseline records observable precedence, not the instant a
person made a decision.

```bash
pose assess discover --if-stale --component <directory>
pose start spec:customer-export --status
```

Assessments reuse unchanged committed component content, engine and validation
matrix. Age alone does not stale them. At closeout, use
`pose assess discover --if-stale --update-state` as required by the workflow.
Use [review and trust](review-and-trust.md) for the exit gate and
[analytics](analytics.md) to inspect usage, waits and delivery outcomes.
