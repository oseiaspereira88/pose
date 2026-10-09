---
spec: pose-attention-projects-every-source
category: added
breaking: false
refs:
---

`pose state --attention` and `pose_obligations` now project four sources they listed as not yet projected: open docs review marks, capability mechanisms with stale triggers, releases in flight past the newest verified one, and legacy review findings past their review date. A project that uses them reads complete coverage and sees what they owe. Obligations of this kind originate on the project itself (`xref:<project>/project:<project>`).
