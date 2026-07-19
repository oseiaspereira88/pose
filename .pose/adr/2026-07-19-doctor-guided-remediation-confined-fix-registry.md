# ADR: Doctor-guided remediation — a confined fix registry, additive JSON schema

## Status
Accepted (2026-07-19) — spec `pose-doctor-guided-remediation`

## Context

`pose doctor` (spec `pose-doctor`) was strictly read-only: every finding had
a `check`/`level`/`message`/`hint`, but nothing distinguished a finding a
human must act on externally (missing `git`) from one POSE itself could
safely repair (a missing pre-commit hook, a stale `.mcp.json`, a dangling
`.claude/skills` symlink) — every first-run failure meant reading a hint and
running a second command by hand. The existing JSON output
(`{findings, errors, warnings}`) is a public contract other tooling can
already consume (the installer E2E and the release compatibility gate both
parse `pose doctor --json`), so any change had to be additive, never a
rename or removal.

Alternatives considered:

1. **A generic "auto-fix everything" flag that re-runs `pose install`.**
   Simplest to implement, but violates the spec's own constraint (default
   to advice/dry-run, explicit apply for mutation) and non-goal (never edit
   arbitrary files silently) — `pose install` touches far more than what's
   actually broken, and doctor's job is diagnosis, not a re-install trigger.
2. **Free-form fix functions keyed by ad hoc logic scattered across the
   diagnostic pass.** Fast to write, but nothing then guarantees a fixable
   finding's action is actually confined and reversible — the exact
   "overconfident fixes can damage custom setups" risk the spec calls out.
3. **A small, explicit fix registry (`doctorFixRegistry`) mapping a check
   code to one confined, reversible, already-idempotent action, with
   classification (fixable/detectable/blocked) derived centrally from that
   registry, and every JSON field this spec adds appended — never
   replacing — the existing shape.**

## Decision

Option 3.

- **Classification is structural, not per-call-site:** `classifyFinding(check,
  level)` returns `n/a` for every `ok` finding, `fixable` when the check code
  is a key in `doctorFixRegistry`, `blocked` for the two external-toolchain
  checks (`deps.git`, `deps.go`), and `detectable` for everything else
  (schema drift, missing `.pose/`) — so every `add(check, level, message,
  hint)` call site in the diagnostic pass is untouched; only three checks
  needed to actually join the registry to become fixable.
- **The fix registry currently has exactly three confined entries**, each
  reusing an existing, already-tested code path rather than a new one:
  `hooks.pre-commit` calls `cmdHooks(root, []string{"install"}, ...)`
  directly; `mcp.config` calls the existing `configureMCP()` (used by
  `cmdInstall`, already preserves custom configuration and only migrates
  recognized legacy entries); `skills.symlinks` calls a newly-extracted
  `recreateClaudeSkillSymlinks()` (the exact block `cmdInstall` already ran,
  now shared instead of duplicated). None of the three touch instance
  content (specs, knowledge, `AGENTS.md`/`POSE.md`) — the non-goal is
  structurally unreachable, not just documented.
  `schema.version` was deliberately left out of the registry even though
  `pose upgrade` could resolve it: an upgrade is a broader, already-explicit
  operation with its own dry-run/idempotency contract
  (`pose-upgrade-compatibility-lab`) — collapsing it into doctor's confined
  fix set would blur two different consent boundaries for no real benefit.
- **`--fix` defaults to preview; `--yes` is required to mutate; `--only
  <code>` scopes to one check** — matching the constraint literally: bare
  `pose doctor --fix` (or `--fix --dry-run`) lists what would happen and
  changes nothing; only `--fix --yes` applies, then immediately reruns the
  full diagnostic pass and reports per-check before/after (recheck, R3).
  Every registered fix is naturally idempotent (reinstalling an already-
  correct hook/symlink/config is a no-op), so a second `--fix --yes` with
  nothing left to do reports "nothing fixable" rather than re-applying
  redundant work.
- **JSON schema stays additive:** `doctor_schema_version` (new top-level
  int, starts at 1), `evidence`/`remediation_class`/`fix_code` (new finding
  fields), and an optional top-level `fix` object (present only under
  `--fix`) are all pure additions — `check`/`level`/`message`/`hint` and the
  top-level `findings`/`errors`/`warnings` are byte-identical to the
  pre-spec shape, verified by the pre-existing `TestDoctorHealthyAndBrokenInstance`
  still passing unmodified.
- **Redaction is defense-in-depth, not a new detector:** `pose doctor` never
  echoes raw file content today, but `redactSecretShapedContent` (reusing
  `skills_check.go`'s `secretLikePatterns`) is applied to every
  message/evidence/hint/fix-error string regardless, so a future check that
  *does* start surfacing file content can't accidentally leak a credential
  through doctor's output.

## Consequences

- Positive: adding a fourth fixable check later is a one-line registry
  addition plus reusing (or writing) one confined, idempotent function —
  the classification and CLI plumbing need no changes.
- Positive: every fix action was already exercised by an existing test
  path before this spec (`cmdHooks`, `configureMCP`, the skill-symlink
  block) — `doctor --fix` composes proven building blocks instead of
  introducing new mutation logic.
- Negative: `schema.version` (arguably the most common real-world doctor
  finding) stays `detectable`, not `fixable` — an operator still runs
  `pose upgrade` by hand. Acceptable: upgrade is already a well-documented,
  low-friction explicit command, and the spec's constraint favors narrow
  confined fixes over broad convenience.
- Neutral: the `fix` JSON object's shape (`mode`, `candidates` in dry-run,
  `results` in apply) is new and unversioned beyond the top-level
  `doctor_schema_version` — acceptable since it did not exist before this
  spec, so there is no prior shape to preserve for it specifically.
