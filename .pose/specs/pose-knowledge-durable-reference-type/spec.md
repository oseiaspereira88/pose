---
slug: pose-knowledge-durable-reference-type
status: draft
created_at: 2026-08-15
completed_at:
supersedes:
depends_on:
priority: 3
components: pose-mcp
delivers:
---

# Spec: pose-knowledge-durable-reference-type

---

## 1. Intent

### Goal
Give a durable, non-architectural fact or convention a correctly-fitting
home in POSE's governance model, instead of forcing it into a
`decision-log` (which implies a decision was made and a review is due) or
an ADR (scoped to structural/architectural decisions).

### Business value
`github.com/oseiaspereira88/pose#25`: `.pose/knowledge/`'s three types
(`handoff`, `decision-log`, `note`) are all governed by
`knowledge-governance.md`, which blocks creation without `expires_at` —
TTL 30 days default, 90 days maximum, no exception. Every knowledge
artifact is deliberately ephemeral by design. The only permanent record is
the ADR, explicitly scoped to structural/architectural decisions (choosing
between significant options, changing a public contract, accepting a
trade-off). Neither fits a fact like "GitHub's closing-keyword parser
doesn't accept a URL-prefixed issue reference" — durable (won't become
false or need re-review on any predictable timeline) but not architectural
(nothing was decided or traded off). Recorded as a `decision-log` for lack
of a better option (`.pose/knowledge/2026-08-15-decision-log-github-issue-
closing-keyword-format.md`), which will resurface in
`pose knowledge-housekeeping list-expired` in 90 days for no substantive
reason.

### Constraints
- Whatever the resolution, it must not weaken `knowledge-governance.md`'s
  TTL discipline for the three existing types — this is about adding (or
  explicitly declining to add) a distinct category, not loosening the
  existing one.
- Must not duplicate what `.pose/rules/` already does if a rule is in fact
  the correct existing home — see Decision 1 below, to be resolved before
  implementation starts.

### Non-goals
- Changing ADR's scope or lifecycle.
- Retroactively re-triaging every existing `decision-log`/`note` artifact
  — only the mechanism/guidance for *new* durable-but-non-architectural
  content is in scope.

---

## 2. Requirements

### Functional
- R1: A durable, non-architectural fact or convention shall have a
  documented, correctly-fitting home in POSE's governance model — either
  a new knowledge type exempt from mandatory `expires_at` (or with a
  materially longer, review-not-expire cadence), or explicit guidance
  routing this content to `.pose/rules/` instead, whichever Decision 1
  resolves to.
- R2: Whichever resolution is chosen, `pose-knowledge`'s skill and
  `knowledge-governance.md` shall document how to tell a `decision-log`/
  `note` apart from this new home, so a future author doesn't face the
  same "closest available type" guess this issue's investigation had to
  make.

### Non-functional
- No regression to `pose knowledge-check --strict`'s existing TTL
  enforcement for `handoff`/`decision-log`/`note`.

---

## 3. Technical Plan

### Affected areas
- `pose-mcp/internal/pose/knowledge.go` (if a new type is added)
- `.pose/rules/knowledge-governance.md`
- `.agents/skills/pose-knowledge/SKILL.md`

---

## 4. Tasks

### Planning
- [x] Confirm the gap empirically: recorded
      `.pose/knowledge/2026-08-15-decision-log-github-issue-closing-
      keyword-format.md` as a `decision-log` for lack of a better fit,
      documented the mismatch in this spec's Business value
- [ ] Resolve Decision 1 (new type vs. `.pose/rules/` routing) before
      moving to in-progress

### Implementation
- [ ] TBD — depends on Decision 1

### Validation
- [ ] TBD — depends on Decision 1

---

## 5. Decisions

### Decision 1
- Date: 2026-08-15
- Context: two candidate resolutions were identified during the issue
  investigation, not yet chosen between:
  (a) add a fourth knowledge type (e.g. `reference`) exempt from mandatory
  `expires_at`, or with a materially longer ceiling reviewed rather than
  expired-and-archived; (b) document that this content belongs in
  `.pose/rules/` instead — already the durable, non-expiring home for team
  conventions — and treat `.pose/knowledge/` as strictly for time-bound
  operational memory, no code change needed.
- Decision: not yet made — deferred to implementation. Whoever picks this
  up should first check whether `.pose/rules/` content is actually
  surfaced with enough discoverability for a fact like the trailer-format
  one (rules are consulted via `pose suggest <workflow> --path <dir>`,
  which is path/component-scoped — worth confirming a cross-cutting,
  no-specific-path convention like this one would actually reach an agent
  before adding new schema surface for the same job).
- Rationale: recording the fork explicitly rather than picking one
  prematurely — this spec's own Intent is itself an example of exactly the
  kind of durable-but-non-architectural content in question, and forcing
  a decision here without checking (b)'s discoverability first would risk
  building (a) for something (b) already solves.
- Consequences: this spec stays `draft`/`in-progress` with an unresolved
  fork until Decision 1 is closed; Implementation/Validation tasks above
  are placeholders pending that.

---

## 6. Validation

### Strategy
To be defined once Decision 1 resolves: either regression tests for a new
knowledge type's parsing/TTL-exemption in `pose-mcp/internal/pose`, or a
documentation-only validation (skill/rule content review, no code change).

### Known gaps
- Decision 1 is open; this spec cannot close until it resolves.

---

## 7. Final Report

### Follow-ups
- [open]
