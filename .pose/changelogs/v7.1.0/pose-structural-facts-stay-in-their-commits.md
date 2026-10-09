---
spec: pose-structural-facts-stay-in-their-commits
category: changed
breaking: false
refs:
---

Structural review facts now describe what the spec's own commits changed. A subject path used to be compared between the first and the last attributed commit, so changes other specs or releases made to the same file in between were reported, and charged, as this spec's. Newly sealed bundles record the content each spec's commits produced and compare that; a fact that comes from a commit shared with other specs names them and asks the reviewer to confirm it. Bundles sealed earlier keep their reading.
