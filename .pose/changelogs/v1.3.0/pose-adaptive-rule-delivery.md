---
spec: pose-adaptive-rule-delivery
category: added
breaking: false
refs: ISSUE#21, ISSUE#24
---

`pose doctor` now recommends installing the domain rule extension matching a detected module's stack (Go → `pose-rule-backend-go`, a Node module with an actual React dependency → `pose-rule-frontend-react`) when it isn't installed yet — advisory only, and a plain Node.js backend is never recommended the React rule.
