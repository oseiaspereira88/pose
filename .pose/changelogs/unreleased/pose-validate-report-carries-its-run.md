---
spec: pose-validate-report-carries-its-run
category: fixed
breaking: false
refs:
---

`pose validate --report` records the run it made: the report lists the commands
executed and the Result line, and derives its outcome from them.

It never had. The report looked for a `pose-validate.latest.log` that native
validation does not write, so every report said `_Fill manually_` and `_No
validation output detected_` beside an outcome recorded as manual — a pass
nothing in the report supported. The run's output is now handed to the report
directly; no log file is written to the tree.
