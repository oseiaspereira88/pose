---
spec: pose-diagnose-invisible-governance-failures
category: added
breaking: false
refs:
---

`pose doctor` now reports two conditions that used to surface only as a
downstream symptom: a review profile demanding an evidence class no registered
check may emit, and a check declaring no evidence class at all, whose results
are discarded when a review collects evidence. `pose artifact-check` also prints
the resolved change set's base, head and commit count in its default output, so
an inflated change set can be diagnosed without re-running with `--json`.
