---
spec: pose-transfer-preserves-obligations
category: changed
breaking: false
refs:
---

Spec transfer now carries action requests safely: the preview lists the source spec's requests with their disposition, retiring the source invalidates every open or answered one exactly once (a resumed transfer records nothing twice), a request opened after the preview makes the plan stale, and nothing answered in the source project authorizes or restricts the destination.
