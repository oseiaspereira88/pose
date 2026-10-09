---
spec: pose-attention-within-a-second
category: changed
breaking: false
refs:
---

`pose state --attention` reads each spec and each review bundle's scope once per answer instead of once per spec: about 1.3 s on a large instance where the agency-readiness pilot measured 30 to 37 s, with the same obligations.
