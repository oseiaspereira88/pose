---
spec: pose-design-delta-batched-git-reads
category: changed
breaking: false
---

Structural assessment reuses one bounded Git batch reader for blob reads within
an assessment, reducing process creation while retaining path, symlink and byte
limits. The process is closed and reaped at the end of the operation.
