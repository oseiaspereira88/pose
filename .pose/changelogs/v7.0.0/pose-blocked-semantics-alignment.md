---
spec: pose-blocked-semantics-alignment
category: changed
breaking: false
refs:
---

`blocked` now means one thing everywhere: a non-terminal operational condition, never a delivery outcome. Readiness reports `terminal` and, for a blocked spec, `cause: dependency` with its unmet prerequisites or `cause: unknown`; `pose state` lists blocked specs with that cause; `pose lint-spec` warns when a blocked spec declares no prerequisite. `pose adoption-metrics` keeps the v1 success ratio unchanged and adds a labelled `task_success_ratio_v2` that reads flat and folder specs alike and keeps blocked out of resolved outcomes. No command rewrites a legacy blocked spec, and spec transfer keeps using the status as its staging guard.
