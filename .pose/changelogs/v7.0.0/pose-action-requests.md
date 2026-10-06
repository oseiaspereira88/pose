---
spec: pose-action-requests
category: added
breaking: false
refs:
---

POSE can record a material request to a person or an external system — a decision, approval, input, external operation or acceptance — with `pose action open`: question, options and consequences, recommendation, recipient principal or role (or `unassigned`), qualified targets, per-phase effects and an optional subject. Each request is an append-only journal under `.pose/actions/` that survives sessions with its id and request digest, appears in `pose state --attention` and `pose_obligations` as an actor obligation restricting only the phases it names, and is readable over MCP with `pose_action_requests`. The materiality criterion is part of the contract: an ordinary question is not a request.
