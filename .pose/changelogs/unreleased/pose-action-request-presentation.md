---
spec: pose-action-request-presentation
category: added
breaking: false
refs:
---

`pose action list --present` (and `pose_action_requests` with `present: true`) groups the requests still waiting for an answer for one conversation: by origin and the earliest phase they restrict, with what blocks start or execution first and release-only questions marked as able to wait. Every request keeps its own id, digest and answer, the person sees exactly what the answer binds to, and settled requests are not asked again.
