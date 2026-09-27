---
spec: pose-check-dor-accepts-qualified-refs
category: fixed
breaking: false
refs:
---

`pose check` no longer reports "transition to in-progress without Definition of Ready" for a spec whose `depends_on` holds a qualified `xref:` reference that `pose lint-spec --ready-check` accepts. Both gates now read dependencies with the same parser, and a malformed reference still fails both.
