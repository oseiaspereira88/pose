---
category: fixed
breaking: true
---

The first amendment now preserves recorded atomic-start nodes. In-progress lint warns about pending history changes, and close refuses opted-in histories that done lint would reject, including malformed logs. Existing journals stay append-only and no history is enabled automatically. Update shared writers together so amendment append and close use the same lifecycle lock.
