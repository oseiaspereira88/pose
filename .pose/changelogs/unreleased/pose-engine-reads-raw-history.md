---
spec: pose-engine-reads-raw-history
category: security
breaking: false
---
A replace ref or a graft can no longer redirect what the engine reads: `pose release check` no longer accepts a tag with no snapshot in its tree.
