---
spec: pose-capability-catalog
category: added
breaking: false
refs:
---

Every adoptable capability is in one catalog, and `pose adopt` decides each: review-policy capabilities, criterion reuse, the Definition of Ready, the review overlays (`overlay:<profile>`), qualified references and spec authority transfer (raising `schema_version` as their contract requires), signed attestations and verified identity. `pose adopt --list` shows each with its state (on, off, declined, deferred, needs-setup), its day-to-day effect, its introducing version and whether a new instance gets it. Missing requirements and prerequisites are refused with what is missing; `--decline` and `--defer` record the decision in `.pose/policy/adoption-decisions.json` so it is not asked again. A new instance now also adopts the Definition of Ready.
