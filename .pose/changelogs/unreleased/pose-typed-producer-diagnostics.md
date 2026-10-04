---
spec: pose-typed-producer-diagnostics
category: added
breaking: false
refs:
---

Pending-state producers now emit typed diagnostics alongside their prose. `closeout-check --json` and `pose_closeout_state` carry `diagnostics` (code, domain, refs and satisfaction condition, with text blockers the engine cannot type yet marked `opaque`) and a typed `next_step`; readiness `waiting_on` entries carry a `code`; auto-attest pendencies carry `code`, `source` and `condition`; start previews list owed review criteria as typed items. Codes come from one catalog and do not change with wording or locale.
