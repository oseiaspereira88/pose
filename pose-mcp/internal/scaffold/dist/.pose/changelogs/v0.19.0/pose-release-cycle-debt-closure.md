---
spec: pose-release-cycle-debt-closure
category: fixed
breaking: false
refs:
---

`pose release prepare` no longer breaks the structural gate of the specs it
consumes: a spec that declared its changelog fragment as an artifact now has
that claim repointed to the archived path as part of the same transaction, so
`pose check --strict` stays green after a cut instead of needing a manual repair
pass.

A follow-up whose `(owner:… crit:… review:…)` group wrapped onto its own line
was parsed as if the group were absent, leaving the item silently unowned — and
its text truncated at the first line. Wrapped follow-ups are now read whole.

The installer's provider-download path — the one every `curl | bash` user takes
— is covered by tests for the first time, including rejection of a truncated
archive.
