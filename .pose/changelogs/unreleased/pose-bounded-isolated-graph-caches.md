---
spec: pose-bounded-isolated-graph-caches
category: changed
breaking: false
---

The caches of rebuildable data are bounded in bytes and isolated. The parsed
delivery graph, the design-delta report, the test catalog and the paths a commit
changed share one cache that evicts by retained bytes under a global budget and a
budget per worktree, produces a missed key once however many callers ask, lets a
waiter leave without disturbing the others, and never remembers a failure. A
returned graph shares no map or slice with the cache: the attribute maps of the
nodes and the refs of the roadmap criteria were shared before. Read-only
consumers can view the graph without a copy. The test catalog is keyed by the
validation matrix and expires after 30 seconds.
