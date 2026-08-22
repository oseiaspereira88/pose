---
name: pose-spec-closeout
description: Use to close a completed POSE spec by setting status done, recording completion date, and dispositioning every follow-up so the backlog remains live and deduplicated. Trigger keywords - closeout, close spec, complete spec, mark done, follow-up, triage, spec lifecycle, completed_at.
when_to_use: Feature, bugfix, or refactor implementation has passed deterministic validation and its spec needs formal closure. Use as the final workflow step before claiming delivery.
pose_schema_range: "1-1"
clients: agents-skills, mcp, claude-code
capabilities: read, spec-write
---

# Skill: pose-spec-closeout

Close a spec lifecycle and triage every follow-up without silently losing intent.

## Required reading

1. The spec under `.pose/specs/<slug>.md` or `.pose/specs/<slug>/spec.md`.
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

1. Run `pose followups --all` to aggregate backlog and lexical near-duplicate candidates.
2. Judge semantic equivalence yourself; lexical candidates are hints, not verdicts.
3. Stop and obtain user confirmation before writing `spawned`, `covered`, or `duplicate`. These transitions create work or silently discard an item if wrong. `open`, `done`, and `wont-do` do not require confirmation.
4. Ensure every target slug exists and does not point back to the current spec.

## Steps

1. Confirm strict deterministic validation passed for affected modules (`pose validate --strict --module <affected-path>`).
2. Verify that all implementation commits modifying the spec's declared `### Artifacts` carry a `POSE-Spec: <slug>` trailer in their commit message. Commits lacking this trailer cannot be attributed during `pose close` or `pose artifact-check`.
3. Run a separate review pass: prepare and seal via `pose review bundle spec:<slug> --seal`, attest via `pose review auto-attest <bundle-id> --reviewer agent:<id> --apply` (or `pose review attest`), and verify with `pose review verify spec:<slug>` (or use `pose review record spec:<slug> ... --apply` for legacy policies without review bundles).
4. Require `pose review verify spec:<slug>` and `pose review-check spec:<slug>`; remediate, revalidate and supersede stale or rejected attempts.
5. Inspect `pose followups --all` and, if useful, lower `--similarity` to broaden candidates.
6. Propose each consequential disposition and obtain confirmation before writing it.
7. Apply `pose close spec:<slug>`; use a manual lifecycle edit only when the Git workflow requires it and preserve the same gate.
8. Produce a **changelog fragment** for the delivered spec:
   ```bash
   cp .pose/templates/changelog-fragment.md .pose/changelogs/unreleased/<slug>.md
   # fill category/breaking and the user-facing summary (derive from Intent, not implementation)
   ```
   Internal work with no user-facing effect: set `changelog: none` in the spec
   frontmatter instead of creating a fragment. `pose check` warns on done specs
   without a fragment (post-adoption).
9. Run `pose lint-spec <slug> --strict`.
10. Create any confirmed successor spec with `pose new-spec <slug>` (defaults to dated flat `.pose/specs/YYYY-MM-DD-<slug>.md`) and revalidate its intent instead of copying follow-up text verbatim.
11. Run `pose assess discover --update-state` upon spec closure to update dynamic platform assessments.
12. Inspect residual live backlog with `pose followups --open --json`.
13. When Contributor Mode is active, if the delivery cycle revealed POSE engine friction, false-positive linters, or tooling gaps, stage a sanitized feedback report with `pose contribute stage --type enhancement --title "<summary>"`.

## Output requirements

- `status: done` and populated `completed_at`.
- Changelog fragment in `.pose/changelogs/unreleased/<slug>.md` (or `changelog: none` in the spec frontmatter).
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
