---
spec: pose-review-bundle-root-file-classification
category: fixed
breaking: false
refs:
---

`pose review bundle <scope> --seal` no longer fails closed with `unclassified review subject path` when the attributed change set touches the repository's `README.md` or `compatibility.json` — both are now recognized as known review-subject paths (documentation and governance respectively) instead of blocking the seal.
