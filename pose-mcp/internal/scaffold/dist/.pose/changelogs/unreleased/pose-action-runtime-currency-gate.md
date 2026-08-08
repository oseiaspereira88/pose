---
spec: pose-action-runtime-currency-gate
category: added
breaking: false
refs:
---

A deprecated action runtime now fails the build instead of annotating a green
run. Every pinned action's declared runtime is recorded in
`.github/action-runtimes.json`; an offline check fails on a deprecated runtime,
an unrecorded action or a record left behind by a bump, and a CI step re-resolves
each `action.yml` at its pinned ref so the record cannot quietly disagree with
reality. Building the record immediately found what it exists for:
`goreleaser/goreleaser-action` was still on Node.js 20, having been left out of
the earlier bump that covered only first-party actions. It is now on v7.2.3 and
the repository has no deprecated runtime left.
