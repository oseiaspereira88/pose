---
slug: pose-v1-2-2-changelog-review
status: done
created_at: 2026-08-15
completed_at: 2026-08-15
changelog: none
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
- [x] Re-read both closed specs' Final Reports in full
      (`pose-review-bundle-root-file-classification`,
      `pose-review-scope-trailer-check`) against their fragments

### Implementation
- [x] Correct fragment wording/frontmatter in place if any drift is found
- [x] Record the go/no-go cut decision (R3) in Decisions below

### Validation
- [x] `pose release status` confirms v1.2.1 is still `verified` and no
      release is `prepared` yet (nothing to conflict with)

---

## 5. Decisions

### Decision 1
- Date: 2026-08-15
- Context: R1/R2 comparative read of both pending fragments against their
  closed specs' Final Reports.
- Findings:
  - `pose-review-bundle-root-file-classification.md`: `category: fixed`,
    `breaking: false`, body accurately describes the shipped
    `reviewBundlePathClass` classification fix. No drift. `refs:` correctly
    empty — this fix was discovered mid-session while closing an unrelated
    spec, not tied to any tracked issue.
  - `pose-review-scope-trailer-check.md`: `category: added`,
    `breaking: false`, body describes the *final, corrected* `doctor` check
    name (`review.scope-change-set`) and behavior — not the initial
    trailer-based premise the spec's own Decision 2 walked back. No content
    drift. One formatting drift found and fixed: `refs: [ISSUE#17]` used
    YAML flow-list brackets, inconsistent with every other fragment in this
    repository's history (`refs: ISSUE#17`, bare); corrected to match.
  - R2 (overlap check): the two fragments cover distinct mechanisms
    (review-subject path classification vs. a new `doctor` diagnostic) with
    no shared wording or redundant claims — no consolidation needed.
- Decision (R3): **Go.** Both fragments are accurate and ready as-is; v1.2.2
  can be cut whenever desired. Not cutting automatically as part of this
  spec — `pose release prepare --version v1.2.2` is a separate, explicit
  action (skill `pose-release-closeout`) this spec does not take.
- Rationale: no defects found worth blocking on; the one drift (refs
  bracket syntax) was cosmetic and is now fixed. There's no repository
  policy requiring a minimum fragment count before cutting (v1.2.1 shipped
  with exactly one), so waiting for more candidates is a preference, not a
  requirement — left to the user's call.
- Consequences: none beyond the one-line fragment fix; no code or other
  specs touched.

### Decision 2
- Date: 2026-08-15
- Context: `pose close` rejected this spec with `undeclared-delivery:
  changed delivery root pose-mcp has no delivery declaration`, despite
  `### Artifacts` declaring `- none: governance` (no source claims at all).
  Traced to `LoadDeliveryPolicy` (`pose-mcp/internal/pose/delivery_surface.go`
  ~line 96): it derives an *implicit* whole-module root
  `DeliveryRoot{Path: "pose-mcp", ...}` from `.pose/indexes/module-metadata.json`
  whenever that module has `deliveryRole`/`entrypoints` set — in addition to
  the three explicit `pose-mcp/internal/{cli,mcpserver,pose}` sub-roots in
  `.pose/policy/delivery.json`. The trigger was this spec's own second
  `pose report --change-from b48e462 --change-to 6c92e3a`: that range
  unintentionally squashed in an *intervening* evidence-refresh commit
  (`46b5216`) that synced `pose-mcp/internal/scaffold/dist/...` as a
  mechanical side effect of `go generate ./internal/scaffold` — a path the
  *artifact* policy's exclusions list explicitly excludes from governance,
  but delivery-root matching doesn't consult that same exclusion list.
  First attempted fix was declaring `delivers: capability:pose-mcp`, which
  traded that error for a new one, `unconnected-artifact: delivery target
  has no structured artifact provenance` — a `none:` spec has no artifact
  claims to connect a delivery target to, so the two frontmatter fields
  are mutually exclusive in this schema.
- Options considered: (a) declare a delivery target anyway, accepting the
  `unconnected-artifact` contradiction is unresolvable for a `none:` spec;
  (b) fabricate an artifact claim for the scaffold-mirror path just to
  connect the delivery target, even though that path is a mechanical,
  policy-excluded byproduct, not something this spec designed; (c) remove
  the wide, over-squashed change set and replace it with commit-scoped
  single-commit ranges that never cross a pose-mcp/-touching commit, and
  drop `delivers:` again since nothing pose-mcp-shaped is genuinely in
  this spec's change set once correctly scoped.
- Decision: (c). Deleted the over-wide persisted change set
  (`.pose/reports/*-follow-up.{md,jsonl}`, `range:b48e462..6c92e3a`) and
  replaced it with two single-commit ranges
  (`range:6c92e3a~1..6c92e3a`, `range:74d8067~1..74d8067`), each touching
  only `.pose/specs/pose-v1-2-2-changelog-review/spec.md`.
- Rationale: (a)/(b) both accept or manufacture provenance this spec
  doesn't actually have — the honest fix is correcting the change-set
  boundary that was wrong in the first place, not working around its
  consequence. A `pose report` range should track the actual commit(s)
  that deliver a spec's content, not incidentally absorb whatever
  evidence-refresh housekeeping happened to land between them in git
  history.
- Consequences: confirms a durable operating lesson — when persisting
  change sets across several closeout-evidence commits for the same spec,
  prefer several narrow single-commit `pose report` calls over one wide
  range spanning intermediate index/validate/scaffold-sync commits,
  *especially* for a spec with no real `delivers:` target to fall back on.

---

## 6. Validation

### Strategy
Manual comparative read: each fragment's body against its spec's "Delivered
scope"/"Files and modules changed" sections, plus `pose lint-spec <slug>
--strict` on both closed specs to confirm nothing about their frontmatter
changed underneath this review (they must stay `status: done`, untouched).

### Requirement trace
- R1 [satisfied] Decision 1 — both fragments compared against Final Report
  content; one wording/formatting drift found (refs bracket syntax) and
  corrected.
- R2 [satisfied] Decision 1 — confirmed no overlap between the two
  fragments.
- R3 [satisfied] Decision 1 — recorded explicit "Go" with rationale.

### Known gaps
- None.

---

## 7. Final Report

### Delivered scope
Reviewed both pending `.pose/changelogs/unreleased/` fragments against
their closed specs' Final Reports; corrected one formatting drift
(`refs: [ISSUE#17]` → `refs: ISSUE#17` in
`pose-review-scope-trailer-check.md`, matching this repository's
established bare-reference convention); confirmed no overlap between the
two; recorded an explicit "Go" decision for cutting v1.2.2, left for the
user to trigger.

### Files and modules changed
- `.pose/changelogs/unreleased/pose-review-scope-trailer-check.md`: one-line
  `refs:` formatting fix.

### Validation executed
- `pose lint-spec pose-review-bundle-root-file-classification --strict`:
  SUCCESS.
- `pose lint-spec pose-review-scope-trailer-check --strict`: SUCCESS.
- `pose release status`: v1.2.1 `verified`, no release `prepared`, 2
  pending fragments (matches this review's scope exactly).

### Residual risks
- None.

### Follow-ups
- [open] Actually run `pose release prepare --version v1.2.2` when the user
  chooses to cut (owner:@pose-maintainers crit:low review:2026-09-01)
- [open] `cmdArtifactCheck` (`pose-mcp/internal/cli/artifact_integrity.go`
  ~line 318-332) reports a spurious `resolvability` error on any spec whose
  `### Artifacts` declares `- none: <reason>` — it validates `claim.Path`
  for every claim regardless of `Action`, and a none-action claim's `Path`
  is always empty by construction. Found running `pose artifact-check
  --spec pose-v1-2-2-changelog-review --strict` (exit 1) even though `pose
  check --strict`'s own delivery-graph build handles none-claims correctly
  (exit 0) — the bug is confined to this one standalone command's extra
  validation loop. Fix: skip claims with `Action == "none"` in that loop
  (owner:@pose-maintainers crit:low review:2026-09-15)
