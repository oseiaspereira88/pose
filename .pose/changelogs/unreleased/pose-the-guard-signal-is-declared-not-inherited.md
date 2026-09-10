---
spec: pose-the-guard-signal-is-declared-not-inherited
category: changed
breaking: false
refs:
---

The cross-version compatibility guard reads `POSE_RELEASE_HISTORY_AVAILABLE`, a
variable this repository's workflows declare, instead of inferring from the
provider's environment.

It had been wrong twice in opposite directions, both times for reading a signal
someone else owns. Keying on `CI` broke every repository that vendors the engine
and runs its suite. Keying on `GITHUB_REPOSITORY` fixed that and left the other
half open: the variable is the provider's to set, so on a provider that does not
set it this repository's own CI would skip in silence — the hole the guard exists
to close.

A declared variable is set only where a checkout was prepared with the release
history, cannot be inherited by a consumer, and can be asserted statically. The
workflow contract now requires every job that runs the Go suite to declare it,
alongside the full-history checkout it promises.

That assertion found a third such job on its first run — the release workflow's
own `Tests + installer E2E`, the run whose verdict cuts the release, which both
earlier fixes had enumerated by hand and missed.
