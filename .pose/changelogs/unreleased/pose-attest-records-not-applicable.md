---
spec: pose-attest-records-not-applicable
category: added
breaking: false
refs:
---

`pose review attest` accepts `--criterion ID|disposition|evidence|rationale`,
so a reviewer can record the dispositions the engine already models. Every
required criterion used to become `passed` with evidence picked from the
supplied refs, which was harmless while nothing checked that evidence and a
dead end once a `passed` must cite sealed evidence of a demanded class: a
criterion that genuinely does not apply could only be claimed as passed on
evidence that does not support it, or left blocking the closeout.

`not-applicable` without a rationale is refused at the flag rather than only at
sealing, and a criterion nobody names keeps the current default.
