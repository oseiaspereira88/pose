---
spec: review-check-names-a-negative-verdict
category: fixed
breaking: false
---

`pose review-check` and `pose check --strict` now report the review that blocks a closed scope when its newest review rejected it or requested changes: the attestation, its decision and each open finding. They used to say `no review attempt exists`, while `pose review verify` showed the findings.
