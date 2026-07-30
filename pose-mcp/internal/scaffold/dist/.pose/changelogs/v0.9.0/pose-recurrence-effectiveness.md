---
spec: pose-recurrence-effectiveness
category: added
breaking: false
refs:
---

The recurrence loop now measures itself: escalations register their
intervention and observation window, and `pose recurrence-effect` compares
failure rates — and optional duration/cost telemetry from `pose report` —
before and after, with data-quality warnings for sparse samples or incomplete
windows. An ineffective intervention demands a governed follow-up instead of
silent acceptance, and `--fail-ineffective` can make that blocking by policy.
