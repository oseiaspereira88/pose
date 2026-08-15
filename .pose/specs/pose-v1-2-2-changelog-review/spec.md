---
slug: pose-v1-2-2-changelog-review
status: draft
created_at: 2026-08-15
completed_at:
supersedes:
depends_on:
priority: 3
components: pose-mcp
delivers:
---

# Spec: pose-v1-2-2-changelog-review

---

## 1. Intent

### Goal
Revisit the two pending fragments in `.pose/changelogs/unreleased/` — for
`pose-review-bundle-root-file-classification` and
`pose-review-scope-trailer-check` — for accuracy and completeness before
`pose release prepare --version v1.2.2` is run, and decide whether to cut
now or let more candidates accumulate first.

### Business value
Both fragments were authored in the same session that shipped them, close
to the code, without a second look once the full set of closed-today specs
was known together. `pose-review-scope-trailer-check` in particular went
through a mid-implementation course-correction (its own Decision 2): the
fragment's wording should be checked against the *final*, corrected
behavior — not the first draft. Reviewing before cutting is cheap; a wrong
or stale changelog entry shipped in a release is not (`.pose/changelogs/v*.md`
are immutable once a version is tagged).

### Constraints
- Read-only review of existing fragments and already-closed specs; do not
  reopen or re-implement `pose-review-bundle-root-file-classification` or
  `pose-review-scope-trailer-check` — both are `status: done` and released
  code is out of scope here.
- The actual `pose release prepare`/publish/verify cycle is a separate,
  later action (skill `pose-release-closeout`), not part of this spec.

### Non-goals
- Deciding the next version number's scope beyond what these two fragments
  already cover — no new feature work is pulled in by this spec.

---

## 2. Requirements

### Functional
- R1: For each pending fragment, confirm its `category`/`breaking`/`refs`
  frontmatter and body match the corresponding spec's Final Report
  ("Delivered scope") exactly as closed, not as first drafted.
- R2: Confirm the two fragments do not overlap or need consolidating (they
  touch different mechanisms — review-subject classification vs. a new
  doctor check — but both are review-bundle-adjacent, worth a deliberate
  check).
- R3: Record an explicit go/no-go decision on cutting v1.2.2 now versus
  waiting for more candidates, with rationale.

### Compatibility
- Neither fragment is `breaking: true`; confirm that classification still
  holds under review.

---

## 3. Technical Plan

### Artifacts
- none: governance

---

## 4. Tasks

### Planning
- [ ] Re-read both closed specs' Final Reports in full
      (`pose-review-bundle-root-file-classification`,
      `pose-review-scope-trailer-check`) against their fragments

### Implementation
- [ ] Correct fragment wording/frontmatter in place if any drift is found
- [ ] Record the go/no-go cut decision (R3) in Decisions below

### Validation
- [ ] `pose release status` confirms v1.2.1 is still `verified` and no
      release is `prepared` yet (nothing to conflict with)

---

## 6. Validation

### Strategy
Manual comparative read: each fragment's body against its spec's "Delivered
scope"/"Files and modules changed" sections, plus `pose lint-spec <slug>
--strict` on both closed specs to confirm nothing about their frontmatter
changed underneath this review (they must stay `status: done`, untouched).

### Known gaps
- None yet — to be filled once the review runs.

---

## 7. Final Report

### Follow-ups
- [open] 
