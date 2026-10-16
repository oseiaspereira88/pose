---
title: Writers publish through one exclusive, compare-and-swap publisher
status: accepted
date: 2026-10-16
decision_type: architecture
---

# Writers publish through one exclusive, compare-and-swap publisher

## Context

Spec `pose-writer-coordination-and-atomic-publication` (B14) reproduced on the
engine at `1bb5111e` how writers lose or misreport effects (front 9 CONS-04,
CONS-06, CONS-09; probes in `.pose/results/b14-front09-probes.json`):

- `writeAtomic`, the helper behind indexes, state, reports, installs and
  managed manuals, staged every file at a fixed `.<name>.tmp`. Two writers of
  the same file shared it: one could publish the other's bytes and the second
  fail its rename. Another writer's staging at that path stopped `pose index`
  outright (P01). The start record, the action policy and adoption policy
  files used fixed `<path>.tmp` names of their own.
- Mutable records were read, changed and written back with no check that
  they were still what was read: the state refresh overwrote a curated
  section edited while it rendered, the action policy overwrote a concurrent
  edit, and `pose close` wrote `status: done` over whatever the spec held
  when it wrote — not the bytes its gates judged.
- Start used a per-spec lock, transfer a per-root lock, and close none: a
  close and a start, or a transfer and a close, of the same spec could
  interleave.
- `pose state refresh` published the state, then failed to append its
  history and reported a plain failure (P02); the next append after a torn
  history line was glued to it (P03).

## Options considered

1. **A transaction across files and roots.** Not available on the
   filesystems POSE runs on without a daemon; promises atomicity it cannot
   keep. Index generations are B15; release recovery is B41.
2. **Fix each writer locally.** Leaves the next writer to repeat the fixed
   temporary file and the missing precondition.
3. **One publisher every writer uses, with an explicit precondition and
   artifact class, and the spec's existing start lock shared by every
   lifecycle writer.**

Option 3 is chosen.

## Decision

**Publisher** (`internal/pose/writer_publication.go`, `PublishFile`). Each
call stages its bytes in a temporary file only it creates, next to the target
(`.<name>.*.tmp`), and on any failure removes only that file. It publishes:

- with `ExpectAny` by rename — last writer wins, for derived artifacts their
  writer recomputes from inputs;
- with `ExpectDigest` (the SHA-256 of the bytes the writer read) by rename
  only if the file still holds those bytes — the read-modify-write of a
  mutable record; otherwise a `*PublishConflictError` (`ErrPublishConflict`)
  and nothing is published;
- with `ExpectAbsent` by hard link, which fails if the file exists —
  immutable records; a retry that finds the same bytes converges, other bytes
  conflict.

Artifact classes set durability: `authoritative` (specs, start records,
policies, the project state, whose curated sections the engine cannot
recompute) syncs the staged file and then the directory (except on Windows,
whose rename is journaled); `derived` (indexes, reports, manuals) is
published whole but not synced, so a crash may lose the last publication and
never presents a torn one. A directory sync that fails after the rename is a
`*PublishDurabilityError` (`ErrPublishUnsynced`): the file is published and
its durability unconfirmed, never a failure without effect. Where hard links
are unavailable, an immutable record is created exclusively and written in
place instead (the readers of immutable records refuse a write cut short),
so start keeps working on such filesystems. A staging file a crash leaves
(`.<name>.<digits>.tmp`) is git-ignored in this repository and is not a
change for the closeout context check. The precondition compares bytes, so it detects a
writer that ignores POSE's locks, but between the check and the rename a
non-cooperating writer can still win: writers that coordinate hold the
resource's lock around read, compute and publish.

Writers moved to it: `writeAtomic` (`derived`, `ExpectAny`; every caller loses
the shared temporary file), the start record (`authoritative`; the baseline
`ExpectAbsent`, so a write cut short publishes nothing, and the update under
the lifecycle lock), the action policy (`authoritative`, `ExpectDigest` of the
document it loaded), the adoption policy files (`authoritative`), the state
(`derived`, `ExpectDigest` of the state it read, so a curated edit made while
it renders is a conflict, never overwritten) and the close transition
(`authoritative`, `ExpectDigest` of the bytes the gates judged). The other
writers of the state — the pending-refresh marker and the architecture
section `assess discover --update-state` writes — publish over the state they
read too; the marker, which runs after a failed refresh, falls back to an
in-place write only when the directory cannot take a staging file, after
checking the file still holds the bytes read. The artifact backfill rewrites
specs under their lifecycle lock and over the bytes it read. The start
record update keeps `ExpectAny`, because it runs under the lifecycle lock
that coordinates every writer of that record.

**Coordination.** The per-spec start lock of B13 (`.pose/starts/.<slug>.lock`,
owner recorded, never broken by age) is the spec's lifecycle lock, taken by
`LockSpecLifecycle` for start, cancel, the close transition and a transfer —
in the source and the destination project, after the root locks, in
canonical order (root path, then slug) — and by the artifact backfill. It
never waits, so a held lock is a refusal with its owner's state: a running
start is `start-in-progress`, any other running writer
`spec-lifecycle-in-progress` with its operation, and a transfer refused by
either is `concurrent-lifecycle-writer`; lifecycle writers cannot deadlock.
Roots and specs not involved are not locked.

**Close.** `cmdCloseLocal` records the digest of the spec or roadmap its gates
judge, takes the lifecycle lock for a spec, judges the authorization again
inside the protected region — a review that stopped being approved, or an
action request that now restricts closeout, refuses the transition — and
publishes it only over those bytes: an edit made while the gates ran is a
`publish-conflict`, the edit stays, and `pose close` again judges the current
bytes.

**State refresh.** A refresh that published the state and could not append
its history returns a partial result (`state.published=`, an error naming
both effects, refresh log `result: partial`) instead of a plain failure; the
next refresh appends its own entry and none is made up for the failed one.
A hook-triggered refresh in best-effort mode does the same without marking
`refresh_pending` (the state is current); `pose state` reports the partial
refresh as `history-not-appended`, and the event is not deduplicated, so its
next run appends the entry. An
append after a torn last line starts on a new line, so the torn bytes stay
as they were — an unreadable line the reader skips — and the new entry reads.

## Consequences

- No writer shares staging; mutable records are published over what was read
  or refused; immutable ones are never overwritten; lifecycle writers of a
  spec exclude each other.
- New refusals: `publish-conflict` from close, state refresh and the action
  policy; `concurrent-lifecycle-writer` from transfer; a close while a start
  or transfer holds the spec. All are additions, classified minor.
- Tests that injected a failure by planting a directory on a fixed temporary
  path use `PublishFault`, a hook only tests set, at the publication point.
- Writers not moved, declared as limits: the validation matrix seed
  (`stack_seed.go`, `wizard.go`), the managed manual merge
  (`managed_docs.go`), `project.json`, the capability assessment files and the
  extension lock stage through `writeAtomic` (no shared temporary file) but
  without a precondition: they run at install, update or assessment time, by
  one operator, and their content is recomputed from inputs. Moving them to
  preconditions is left to the read-view and runtime units (B34, B35).
- Limits: no atomicity across files or roots (index generations are B15);
  non-cooperating writers can still race the check-then-rename window; power
  loss is not qualified beyond `fsync` of file and directory; without hard
  links an immutable record is written in place and can be cut short; MCP has no lifecycle writer route (start,
  close, transfer and state refresh are CLI), so its parity is in the readers
  of these publications.

## Falsifier

This decision is wrong if two writers can publish each other's staged bytes,
if a mutable record is published over bytes its writer did not read, if an
immutable one is overwritten, if a close publishes over bytes its gates did
not judge, if two lifecycle writers of one spec interleave, or if a refresh
that published the state reports that nothing happened. The regressions in
`internal/pose/writer_publication_test.go` and
`internal/cli/writer_publication_test.go` assert each of these.
