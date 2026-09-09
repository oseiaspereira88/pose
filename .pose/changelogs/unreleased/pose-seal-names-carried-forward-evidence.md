---
spec: pose-seal-names-carried-forward-evidence
category: added
breaking: false
refs:
---

Sealing a review bundle now names the evidence that ran against a commit other
than the head the bundle approves. The engine already accepts such a result as
current — by provenance digest, or because the scope is closed — and never said
so, leaving a reviewer unable to tell a result produced against this subject
from one carried forward.

It is a warning, not a blocker: the engine's decision about currency does not
change, and blocking would refuse the closed-scope case the bridge exists for.
