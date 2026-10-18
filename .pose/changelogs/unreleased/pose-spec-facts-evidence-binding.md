---
spec: pose-spec-facts-evidence-binding
category: changed
breaking: false
---

`pose specs facts` no longer presents every indexed validation result of a spec's module as
the spec's evidence. `checks` now lists the results the index ties to the spec by scope
provenance, with the basis and whether each ran against the head of the spec's change sets;
the rest of what the index holds appears under `candidates`, printed as `CANDIDATE, not evidence
of this spec` with the reason (a declared module, or an empty `components`, is not
provenance). A result that ran against another head is kept and labelled, not dropped.

The facts also say how fresh the index is: `fresh`, `stale` (an input changed since it was
published), `unknown` (never published as a set) or `unavailable`, with a limitation that points
to `pose index` when the answer cannot be established. Reading the facts rebuilds nothing and
concludes nothing.

Classified minor: the `checks` field of the JSON narrows to associated results (candidates are
a new field); a consumer that copied `checks` as proof of execution now gets only what the index
ties to the spec.
