---
title: Index sets are published as verified generations
status: accepted
date: 2026-10-17
decision_type: architecture
---

# Index sets are published as verified generations

## Context

Spec `pose-index-generation-integrity` (B15, front 9 CONS-05). `pose index`
computes seven derived files — `repo-map.json`, `services.json`, `packages.json`,
`spec-graph.json`, `roadmaps.json`, `delivery-integrity.json`, `releases.json` —
and, until now, wrote them one after the other through `map` iteration order. Each
file was published whole (B14), but the seven are seven names:

- a write that failed in the middle (an output that could not be staged, a
  crash, `ENOSPC`) left the files before it new and the files after it old, and
  every reader accepted the mixture: the component map of one run beside the
  delivery graph of another;
- two direct writers of `delivery-integrity.json` (the refresh at the end of
  `pose artifact-check`, and the bootstrap of `pose review bundle`) changed that
  one file alone, so no file could tell which others it belonged with;
- a reader that read two of the files could read them from two moments.

An atomic file is not an atomic set. There is no transaction across files on
the filesystems POSE runs on, and a daemon is out of scope (B34/B35).

## Options considered

1. **Directory swap.** Write the set to a new directory and rename it over
   `.pose/indexes`. Not atomic over a non-empty directory on Linux, breaks the
   curated files that live beside the computed ones (`module-metadata.json`,
   `validation-matrix.json`, `task-map.json`, `extensions.lock.json`), and every
   reader's path.
2. **A manifest only.** Keep the seven names and add a file that lists their
   digests. A reader can detect a mixture but, during a publication, every name
   may already be new while the manifest is old, so a reader that wanted the old
   set has nowhere to read it from.
3. **Immutable generations plus a descriptor, with the legacy names kept.**
   Write each generation under its own directory, publish the legacy names from
   the same bytes, and commit by publishing the descriptor last. Readers fix the
   generation the descriptor names and read its immutable copy.

Option 3 is chosen.

## Decision

**Generation.** The seven files above, and nothing else, are a generation. Its
descriptor, `.pose/indexes/generation.json`, holds `schema_version`, a
`generation_id`, each file's name, SHA-256 and size, and `inputs`: the digests of
the governed inputs the set was computed from (specs, roadmaps, policy, module
metadata, validation matrix). The id is `gen-` plus 16 hex digits of the digest of
files and inputs, so the same inputs name the same generation and a retry converges
on it. The descriptor carries no time. Git history is read by the delivery graph and
is not pinned: a commit would change the id on every commit; limits are declared.

**Publication** (`Store.PublishIndexGeneration`). It holds the **generation lock**
(`.pose/indexes/.generation.lock`, owner recorded as in B13) from the read of the
current descriptor to the end of retention: two publishers never accept the same
predecessor, and a retention step never runs against a generation another publisher
has since made live. The lock never waits and is never broken by age: a lock another
call holds is `ErrIndexGenerationBusy` with the owner's state and nothing is written;
only a lock whose owner is proven dead (same host, process gone) is removed, under an
exclusive lock of the indexes directory, because the publication it guarded is
idempotent and its outputs are derived. The commit remains a compare-and-swap over
the bytes read, so a writer that ignores the lock is still refused. The phases, each
file through the B14 publisher:

0. when there is no descriptor yet, a **pending marker** (`.generation-pending`) is
   written before any legacy name changes and removed after the commit: a first
   publication that stopped halfway leaves legacy files that may mix two runs, and
   readers report that repository as `unavailable`, not `legacy-unversioned`;
1. the files are written under `.pose/indexes/generations/<id>/`, each with
   `ExpectAbsent` (immutable; the same bytes converge) and read back against the
   descriptor;
2. the legacy names `.pose/indexes/<name>` are published from the same bytes
   (`ExpectAny`, derived);
3. the descriptor is published over the one the writer read (`ExpectDigest`,
   authoritative: synced). This is the commit.

A failure before step 3 leaves the previous descriptor, and so the previous
generation, in force; a failure of step 3 whose cause is the unconfirmed durability of
the directory (`ErrPublishUnsynced`) is a **committed** generation: the receipt says
so (`Committed`) and the CLI reports the new generation as current, not the old one; the receipt says which phase failed and `pose index` says the
previous generation stays. A failure of retention after step 3 is a warning in the
receipt and never a failed publication. The same set published twice writes
nothing (`unchanged`).

**Reading** (`Store.ReadIndexGeneration`, `Store.ReadIndex`). A reader reads the
descriptor, validates it (schema, file set and order, SHA-256 shape, id equal to
the digest of its content, no unknown field), then reads each file from the
generation's immutable copy and, when that is absent or does not match, from the
legacy name, and checks it against the descriptor. It reads the descriptor again at
the end and starts over, at most three times, if it changed. The result is one of:

- `current`: whole and identified; the bytes are the ones read, which is what
  "fixing the generation" means for the operation;
- `legacy-unversioned`: there is no descriptor (a repository that never ran
  `pose index` since this contract); the files are returned as before and nothing
  claims they belong together;
- `unavailable`: a descriptor exists and the set does not verify, or it cannot be
  read; the reason names the file or the field. It is an error wrapping
  `ErrIndexGenerationUnavailable`, never an empty or a mixed result.

`StaleInputs` compares the bindings with the inputs as they are now; a changed
input makes the set stale, not unavailable.

**One generation per operation.** `Store.PinIndexes` reads the set once and returns a
store whose index reads all come from it. The review plan, the review bundle (its plan and
its evidence), the closeout check and the start obligations pin at their entry
(a pin already taken is reused, never retaken), so what they combine comes from the
same generation even if a publication commits meanwhile; an operation that reads one
index needs no pin. A pin taken while the repository has no descriptor keeps no bytes:
if a first publication starts (pending marker or descriptor) it reads as
`unavailable`, and an unpinned legacy read is checked again after the file is read.

**Replacement.** `Store.ReplaceIndex` reads the current generation under the same lock
and commits the new one over the exact bytes of that descriptor: the publication is
refused before anything is written, and the commit is a compare-and-swap over them, if
a writer that ignores the lock published in between, so a generation is never silently
reverted. `Store.ReplaceIndexOrLegacy` takes the lock for callers that, before the first
generation, write the file themselves (the delivery refresh): the decision that the
repository is legacy and the write happen under the same lock.

**Consumers.** `GetDeliveryIntegrity` (and with it spec facts, the MCP
`pose_delivery_integrity` tool, the obligation projection and the closeout), the
review plan's component map and observed paths, the obligation snapshot's index
digest and `pose suggest` read through `ReadIndex`. An unavailable set stops a
reader that needs it (`GetDeliveryIntegrity`, the review plan's component map) and
is recorded as a warning or a limitation where the consumer can go on without it
(observed paths in a review plan, the snapshot's index digest becomes
`unavailable`). The two direct writers use `Store.ReplaceIndex`, which publishes a
new generation that differs in one file; before the first generation exists it
returns `ErrIndexGenerationLegacy` and they write the file as they always did. A
set that does not verify is not built upon, and the refresh says so on stderr.

**CLI.** `pose index --verify` reads without writing and exits 1 unless the set is
`current`, the legacy names under `.pose/indexes` are those of the generation
(`Diverged`: a file edited by hand, or a publication that has not committed) and its
inputs hold; each other state is named.

**Retention.** The generation the descriptor names when retention runs, and the one the
call published, are live; the newest others, up to three, stay;
nothing younger than ten minutes is removed, which protects a reader that is still
copying a generation it fixed; only directories named `gen-<16 hex>` directly under
`generations/` that are not links are removed, and a `generations/` that is itself a
link is refused for publication and retention alike. `generations/` is git-ignored; the
descriptor is committed with the indexes.

**Distribution.** `generation.json` and `generations/` are never synced by `pose
update` and never seeded: a descriptor next to files it does not describe would
turn a valid set into an unavailable one.

## Consequences

- No reader can take a mixture for a set. A mixture is `unavailable`, with a reason,
  for the CLI, the MCP and every internal consumer, because they read through one
  function.
- **Compatibility, classified minor.** A repository with no descriptor behaves as
  before. After the first `pose index`, engine readers read the generation, so a
  generated index edited by hand has no effect while the generation's copy exists
  (and makes the set `unavailable` in a clone without the copy); `pose index
  --verify` reports the file. Tests and fixtures that hand-wrote `repo-map.json`
  after an install now replace it through `ReplaceIndex`. `pose index --verify` and
  the files `generation.json` and `generations/` are additions. The copy is
  preferred over the legacy name because the legacy names are rewritten before the
  commit: a reader that preferred them would see a publication in progress.
- The legacy names are written before the commit. A tool that reads them directly,
  without the descriptor, can still see a mixture while a publication is running;
  that is the declared limit of legacy readers, and no engine reader is one.
- A clone has the descriptor but not `generations/` (git-ignored): it verifies the
  legacy names against the descriptor, which is the same check.
- Limits: no atomicity across files without a daemon; Git history is not pinned by
  the descriptor; retention is by count and age, not by tracking readers across
  processes; a reader holds the bytes it read, so a pruned generation never changes
  what an operation already fixed; power loss is not qualified beyond B14's `fsync`
  of the descriptor. B34/B35 may build a read view on this contract.

## Falsifier

This decision is wrong if a reader can return files of two generations as one set,
if a failed or interrupted publication changes which generation is current, if a
retry produces a second identity for the same inputs, if a set that does not verify
is read as empty or complete, or if retention removes the current generation or a
directory it did not create. `internal/pose/index_generation_test.go` asserts each of
these, including a stop at every phase and readers racing a publisher.
