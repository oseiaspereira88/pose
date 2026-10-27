---
spec: pose-delegation-mcp
category: added
breaking: false
---

Add the MCP tools `pose_delegate_start`, `pose_delegate_status`, `pose_delegate_wait`, `pose_delegate_resume` and `pose_delegate_handoff`. They run `pose delegate … --json` in the resolved project, so plans, refusals and runs are the CLI's: starts, resumes and handoffs are detached, a status never blocks, a wait is bounded (300 seconds at most) and returns `still-running` when it passes. Refusals carry a `reason_code` shared with the CLI's error documents; results carry the normalized summary only. `pose delegate resume` and `handoff` take `--detach`.
