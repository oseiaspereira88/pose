# ADR: Durable non-architectural knowledge belongs in rules, not a new type

## Status
Accepted (2026-08-15) — spec `pose-knowledge-durable-reference-type`,
`github.com/oseiaspereira88/pose#25`

## Context

`.pose/knowledge/`'s three types (`handoff`, `decision-log`, `note`) all
require `expires_at` — 30-day default, 90-day maximum, no exception
(`.pose/rules/knowledge-governance.md`). Every knowledge artifact is
deliberately ephemeral. The only permanent record in POSE's governance model
is the ADR, explicitly scoped to structural/architectural decisions. A
durable-but-non-architectural fact — one that will not become false or need
re-review on any predictable timeline, but that also does not represent a
decision or trade-off — fits neither. The motivating case:
`.pose/knowledge/2026-08-15-decision-log-github-issue-closing-keyword-format.md`
records that GitHub's closing-keyword parser rejects a URL-prefixed issue
reference (`github.com/owner/repo#N`), accepting only `#N`/`owner/repo#N`.
Nothing was decided here and nothing will change on a review cadence; it was
filed as a `decision-log` purely for lack of a better fit, and will resurface
in `pose knowledge-housekeeping list-expired` in 90 days for no substantive
reason.

Two candidate resolutions were identified, not yet chosen between:

1. **Add a fourth knowledge type** (e.g. `reference`), exempt from mandatory
   `expires_at` or governed by a materially longer review-not-expire
   cadence.
2. **Route this content to `.pose/rules/` instead** — already POSE's
   durable, non-expiring home for team conventions — and keep
   `.pose/knowledge/` strictly for time-bound operational memory, with no
   schema change.

Option 2 was blocked on one open question: is `.pose/rules/` content
actually discoverable for a fact with no natural path/component scope (the
trailer-format fact applies to every commit, not to one directory)?
`pose suggest <task> --path <dir>` was assumed to map rules by
path/component, which would make it a poor fit for a cross-cutting
convention.

**Investigation of `pose-mcp/internal/cli/suggest.go` and
`.pose/indexes/task-map.json` found that assumption wrong.** Each task type
in `task-map.json` carries a base `rules` array (e.g. `bugfix` and `feature`
both list `["security", "documentation-style"]`; `knowledge` lists
`["knowledge-governance", "documentation-style"]`) that `cmdSuggest`
(`suggest.go:79`) returns unconditionally for that task type — no `--path`
or `--domain` required. `--path`/`--domain` only resolves an *additional*
layer, `rules_by_domain` (`suggest.go:80-83`), used exclusively for
language/component-specific rules (`frontend-react`, `backend-go`,
`kubernetes`) that legitimately vary by where the change lands. A
cross-cutting convention with no natural path scope does not need that
layer at all: adding it to a task type's base `rules` list makes it surface
on every invocation of `pose suggest <task>` for that task type, regardless
of path — and every `pose-*` skill's Required Reading step already mandates
running `pose suggest <task> [--path <dir>]` before starting work (see
`pose-bugfix`, `pose-feature` skills), so the base list reaches an agent
unconditionally at the point where it matters.

This reverses the risk calculus: `.pose/rules/` already has *stronger*,
not weaker, discoverability for exactly this kind of fact than
`.pose/knowledge/` does today. Knowledge artifacts are discovered by
opportunistic search (`find .pose/knowledge -name '*<topic>*.md'`, per the
`pose-knowledge` skill) — nothing injects them into a workflow's required
reading. Rules, by contrast, are injected automatically by the very
tool (`pose suggest`) every governed workflow already calls first.

## Decision

- **No fourth knowledge type.** `.pose/knowledge/` stays strictly
  handoff/decision-log/note, all mandatorily TTL'd — no exemption, no
  longer-ceiling carve-out. This preserves `knowledge-governance.md`'s TTL
  discipline exactly as-is (a stated constraint of spec
  `pose-knowledge-durable-reference-type`).
- **Durable, non-architectural facts and conventions belong in
  `.pose/rules/`.** A rule is the correct home when the content (a) will
  not become false or need review on a predictable timeline, (b) does not
  represent a decision or trade-off (ruling out both `decision-log` and
  ADR), and (c) is a convention or fact an agent should be told before or
  during relevant work — the exact shape `pose suggest` rule injection
  serves.
- **No code change to `pose suggest`/`task-map.json`'s mechanism.** The
  base-vs-domain rule layering already supports cross-cutting rules with no
  path scoping; a new durable fact is added either to an existing rule file
  or as a new one, then wired into the relevant task type(s)' base `rules`
  array in `task-map.json` (e.g. a commit-convention fact would join
  `bugfix`, `feature`, and `release-closeout`'s base list, not a
  domain-specific one).
- **Documentation-only follow-through**, tracked under spec
  `pose-knowledge-durable-reference-type`: `.pose/rules/knowledge-governance.md`
  and the `pose-knowledge` skill gain guidance distinguishing "this is a
  `decision-log`" (a decision was made, a review trigger exists) from
  "this belongs in `.pose/rules/`" (nothing was decided, nothing expires),
  so a future author does not have to guess the way this ADR's motivating
  case did.
- **No retroactive re-triage.** The existing
  `github-issue-closing-keyword-format` decision-log stays as-is and expires
  on its own schedule; only the mechanism/guidance for *new* content is in
  scope (an explicit non-goal of the spec).

### Rejected trade-off

Option 1 (fourth knowledge type) was rejected: it would duplicate a
discoverability mechanism `.pose/rules/` + `pose suggest` already provides,
with a weaker guarantee (opportunistic search vs. automatic injection into
every governed workflow's required reading), plus new schema surface
(`knowledge.go`'s `Type` enum, TTL-exemption logic, `knowledge-check`
handling) to build and maintain for no discoverability gain.

## Consequences

- Positive: closes the gap `github.com/oseiaspereira88/pose#25` reported
  without adding a fourth knowledge type or weakening
  `knowledge-governance.md`'s TTL enforcement for the three existing types.
- Positive: `.pose/rules/` gains an explicit, documented use case
  (durable non-architectural fact/convention) distinct from its existing
  role (team/language conventions), formalized via
  `knowledge-governance.md` and the `pose-knowledge` skill rather than left
  implicit.
- Trade-off: a rule file has no frontmatter TTL/owner/sensitivity
  bookkeeping the way a knowledge artifact does — durable facts placed in
  `.pose/rules/` rely on ordinary code review for quality control, not
  `pose knowledge-check`. Acceptable: the content in question is by
  definition not time-bound, so TTL bookkeeping was never buying anything
  for it.
- Residual: nothing prevents a future author from still defaulting to
  `decision-log` out of habit; the documentation follow-through (spec
  `pose-knowledge-durable-reference-type`, R2) is what closes that gap, not
  this ADR alone — tracked there, not here.
