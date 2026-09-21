---
spec: check-worker-count-is-the-machines
category: added
breaking: false
---

`POSE_CHECK_WORKERS` sets the worker pool `pose check` uses for its per-spec gate work.
The default stays the core count: eight workers looked faster than sixteen over two
runs and did not over five, so nothing was changed on a difference that noise explains.
An unusable value is ignored rather than failing the gate, and the pool is still clamped
by the number of items.
