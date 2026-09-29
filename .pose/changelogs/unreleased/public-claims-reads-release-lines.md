---
spec: public-claims-reads-release-lines
category: fixed
breaking: false
---

`pose public-claims` reads release-line claims such as `POSE 6.x` and compares them with the released major. The docs site said "Applies to: POSE 5.x (current stable)" on every page through the 6.0 releases, and the gate passed because its prose pattern needed a minor digit. The pages now state 6.x, and each one is a declared surface.
