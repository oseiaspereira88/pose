---
spec: pose-git-and-dispatch-snapshot-integrity
category: changed
breaking: false
---

Closeout planning and delegated dispatch refuse unavailable Git observations and
hidden edits; directory prefixes no longer authorize arbitrary derived writes.
Dispatch checks sealed inputs and origin freshness before and after execution,
retaining stale runs as failed history. MCP `pose_closeout_state` accepts optional
`include_plan: true` for the same spec execution plan as the CLI.
