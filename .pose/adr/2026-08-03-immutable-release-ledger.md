---
id: ADR-018
title: Immutable release snapshots and evidence-backed lifecycle
status: accepted
date: 2026-08-03
supersedes:
superseded_by:
---

# ADR-018: Immutable release snapshots and evidence-backed lifecycle

## Context

Tags and provider releases could exist while changelog fragments remained
forever unreleased. A tag-triggered workflow may also fail after the tag exists,
so tag presence cannot prove publication. Rendering notes from the live pending
queue made published content mutable and erased the boundary between candidate
work and released work.

## Decision

POSE freezes selected fragments, canonical notes and declared version evidence
in an immutable version manifest before tagging. Lifecycle facts are replayed
from append-only, provider-neutral evidence events. `tagged`, `published` and
`verified` remain distinct; verification binds the exact publication evidence
and asset digests. Provider workflows retain minimized evidence for reviewed
import to the default branch and never directly strengthen local state.

Preparation is dry-run by default and moves only explicitly selected fragments.
Git mutation remains outside the read-only core and may neither overwrite tags
nor force-push. Historical backfill reports confidence and gaps without
fabricating manifests, notes or provider proof.

## Consequences

- Tagged CI can publish only a committed prepared snapshot.
- Local status honestly remains `tagged` when provider evidence is absent.
- The next pending queue contains only work created after the cut.
- Publication evidence can arrive asynchronously without mutating the tag.
- Existing tags through the policy cutoff remain migration findings, not
  invented releases.

## Alternatives rejected

- Treating tag presence as release completion cannot represent failed CI.
- Live provider queries are nondeterministic and unavailable offline.
- Consolidated notes alone lose per-spec provenance; fragment directories alone
  are inconvenient publication inputs.
