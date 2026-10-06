---
spec: pose-atomic-start-adoption-cutoff
category: added
breaking: false
refs:
---

`atomic_start_adopted_at: YYYY-MM-DD` limits atomic start to specs created on or after the date. An older spec already in progress without a start is reported as `legacy-unbaselined` with a note and blocks nothing; without the date every such spec still needs reconciliation, and a malformed date is refused.
