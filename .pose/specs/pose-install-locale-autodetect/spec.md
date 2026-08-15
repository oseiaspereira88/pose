---
slug: pose-install-locale-autodetect
status: in-progress
created_at: 2026-08-15
completed_at:
supersedes:
depends_on:
priority: 1
components: pose-mcp, cli
delivers: capability:pose-mcp
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

### Artifacts
- modified: pose-mcp/internal/cli/install.go
- modified: pose-mcp/internal/cli/localization_docs_test.go

### Delivery targets
- capability:pose-mcp module:pose-mcp profile:composed-capability entrypoint:pose-mcp/cmd/pose/main.go

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
- [x] Confirm the exact gating condition: an explicit `--locale` (any
      value, including `en`) must always win over detection — see
      Decision 1, this could not be inferred from `resolveDocLocale`'s own
      short-circuit alone

### Implementation
- [x] Track `localeExplicit` when `--locale` is parsed; call
      `machineryLocale(dist, target, locale)` only when `!localeExplicit`,
      right before `deliverMachinery` (covers both the machinery copy and
      the `AGENTS.md`/`POSE.md` docs step, which both read the same
      `locale` variable)

### Validation
- [x] Regression tests in `localization_docs_test.go`: reinstall without
      `--locale` preserves detected pt-BR; fresh install without `--locale`
      still defaults to `en`; explicit `--locale en` (with `--force`)
      overrides detection
- [x] Existing fresh-install tests unaffected
- [x] Manually reverified against `~/GolandProjects/codass`: `pose install
      .` and `pose update --force` (both no `--locale`) now correctly log
      `locale: pt-BR` and produce zero diff; reverted afterward
- [x] `go -C pose-mcp test ./...`, `go -C pose-mcp vet ./...`, `gofmt -l`

---

## 5. Decisions

### Decision 1
- Date: 2026-08-15
- Context: an unconditional `locale = machineryLocale(dist, target, locale)`
  before every `cmdInstall` call — the simplest possible fix, and what
  `cmdUpdate`'s non-force path itself does — was tried first. It correctly
  fixed R1 (detect pt-BR when `--locale` is absent) but broke R3 for one
  specific input: `cmdInstall(..., "--locale", "en")` on an already
  pt-BR-localized target still produced pt-BR content, because
  `resolveDocLocale`'s short-circuit is `if preferred != "" && preferred !=
  "en"` — an explicit `en` is indistinguishable from the "no preference"
  default, so it falls through to content-based detection and gets
  overridden back to pt-BR. Caught by
  `TestInstallExplicitLocaleOverridesDetection` before this was assumed
  fixed. (This same ambiguity already exists, unfixed, in `cmdUpdate`'s own
  non-force path — `pose update --locale en` on an already pt-BR project
  would hit the identical override; out of scope here, since this spec's
  R3 only commits to "unchanged from today's contract" for `cmdInstall`.)
- Decision: track an explicit `localeExplicit` bool at flag-parse time;
  call `machineryLocale()` only when `!localeExplicit`.
- Rationale: distinguishing "the caller said `en`" from "nothing was said"
  requires information `resolveDocLocale` structurally cannot have (a
  three-state signal collapsed into a two-state string) — the caller
  (`cmdInstall`) is the only place that still knows which one happened.
- Consequences: `cmdInstall` now correctly handles the explicit-en case
  that `cmdUpdate`'s existing non-force path does not; the two diverge
  slightly, but only where `cmdUpdate`'s existing behavior was itself the
  bug (documented above, not touched by this spec).

### Decision 2
- Date: 2026-08-15
- Context: while validating R3 with `--locale en` (no `--force`) against an
  already pt-BR fixture, the resulting `AGENTS.md` still contained
  Portuguese content — not a locale-resolution bug, but a separate,
  pre-existing property of `MergeManagedDoc`: it matches engine-owned
  sections between canonical and local text **by heading string**. A
  Portuguese heading (`## Contexto do projeto`) never matches its English
  counterpart (`## Project context`), so the merge treats every Portuguese
  section as instance-added content and appends it after the (correctly
  resolved, correctly English) canonical sections — both languages end up
  concatenated in the same file. `--force` bypasses the merge entirely
  (wholesale overwrite) and produces clean English, confirming the locale
  resolution itself was correct; only the *unforced* merge path has this
  gap.
- Decision: scope this spec to locale *resolution* only (R1-R3, all about
  which content `deliverMachinery`/the docs step reads from) and adjusted
  `TestInstallExplicitLocaleOverridesDetection` to assert against the
  `--force` (full-overwrite) path, which is what a real "switch this
  project's language" operation would use in practice. Filed the
  merge-by-heading limitation as a follow-up rather than fixing it here.
- Rationale: fixing cross-language section matching in `MergeManagedDoc`
  (e.g. by a language-neutral section ID instead of heading text) is a
  separate, larger change to the merge contract — the anti-pattern this
  workflow explicitly warns against (`bugfix.md`: "no parallel
  refactoring"). It also only matters for the *unforced* reinstall-with-a-
  different-explicit-locale case, which is narrower and less surprising
  than R1's silent revert (the unforced, no-`--locale` case correctly
  produces zero drift today, since canonical and local headings match once
  detection resolves to the same language).
- Consequences: an operator who explicitly reruns `pose install --locale
  en` (no `--force`) on a pt-BR project without also passing `--force`
  gets a file with both languages concatenated, not a clean switch — a
  real but narrower and less silent gap than R1's, tracked as a follow-up.

---

## 6. Validation

### Strategy
Unit-level regression in `pose-mcp/internal/cli`: install a pt-BR fixture,
rerun `cmdInstall` without `--locale`, assert zero content drift (matching
the manual reproduction against codass). A second case confirms a fresh
target (no prior `POSE.md`) still defaults to `en`. A third confirms an
explicit `--locale en` (via `--force`, see Decision 2) overrides detection.

### Requirement trace
- R1 [satisfied] `TestInstallReinstallDetectsExistingLocaleWithoutFlag`;
  reverified against `~/GolandProjects/codass`.
- R2 [satisfied] `TestInstallFreshWithoutLocaleStillDefaultsToEnglish`.
- R3 [satisfied] `TestInstallExplicitLocaleOverridesDetection`; see
  Decision 1 for the edge case that required `localeExplicit` rather than
  an unconditional detection call.

### Known gaps
- The transactional/rollback finding (issue #18, secondary) is explicitly
  out of scope — tracked as a follow-up, not fixed here.
- The merge-by-heading language-matching limitation found in Decision 2 is
  a second, narrower follow-up — also not fixed here.

---

## 7. Final Report

### Delivered scope
`cmdInstall` now detects an already-installed target's existing locale
(reusing `machineryLocale`/`resolveDocLocale`, the same mechanism
`cmdUpdate`'s non-force path already used) whenever `--locale` is not
passed explicitly, fixing the silent English revert in both
`pose install <existing-target>` and `pose update --force`. An explicit
`--locale` (any value) still always wins, including the `en`-vs-detection
edge case a naive fix would have missed (Decision 1).

### Files and modules changed
- `pose-mcp/internal/cli/install.go`: `localeExplicit` tracking at flag
  parse time; conditional `machineryLocale()` call before machinery/docs
  delivery.
- `pose-mcp/internal/cli/localization_docs_test.go`: three new regression
  tests (`TestInstallReinstallDetectsExistingLocaleWithoutFlag`,
  `TestInstallFreshWithoutLocaleStillDefaultsToEnglish`,
  `TestInstallExplicitLocaleOverridesDetection`).

### Validation executed
- `go -C pose-mcp test ./internal/cli/... -run TestInstall`: SUCCESS (8/8,
  including the 3 new tests).
- `go -C pose-mcp test ./...`: SUCCESS.
- `go -C pose-mcp vet ./...`: SUCCESS.
- `gofmt -l`: clean.
- Manual reverification against `~/GolandProjects/codass` (real pt-BR
  project, not a fixture): `pose install .` and
  `pose update --no-self --force` (both without `--locale`) now log
  `locale: pt-BR` and produce zero diff; reverted, working tree left clean.

### Residual risks
- None beyond the two documented follow-ups (transactional gate;
  merge-by-heading language matching), both pre-existing and out of scope.

### Follow-ups
- [open] `pose install`'s final `--strict` gate (install.go:262) runs after
  every file is already written and does not roll back on failure —
  recoverable via `.pose-backup` + git, but not transactional. Consider
  either gating before mutation (dry-run the check first) or explicitly
  documenting that a failed install/update can still leave mutated files
  (owner:@pose-maintainers crit:medium review:2026-09-15)
- [open] `MergeManagedDoc` matches sections by heading text, so an explicit
  `--locale` switch on `AGENTS.md`/`POSE.md` without `--force` concatenates
  both languages instead of cleanly switching (Decision 2) — only the
  merge (unforced) path is affected; `--force` already produces a clean
  result. Consider a language-neutral section identifier if this proves to
  matter in practice (owner:@pose-maintainers crit:low review:2026-10-01)
