---
spec: pose-progressive-spec-surface
category: added
breaking: false
refs:
---

Small changes can use a smaller spec: `pose new-spec <slug> --surface minimal` keeps intent, requirements, artifacts, the requirement trace and follow-ups, and drops the planner-local Tasks section and the report sections that `pose specs facts <slug>` now derives from the delivery index (paths changed and checks run). `lint-spec` honours the surface for Tasks only; every lifecycle gate stays as strict, and derived facts never include intent, rationale or accepted risk.
