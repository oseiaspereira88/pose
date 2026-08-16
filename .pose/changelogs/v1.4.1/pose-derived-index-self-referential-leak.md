---
spec: pose-derived-index-self-referential-leak
category: fixed
breaking: false
refs:
---

The embedded scaffold no longer ships pose-dist's own computed `.pose/indexes/{repo-map,services,packages,spec-graph,roadmaps,delivery-integrity,releases,extensions.lock}.json` content to installed instances (the same leak class already fixed for `.pose/policy/{delivery,artifacts}.json` and `module-metadata.json`/`validation-matrix.json`). `pose install`/`pose update` now recompute these indexes for the target immediately after seeding any of them, so a target ends up with its own real state instead of a stale snapshot of pose-dist's own graph.
