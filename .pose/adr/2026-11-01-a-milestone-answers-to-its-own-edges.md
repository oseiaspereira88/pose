# ADR: A milestone answers to its own edges

## Status
Accepted (2026-11-01), spec `pose-milestone-closes-when-reached`.

## Context

The federated acceptance of a milestone was built from the whole roadmap with the other milestones removed, so the `depends_on` of the roadmap stayed. Every milestone of `delegation` was blocked until `delegated-review` closed, although their order was already declared by `after:`. The same dependency binds differently once a roadmap has milestones: their order is what makes it possible to close part of a roadmap at all.

## Decision

- `depends_on` of a roadmap binds the close of that roadmap. It is not an edge of its milestones.
- A milestone answers to its `after:` edges, its member specs and its own `consumes`.
- A milestone that needs the prerequisite of its roadmap declares it in `after:`. That form already works for a milestone or a spec, and for a roadmap as `roadmap:<slug>`.
- A bundle sealed under the wider rule keeps its approval while the `depends_on` of the roadmap still names the missing edge.

## Consequences

- A milestone can close while the prerequisite of its roadmap is open. Its close states that milestone, and says nothing of the roadmap.
- `roadmap:<slug>` still waits for every prerequisite, so the outcome of a roadmap is not weakened.
- New manifests of milestones are narrower than old ones, and the comparison of retained approvals carries one more case.

## Alternatives

- Keeping the rule and documenting it: the order declared in `after:` would keep meaning nothing until the prerequisite closed.
- A reseal of every affected milestone: eleven reviews of unchanged work.
- Marking the edge in the manifest as roadmap-level: a schema change to preserve an edge that no longer binds.
