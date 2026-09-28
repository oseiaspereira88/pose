---
spec: check-builds-the-delivery-graph-once
category: fixed
breaking: false
---

`pose check` builds the delivery graph once instead of once per completed spec. It
rebuilt the whole graph for every done spec, which on this repository was 117 builds
at about five seconds each: 600 of the 635 seconds the gate took, growing with every
spec closed. `check --strict` now completes in 39 seconds with byte-identical
findings. The reuse is possible because `focusSurfaceGraph` no longer filters through
the caller's backing array, which had been truncating the graph it was handed.
