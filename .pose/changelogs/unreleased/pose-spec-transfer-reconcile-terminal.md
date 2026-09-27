---
spec: pose-spec-transfer-reconcile-terminal
category: added
breaking: false
refs:
---

`pose spec-transfer preview --mode reconcile-terminal` retires a coordinator spec onto an executor in another project that is already `done` with a terminal closeout. The executor is never written: apply and resume only verify its digest. The requirement map is N:M over every source requirement, may target another spec through `destination_ref`, and records a `rationale` for `withdrawn` and `pending`; an owed requirement must land in an open spec and is kept in the redirect as `open_obligations`. `--map-file` reads the map as JSON. Plans and redirects of this mode use schema version 2, which older engines refuse. Ordinary transfers are unchanged.
