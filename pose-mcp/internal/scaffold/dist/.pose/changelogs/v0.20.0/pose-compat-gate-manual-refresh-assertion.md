---
spec: pose-compat-gate-manual-refresh-assertion
category: fixed
breaking: false
refs:
---

The release compatibility gate no longer fails an upgrade whose managed manual
legitimately changed. It asserted that `AGENTS.md` was byte-identical before and
after, which contradicted the upgrade contract adopted in 0.18.0 — canonical
manual content is supposed to reach installed repositories. The check now
requires what the contract actually promises: whatever the instance wrote
survives, in the manual or in its `.pose-backup`.
