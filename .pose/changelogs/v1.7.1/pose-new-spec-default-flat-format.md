---
spec: pose-new-spec-default-flat-format
category: fixed
breaking: false
refs: ISSUE#33
---

Resolved issue #33: Changed `pose new-spec` default layout to modern date-prefixed flat file `.pose/specs/YYYY-MM-DD-<slug>.md` so that newly scaffolded specifications immediately conform (`pose spec-format status` reports `conforming: true`). Added `--folder` flag for creating date-prefixed folder envelope when specs include amendments, and enhanced `pose spec-format migrate` to convert dated folders to flat files while preserving folders with amendments.
