---
spec: pose-class-producers-reads-the-disjunction
category: fixed
breaking: false
refs:
---

`validate.class-producers` reports a review criterion none of whose accepted
evidence classes is produced, instead of reporting each class on its own — and
skips a profile whose language selector names nothing the repository has.

Both were false alarms, and both surfaced only by acting on the check's own
output. `evidence_classes` is a disjunction: the attestation cites one of them
and the first match satisfies the criterion. Reporting classes individually named
`e2e` while every criterion listing it also accepted `integration` or `unit` —
a true statement naming nothing to fix, which is how a check earns being ignored.
And `a11y` was named for a profile selected by `languages: [javascript,
typescript]` in a repository that has none, so the criterion could never apply.

Neither correction weakens it. The case it exists for still fires: a criterion
whose only accepted class lost its producer.

Only the language selector is evaluated, and only to skip — a profile selected by
component id or delivery kind is unaffected.
