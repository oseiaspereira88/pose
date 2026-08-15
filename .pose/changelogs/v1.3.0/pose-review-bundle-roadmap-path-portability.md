---
spec: pose-review-bundle-roadmap-path-portability
category: fixed
breaking: false
refs: ISSUE#26
---

A sealed roadmap review bundle no longer bakes in the absolute checkout path it was sealed from — it previously read as stale ("changed") on any machine or CI runner other than the one that sealed it, permanently blocking `pose check --strict` there.
