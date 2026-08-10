---
spec: pose-container-build-gate
category: added
breaking: false
refs:
---

CI now builds both delivery container images and starts them. They are composed
in production by the harne8 monorepo from this repository, and nothing here built
either one — so a broken Dockerfile surfaced when that stack came up, in a
different repository. The gate uses the same build contexts the production
compose file uses, which are not interchangeable, and requires each entrypoint to
announce itself: an image that builds and cannot start is the failure that
reaches a compose stack as a crash-loop.
