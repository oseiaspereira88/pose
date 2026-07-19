---
spec: pose-dora-adoption-metrics
category: added
breaking: false
refs:
---

New `pose record-deployment` / `pose record-incident` commands ingest
explicit delivery events (never inferred from commits), and
`pose dora-metrics` computes the five current DORA metrics from them —
each reporting `unavailable` with a reason rather than a fabricated zero
when there's no real data for that metric's denominator. `pose
adoption-metrics` reports activation, time-to-first-gate, retention and
task success derived entirely from data POSE already owns (specs,
workflow history) — no new events needed. Every metric is a
team/application aggregate; the event schema has no identity field, so
individual ranking is structurally impossible. `pose events-housekeeping`
handles retention/deletion of stored events.
