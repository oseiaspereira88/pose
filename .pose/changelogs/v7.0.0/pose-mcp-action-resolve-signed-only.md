---
spec: pose-mcp-action-resolve-signed-only
category: added
breaking: false
refs:
---

New MCP tool `pose_action_resolve` (risk class `governance-write`) records an answer to an action request only with the principal's proof: an SSH signature by a key registered to it (`pose identity add`) or a trusted issuer's claim. Its preview returns the exact statement to sign and the `ssh-keygen -Y sign -n pose-action-answer` command, so an agent can relay a person's answer without being able to forge it; a call without a proof is refused under any identity assurance.
