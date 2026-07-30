---
spec: pose-mcp-protocol-completeness
category: added
breaking: false
refs:
---

`pose_list_specs`, `pose_list_roadmaps`, `pose_list_knowledge` and
`pose_list_reports` now support opaque, versioned cursor pagination
(`cursor`/`limit` in, `next_cursor` out) — fully additive, so omitting both
arguments returns the exact response shape as before. The MCP tool catalog's
stability within a server process is now verified rather than assumed;
resources and prompts remain deliberately unimplemented (see the MCP server
docs) since the existing typed tools already serve their governance use case
more safely.
