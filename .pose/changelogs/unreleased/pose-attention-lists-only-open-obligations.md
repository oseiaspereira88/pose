---
spec: pose-attention-lists-only-open-obligations
category: fixed
breaking: false
refs:
---

Attention no longer lists an answered, waived or cancelled obligation under `for_actor` or `gates`. An answered action request used to keep appearing as something waiting on the person even after it stopped restricting any phase; it now stays only in the obligation report, and reappears if its answer is invalidated.
