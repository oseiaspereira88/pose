---
spec: pose-component-evidence-is-not-inherited-upward
category: fixed
breaking: true
refs:
---

A validation result from a directory inside a component no longer satisfies a
criterion or delivery target about the component. It covered one directory of it
and was accepted as covering all of it.

That is the same shape as the sibling case closed one spec earlier: real
evidence, of the class the criterion demands, silent about most of what it was
asked about. `moduleMatchesTarget` accepted a path prefix in either direction,
so refusing siblings left containment open in both readings.

The other direction stays, because it is a claim the result actually supports: a
module-wide run exercises its subtree, and running checks once at the module root
is how nearly every project is laid out. A result registered for `web` still
answers for a target at `web/api`. The repository root is untouched — a
single-module project is its own component.

A scope whose only evidence sits in a directory inside the component now has
none, and stops closing until a check is registered for the component or the
target is declared where the evidence already is.
