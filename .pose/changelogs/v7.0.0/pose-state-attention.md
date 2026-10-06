---
spec: pose-state-attention
category: added
breaking: false
refs:
---

`pose state --attention [--scope <ref>] [--actor <id|role>] [--phase <p>] [--kind <c>] [--json]` answers what is still owed, by whom and restricting which phase: coverage first (an answer is `INCOMPLETE` while any producer failed or is not integrated, so an empty group never reads as nothing owed), then what needs the actor or any person or role, then what restricts start, execution, review, closeout and release, then advisory residual debt. The new MCP tool `pose_obligations` returns the same obligation ids from the same domain function. Attention is not a gate.
