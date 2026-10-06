---
spec: pose-followup-dispositions-accept-qualified-refs
category: added
breaking: false
refs:
---

A `covered`, `duplicate` or `spawned` follow-up can name a spec in another project as `xref:<project>/spec:<slug>`. `lint-spec` verifies it against the other project when that project is bound in `POSE_PROJECT_ROOTS`, accepts it when it is not, and refuses a malformed target or one that names a non-spec artifact.
