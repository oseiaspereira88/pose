---
spec: pose-mcp-project-scope-contract
category: changed
breaking: false
refs:
---

Every `pose_*` MCP tool now advertises the same `project_id` schema (all 20,
up from 11). Unknown and ambiguous project selection return distinct
structured errors (`structuredContent.error_code`) instead of opaque text,
never leaking the resolved filesystem root. The new opt-in
`POSE_MCP_STRICT_PROJECT_SELECTION` flag fails closed on implicit project
selection once a deployment has onboarded more than one project;
single-project deployments are provably unaffected.
