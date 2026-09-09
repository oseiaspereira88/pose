---
spec: pose-evidence-scoped-to-component
category: fixed
breaking: true
refs:
---

A criterion or tool scoped to a component is rejected when it cites evidence
from outside it. Evidence was matched by reference and class alone, so a backend
criterion could be satisfied by a frontend sibling's integration result — real,
of a demanded class, and silent about the thing the criterion is about.

The sealed plan now records which components each profile was selected for.
Without it the bundle could not decide this even in principle: a criterion names
its profiles and nothing mapped a profile to what it matched. Reading that from
the current policy at verification time would judge an immutable bundle by
today's configuration, which is what sealing exists to prevent.

A criterion governed by any base profile still answers for every component and is
not narrowed. The bundle payload gains a field, so a bundle sealed by this
release does not digest the same as one sealed before it.
