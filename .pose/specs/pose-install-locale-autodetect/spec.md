---
slug: pose-install-locale-autodetect
status: draft
created_at: 2026-08-15
completed_at:
supersedes:
depends_on:
priority: 1
components: pose-mcp, cli
delivers:
---

# Spec: pose-install-locale-autodetect

---

## 1. Intent

### Goal
Make `cmdInstall` (`pose-mcp/internal/cli/install.go`) detect an already-
installed target's existing locale the same way `cmdUpdate`'s non-force path
already does, instead of silently defaulting to English whenever `--locale`
isn't passed explicitly.

### Business value
`github.com/oseiaspereira88/pose#18` (open): reproduced against
`~/GolandProjects/codass`, a real pt-BR-localized project. Two real, easy-to-
trigger paths silently revert an already-localized project's entire
machinery (`.pose/rules`, `.pose/templates`, `.pose/workflows`,
`.agents/skills`, `AGENTS.md`, `POSE.md`) to English, with no warning that
the locale changed:
1. `pose install <target>` run again on an existing instance (e.g. to repair
   something) — no locale detection at all.
2. `pose update --force` without `--locale` — its force branch
   (`maintenance.go:151-163`) delegates entirely to `cmdInstall`, bypassing
   the detection the `!force` branch already performs correctly via
   `machineryLocale()` (`maintenance.go:145`, reads the existing `POSE.md`).

`pose update` (no `--force`) already gets this right — it is specifically
the `cmdInstall` code path (used directly, or via `update --force`) that
lacks the detection. A user who reaches for `--force` or reruns `install` to
fix something unrelated has no reason to expect their project's language to
flip.

### Constraints
- Do not change behavior for a genuinely fresh install (no pre-existing
  `POSE.md`): it must keep defaulting to `en` when `--locale` is omitted —
  there is nothing to detect yet.
- Do not change behavior when `--locale` is passed explicitly — an explicit
  flag always wins, matching `cmdUpdate`'s existing contract.
- Reuse `machineryLocale()`/`resolveDocLocale` rather than inventing a
  second detection mechanism that could drift from `cmdUpdate`'s.

### Non-goals
- Making `pose install`/`pose update --force` transactional (the secondary
  finding in issue #18: the final `--strict` gate runs after files are
  already written and does not roll back on failure). Real, but lower
  severity — recoverable via the `.pose-backup` files already written and
  tracked in git. Follow-up, not this spec's fix.

---

## 2. Requirements

### Functional
- R1: When `cmdInstall` runs against a target where `.pose/POSE.md` already
  exists and `--locale` was not passed, it shall detect the existing
  locale via the same mechanism `cmdUpdate`'s non-force path uses
  (`machineryLocale`/`resolveDocLocale` reading the existing `POSE.md`)
  instead of defaulting to `en`.
- R2: When `.pose/POSE.md` does not exist (fresh install) and `--locale`
  was not passed, `cmdInstall` shall keep defaulting to `en` — unchanged.
- R3: When `--locale <tag>` is passed explicitly to `cmdInstall` (directly,
  or via `pose update --force --locale <tag>`), it shall take precedence
  over detection — unchanged from today's contract.

### Non-functional
- No new external dependency or network call; detection reads only local
  files already on disk (same cost `cmdUpdate` already pays).

### Compatibility
- Purely corrective for an already-installed target; does not change the
  on-disk schema or CLI flag surface.

---

## 3. Technical Plan

### Affected areas
- `pose-mcp/internal/cli/install.go`

### Technical risks
- Low: mirrors an already-shipped, already-tested detection path
  (`cmdUpdate`'s non-force branch); the risk is confined to correctly
  gating it on "target already has a POSE.md" so a fresh install's `en`
  default is untouched.

---

## 4. Tasks

### Planning
- [x] Reproduce against `~/GolandProjects/codass` (real pt-BR project):
      `pose update --no-self` (no-op, correct) vs `pose update --no-self
      --force` (72 files reverted to English) vs `pose install <same-dir>`
      (same regression, no `--force` needed)
- [x] Trace root cause to `cmdInstall`'s hardcoded `locale = "en"`
      (install.go:25) never being overridden by `machineryLocale()`
      detection, which only `cmdUpdate`'s `!force` branch calls
- [ ] Confirm the exact gating condition (existing `POSE.md` vs. any other
      already-installed signal) before implementing

### Implementation
- [ ] Call `machineryLocale()`-equivalent detection in `cmdInstall` when
      target already has `.pose/POSE.md` and `--locale` is unset

### Validation
- [ ] Regression test: re-running `cmdInstall` on an already pt-BR-localized
      fixture without `--locale` produces zero diff (mirrors
      `TestDoctorSilentOnLegitimatePolicyRoots`-style fixture pattern)
- [ ] Existing fresh-install tests still default to `en` unchanged
- [ ] `pose validate --tolerant --module pose-mcp/internal/cli`

---

## 6. Validation

### Strategy
Unit-level regression in `pose-mcp/internal/cli`: install a pt-BR fixture,
rerun `cmdInstall` without `--locale`, assert zero content drift (matching
the manual reproduction against codass). A second case confirms a fresh
target (no prior `POSE.md`) still defaults to `en`. A third confirms
`--locale` still overrides detection either way.

### Known gaps
- The transactional/rollback finding (issue #18, secondary) is explicitly
  out of scope — tracked as a follow-up, not fixed here.

---

## 7. Final Report

### Follow-ups
- [open] `pose install`'s final `--strict` gate (install.go:262) runs after
  every file is already written and does not roll back on failure —
  recoverable via `.pose-backup` + git, but not transactional. Consider
  either gating before mutation (dry-run the check first) or explicitly
  documenting that a failed install/update can still leave mutated files
  (owner:@pose-maintainers crit:medium review:2026-09-15)
