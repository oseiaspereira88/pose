---
spec: pose-v6-2-0-release-stability
category: fixed
breaking: false
refs: ISSUE#121, ISSUE#122
---

Contributor-mode manuals agree that agents request confirmation before staging
local feedback and again before submitting it upstream. Review bundles can seal
a root `.env.example` with its content intact. Package-channel checks inspect
the installed binary in a fresh project instead of the source checkout.
