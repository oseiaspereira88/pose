---
slug: pose-one-follow-up-format
status: in-progress
completed_at:
created_at: 2026-09-10
supersedes:
depends_on: pose-followup-ownership-sla
priority: 0
components: pose-mcp
task_type: bugfix
delivers:
---

# Spec: One follow-up format, shown where follow-ups are written

## 1. Intent

### Goal
Every place POSE tells an author how to write a follow-up shall show the one
format its parser reads, and the lint shall read follow-ups the way `pose
followups` does and say so when ownership is written any other way.

### Business value
`pose followups` reads ownership from one place: a trailing parenthesized group,
`(owner:@alias crit:low|medium|high review:YYYY-MM-DD)`. Anything else is
ignored without a word, and the item reads as unowned, with no criticality and
no review date — so it never becomes overdue.

Fifty-five follow-ups in this repository, written since 2026-09-07, put the
fields after a dash instead: `— owner:unowned crit:low review:…`. Twenty-seven
are open. None surfaced, for two reasons measured here:

- The spec template's `### Follow-ups` section — where an author actually writes
  one — explains the dispositions and shows `- [open] ` with no metadata. The
  format lived in POSE.md, an ADR and the docs site; one docs page even showed
  it without its parentheses.
- The lint checks ownership only on `done` specs, and treats metadata outside
  the group as simply absent. All fifty-five sat on specs still `in-progress`.

Measuring the lint turned up a second defect in the same place. It read only a
bullet's first line, while `pose followups` joins continuation lines. A wrapped
ownership group was therefore owned for one reader and "unowned" for the other:
the lint warned on three items `pose followups` reads correctly. The fix for
wrapping had been made in one of the two readers.

The same drift existed in harne8, eighteen open items with two `crit:high`
among them. That repository is migrated separately.

### Constraints
- The new diagnostic is a warning: an unowned item is already a warning at
  closeout, and ownership written in the wrong place is the same fact.

### Non-goals
- Accepting the dash form. Two formats would be the problem this removes.
- Giving the migrated items an owner. They were written `owner:unowned`; the
  migration moves the fields, and assigning owners is triage.

---

## 2. Requirements

### Functional
- R1: The spec template, in both locales, shall show the canonical follow-up
  line and state that ownership written any other way is ignored.
- R2: The closeout skill, in both locales, shall show the same line, and no
  guidance shall describe `[open]` as ownerless.
- R3: POSE.md and the docs site shall state the same rule, and show the group
  with its parentheses.
- R4: The lint shall join a follow-up's continuation lines as `pose followups`
  does.
- R5: The lint shall warn, in any spec status, when an open follow-up carries
  ownership fields outside the trailing group.
- R6: This repository's follow-ups shall use the canonical format.

### Non-functional
- The embedded copies match their sources (`go generate`).

---

## 3. Technical Plan

### Affected areas
- `pose-mcp/internal/cli/lintspec.go`, `followups.go` — reading and diagnosing
- `.pose/templates/spec.md`, `.agents/skills/pose-spec-closeout/SKILL.md`,
  `POSE.md`, their pt-BR translations and embedded copies
- `docs-site/docs/concepts.md`, `architecture.md`

### Artifacts
- created: .pose/specs/2026-09-10-pose-one-follow-up-format.md
- renamed: .pose/changelogs/unreleased/pose-one-follow-up-format.md -> .pose/changelogs/v5.0.2/pose-one-follow-up-format.md
- modified: pose-mcp/internal/cli/lintspec.go
- modified: pose-mcp/internal/cli/followups.go
- modified: pose-mcp/internal/cli/followups_owner_test.go
- modified: .pose/templates/spec.md
- modified: locales/pt-BR/.pose/templates/spec.md
- modified: .agents/skills/pose-spec-closeout/SKILL.md
- modified: locales/pt-BR/.agents/skills/pose-spec-closeout/SKILL.md
- modified: POSE.md
- modified: locales/pt-BR/POSE.md
- modified: docs-site/docs/concepts.md
- modified: docs-site/docs/architecture.md
- modified: pose-mcp/internal/scaffold/dist/.pose/templates/spec.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/.pose/templates/spec.md
- modified: pose-mcp/internal/scaffold/dist/.agents/skills/pose-spec-closeout/SKILL.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/.agents/skills/pose-spec-closeout/SKILL.md
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md

### Technical risks
- A follow-up whose prose mentions `owner:` or `review:` without being metadata
  would draw the warning. It is a warning, and the template now says where the
  fields go.

---

## 4. Tasks

### Implementation
- [x] Increment 1: Show the canonical line where follow-ups are written (R1, R2, R3)
- [x] Increment 2: The lint joins continuation lines (R4)
- [x] Increment 3: Warn on ownership outside the trailing group (R5)
- [x] Increment 4: Migrate this repository's follow-ups (R6)

### Validation
- [x] Both new lint tests fail against the previous lint

---

## 5. Decisions

### Decision 1
- Date: 2026-09-10
- Context: the dash form could be made a second accepted syntax, which would
  have fixed fifty-five items without touching them.
- Decision: keep one format and migrate.
- Rationale: the defect is an author not knowing which format is read. A second
  syntax makes that question harder, and every reader of follow-ups — the lint,
  `pose followups`, the MCP projection, the portfolio — would have to learn it.

### Decision 2
- Date: 2026-09-10
- Context: the misplaced-ownership diagnostic could be an error at closeout, as
  a malformed group is.
- Decision: a warning, in every status.
- Rationale: an error would break `pose check --strict` for instances with
  done specs written this way, on adopting a patch. Warning before closeout is
  what would have caught these fifty-five: they were all `in-progress`.

---

## 6. Validation

### Strategy
Measure both readers before and after on this repository, and require the new
lint behaviour to fail against the previous lint.

### Deterministic checks

#### Test
- Command: `go test -count=1 ./...`
- Scope: `pose-mcp`
- Expected: exit 0

#### Lint
- Command: `pose lint-spec --all`
- Scope: this instance
- Expected: no "outside the trailing group" warning

### Execution log
- Date: 2026-09-10
- Environment: local, Go 1.26
- Notes: before the migration, the new lint warned on exactly the 27 open
  dash-form items, and its "unowned" warnings fell from 20 to 17 — the three
  wrapped groups it now reads. After migrating 55 lines in 41 specs the warning
  count is 0, and the 27 items carry `crit:low` and a review date in `pose
  followups`. Both new lint tests fail with the previous `lintspec.go`.
  `pose lint-spec --all` still exits 1, as it does on 5.0.1, for four errors in
  `pose-manual-and-cli-command-parity`, a done spec missing sections.

### Results summary
- Successes: R1–R6 verified.
- Failures: none.

### Requirement trace
- R1 [satisfied] <both spec templates show `- [<disposition>] <text> (owner:@alias crit:low|medium|high review:YYYY-MM-DD)` and state that any other placement is ignored>
- R2 [satisfied] <both closeout skills show the same line; the pt-BR table no longer calls `[open]` ownerless; both list the misplacement as an anti-pattern>
- R3 [satisfied] <POSE.md in both locales and docs-site concepts/architecture state the rule; architecture.md now shows the group with its parentheses>
- R4 [satisfied] <TestLintReadsAWrappedOwnershipGroupLikeFollowups lints a done spec whose open item wraps before its group and requires no unowned warning>
- R5 [satisfied] <TestLintWarnsOwnershipOutsideTheTrailingGroup requires the warning, naming the format, for in-progress and done specs, and none for the canonical group>
- R6 [satisfied] <no `— owner:` follow-up remains under .pose/specs; `pose lint-spec --all` reports no misplaced ownership>

### Known gaps
- The migrated items are still `owner:unowned`; they now carry a criticality and
  a review date, so they surface when due.

---

## 7. Final Report

### Summary
The format POSE reads is the format POSE shows, and the lint says so when a
follow-up is written any other way.

### Follow-ups
