---
slug: pose-knowledge-durable-reference-type
status: in-progress
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
- `.pose/rules/knowledge-governance.md`
- `.agents/skills/pose-knowledge/SKILL.md`
- `locales/pt-BR/.pose/rules/knowledge-governance.md`
- `locales/pt-BR/.agents/skills/pose-knowledge/SKILL.md`
- `pose-mcp/internal/scaffold/dist/**` (regenerated, not hand-edited)

### Artifacts
- modified: .pose/rules/knowledge-governance.md
- modified: .agents/skills/pose-knowledge/SKILL.md
- modified: locales/pt-BR/.pose/rules/knowledge-governance.md
- modified: locales/pt-BR/.agents/skills/pose-knowledge/SKILL.md
- modified: pose-mcp/internal/scaffold/dist/.pose/rules/knowledge-governance.md
- modified: pose-mcp/internal/scaffold/dist/.agents/skills/pose-knowledge/SKILL.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/.pose/rules/knowledge-governance.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/.agents/skills/pose-knowledge/SKILL.md
- created: .pose/adr/2026-08-15-durable-non-architectural-knowledge-belongs-in-rules-not-a-new-type.md
- created: .pose/changelogs/unreleased/pose-knowledge-durable-reference-type.md

---

## 4. Tasks

### Planning
- [x] Confirm the gap empirically: recorded
      `.pose/knowledge/2026-08-15-decision-log-github-issue-closing-
      keyword-format.md` as a `decision-log` for lack of a better fit,
      documented the mismatch in this spec's Business value
- [x] Resolve Decision 1 (new type vs. `.pose/rules/` routing) before
      moving to in-progress — see Decision 1 and
      `.pose/adr/2026-08-15-durable-non-architectural-knowledge-belongs-in-rules-not-a-new-type.md`

### Implementation
- [x] Added a "When this is not the right home" section to
      `.pose/rules/knowledge-governance.md` (EN + pt-BR): durable facts
      with no review trigger route to `.pose/rules/`, wired into a task
      type's base `rules` array in `.pose/indexes/task-map.json` rather
      than filed as `decision-log`
- [x] Added the same distinction to `.agents/skills/pose-knowledge/
      SKILL.md` (EN + pt-BR), right after the artifact-type list
- [x] Regenerated `pose-mcp/internal/scaffold/dist/**` via
      `go generate ./internal/scaffold` so the embedded copies used by
      `pose install`/`pose update` on other instances match

### Validation
- [x] `pose check --strict` after edits: clean
- [x] Manual content review: no change to `expires_at` enforcement wording
      or the three existing artifact types' definitions — new section is
      additive
- [x] Confirmed no retroactive re-triage: `github-issue-closing-keyword-
      format.md` untouched, still expires 2026-11-13 on its own schedule

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
- Decision: **(b) — route to `.pose/rules/`, no fourth knowledge type.**
  Investigated `pose-mcp/internal/cli/suggest.go` and
  `.pose/indexes/task-map.json`: each task type's base `rules` array is
  returned by `pose suggest <task>` unconditionally, with no `--path`
  required; `--path`/`--domain` only resolves an *additional*
  `rules_by_domain` layer for language/component-specific rules. A
  cross-cutting convention with no natural path scope belongs in a task
  type's base `rules` list, which every `pose-*` skill's Required Reading
  step already surfaces before work starts — stronger discoverability than
  `.pose/knowledge/`'s opportunistic `find`-based search offers today. Full
  rationale, rejected trade-off and consequences recorded in
  `.pose/adr/2026-08-15-durable-non-architectural-knowledge-belongs-in-rules-not-a-new-type.md`.
- Rationale: recording the fork explicitly rather than picking one
  prematurely — this spec's own Intent is itself an example of exactly the
  kind of durable-but-non-architectural content in question, and forcing
  a decision here without checking (b)'s discoverability first would risk
  building (a) for something (b) already solves. The discoverability check
  confirmed (b) is not just adequate but *stronger* than (a) would have
  been.
- Consequences: implementation scope is documentation-only — no
  `pose-mcp/internal/pose/knowledge.go` change. `.pose/rules/
  knowledge-governance.md` and `.agents/skills/pose-knowledge/SKILL.md`
  gain guidance distinguishing "this is a `decision-log`" from "this
  belongs in `.pose/rules/`"; no retroactive re-triage of the existing
  `github-issue-closing-keyword-format` artifact (out of scope per this
  spec's Non-goals).

---

## 6. Validation

### Strategy
Documentation-only: content review of the added guidance in both locales
plus the ADR, `pose check --strict` for structural conformance, and a scan
confirming the existing `github-issue-closing-keyword-format` artifact was
left untouched (Non-goals explicitly exclude retroactive re-triage).

### Requirement trace
- R1 [satisfied] Decision 1 resolved to routing durable, non-architectural
  content to `.pose/rules/`; documented in the ADR and referenced from
  Decision 1 below.
- R2 [satisfied] `.pose/rules/knowledge-governance.md` and
  `.agents/skills/pose-knowledge/SKILL.md` (EN + pt-BR) both gained the
  "not a decision-log if nothing was decided/nothing expires" distinction.

### Known gaps
- None identified. No code path changes, so no regression surface beyond
  the doc/skill content itself, covered by manual review above.

---

## 7. Final Report

### Delivered scope
Resolved the open fork via ADR
`2026-08-15-durable-non-architectural-knowledge-belongs-in-rules-not-a-new-type`:
no fourth knowledge type. `.pose/knowledge/` stays strictly
`handoff`/`decision-log`/`note`, all mandatorily TTL'd. Durable,
non-architectural facts and conventions route to `.pose/rules/` instead —
confirmed adequately discoverable because a task type's base `rules` array
in `.pose/indexes/task-map.json` is returned by `pose suggest <task>`
unconditionally (no `--path` required); `--path`/`--domain` only resolves
an additional layer for language/component-specific rules.
`knowledge-governance.md` and the `pose-knowledge` skill (EN + pt-BR) now
document the distinction so a future author does not have to guess.

### Files and modules changed
- `.pose/rules/knowledge-governance.md`,
  `locales/pt-BR/.pose/rules/knowledge-governance.md`: new "When this is
  not the right home" section.
- `.agents/skills/pose-knowledge/SKILL.md`,
  `locales/pt-BR/.agents/skills/pose-knowledge/SKILL.md`: routing guidance
  after the artifact-type list.
- `pose-mcp/internal/scaffold/dist/**`: regenerated via
  `go generate ./internal/scaffold`.
- `.pose/adr/2026-08-15-durable-non-architectural-knowledge-belongs-in-rules-not-a-new-type.md`:
  new ADR.
- `.pose/changelogs/unreleased/pose-knowledge-durable-reference-type.md`:
  new fragment.

### Validation executed
- `pose check --strict`: SUCCESS.
- Manual review: existing TTL enforcement and artifact-type definitions
  unchanged; new content additive only.

### Residual risks
- None identified. Documentation-only change; no code path affected.

### Follow-ups
- [wont-do: retroactive re-triage of `github-issue-closing-keyword-format.md` explicitly excluded by this spec's Non-goals — it expires on its existing 2026-11-13 schedule with no action needed]
