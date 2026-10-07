---
spec: pose-recurrence-check-resolved-clusters
category: fixed
breaking: false
refs:
---

`pose recurrence-check` no longer counts a failure that a later `pass` of the same `stable_hash` settled: it groups the window's history by task, report type and `stable_hash`, flags a task only for failures after a cluster's latest `pass` (or with no `pass` in the window), and never lets a pass of another `stable_hash` settle a failure. Settled clusters stay visible as `resolved` findings and in `recurrence.resolved_clusters`; `--include-pass` keeps counting every record and the exit code is unchanged.
