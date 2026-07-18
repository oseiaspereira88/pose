---
name: pose-spec-closeout
description: Use to close a completed POSE spec by setting status done, recording completion date, and dispositioning every follow-up so the backlog remains live and deduplicated. Trigger keywords - closeout, close spec, complete spec, mark done, follow-up, triage, spec lifecycle, completed_at.
when_to_use: Feature, bugfix, or refactor implementation has passed deterministic validation and its spec needs formal closure. Use as the final workflow step before claiming delivery.
---

# Skill: pose-spec-closeout

Close a spec lifecycle and triage every follow-up without silently losing intent.

## Required reading

1. The spec under `.pose/specs/<slug>/spec.md`.
2. [`.pose/templates/spec.md`](../../../.pose/templates/spec.md).
3. [AGENTS.md](../../../AGENTS.md).

## Lifecycle

The normal path is `draft` to `in-progress` to `done`. Alternative terminal
states are `blocked`, `superseded`, and `abandoned`. Set `completed_at` only
when transitioning to `done`.

## Follow-up dispositions

| Disposition | Use when |
|---|---|
| `[open]` | Relevant live backlog without a dedicated spec |
| `[spawned: <slug>]` | Created or seeded a new spec |
| `[covered: <slug>]` | Already delivered by another existing spec |
| `[duplicate: <slug>]` | Duplicates a follow-up already triaged elsewhere |
| `[done]` | Resolved directly without another spec |
| `[wont-do: <reason>]` | Intentionally declined with rationale |

`[open]` is a deliberate live disposition, not an untriaged item.

## Deterministic, semantic, and human triage

1. Run `./pose followups --all` to aggregate backlog and lexical near-duplicate candidates.
2. Judge semantic equivalence yourself; lexical candidates are hints, not verdicts.
3. Stop and obtain user confirmation before writing `spawned`, `covered`, or `duplicate`. These transitions create work or silently discard an item if wrong. `open`, `done`, and `wont-do` do not require confirmation.
4. Ensure every target slug exists and does not point back to the current spec.

## Steps

1. Confirm strict deterministic validation passed for affected modules.
2. Inspect `./pose followups --all` and, if useful, lower `--similarity` to broaden candidates.
3. Propose each consequential disposition and obtain confirmation before writing it.
4. Set `status: done` and the real `completed_at` date.
5. Run `./pose lint-spec <slug> --strict`.
6. Create any confirmed successor spec and revalidate its intent instead of copying follow-up text verbatim.
7. Inspect residual live backlog with `./pose followups --open --json`.

## Output requirements

- `status: done` and populated `completed_at`.
- Valid disposition on every Final Report follow-up.
- User confirmation before `spawned`, `covered`, or `duplicate`.
- Successful strict spec lint.
- Confirmed successor specs with independently validated intent.

## Anti-patterns

- Closing before deterministic validation.
- Reusing follow-ups automatically without confirmation.
- Treating lexical candidates as semantic verdicts.
- Deleting history instead of using `wont-do`.
- Using `open` as a dumping ground when no real intent remains.
