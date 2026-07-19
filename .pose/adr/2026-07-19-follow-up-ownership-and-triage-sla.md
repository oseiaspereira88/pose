# ADR: Follow-up ownership and triage SLA

## Status
Accepted (2026-07-19) — spec `pose-followup-ownership-sla`

## Context

Open follow-ups had a disposition but no owner, urgency or review date — a
permanent unowned text backlog in the making. The roadmap requires residual
work to carry durable identifiers and owners; the spec constrains SLAs to
*triage promises*, never unconditional implementation deadlines, and warns
against blocking policies that freeze delivery.

Alternatives considered:

1. **External issue tracker** — explicit non-goal; POSE artifacts stay in
   the repository and portable.
2. **Frontmatter-level ownership per spec** — too coarse: one spec can leave
   several follow-ups with different owners and urgencies.
3. **Inline ownership group per follow-up** — a trailing
   `(owner:@alias crit:low|medium|high review:YYYY-MM-DD [by:@actor])`
   group on the bullet, parsed by the same follow-up engine.

## Decision

Option 3:

- **Syntax (additive):** the trailing parenthesized group with `owner:`
  (required), `crit:` and `review:` (required together with owner) and
  optional `by:` recording who set the disposition. Aliases only — no
  personal contact data; external identity mapping stays outside the
  repository.
- **Legacy migration:** follow-ups without the group parse as
  `owner:unowned`. On `done` specs they produce a closeout **warning** (the
  remediation window); malformed or partial groups are **errors**. All seven
  open follow-ups in this repository were migrated to owned entries at
  adoption.
- **Projections:** `pose followups` reports `overdue=` and `unowned=` in the
  header, filters with `--overdue` and `--owner <alias>`, marks expired
  reviews `OVERDUE`, and exposes the fields in `--json`.
- **Risk-based blocking (R2):** enforcement is opt-in — `--fail-overdue`
  exits non-zero when any open follow-up has an expired review date. CI and
  the quarterly audit may adopt it; the default stays non-blocking so an
  expired triage date cannot freeze delivery by itself.
- **Disposition integrity (R3):** the existing closeout gate keeps rationale
  (`wont-do: reason`) and validated targets (`spawned/covered/duplicate`
  must reference existing specs); `by:` adds optional, minimized actor
  attribution. Append-only disposition history belongs to
  `pose-spec-amendment-history` (next milestone), not this contract.

## Consequences

- Positive: the live backlog answers "whose, how urgent, when reviewed" —
  and the quarterly audit can enumerate expired triage promises
  deterministically.
- Positive: ownership is portable (aliases) and additive (legacy files stay
  readable).
- Trade-off: closeout of a spec with open follow-ups costs one metadata
  group per item; unowned residual work is exactly what the spec set out to
  eliminate.
- Residual: an SLA can be renewed forever by editing `review:`; the
  amendment-history spec will make such renewals visible and reviewable.
