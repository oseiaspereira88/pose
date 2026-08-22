---
spec: pose-review-overlay-uninstalled-rules-graceful-degradation
category: fixed
breaking: false
---

Fixed review bundle preparation and sealing crash when configured overlay profiles reference rule IDs provided by uninstalled extensions (`backend-go`, `frontend-react`). Missing extension rules now degrade gracefully to plan warnings instead of hard blockers.
