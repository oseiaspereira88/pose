# ADR: Retired machinery files stay on disk, never auto-migrated by pose update

## Status
Accepted (2026-08-15) — spec `pose-domain-rule-extension-migration`,
`github.com/oseiaspereira88/pose#24`

## Context

`pose-domain-rule-extension-migration` removes `backend-go.md` and
`frontend-react.md` from core machinery (`.pose/rules/`), delivering them
exclusively as extensions from then on (the pattern
`pose-rule-kubernetes` already proved). Every already-installed instance
that received either file through an earlier `pose install`/`pose update`
has it on disk today. Removing a file from the engine's shipped set changes
what a subsequent `pose update` does for those instances, and this spec's
own Decision 1 flagged the question as unresolved: silently dropping
content an agent may depend on is a regression, not a cleanup, so the
compatibility strategy needed its own scrutiny rather than an inline
default.

Two candidate strategies were on the table:

1. **Auto-migrate**: `pose update` detects that an instance already has a
   file being retired and actively installs the matching extension in its
   place, preserving effective content across the transition.
2. **Leave in place, stop re-seeding**: already-delivered files stay
   exactly as they are; the engine simply stops re-copying them. Adoption
   of the extension is a separate, user-initiated step.

Reading `deliverMachinery` (`pose-mcp/internal/cli/machinery.go`) before
deciding: it walks `machineryRoots` from the *current* engine source only
— a file no longer present in that source is never visited by the walk,
and nothing in `deliverMachinery` ever deletes or overwrites a file that
drops out of the shipped set (the existing "deletion record" logic only
concerns files the *instance itself* removed, so they are not
resurrected — a different case). Strategy 2 is therefore not a design
choice requiring new code: it is `deliverMachinery`'s existing behavior,
unconditionally, the moment `backend-go.md`/`frontend-react.md` leave
`.pose/rules/`. Strategy 1 would require new logic with real failure
modes of its own — deciding when a target-modified copy of the file
should still be "migrated" over, and doing so from inside `pose update`,
a command whose existing contract is refresh machinery, not install
extensions (a separate, explicitly user-triggered, signed/transactional
mechanism by design).

## Decision

- **Strategy 2, plus an explicit discoverability signal.** Already-
  delivered `backend-go.md`/`frontend-react.md` are left untouched on any
  instance that has them; `pose update` simply stops re-seeding them,
  which is `deliverMachinery`'s default behavior for any file removed
  from the shipped source — no new deletion or migration logic is
  written.
- **`pose doctor` gains an advisory check** (implemented as part of
  `pose-domain-rule-extension-migration`, not this ADR) that compares an
  instance's `machinery-manifest.json` history against the current
  machinery root set: a path the manifest recorded as previously
  delivered, still present on disk, but no longer in the current
  machinery walk, surfaces as "no longer delivered by machinery — install
  `pose-rule-<name>` to keep receiving updates." Advisory only, never
  blocking, matching every other `pose doctor` check's contract.
- **No auto-install of the extension from `pose update`.** Extension
  installation stays exclusively user-triggered
  (`pose extension install`), preserving the existing "data-only,
  transactional, explicit" contract that mechanism was built on.

### Rejected trade-off

Auto-migration (strategy 1) was rejected: it would blur `pose update`'s
contract (machinery refresh) with `pose extension install`'s (signed,
transactional, explicit package install), for a benefit — automatically
preserving content that is not actually going stale, just no longer
re-delivered — that the advisory check already delivers with far less
risk and no new failure surface.

## Consequences

- Positive: zero new deletion/migration code path; the compatibility
  guarantee ("nothing an agent depends on disappears without warning")
  is met by discoverability, not by silent mutation.
- Positive: keeps `pose update` and `pose extension install` cleanly
  separated — a target repository's rule content only ever changes
  through an action a human or agent explicitly took.
- Trade-off: an instance that never runs `pose doctor` will not learn its
  `backend-go.md`/`frontend-react.md` is now stale-but-present content
  until it does. Accepted: the same is already true of every other
  `pose doctor` advisory today, and the file itself does not become
  wrong or harmful by staying in place — it simply stops receiving
  updates.
- Residual: `pose-domain-rule-extension-migration`'s implementation must
  actually build the doctor check described above — this ADR fixes the
  strategy, not the code.
