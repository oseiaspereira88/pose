---
spec: pose-phase-scoped-readiness
category: added
breaking: false
refs:
---

Readiness can now be read per phase: `pose_spec_readiness` with `phases: true` and `pose state --attention --scope spec:<slug>` say whether start, execution, review, closeout and release are clear, restricted, partially restricted on named nodes, or unknown because a producer that could restrict them was not read. A closeout-only request no longer reads as a stop to implementation, a restriction on one requirement is reported as partial without claiming the rest is independent, and the legacy `ready` keeps its meaning.
