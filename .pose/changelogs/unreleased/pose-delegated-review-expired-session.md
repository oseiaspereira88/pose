---
spec: pose-delegated-review-expired-session
category: changed
breaking: false
---
A review round whose attempt failed while continuing a session opens a new session on its next attempt, with the reason recorded, instead of resuming the same dead session.
