---
spec: pose-review-scope-trailer-check
category: added
breaking: false
refs: ISSUE#17
---

`pose doctor` gained a new check, `review.scope-change-set`: it warns when a spec with `delivers:` populated has no change set recorded in `.pose/reports/history/` (via `pose report --change-from`/`--change-to`), so `pose review bundle --seal` is caught failing with "no immutable attributed change set exists" before it happens, not after — and points directly at the `pose report` command that resolves it.
