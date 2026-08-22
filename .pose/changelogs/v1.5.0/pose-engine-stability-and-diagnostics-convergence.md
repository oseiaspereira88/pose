---
spec: pose-engine-stability-and-diagnostics-convergence
category: fixed
breaking: false
---

Hardened engine stability by adding Git commit trailer recognition to `pose doctor`, eliminating spurious `resolvability` errors in `pose artifact-check` for `none`-action claims, normalizing `.gitignore` trailing slash lookups across module discovery walkers, and replacing test self-execution with stable binaries.
