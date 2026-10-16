---
spec: pose-writer-coordination-and-atomic-publication
category: changed
breaking: false
---

Writers no longer share a staging file. Indexes, state, reports, manuals and
installs were staged at a fixed `.<name>.tmp`, so two writers could publish
each other's bytes, and another writer's staging there stopped `pose index`;
every publication now stages in a file only it uses and removes only that
file on failure. The start record, the action policy and adoption policy
files left their fixed `<path>.tmp` too.

Mutable records are published only over the bytes their writer read:
`pose state refresh` refuses with `publish-conflict` instead of overwriting a
curated section edited while it rendered, the action policy refuses instead
of overwriting a concurrent edit, and `pose close` publishes the transition
only over the bytes its gates judged. Immutable records (the start baseline)
are never overwritten, and a baseline write cut short publishes nothing.

Start, close and transfer of a spec share its lifecycle lock: a close while a
start or a transfer holds the spec is refused with the lock's owner, and a
transfer of a spec another lifecycle call holds is refused as
`concurrent-lifecycle-writer`.

A `pose state refresh` that published the state and could not append its
history now says so (`state.published=`, refresh log `result: partial`)
instead of reporting a plain failure, and a history entry is never glued to a
torn last line.
