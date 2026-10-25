---
spec: pose-closeout-terminal-verification
category: fixed
breaking: true
refs:
---

A done lifecycle no longer certifies terminal closeout or fabricated execution steps. Plans, CLI and MCP disclose the applicable verification basis and historical limits; invalid proof/history remains blocked without reopening or rewriting records. Repeated close/apply/resume verifies a done scope without modifying its checkpoint, lifecycle or review. Consumers must handle the new `not-run` plan step state.
