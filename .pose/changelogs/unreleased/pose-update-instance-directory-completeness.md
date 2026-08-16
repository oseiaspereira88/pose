---
spec: pose-update-instance-directory-completeness
category: fixed
breaking: false
refs:
---

`pose update` without `--force` now creates every directory the instance contract requires (`instanceDirs`), not a hand-picked subset of 4 — previously a plain update could report `Result: SUCCESS` on an old instance while `.pose/assessments` (among others) stayed missing, and the instance's own very next `pose check --strict` failed with a broken reference.
