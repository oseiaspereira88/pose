---
spec: pose-delivery-later-removal
category: added
breaking: false
---

A spec can remove a file that earlier specs declare `created` or `modified`. Their existence check is satisfied — and the delivery-integrity graph records an edge `removed-by` — when the later spec declares `removed:` for the path, its attributed change set contains the commit that deleted it, and the earlier spec's latest change to the path is a Git ancestor of that commit; `pose index`, `check --strict`, `artifact-check` and `surface-check` apply the same rule. A change made after the removal, or a removal its spec never performed, keeps the existence finding, which now says why. `pose-mcp/internal/cli/review_dispatch.go`, kept only because of this gap, is removed.
