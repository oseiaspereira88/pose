---
spec: pose-report-a-demanded-class-nothing-produces
category: added
breaking: false
refs:
---

`pose doctor` reports an evidence class the selected review profiles demand that
no registered check in the repository produces.

`review.evidence-vocabulary` asks whether a class *could* be emitted by some
check somewhere — it holds a profile to the closed vocabulary. It says nothing
about whether this repository generates that evidence, and the two came apart
when `go vet` moved from `build` to `lint`: `build` stayed a valid class, the
profiles kept demanding it, and nothing produced it any more. A criterion
depending on evidence the repository does not generate, with nothing saying so.

It is a warning rather than a refusal, because evidence can be imported from
elsewhere. A module override counts as a producer: the question is whether the
repository generates the class, not which stack does.

Run against POSE's own repository it has two findings on the first execution:
the selected profiles demand `a11y` and `e2e`, and no registered check emits
either.
