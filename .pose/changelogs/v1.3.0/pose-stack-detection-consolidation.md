---
spec: pose-stack-detection-consolidation
category: added
breaking: false
refs: ISSUE#21
---

`pose install`/`pose init` now discovers the target repository's real modules (Node, Go, Rust, Java, Python, .NET) and seeds `.pose/indexes/module-metadata.json` with them — additive only, never overwriting a hand-authored entry. A brownfield repository no longer keeps a permanently empty or static module-metadata file.
