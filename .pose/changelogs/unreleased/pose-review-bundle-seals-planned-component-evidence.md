---
spec: pose-review-bundle-seals-planned-component-evidence
category: fixed
breaking: false
refs:
---

A sealed review bundle now carries the current validation evidence of every module its plan asks a per-component `validate` tool for, not only the modules of the spec's delivery targets. A spec that delivers in one module and changes others can cite each component's own evidence, instead of being blocked with "cites evidence from" a sibling. Scopes without delivery targets seal as before.
