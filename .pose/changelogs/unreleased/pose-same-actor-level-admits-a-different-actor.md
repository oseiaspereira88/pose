---
spec: pose-same-actor-level-admits-a-different-actor
category: fixed
breaking: false
refs:
---

Under verified identity, a review policy requiring `same-actor-separate-execution` no longer refuses a signed review by another principal in another execution; only the implementation's own run is refused, consistent with `different-actor` and `mandatory-human` above it.
