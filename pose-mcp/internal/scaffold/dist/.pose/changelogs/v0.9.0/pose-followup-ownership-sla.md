---
spec: pose-followup-ownership-sla
category: added
breaking: false
refs:
---

Open follow-ups now carry an owner, criticality and next-review date —
`(owner:@alias crit:high review:2026-10-01)` — parsed and enforced at
closeout. `pose followups` reports overdue and unowned counts, filters by
owner or expired review, and the opt-in `--fail-overdue` flag turns expired
triage promises into a blocking policy gate.
