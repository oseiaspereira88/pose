---
spec: pose-adopt-request-keeps-the-reason
category: fixed
breaking: false
refs:
---

A capability declined or deferred through a configuration-review answer is now recorded only with a reason: the one given in `pose action resolve … --reason`, or `pose adopt --request <id> --reason <text>`. It used to be recorded as "answered decline in act-…", so the reason a project gave was lost.
