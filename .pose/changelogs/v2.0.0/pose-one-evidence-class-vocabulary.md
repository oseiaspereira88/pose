---
spec: pose-one-evidence-class-vocabulary
category: changed
breaking: true
refs:
---

A review profile may now declare only the evidence classes a registered check
may emit. Two lists governed this and disagreed in both directions: ten names a
profile could demand were impossible to satisfy — `contract`, `integration-test`,
`lint`, `manual-review`, `observability`, `requirement-trace`, `security-scan`,
`test`, `typecheck`, `validation` — and three a check could emit could never be
demanded: `contrast`, `design-system`, `visual-regression`. They agreed on six
of nineteen.

A profile demanding an unproducible class plans a gate only a fabricated
disposition can pass, which is the failure the four specs before it in
this release closed one consumer at a time. Refusing the profile is where it stops being expressible,
and it makes those downstream guards unnecessary: the criterion blocker, the
tool class drop and the filter behind both are removed.

**A profile declaring one of the ten dropped names now fails to load**, which
fails review planning rather than degrading it. `pose doctor`'s
`review.evidence-vocabulary` has reported exactly this since 1.8.0. The profiles
shipped with POSE were reconciled to the intersection earlier in this same
release and load unchanged.
