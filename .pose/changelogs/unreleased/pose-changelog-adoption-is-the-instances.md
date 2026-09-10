---
spec: pose-changelog-adoption-is-the-instances
category: fixed
breaking: false
refs:
---

The shipped changelog policy carries an empty `adopted_at`, and `pose install`
and `pose update` stamp the day the instance received it.

`.pose/policy/changelog.json` was synced byte-for-byte into the scaffold, so
every install inherited POSE's own adoption date. For a project starting today
that reads as "gate everything", harmless by accident. For a project migrating in
with a history of specs it is not: every spec completed before that date is
silently exempt from needing a changelog fragment, by a date belonging to someone
else — the gate looks adopted and covers less than the project thinks.

Same family as the self-referential delivery roots of issue #17, in a file that
fix did not cover.

Shipping an empty date alone would have broken every fresh install, because
`pose check` refused one with `adopted_at is required`. An empty date now means
the contract is not adopted rather than that the policy is malformed, and the
stamp means a fresh install does not stay in that state. An instance that already
carries a date keeps it.
